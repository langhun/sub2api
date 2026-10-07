package smartrouting

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type routeFeedback struct {
	mu                 sync.Mutex
	accounts           map[string]accountCircuit
	queues             [8][]service.SmartRoutingAttempt
	workers            [8]bool
	lastCleanup        time.Time
	outcomeQueues      [8][]service.SmartRoutingOutcome
	outcomeWorkers     [8]bool
	outcomeLastCleanup time.Time
	auditQueues        [8][]routeAuditEvent
	auditWorkers       [8]bool
	auditDroppedCount  atomic.Int64
}

const routeAuditQueueCapacity = 1024

type routeAuditEventKind uint8

const (
	routeAuditDecision routeAuditEventKind = iota + 1
	routeAuditSelectedAccount
	routeAuditNoRoute
)

// routeAuditEvent is intentionally self-contained. Events are copied out of
// request state before enqueueing so the bounded writer never retains mutable
// request objects or a cancelled request context.
type routeAuditEvent struct {
	kind            routeAuditEventKind
	requestID       string
	apiKeyID        int64
	profileID       int64
	requestedModel  string
	endpointType    string
	selectedGroupID int64
	selectedAccount int64
	strategy        string
	scoreSnapshot   []byte
	fallbackDepth   int
	decisionReason  string
}

func (m *Module) RoutingAccountMetrics(ctx context.Context, groupID int64, model, endpoint string, stream bool) map[int64]service.SmartRoutingAccountMetric {
	if m.repo == nil || m.repo.db == nil {
		return nil
	}
	readCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	rows, err := m.repo.db.QueryContext(readCtx, `SELECT account_id,
		COUNT(*) FILTER (WHERE created_at>=NOW()-INTERVAL '15 minutes'),
		COUNT(*) FILTER (WHERE success AND account_speed_ms>0 AND created_at>=NOW()-INTERVAL '15 minutes'),
		COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY account_speed_ms) FILTER (WHERE success AND account_speed_ms>0 AND created_at>=NOW()-INTERVAL '15 minutes'),0),
		COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY account_speed_ms) FILTER (WHERE success AND account_speed_ms>0 AND created_at>=NOW()-INTERVAL '15 minutes'),0),
		COUNT(*) FILTER (WHERE NOT success AND created_at>=NOW()-INTERVAL '15 minutes')::float / GREATEST(COUNT(*) FILTER (WHERE created_at>=NOW()-INTERVAL '15 minutes'),1),
		COUNT(*),
		COUNT(*) FILTER (WHERE success AND account_speed_ms>0),
		COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY account_speed_ms) FILTER (WHERE success AND account_speed_ms>0),0),
		COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY account_speed_ms) FILTER (WHERE success AND account_speed_ms>0),0),
		COUNT(*) FILTER (WHERE NOT success)::float / GREATEST(COUNT(*),1)
        FROM (
			SELECT DISTINCT ON (request_id, group_id, account_id) account_id, success, account_speed_ms, created_at
            FROM custom_route_attempts
            WHERE group_id=$1 AND requested_model=$2 AND endpoint_type=$3 AND stream=$4
			  AND created_at>=NOW()-INTERVAL '1 hour'
            ORDER BY request_id, group_id, account_id, attempt DESC
        ) final_attempts GROUP BY account_id`, groupID, model, endpoint, stream)
	if err != nil {
		slog.Warn("smart_routing_account_metrics_failed", "error", err)
		return nil
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]service.SmartRoutingAccountMetric{}
	for rows.Next() {
		var id int64
		var recentSamples, recentSpeedSamples, fallbackSamples, fallbackSpeedSamples int64
		var recentSpeed, recentP90, recentError, fallbackSpeed, fallbackP90, fallbackError float64
		if rows.Scan(&id, &recentSamples, &recentSpeedSamples, &recentSpeed, &recentP90, &recentError, &fallbackSamples, &fallbackSpeedSamples, &fallbackSpeed, &fallbackP90, &fallbackError) != nil {
			return nil
		}
		metric := service.SmartRoutingAccountMetric{Samples: recentSamples, SpeedSamples: recentSpeedSamples, SpeedMs: recentSpeed, SpeedP90Ms: recentP90, ErrorRate: recentError}
		if recentSamples < minSamples && fallbackSamples >= minSamples {
			metric.Samples, metric.ErrorRate = fallbackSamples, fallbackError
		}
		if recentSpeedSamples < minSamples && fallbackSpeedSamples >= minSamples {
			metric.SpeedSamples, metric.SpeedMs, metric.SpeedP90Ms = fallbackSpeedSamples, fallbackSpeed, fallbackP90
		}
		out[id] = metric
	}
	if rows.Err() != nil {
		return nil
	}
	return out
}

