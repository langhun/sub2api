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
	ProbeToken                   string
	StatusCode                   int
	// HealthFinal marks an attempt that represents the website-level outcome.
	// Pool-mode retries remain telemetry-only until the handler exhausts the
	// website's own retry budget.
	HealthFinal bool
	// HealthOnly prevents the terminal pool failure notification from being
	// persisted as a duplicate request attempt.
	HealthOnly bool
	SpeedMs    int64
	CreatedAt  time.Time
}

type SmartRoutingAttemptObserver interface {
	ObserveRoutingAttempt(SmartRoutingAttempt)
}

type SmartRoutingAccountMetric struct {
	SpeedMs   float64
	ErrorRate float64
	// Samples is the number of request-level outcomes in the metric window.
	// Speed preference is ignored until this reaches the routing cold-start
	// threshold, so one lucky request cannot label an account as fastest.
	Samples int64
}

type SmartRoutingAccountMetricProvider interface {
	RoutingAccountMetrics(context.Context, int64, string, string, bool) map[int64]SmartRoutingAccountMetric
}

// SmartRoutingAccountAdmission is implemented by the Overlay. A recovery lease
// belongs to one account, never to an entire group of otherwise healthy peers.
type SmartRoutingAccountAdmission interface {
	AcquireRoutingAccount(context.Context, int64, string, string, bool) (string, bool)
	ReleaseRoutingAccount(int64, string, string, bool, string)
}

const SmartRoutingAccountUnavailableReason GatewayFailureReason = "smart_routing_account_unavailable"

func BeginSmartRoutingAttempt(ctx context.Context, accountID int64) (func(), error) {
	s := SmartRoutingStateFromContext(ctx)
	if s == nil || !s.SmartRoutingActive {
		return func() {}, nil
	}
	controller, ok := s.Observer.(SmartRoutingAccountAdmission)
	if !ok {
		return func() {}, nil
	}
	token, admitted := controller.AcquireRoutingAccount(ctx, accountID, s.RequestedModel, s.EndpointType, s.Stream)
	if !admitted {
		if s.FailedAccountIDs == nil {
			s.FailedAccountIDs = map[int64]struct{}{}
		}
		s.FailedAccountIDs[accountID] = struct{}{}
		if s.AccountExclusions == nil {
			s.AccountExclusions = map[int64]string{}
		}
		s.AccountExclusions[accountID] = "account unavailable at final admission"
		return func() {}, &UpstreamFailoverError{StatusCode: 503, Reason: SmartRoutingAccountUnavailableReason}
	}
	s.ProbeToken = token
	model, endpoint, stream := s.RequestedModel, s.EndpointType, s.Stream
	return func() {
		controller.ReleaseRoutingAccount(accountID, model, endpoint, stream, token)
		s.ProbeToken = ""
	}, nil
}

func LoadSmartRoutingAccountMetrics(ctx context.Context, resolver SmartRoutingResolver, groupID *int64) {
	s := SmartRoutingStateFromContext(ctx)
	if s == nil || !s.SmartRoutingActive || s.NoRoute || groupID == nil {
		return
	}
	if provider, ok := resolver.(SmartRoutingAccountMetricProvider); ok {
		s.AccountMetrics = provider.RoutingAccountMetrics(ctx, *groupID, s.RequestedModel, s.EndpointType, s.Stream)
	}
}

