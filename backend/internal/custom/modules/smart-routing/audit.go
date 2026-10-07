package smartrouting

import (
	"context"
	"database/sql"
	"encoding/json"
	"hash/fnv"
	"log/slog"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func requestIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	requestID, ok := ctx.Value(ctxkey.RequestID).(string)
	return requestID, ok && requestID != ""
}

// enqueueAudit is a bounded, best-effort telemetry queue. It starts at most
// one worker per shard and drops telemetry under sustained overload rather
// than allowing request traffic to create an unbounded number of goroutines.
func (m *Module) enqueueAudit(event routeAuditEvent) {
	if m == nil || m.repo == nil || m.repo.db == nil || event.requestID == "" {
		return
	}
	shard := auditShard(event.requestID)
	f := &m.feedback
	f.mu.Lock()
	if len(f.auditQueues[shard]) >= routeAuditQueueCapacity {
		f.mu.Unlock()
		dropped := f.auditDroppedCount.Add(1)
		slog.Warn("smart_routing_audit_queue_full", "request_id", event.requestID, "kind", event.kind, "total_dropped", dropped)
		return
	}
	// Copy the snapshot bytes so callers can safely reuse their JSON buffer.
	if event.scoreSnapshot != nil {
		event.scoreSnapshot = append([]byte(nil), event.scoreSnapshot...)
	}
	f.auditQueues[shard] = append(f.auditQueues[shard], event)
	if !f.auditWorkers[shard] {
		f.auditWorkers[shard] = true
		go m.flushAudit(shard)
	}
	f.mu.Unlock()
}

func auditShard(requestID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(requestID))
	return int(h.Sum32() % 8)
}

// startDecisionRetention owns a dedicated timer because preview and no-route
// traffic may not produce attempt/outcome events. Retention therefore cannot
// be coupled to either feedback worker's activity.
func (m *Module) startDecisionRetention() {
	if m == nil || m.repo == nil || m.repo.db == nil {
		return
	}
	m.decisionCleanup.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				m.cleanupDecisionRows()
			}
		}()
	})
}

func (m *Module) cleanupDecisionRows() {
	if m == nil || m.repo == nil || m.repo.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := m.repo.db.ExecContext(ctx, `DELETE FROM custom_route_decisions WHERE ctid IN (SELECT ctid FROM custom_route_decisions WHERE created_at < NOW()-INTERVAL '24 hours' LIMIT 10000)`); err != nil {
		slog.Warn("smart_routing_decision_cleanup_failed", "error", err)
	}
}

func (m *Module) flushAudit(shard int) {
	f := &m.feedback
	for {
		f.mu.Lock()
		if len(f.auditQueues[shard]) == 0 {
			f.auditWorkers[shard] = false
			f.mu.Unlock()
			return
		}
		event := f.auditQueues[shard][0]
		f.auditQueues[shard][0] = routeAuditEvent{}
		f.auditQueues[shard] = f.auditQueues[shard][1:]
		f.mu.Unlock()

		writeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		var err error
		switch event.kind {
		case routeAuditDecision:
			_, err = m.repo.db.ExecContext(writeCtx, `INSERT INTO custom_route_decisions (request_id,api_key_id,profile_id,requested_model,endpoint_type,selected_group_id,strategy,score_snapshot,fallback_depth,decision_reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, event.requestID, event.apiKeyID, event.profileID, event.requestedModel, event.endpointType, event.selectedGroupID, event.strategy, event.scoreSnapshot, event.fallbackDepth, event.decisionReason)
		case routeAuditSelectedAccount:
			// Events for one request are FIFO on a single shard. The newest
			// unassigned row is therefore the decision immediately preceding
			// this scheduler admission, even when the request retries a group.
			updated := false
			for attempt := 0; attempt < 5; attempt++ {
				var result sql.Result
				result, err = m.repo.db.ExecContext(writeCtx, `UPDATE custom_route_decisions SET selected_account_id=$1 WHERE id=(SELECT id FROM custom_route_decisions WHERE request_id=$2 AND api_key_id=$3 AND selected_group_id=$4 AND selected_account_id IS NULL ORDER BY created_at DESC, id DESC LIMIT 1)`, event.selectedAccount, event.requestID, event.apiKeyID, event.selectedGroupID)
				if err == nil {
					if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows > 0 {
						updated = true
						break
					}
				}
				if attempt < 4 {
					timer := time.NewTimer(20 * time.Millisecond)
					select {
					case <-timer.C:
					case <-writeCtx.Done():
						timer.Stop()
					}
				}
			}
			if !updated && err == nil {
				slog.Debug("smart_routing_selected_account_unmatched", "request_id", event.requestID, "group_id", event.selectedGroupID, "account_id", event.selectedAccount)
			}
		case routeAuditNoRoute:
			_, err = m.repo.db.ExecContext(writeCtx, `INSERT INTO custom_route_decisions (request_id,api_key_id,profile_id,requested_model,endpoint_type,selected_group_id,strategy,score_snapshot,fallback_depth,decision_reason) VALUES ($1,$2,$3,$4,$5,NULL,$6,$7,$8,'no_available_route')`, event.requestID, event.apiKeyID, event.profileID, event.requestedModel, event.endpointType, event.strategy, event.scoreSnapshot, event.fallbackDepth)
		}
		cancel()
		if err != nil {
			slog.Warn("smart_routing_audit_write_failed", "request_id", event.requestID, "kind", event.kind, "error", err)
		}
	}
}

func (m *Module) recordNoRoute(ctx context.Context, key *service.APIKey, profile *Profile, model string, candidates []Group) {
	if m.repo == nil || m.repo.db == nil || key == nil || profile == nil {
		return
	}
	requestID, ok := requestIDFromContext(ctx)
	if !ok {
		return
	}
	state := service.SmartRoutingStateFromContext(ctx)
	if state == nil {
		return
	}
	snapshot, _ := json.Marshal(map[string]any{"candidates": candidates, "continuity_locked": state.ContinuityLocked, "switches": state.GroupSwitches, "capacity_retries": state.CapacityRetries, "attempts": state.Attempt, "excluded_account_count": len(state.FailedAccountIDs), "account_exclusions": state.AccountExclusions})
	keyID, profileID, endpoint, depth := key.ID, profile.APIKeyID, state.EndpointType, state.GroupSwitches
	strategy := profile.Strategy
	m.enqueueAudit(routeAuditEvent{kind: routeAuditNoRoute, requestID: requestID, apiKeyID: keyID, profileID: profileID, requestedModel: model, endpointType: endpoint, strategy: strategy, scoreSnapshot: snapshot, fallbackDepth: depth})
}
