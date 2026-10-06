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
	// SpeedMs is the cumulative speed sample for the selected group. It
	// includes same-group account failover, pool retries and their backoff, but
	// is only non-zero for a successful request with a valid terminal speed.
	SpeedMs int64
	// AccountSpeedMs is the cumulative speed sample for one upstream website
	// entry. It includes pool retries for the same account and resets when the
	// scheduler changes account or group.
	AccountSpeedMs int64
	// RequestElapsedMs is retained for request-level diagnostics. It must not
	// be used as an account speed sample after a cross-account or cross-group
	// switch.
	RequestElapsedMs int64
	CreatedAt        time.Time
}

type SmartRoutingAttemptObserver interface {
	ObserveRoutingAttempt(SmartRoutingAttempt)
}

type SmartRoutingAccountMetric struct {
	SpeedMs    float64
	SpeedP90Ms float64
	ErrorRate  float64
	// Samples is the number of request-level outcomes in the metric window.
	// It is used for the reliability of the success-rate dimension.
	Samples int64
	// SpeedSamples counts only successful request-level outcomes with a
	// positive speed sample. It is the reliability gate for speed preference;
	// total request samples must not make one lucky success look authoritative.
	SpeedSamples int64
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
		MarkSmartRoutingAccountFailed(ctx, accountID, s.CurrentGroupID)
		if s.AccountExclusions == nil {
			s.AccountExclusions = map[int64]string{}
		}
		s.AccountExclusions[accountID] = "account unavailable at final admission"
		return func() {}, &UpstreamFailoverError{StatusCode: 503, Reason: SmartRoutingAccountUnavailableReason}
	}
	s.ProbeToken = token
	// Keep one request-level origin for outcome diagnostics, while maintaining
	// independent group/account origins for latency attribution.
	now := time.Now()
	if s.SmartRoutingStartedAt.IsZero() {
		s.SmartRoutingStartedAt = now
	}
	s.CurrentAttemptStartedAt = now
	if s.GroupMetricGroupID != s.CurrentGroupID || s.GroupMetricStartedAt.IsZero() {
		s.GroupMetricGroupID = s.CurrentGroupID
		s.GroupMetricStartedAt = now
	}
	if s.AccountMetricGroupID != s.CurrentGroupID || s.AccountMetricAccountID != accountID || s.AccountMetricStartedAt.IsZero() {
		s.AccountMetricGroupID = s.CurrentGroupID
		s.AccountMetricAccountID = accountID
		s.AccountMetricStartedAt = now
	}
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
		if metric.SpeedSamples < minSamplesForSmartRoutingMetric {
			continue
		}
		v := smartRoutingMetricSpeedMs(metric)
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
		metricSpeed := smartRoutingMetricSpeedMs(m)
		if m.SpeedSamples >= minSamplesForSmartRoutingMetric && metricSpeed > 0 {
			speed = 1
			if maxSpeed > minSpeed {
				speed = (maxSpeed - metricSpeed) / (maxSpeed - minSpeed)
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
		metricSpeed := smartRoutingMetricSpeedMs(metric)
		if metric.SpeedSamples < minSamplesForSmartRoutingMetric || metricSpeed <= 0 {
			continue
		}
		if minSpeed == 0 || metricSpeed < minSpeed {
			minSpeed = metricSpeed
		}
		if metricSpeed > maxSpeed {
			maxSpeed = metricSpeed
		}
	}
	score := func(metric SmartRoutingAccountMetric) float64 {
		speed, success := 0.5, 0.5
		metricSpeed := smartRoutingMetricSpeedMs(metric)
		if metric.SpeedSamples >= minSamplesForSmartRoutingMetric && metricSpeed > 0 {
			speed = 1
			if maxSpeed > minSpeed {
				speed = (maxSpeed - metricSpeed) / (maxSpeed - minSpeed)
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

func smartRoutingMetricSpeedMs(metric SmartRoutingAccountMetric) float64 {
	if metric.SpeedMs <= 0 {
		return 0
	}
	if metric.SpeedP90Ms > 0 {
		// P50 captures normal latency while P90 prevents a route with a bad
		// tail from winning solely because its median is occasionally low.
		return metric.SpeedMs*0.6 + metric.SpeedP90Ms*0.4
	}
	return metric.SpeedMs
}

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
		if healthFinal {
			MarkSmartRoutingAccountFailed(ctx, account.ID, state.CurrentGroupID)
		}
	}
	key, ok := APIKeyFromContext(ctx)
	if !ok {
		return
	}
	state.Attempt++
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	now := time.Now()
	groupSpeed, accountSpeed, requestElapsed := smartRoutingTiming(state, account.ID, stream, duration, firstTokenMs, err == nil, now)
	if requestElapsed > 0 {
		state.RequestElapsedMs = requestElapsed
	}
	if err == nil {
		state.FinalSuccess = true
		state.FinalErrorStatus = 0
		state.FinalGroupID = state.CurrentGroupID
		state.FinalAccountID = account.ID
		if firstTokenMs != nil {
			state.FirstTokenMs = requestElapsed
		}
	}
	state.Observer.ObserveRoutingAttempt(SmartRoutingAttempt{
		RequestID: requestID, APIKeyID: key.ID, GroupID: state.CurrentGroupID,
		AccountID: account.ID, Attempt: state.Attempt, Model: state.RequestedModel,
		Endpoint: state.EndpointType, Stream: stream, Success: err == nil,
		ProbeToken: state.ProbeToken, StatusCode: statusCode, HealthFinal: healthFinal,
		SpeedMs: groupSpeed, AccountSpeedMs: accountSpeed,
		RequestElapsedMs: requestElapsed, CreatedAt: now,
	})
}

// smartRoutingRequestElapsedMs converts an attempt-local result into a
// request-level speed sample. The request origin is intentionally stored in
// SmartRoutingRequestState so pool-mode retries include their previous
// failures and backoff without changing any upstream stream parser.
func smartRoutingTiming(state *SmartRoutingRequestState, accountID int64, stream bool, duration time.Duration, firstTokenMs *int, success bool, now time.Time) (groupSpeed, accountSpeed, requestElapsed int64) {
	localElapsed := smartRoutingLocalElapsed(stream, duration, firstTokenMs)
	if state == nil {
		return 0, 0, localElapsed
	}
	requestElapsed = smartRoutingElapsedAt(state.SmartRoutingStartedAt, state.CurrentAttemptStartedAt, now, stream, duration, firstTokenMs)
	groupElapsed := smartRoutingElapsedAt(state.GroupMetricStartedAt, state.CurrentAttemptStartedAt, now, stream, duration, firstTokenMs)
	accountElapsed := smartRoutingElapsedAt(state.AccountMetricStartedAt, state.CurrentAttemptStartedAt, now, stream, duration, firstTokenMs)
	if success && localElapsed > 0 {
		groupSpeed = groupElapsed
		accountSpeed = accountElapsed
	}
	return groupSpeed, accountSpeed, requestElapsed
}

func smartRoutingLocalElapsed(stream bool, duration time.Duration, firstTokenMs *int) int64 {
	if stream {
		if firstTokenMs == nil || *firstTokenMs <= 0 {
			return 0
		}
		return int64(*firstTokenMs)
	}
	if duration <= 0 {
		return 0
	}
	return duration.Milliseconds()
}

func smartRoutingElapsedAt(start, attemptStart, now time.Time, stream bool, duration time.Duration, firstTokenMs *int) int64 {
	if start.IsZero() {
		return smartRoutingLocalElapsed(stream, duration, firstTokenMs)
	}
	if stream {
		if firstTokenMs == nil || *firstTokenMs <= 0 {
			return 0
		}
		if attemptStart.IsZero() || attemptStart.Before(start) {
			attemptStart = start
		}
		return maxInt64(0, attemptStart.Sub(start).Milliseconds()+int64(*firstTokenMs))
	}
	return maxInt64(0, now.Sub(start).Milliseconds())
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
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
	usageRequestID := resolveUsageBillingRequestID(ctx, "")
	recorder, ok := state.Observer.(SmartRoutingOutcomeRecorder)
	if !ok {
		return
	}
	recorder.RecordRoutingOutcome(SmartRoutingOutcome{
		RequestID: requestID, UsageRequestID: usageRequestID, APIKeyID: key.ID, FinalGroupID: state.FinalGroupID,
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
	MarkSmartRoutingAccountFailed(ctx, account.ID, state.CurrentGroupID)
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
	state := SmartRoutingStateFromContext(ctx)
	if state == nil {
		return excluded
	}
	return SmartRoutingExcludedAccountsForGroup(ctx, state.CurrentGroupID, excluded)
}

// SmartRoutingExcludedAccountsForGroup merges only failures recorded for the
// requested group. Account IDs can be shared by multiple candidate groups;
// keeping the filter group-local prevents a failure in one upstream website
// from suppressing a healthy instance of the same ID in another group.
func SmartRoutingExcludedAccountsForGroup(ctx context.Context, groupID int64, excluded map[int64]struct{}) map[int64]struct{} {
	s := SmartRoutingStateFromContext(ctx)
	if s == nil || !s.SmartRoutingActive || groupID <= 0 || len(s.FailedAccountGroups) == 0 {
		return excluded
	}
	failed := make(map[int64]struct{})
	for accountID, groups := range s.FailedAccountGroups {
		if _, ok := groups[groupID]; ok {
			failed[accountID] = struct{}{}
		}
	}
	if len(failed) == 0 {
		return excluded
	}
	merged := make(map[int64]struct{}, len(excluded)+len(failed))
	for id := range excluded {
		merged[id] = struct{}{}
	}
	for id := range failed {
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
