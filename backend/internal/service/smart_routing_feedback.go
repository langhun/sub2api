package service

import (
	"context"
	"errors"
	"io"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

// SmartRoutingAttempt is immutable: the request-local key can change before
// asynchronous telemetry is persisted. Billing continues to use its own IDs.
type SmartRoutingAttempt struct {
	RequestID                    string
	APIKeyID, GroupID, AccountID int64
	Attempt                      int
	Model, Endpoint              string
	Stream, Success              bool
	RecoveryProbe                bool
	SpeedMs                      int64
	CreatedAt                    time.Time
}

type SmartRoutingAttemptObserver interface {
	ObserveRoutingAttempt(SmartRoutingAttempt)
}

type SmartRoutingAccountMetric struct {
	SpeedMs   float64
	ErrorRate float64
	Blocked   bool
}

type SmartRoutingAccountMetricProvider interface {
	RoutingAccountMetrics(context.Context, int64, string, string, bool) map[int64]SmartRoutingAccountMetric
}

func LoadSmartRoutingAccountMetrics(ctx context.Context, resolver SmartRoutingResolver, groupID *int64) {
	s := SmartRoutingStateFromContext(ctx)
	if s == nil || !s.SmartRoutingActive || s.NoRoute || groupID == nil {
		return
	}
	if provider, ok := resolver.(SmartRoutingAccountMetricProvider); ok {
		s.AccountMetrics = provider.RoutingAccountMetrics(ctx, *groupID, s.RequestedModel, s.EndpointType, s.Stream)
		for id, metric := range s.AccountMetrics {
			if metric.Blocked {
				if s.FailedAccountIDs == nil {
					s.FailedAccountIDs = map[int64]struct{}{}
				}
				s.FailedAccountIDs[id] = struct{}{}
			}
		}
	}
}

func smartRoutingOrderBySpeed(ctx context.Context, accounts []accountWithLoad) []accountWithLoad {
	if !SmartRoutingSpeedPreferred(ctx) || len(accounts) < 2 {
		return accounts
	}
	s := SmartRoutingStateFromContext(ctx)
	minSpeed, maxSpeed := 0.0, 0.0
	for _, a := range accounts {
		v := s.AccountMetrics[a.account.ID].SpeedMs
		if v > 0 && (minSpeed == 0 || v < minSpeed) {
			minSpeed = v
		}
		if v > maxSpeed {
			maxSpeed = v
		}
	}
	if maxSpeed == 0 {
		return accounts
	}
	score := func(id int64) float64 {
		m := s.AccountMetrics[id]
		speed := 0.5
		if m.SpeedMs > 0 {
			speed = 1
			if maxSpeed > minSpeed {
				speed = (maxSpeed - m.SpeedMs) / (maxSpeed - minSpeed)
			}
		}
		return speed*0.7 + (1-m.ErrorRate)*0.3
	}
	ordered := append([]accountWithLoad(nil), accounts...)
	sort.SliceStable(ordered, func(i, j int) bool { return score(ordered[i].account.ID) > score(ordered[j].account.ID) })
	return ordered
}

// Legacy admission has no load batch but must still honor a smart speed policy.
// Return undecided on equal/missing samples so its normal tie-break remains intact.
func smartRoutingPreferAccountBySpeed(ctx context.Context, account, selected *Account) (bool, bool) {
	if !SmartRoutingSpeedPreferred(ctx) || account == nil || selected == nil {
		return false, false
	}
	state := SmartRoutingStateFromContext(ctx)
	score := func(id int64) float64 {
		metric := state.AccountMetrics[id]
		speed := 0.5
		if metric.SpeedMs > 0 {
			speed = 1000 / (1000 + metric.SpeedMs)
		}
		return speed*0.7 + (1-metric.ErrorRate)*0.3
	}
	a, b := score(account.ID), score(selected.ID)
	return a > b, a != b
}

func ObserveSmartRoutingAttempt(ctx context.Context, accountID int64, stream bool, duration time.Duration, firstTokenMs *int, err error) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.Observer == nil || accountID <= 0 || ctx.Err() != nil {
		return
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		var failure *UpstreamFailoverError
		if errors.As(err, &failure) {
			if !failure.ShouldRetryNextAccount() || failure.RequestScopedTransient || failure.Scope == GatewayFailureScopeRequest || !failure.ShouldReportAccountScheduleFailure() {
				return
			}
		} else {
			var streamRead *openAIUpstreamStreamReadError
			var streamEvent *sseStreamErrorEventError
			if !errors.As(err, &streamRead) && !errors.As(err, &streamEvent) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
		}
		if state.FailedAccountIDs == nil {
			state.FailedAccountIDs = map[int64]struct{}{}
		}
		state.FailedAccountIDs[accountID] = struct{}{}
	}
	key, ok := APIKeyFromContext(ctx)
	if !ok {
		return
	}
	state.Attempt++
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	speed := duration.Milliseconds()
	if stream {
		speed = 0
		if firstTokenMs != nil {
			speed = int64(*firstTokenMs)
		}
	}
	state.Observer.ObserveRoutingAttempt(SmartRoutingAttempt{
		RequestID: requestID, APIKeyID: key.ID, GroupID: state.CurrentGroupID,
		AccountID: accountID, Attempt: state.Attempt, Model: state.RequestedModel,
		Endpoint: state.EndpointType, Stream: stream, Success: err == nil,
		RecoveryProbe: state.RecoveryProbe,
		SpeedMs:       speed, CreatedAt: time.Now(),
	})
}

func SmartRoutingSpeedPreferred(ctx context.Context) bool {
	s := SmartRoutingStateFromContext(ctx)
	return s != nil && s.SmartRoutingActive && !s.ContinuityLocked && (s.Strategy == "speed" || (s.Strategy == "custom" && s.SpeedWeight >= 50))
}

func SmartRoutingExcludedAccounts(ctx context.Context, excluded map[int64]struct{}) map[int64]struct{} {
	s := SmartRoutingStateFromContext(ctx)
	if s == nil || !s.SmartRoutingActive || len(s.FailedAccountIDs) == 0 {
		return excluded
	}
	merged := make(map[int64]struct{}, len(excluded)+len(s.FailedAccountIDs))
	for id := range excluded {
		merged[id] = struct{}{}
	}
	for id := range s.FailedAccountIDs {
		merged[id] = struct{}{}
	}
	return merged
}

// SmartRoutingResponseRoutes stores the real owner of a response chain. It is
// unrelated to optional session stickiness and must never weaken authorization.
type SmartRoutingResponseRoutes interface {
	BindRoutingResponse(context.Context, *APIKey, int64, int64, int64, string, time.Duration) error
	LookupRoutingResponse(context.Context, *APIKey, string) (int64, int64, error)
}

// RoutingResponseAccount reads the existing owner and account records together.
// An owner-only primary alias is not evidence that the primary handled a response.
func (s *OpenAIGatewayService) RoutingResponseAccount(ctx context.Context, groupID int64, responseID string, userID, keyID int64) (int64, error) {
	store := s.getOpenAIWSStateStore()
	ownerUser, ownerKey, found, err := store.GetHTTPResponseOwner(ctx, groupID, responseID)
	if err != nil || !found || ownerUser != userID || ownerKey != keyID {
		return 0, err
	}
	return store.GetResponseAccount(ctx, groupID, responseID)
}
