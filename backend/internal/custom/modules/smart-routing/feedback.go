package smartrouting

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type healthState struct {
	Failures                               int
	Completed                              int
	Window, OpenUntil, ProbeUntil, Updated time.Time
	Recovering                             bool
	Successes                              int
	Pending                                int
	Synced                                 bool
}

type routeFeedback struct {
	mu          sync.Mutex
	health      map[string]healthState
	queues      [8][]service.SmartRoutingAttempt
	workers     [8]bool
	lastCleanup time.Time
}

func (m *Module) RoutingAccountMetrics(ctx context.Context, groupID int64, model, endpoint string, stream bool) map[int64]service.SmartRoutingAccountMetric {
	if m.repo == nil || m.repo.db == nil {
		return nil
	}
	readCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	rows, err := m.repo.db.QueryContext(readCtx, `SELECT account_id,
        COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY speed_ms) FILTER (WHERE success AND speed_ms>0),0),
        COUNT(*) FILTER (WHERE NOT success)::float / GREATEST(COUNT(*),1),
		(COUNT(*) FILTER (WHERE NOT success AND created_at >= NOW()-INTERVAL '30 seconds') >= 2
		 AND COUNT(*) FILTER (WHERE NOT success AND created_at >= NOW()-INTERVAL '30 seconds')*2
		 >= COUNT(*) FILTER (WHERE created_at >= NOW()-INTERVAL '30 seconds'))
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
		if rows.Scan(&id, &metric.SpeedMs, &metric.ErrorRate, &metric.Blocked) != nil {
			return nil
		}
		out[id] = metric
	}
	if rows.Err() != nil {
		return nil
	}
	return out
}

func healthKey(groupID int64, model, endpoint string, stream bool) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%t", groupID, model, endpoint, stream)))
	return fmt.Sprintf("custom:route:health:%x", digest)
}

func (m *Module) ObserveRoutingAttempt(a service.SmartRoutingAttempt) {
	key := healthKey(a.GroupID, a.Model, a.Endpoint, a.Stream)
	f := &m.feedback
	f.mu.Lock()
	if f.health == nil {
		f.health = make(map[string]healthState)
	}
	if len(f.health) >= 4096 {
		for k, h := range f.health {
			if h.Pending == 0 && a.CreatedAt.Sub(h.Updated) > 24*time.Hour {
				delete(f.health, k)
			}
		}
	}
	h := f.health[key]
	if a.CreatedAt.Sub(h.Window) > time.Minute {
		h.Failures = 0
		h.Completed = 0
		h.Window = a.CreatedAt
	}
	h.Completed++
	h.Updated = a.CreatedAt
	h.Synced = false
	if a.Success {
		if h.Recovering && a.RecoveryProbe && !a.CreatedAt.Before(h.OpenUntil) {
			h.Successes++
			if h.Successes >= 2 {
				h = healthState{Updated: a.CreatedAt, Pending: h.Pending}
			}
		}
	} else {
		h.Failures++
		h.Successes = 0
		if (h.Failures >= 3 && h.Failures*2 >= h.Completed) || h.Recovering {
			h.Recovering = true
			h.OpenUntil = a.CreatedAt.Add(30 * time.Second)
		}
	}
	// Serialize feedback for each health key, with bounded buffering rather than
	// dropping every observation whenever the database has eight calls in flight.
	digest := sha256.Sum256([]byte(key))
	shard := int(digest[0]) % len(f.queues)
	queued := len(f.queues[shard]) < 512
	if queued {
		h.Pending++
		f.queues[shard] = append(f.queues[shard], a)
	}
	if _, exists := f.health[key]; exists || len(f.health) < 4096 {
		f.health[key] = h
	}
	if queued && !f.workers[shard] {
		f.workers[shard] = true
		go m.flushFeedback(shard)
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
		key := healthKey(a.GroupID, a.Model, a.Endpoint, a.Stream)
		synced := false
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if m.redis != nil {
				_, err := m.redis.Eval(ctx, healthFeedbackScript, []string{key, key + ":failures", key + ":recovering", key + ":success", key + ":total"}, a.Success, a.RecoveryProbe).Result()
				synced = err == nil
				if err != nil {
					slog.Warn("smart_routing_health_write_failed", "error", err)
				}
			}
			if m.repo != nil && m.repo.db != nil && a.RequestID != "" {
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
			}
		}()
		f.mu.Lock()
		if h, exists := f.health[key]; exists {
			h.Pending--
			h.Synced = synced && h.Pending == 0
			f.health[key] = h
		}
		f.mu.Unlock()
	}
}

// A success does not cancel an open circuit. Recovery requires two successful
// half-open probes after the cooldown, preventing one lucky call from flapping.
const healthFeedbackScript = `
if redis.call('EXISTS', KEYS[5]) == 0 then redis.call('DEL', KEYS[2]) end
local total = redis.call('INCR', KEYS[5]); if total == 1 then redis.call('EXPIRE', KEYS[5], 60) end
if ARGV[1] == '1' then
  if ARGV[2] == '1' and redis.call('EXISTS', KEYS[3]) == 1 and redis.call('EXISTS', KEYS[1]) == 0 then
    local n = redis.call('INCR', KEYS[4]); redis.call('EXPIRE', KEYS[4], 86400)
    if n >= 2 then redis.call('DEL', KEYS[2], KEYS[3], KEYS[4], KEYS[5]) end
  end
else
  redis.call('DEL', KEYS[4])
  local n = redis.call('INCR', KEYS[2]); if n == 1 then redis.call('EXPIRE', KEYS[2], 60) end
  if (n >= 3 and n*2 >= total) or redis.call('EXISTS', KEYS[3]) == 1 then
    redis.call('SET', KEYS[1], '1', 'EX', 30)
    redis.call('SET', KEYS[3], '1', 'EX', 86400)
  end
end
return 1`

func (m *Module) routeHealth(ctx context.Context, groupID int64, model string) (blocked, recovering bool) {
	s := service.SmartRoutingStateFromContext(ctx)
	if s == nil {
		return false, false
	}
	key := healthKey(groupID, model, s.EndpointType, s.Stream)
	m.feedback.mu.Lock()
	h := m.feedback.health[key]
	m.feedback.mu.Unlock()
	blocked, recovering = time.Now().Before(h.OpenUntil), h.Recovering && time.Since(h.Updated) < 24*time.Hour
	if m.redis != nil {
		readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		values, err := m.redis.MGet(readCtx, key, key+":recovering").Result()
		if err == nil && len(values) == 2 {
			if h.Synced || h.Updated.IsZero() {
				blocked, recovering = values[0] != nil, values[1] != nil
			} else {
				blocked = blocked || values[0] != nil
				recovering = recovering || values[1] != nil
			}
		}
	}
	return
}

func (m *Module) acquireRecoveryProbe(ctx context.Context, groupID int64, model string) bool {
	s := service.SmartRoutingStateFromContext(ctx)
	if s == nil {
		return true
	}
	key := healthKey(groupID, model, s.EndpointType, s.Stream)
	if m.redis != nil {
		probeCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		ok, err := m.redis.SetNX(probeCtx, key+":probe", "1", 30*time.Second).Result()
		return err == nil && ok
	}
	m.feedback.mu.Lock()
	defer m.feedback.mu.Unlock()
	h := m.feedback.health[key]
	if time.Now().Before(h.ProbeUntil) {
		return false
	}
	h.ProbeUntil = time.Now().Add(30 * time.Second)
	if m.feedback.health == nil {
		m.feedback.health = map[string]healthState{}
	}
	m.feedback.health[key] = h
	return true
}
