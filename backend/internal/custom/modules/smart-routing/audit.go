package smartrouting

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (m *Module) recordNoRoute(ctx context.Context, key *service.APIKey, profile *Profile, model string, candidates []Group) {
	if m.repo == nil || m.repo.db == nil || key == nil || profile == nil {
		return
	}
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	if requestID == "" {
		return
	}
	state := service.SmartRoutingStateFromContext(ctx)
	if state == nil {
		return
	}
	snapshot, _ := json.Marshal(map[string]any{"candidates": candidates, "continuity_locked": state.ContinuityLocked, "switches": state.GroupSwitches, "attempts": state.Attempt, "excluded_account_count": len(state.FailedAccountIDs), "account_exclusions": state.AccountExclusions})
	keyID, profileID, endpoint, depth := key.ID, profile.APIKeyID, state.EndpointType, state.GroupSwitches
	strategy := profile.Strategy
	m.feedback.auditOnce.Do(func() { m.feedback.auditSlots = make(chan struct{}, 64) })
	select {
	case m.feedback.auditSlots <- struct{}{}:
	default:
		slog.Warn("smart_routing_no_route_audit_full", "request_id", requestID)
		return
	}
	go func() {
		defer func() { <-m.feedback.auditSlots }()
		writeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := m.repo.db.ExecContext(writeCtx, `INSERT INTO custom_route_decisions (request_id,api_key_id,profile_id,requested_model,endpoint_type,selected_group_id,strategy,score_snapshot,fallback_depth,decision_reason) VALUES ($1,$2,$3,$4,$5,NULL,$6,$7,$8,'no_available_route')`, requestID, keyID, profileID, model, endpoint, strategy, snapshot, depth)
		if err != nil {
			slog.Warn("smart_routing_no_route_audit_failed", "request_id", requestID, "error", err)
		}
	}()
}
