package smartrouting

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type routeFeedback struct {
	mu          sync.Mutex
	accounts    map[string]accountCircuit
	queues      [8][]service.SmartRoutingAttempt
	workers     [8]bool
	lastCleanup time.Time
	auditOnce   sync.Once
	auditSlots  chan struct{}
}

func (m *Module) RoutingAccountMetrics(ctx context.Context, groupID int64, model, endpoint string, stream bool) map[int64]service.SmartRoutingAccountMetric {
	if m.repo == nil || m.repo.db == nil {
		return nil
	}
	readCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	rows, err := m.repo.db.QueryContext(readCtx, `SELECT account_id,
        COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY speed_ms) FILTER (WHERE success AND speed_ms>0),0),
        COUNT(*) FILTER (WHERE NOT success)::float / GREATEST(COUNT(*),1)
        FROM custom_route_attempts WHERE group_id=$1 AND requested_model=$2 AND endpoint_type=$3 AND stream=$4
        AND created_at>=NOW()-INTERVAL '15 minutes' GROUP BY account_id`, groupID, model, endpoint, stream)
	if err != nil {
		slog.Warn("smart_routing_account_metrics_failed", "error", err)
		return nil
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]service.SmartRoutingAccountMetric{}
	for rows.Next() {
		var id int64
		var metric service.SmartRoutingAccountMetric
		if rows.Scan(&id, &metric.SpeedMs, &metric.ErrorRate) != nil {
			return nil
		}
		out[id] = metric
	}
	if rows.Err() != nil {
		return nil
	}
	return out
}

func (m *Module) ObserveRoutingAttempt(a service.SmartRoutingAttempt) {
	// Publish health before retrying; SQL telemetry must not gate isolation.
	m.observeAccountHealth(a)
	if m.repo == nil || m.repo.db == nil || a.RequestID == "" {
		return
	}
	f := &m.feedback
	f.mu.Lock()
	digest := sha256.Sum256([]byte(accountHealthKey(a.AccountID, a.Model, a.Endpoint, a.Stream)))
	shard := int(digest[0]) % len(f.queues)
	queued := len(f.queues[shard]) < 512
	if queued {
		f.queues[shard] = append(f.queues[shard], a)
		if !f.workers[shard] {
			f.workers[shard] = true
			go m.flushFeedback(shard)
		}
	}
	f.mu.Unlock()
	if !queued {
		slog.Warn("smart_routing_feedback_queue_full")
	}
}

func (m *Module) flushFeedback(shard int) {
	f := &m.feedback
	for {
		f.mu.Lock()
		if len(f.queues[shard]) == 0 {
			f.workers[shard] = false
			f.mu.Unlock()
			return
		}
		a := f.queues[shard][0]
		f.queues[shard][0] = service.SmartRoutingAttempt{}
		f.queues[shard] = f.queues[shard][1:]
		cleanup := time.Since(f.lastCleanup) > time.Minute
		if cleanup {
			f.lastCleanup = time.Now()
		}
		f.mu.Unlock()
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err := m.repo.db.ExecContext(ctx, `INSERT INTO custom_route_attempts
                (request_id,attempt,api_key_id,group_id,account_id,requested_model,endpoint_type,stream,success,speed_ms,created_at)
                VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT DO NOTHING`, a.RequestID, a.Attempt, a.APIKeyID, a.GroupID, a.AccountID, a.Model, a.Endpoint, a.Stream, a.Success, a.SpeedMs, a.CreatedAt)
			if err != nil {
				slog.Warn("smart_routing_attempt_write_failed", "error", err)
			}
			if cleanup {
				_, err = m.repo.db.ExecContext(ctx, `DELETE FROM custom_route_attempts WHERE ctid IN (SELECT ctid FROM custom_route_attempts WHERE created_at < NOW()-INTERVAL '24 hours' LIMIT 10000)`)
				if err != nil {
					slog.Warn("smart_routing_attempt_cleanup_failed", "error", err)
				}
			}
		}()
	}
}