func (m *Module) ObserveRoutingAttempt(a service.SmartRoutingAttempt) {
	// Pool-mode attempts become website-level health failures only after the
	// handler confirms that the website's internal retry budget is exhausted.
	if a.HealthFinal {
		m.observeAccountHealth(a)
	}
	if a.HealthOnly {
		return
	}
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

func (m *Module) RecordRoutingOutcome(a service.SmartRoutingOutcome) {
	if m == nil || m.repo == nil || m.repo.db == nil || a.RequestID == "" || a.APIKeyID <= 0 {
		return
	}
	f := &m.feedback
	f.mu.Lock()
	digest := sha256.Sum256([]byte(a.RequestID))
	shard := int(digest[0]) % len(f.outcomeQueues)
	queued := len(f.outcomeQueues[shard]) < 512
	if queued {
		f.outcomeQueues[shard] = append(f.outcomeQueues[shard], a)
		if !f.outcomeWorkers[shard] {
			f.outcomeWorkers[shard] = true
			go m.flushOutcomes(shard)
		}
	}
	f.mu.Unlock()
	if !queued {
		slog.Warn("smart_routing_outcome_queue_full")
	}
}

func (m *Module) flushOutcomes(shard int) {
	f := &m.feedback
	for {
		f.mu.Lock()
		if len(f.outcomeQueues[shard]) == 0 {
			f.outcomeWorkers[shard] = false
			f.mu.Unlock()
			return
		}
		a := f.outcomeQueues[shard][0]
		f.outcomeQueues[shard][0] = service.SmartRoutingOutcome{}
		f.outcomeQueues[shard] = f.outcomeQueues[shard][1:]
		cleanup := time.Since(f.outcomeLastCleanup) > time.Minute
		if cleanup {
			f.outcomeLastCleanup = time.Now()
		}
		f.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := m.repo.db.ExecContext(ctx, `INSERT INTO custom_route_outcomes
            (request_id,api_key_id,final_group_id,final_account_id,final_success,attempts,group_switches,duration_ms,first_token_ms,error_status)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
            ON CONFLICT (request_id) DO UPDATE SET final_group_id=EXCLUDED.final_group_id,
            final_account_id=EXCLUDED.final_account_id, final_success=EXCLUDED.final_success,
            attempts=EXCLUDED.attempts, group_switches=EXCLUDED.group_switches,
            duration_ms=EXCLUDED.duration_ms, first_token_ms=EXCLUDED.first_token_ms,
            error_status=EXCLUDED.error_status`, a.RequestID, a.APIKeyID, nullableID(a.FinalGroupID), nullableID(a.FinalAccountID), a.FinalSuccess, a.Attempts, a.GroupSwitches, a.DurationMs, a.FirstTokenMs, a.ErrorStatus)
		if err != nil {
			slog.Warn("smart_routing_outcome_write_failed", "request_id", a.RequestID, "error", err)
		}
		// Outcome telemetry is retained for a short window, but the usage history
		// needs a durable request-level marker so old records can still explain
		// that smart routing was used after telemetry cleanup.
		if a.UsageRequestID != "" {
			if _, markerErr := m.repo.db.ExecContext(ctx, `INSERT INTO custom_usage_log_routing_marks (request_id, api_key_id)
		        VALUES ($1,$2) ON CONFLICT (request_id, api_key_id) DO NOTHING`, a.UsageRequestID, a.APIKeyID); markerErr != nil {
				slog.Warn("smart_routing_usage_marker_write_failed", "request_id", a.UsageRequestID, "error", markerErr)
			}
		}
		if cleanup {
			_, err = m.repo.db.ExecContext(ctx, `DELETE FROM custom_route_outcomes WHERE ctid IN (SELECT ctid FROM custom_route_outcomes WHERE created_at < NOW()-INTERVAL '24 hours' LIMIT 10000)`)
			if err != nil {
				slog.Warn("smart_routing_outcome_cleanup_failed", "error", err)
			}
		}
		cancel()
	}
}

func nullableID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
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
                (request_id,attempt,api_key_id,group_id,account_id,requested_model,endpoint_type,stream,success,speed_ms,account_speed_ms,created_at)
                VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT DO NOTHING`, a.RequestID, a.Attempt, a.APIKeyID, a.GroupID, a.AccountID, a.Model, a.Endpoint, a.Stream, a.Success, a.SpeedMs, a.AccountSpeedMs, a.CreatedAt)
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