func smartRoutingOrderBySpeed(ctx context.Context, accounts []accountWithLoad) []accountWithLoad {
	if !SmartRoutingSpeedPreferred(ctx) || len(accounts) < 2 {
		return accounts
	}
	s := SmartRoutingStateFromContext(ctx)
	minSpeed, maxSpeed := 0.0, 0.0
	for _, a := range accounts {
		metric := s.AccountMetrics[a.account.ID]
		if metric.Samples < minSamplesForSmartRoutingMetric {
			continue
		}
		v := metric.SpeedMs
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
		speed, success := 0.5, 0.5
		if m.Samples >= minSamplesForSmartRoutingMetric && m.SpeedMs > 0 {
			speed = 1
			if maxSpeed > minSpeed {
				speed = (maxSpeed - m.SpeedMs) / (maxSpeed - minSpeed)
			}
		}
		if m.Samples >= minSamplesForSmartRoutingMetric {
			success = 1 - m.ErrorRate
		}
		return smartRoutingAccountMetricScore(s, speed, success)
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
	aMetric := state.AccountMetrics[account.ID]
	bMetric := state.AccountMetrics[selected.ID]
	minSpeed, maxSpeed := 0.0, 0.0
	for _, metric := range []SmartRoutingAccountMetric{aMetric, bMetric} {
		if metric.Samples < minSamplesForSmartRoutingMetric || metric.SpeedMs <= 0 {
			continue
		}
		if minSpeed == 0 || metric.SpeedMs < minSpeed {
			minSpeed = metric.SpeedMs
		}
		if metric.SpeedMs > maxSpeed {
			maxSpeed = metric.SpeedMs
		}
	}
	score := func(metric SmartRoutingAccountMetric) float64 {
		speed, success := 0.5, 0.5
		if metric.Samples >= minSamplesForSmartRoutingMetric && metric.SpeedMs > 0 {
			speed = 1
			if maxSpeed > minSpeed {
				speed = (maxSpeed - metric.SpeedMs) / (maxSpeed - minSpeed)
			}
		}
		if metric.Samples >= minSamplesForSmartRoutingMetric {
			success = 1 - metric.ErrorRate
		}
		return smartRoutingAccountMetricScore(state, speed, success)
	}
	a, b := score(aMetric), score(bMetric)
	return a > b, a != b
}

const minSamplesForSmartRoutingMetric = 20

func smartRoutingAccountMetricScore(state *SmartRoutingRequestState, speed, success float64) float64 {
	speedWeight, successWeight := smartRoutingQualityWeights(state)
	return speed*speedWeight/100 + success*successWeight/100
}

// The Overlay resolves strategy presets before entering the shared scheduler.
// Keep only a neutral fallback here for legacy callers that do not carry the
// optional quality weights.
func smartRoutingQualityWeights(state *SmartRoutingRequestState) (speed, success float64) {
	if state == nil || (state.SpeedWeight <= 0 && state.SuccessWeight <= 0) {
		return 30, 30
	}
	return state.SpeedWeight, state.SuccessWeight
}

func smartRoutingSpeedWeight(ctx context.Context) float64 {
	if state := SmartRoutingStateFromContext(ctx); state != nil {
		speed, _ := smartRoutingQualityWeights(state)
		return speed
	}
	return 30
}

func smartRoutingSuccessWeight(ctx context.Context) float64 {
	if state := SmartRoutingStateFromContext(ctx); state != nil {
		_, success := smartRoutingQualityWeights(state)
		return success
	}
	return 30
}

func ObserveSmartRoutingAttempt(ctx context.Context, account *Account, stream bool, duration time.Duration, firstTokenMs *int, err error) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.Observer == nil || account == nil || account.ID <= 0 || ctx.Err() != nil {
		return
	}
	// Keep the terminal outcome attributable even when the error is request
	// scoped (for example the smart-routing attempt budget) and is therefore
	// intentionally excluded from account-health telemetry.
	state.FinalGroupID = state.CurrentGroupID
	state.FinalAccountID = account.ID
	if failureStatus := smartRoutingFailureStatus(err); failureStatus > 0 {
		state.FinalErrorStatus = failureStatus
	}
	statusCode := 0
	healthFinal := true
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		var failure *UpstreamFailoverError
		if errors.As(err, &failure) {
			if !failure.ShouldRetryNextAccount() || failure.RequestScopedTransient || failure.Scope == GatewayFailureScopeRequest || !failure.ShouldReportAccountScheduleFailure() {
				return
			}
			statusCode = failure.StatusCode
			if account.IsPoolMode() && failure.RetryableOnSameAccount {
				healthFinal = false
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
		if healthFinal {
			state.FailedAccountIDs[account.ID] = struct{}{}
		}
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
	if err == nil {
		state.FinalSuccess = true
		state.FinalGroupID = state.CurrentGroupID
		state.FinalAccountID = account.ID
		if firstTokenMs != nil {
			state.FirstTokenMs = int64(*firstTokenMs)
		}
	}
	state.Observer.ObserveRoutingAttempt(SmartRoutingAttempt{
		RequestID: requestID, APIKeyID: key.ID, GroupID: state.CurrentGroupID,
		AccountID: account.ID, Attempt: state.Attempt, Model: state.RequestedModel,
		Endpoint: state.EndpointType, Stream: stream, Success: err == nil,
		ProbeToken: state.ProbeToken, StatusCode: statusCode, HealthFinal: healthFinal,
		SpeedMs: speed, CreatedAt: time.Now(),
	})
}

func smartRoutingFailureStatus(err error) int {
	var failure *UpstreamFailoverError
	if errors.As(err, &failure) && failure != nil {
		return failure.StatusCode
	}
	return 0
}

// RecordSmartRoutingOutcome writes one request-level result after the gateway
// has reached a terminal success or failure. Implementations must be
// asynchronous/best effort and must never delay the client response.
func RecordSmartRoutingOutcome(ctx context.Context, success bool, groupID, accountID int64, duration time.Duration) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.OutcomeRecorded {
		return
	}
	state.OutcomeRecorded = true
	state.FinalSuccess = success
	if groupID > 0 {
		state.FinalGroupID = groupID
	}
	if accountID > 0 {
		state.FinalAccountID = accountID
	}
	key, _ := APIKeyFromContext(ctx)
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	if requestID == "" || key == nil {
		return
	}
	recorder, ok := state.Observer.(SmartRoutingOutcomeRecorder)
	if !ok {
		return
	}
	recorder.RecordRoutingOutcome(SmartRoutingOutcome{
		RequestID: requestID, APIKeyID: key.ID, FinalGroupID: state.FinalGroupID,
		FinalAccountID: state.FinalAccountID, FinalSuccess: success,
		Attempts: state.TotalAttempts, GroupSwitches: state.GroupSwitches,
		DurationMs: duration.Milliseconds(), FirstTokenMs: state.FirstTokenMs,
		ErrorStatus: state.FinalErrorStatus,
	})
}

// FinalizeSmartRoutingPoolFailure promotes a pool-mode attempt to a website-
// level failure only after the handler has exhausted same-account retries (or
// cannot safely retry after visible output). The original attempt already
// supplied telemetry, so this notification only updates admission health.
func FinalizeSmartRoutingPoolFailure(ctx context.Context, account *Account, err error) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.Observer == nil || account == nil || account.ID <= 0 || ctx.Err() != nil || !account.IsPoolMode() {
		return
	}
	var failure *UpstreamFailoverError
	if !errors.As(err, &failure) || !failure.RetryableOnSameAccount || failure.RequestScopedTransient || failure.Scope == GatewayFailureScopeRequest || !failure.ShouldReportAccountScheduleFailure() {
		return
	}
	if state.FailedAccountIDs == nil {
		state.FailedAccountIDs = map[int64]struct{}{}
	}
	state.FailedAccountIDs[account.ID] = struct{}{}
	state.Observer.ObserveRoutingAttempt(SmartRoutingAttempt{
		AccountID:   account.ID,
		GroupID:     state.CurrentGroupID,
		Model:       state.RequestedModel,
		Endpoint:    state.EndpointType,
		Stream:      state.Stream,
		StatusCode:  failure.StatusCode,
		HealthFinal: true,
		HealthOnly:  true,
		CreatedAt:   time.Now(),
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
