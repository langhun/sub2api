package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type routeAttemptCollector struct{ attempts []SmartRoutingAttempt }

func (c *routeAttemptCollector) ObserveRoutingAttempt(a SmartRoutingAttempt) {
	c.attempts = append(c.attempts, a)
}

type routeOutcomeCollector struct {
	routeAttemptCollector
	outcomes []SmartRoutingOutcome
}

func (c *routeOutcomeCollector) RecordRoutingOutcome(a SmartRoutingOutcome) {
	c.outcomes = append(c.outcomes, a)
}

func TestSmartRoutingFeedbackUsesAttemptIDsWithoutBillingDependency(t *testing.T) {
	collector := &routeAttemptCollector{}
	s := &SmartRoutingRequestState{SmartRoutingActive: true, CurrentGroupID: 2, RequestedModel: "gpt-5", EndpointType: "responses", Observer: collector}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.WithValue(context.Background(), ctxkey.RequestID, "request-1"), &APIKey{ID: 9}), s)
	account11 := &Account{ID: 11}
	account12 := &Account{ID: 12}
	ObserveSmartRoutingAttempt(ctx, account11, true, time.Second, nil, &UpstreamFailoverError{StatusCode: 503})
	s.CurrentGroupID = 21
	ttft := 100
	ObserveSmartRoutingAttempt(ctx, account12, true, 2*time.Second, &ttft, nil)
	require.Len(t, collector.attempts, 2)
	require.Equal(t, "request-1", collector.attempts[0].RequestID)
	require.Equal(t, 1, collector.attempts[0].Attempt)
	require.Equal(t, 2, collector.attempts[1].Attempt)
	require.Equal(t, int64(2), collector.attempts[0].GroupID)
	require.Equal(t, int64(21), collector.attempts[1].GroupID)
	require.False(t, collector.attempts[0].Success)
	require.Equal(t, int64(100), collector.attempts[1].SpeedMs)
	require.Contains(t, SmartRoutingExcludedAccountsForGroup(ctx, 2, nil), int64(11))
	require.NotContains(t, SmartRoutingExcludedAccountsForGroup(ctx, 21, nil), int64(11), "an account failure in group 2 must not suppress the same account ID in group 21")
	ObserveSmartRoutingAttempt(ctx, account12, false, time.Second, nil, errors.New("invalid user parameter"))
	ObserveSmartRoutingAttempt(ctx, account12, false, time.Second, nil, &UpstreamFailoverError{StatusCode: 503, RequestScopedTransient: true})
	ObserveSmartRoutingAttempt(ctx, account12, false, time.Second, nil, &UpstreamFailoverError{Stage: GatewayFailureStageAccountAuth, Scope: GatewayFailureScopeProvider})
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	ObserveSmartRoutingAttempt(cancelCtx, account12, true, time.Second, nil, &UpstreamFailoverError{StatusCode: 503})
	require.Len(t, collector.attempts, 2)
	require.NotContains(t, s.FailedAccountIDs, int64(12), "request/provider credential failures must not quarantine a healthy account")
	s.SmartRoutingActive = false
	ObserveSmartRoutingAttempt(ctx, account12, false, time.Second, nil, nil)
	require.Len(t, collector.attempts, 2, "single-group requests have no new feedback behavior")
}

func TestSmartRoutingFailureScopeAllowsSharedAccountAcrossGroups(t *testing.T) {
	collector := &routeAttemptCollector{}
	state := &SmartRoutingRequestState{
		SmartRoutingActive: true,
		CurrentGroupID:     2,
		RequestedModel:     "gpt-5",
		EndpointType:       "responses",
		Observer:           collector,
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	sharedAccount := &Account{ID: 11}
	failure := &UpstreamFailoverError{StatusCode: 503}

	ObserveSmartRoutingAttempt(ctx, sharedAccount, true, time.Second, nil, failure)
	require.True(t, SmartRoutingAccountFailed(ctx, sharedAccount.ID, 2))
	require.False(t, SmartRoutingAccountFailed(ctx, sharedAccount.ID, 21))
	require.Contains(t, SmartRoutingExcludedAccountsForGroup(ctx, 2, nil), sharedAccount.ID)
	require.NotContains(t, SmartRoutingExcludedAccountsForGroup(ctx, 21, nil), sharedAccount.ID)

	// The same website/account ID can be mounted in both candidate groups. A
	// later failure in group 21 must add that scope without erasing group 2.
	state.CurrentGroupID = 21
	ObserveSmartRoutingAttempt(ctx, sharedAccount, true, time.Second, nil, failure)
	require.True(t, SmartRoutingAccountFailed(ctx, sharedAccount.ID, 2))
	require.True(t, SmartRoutingAccountFailed(ctx, sharedAccount.ID, 21))
	require.Contains(t, SmartRoutingExcludedAccountsForGroup(ctx, 2, nil), sharedAccount.ID)
	require.Contains(t, SmartRoutingExcludedAccountsForGroup(ctx, 21, nil), sharedAccount.ID)
}

func TestSmartRoutingOutcomeUsesFinalRequestResult(t *testing.T) {
	collector := &routeOutcomeCollector{}
	state := &SmartRoutingRequestState{SmartRoutingActive: true, CurrentGroupID: 2, RequestedModel: "gpt-5", EndpointType: "responses", Observer: collector, SmartRoutingStartedAt: time.Now(), TotalAttempts: 2}
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-outcome")
	ctx = context.WithValue(ctx, ctxkey.RequestID, "request-outcome")
	ctx = WithSmartRoutingState(WithAPIKeyContext(ctx, &APIKey{ID: 9}), state)
	account := &Account{ID: 11}
	ObserveSmartRoutingAttempt(ctx, account, true, 2*time.Second, nil, &UpstreamFailoverError{StatusCode: 503})
	state.CurrentGroupID = 21
	ttft := 120
	ObserveSmartRoutingAttempt(ctx, account, true, 500*time.Millisecond, &ttft, nil)
	RecordSmartRoutingOutcome(ctx, true, state.CurrentGroupID, account.ID, 2500*time.Millisecond)
	RecordSmartRoutingOutcome(ctx, false, 2, account.ID, 3*time.Second)
	require.Len(t, collector.outcomes, 1)
	require.Equal(t, "request-outcome", collector.outcomes[0].RequestID)
	require.Equal(t, "client:client-outcome", collector.outcomes[0].UsageRequestID)
	require.True(t, collector.outcomes[0].FinalSuccess)
	require.Equal(t, int64(21), collector.outcomes[0].FinalGroupID)
	require.Equal(t, int64(11), collector.outcomes[0].FinalAccountID)
	require.Equal(t, 2, collector.outcomes[0].Attempts)
	require.Equal(t, int64(120), collector.outcomes[0].FirstTokenMs)
	require.Zero(t, collector.outcomes[0].ErrorStatus)
}

func TestSmartRoutingOutcomeUsesLocalBillingRequestIDWithoutClientID(t *testing.T) {
	collector := &routeOutcomeCollector{}
	state := &SmartRoutingRequestState{SmartRoutingActive: true, Observer: collector}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.WithValue(context.Background(), ctxkey.RequestID, "request-local"), &APIKey{ID: 9}), state)

	RecordSmartRoutingOutcome(ctx, true, 21, 11, time.Second)

	require.Len(t, collector.outcomes, 1)
	require.Equal(t, "request-local", collector.outcomes[0].RequestID)
	require.Equal(t, "local:request-local", collector.outcomes[0].UsageRequestID)
}

func TestSmartRoutingOutcomeCapturesRequestScopedFailureAttribution(t *testing.T) {
	collector := &routeOutcomeCollector{}
	state := &SmartRoutingRequestState{
		SmartRoutingActive:    true,
		CurrentGroupID:        21,
		RequestedModel:        "gpt-5",
		EndpointType:          "responses",
		Observer:              collector,
		SmartRoutingStartedAt: time.Now(),
		TotalAttempts:         1,
	}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.WithValue(context.Background(), ctxkey.RequestID, "request-budget"), &APIKey{ID: 9}), state)
	account := &Account{ID: 12}
	ObserveSmartRoutingAttempt(ctx, account, false, time.Second, nil, &UpstreamFailoverError{
		StatusCode: 503,
		Reason:     SmartRoutingAttemptBudgetExceededReason,
		Scope:      GatewayFailureScopeRequest,
	})
	RecordSmartRoutingOutcome(ctx, false, 0, 0, time.Second)
	require.Len(t, collector.outcomes, 1)
	require.False(t, collector.outcomes[0].FinalSuccess)
	require.Equal(t, int64(21), collector.outcomes[0].FinalGroupID)
	require.Equal(t, int64(12), collector.outcomes[0].FinalAccountID)
	require.Equal(t, 503, collector.outcomes[0].ErrorStatus)
}

func TestSmartRoutingPoolFailureIsFinalizedOnlyAfterHandlerExhaustion(t *testing.T) {
	collector := &routeAttemptCollector{}
	state := &SmartRoutingRequestState{
		SmartRoutingActive: true,
		CurrentGroupID:     2,
		RequestedModel:     "gpt-5",
		EndpointType:       "responses",
		Stream:             true,
		Observer:           collector,
	}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.Background(), &APIKey{ID: 9}), state)
	account := &Account{ID: 11, Type: AccountTypeAPIKey, Credentials: map[string]any{"pool_mode": true}}
	failure := &UpstreamFailoverError{StatusCode: 429, RetryableOnSameAccount: true}

	for i := 0; i < account.GetPoolModeRetryCount()+1; i++ {
		ObserveSmartRoutingAttempt(ctx, account, true, time.Second, nil, failure)
		require.NotContains(t, state.FailedAccountIDs, account.ID)
		require.False(t, collector.attempts[i].HealthFinal)
		require.False(t, collector.attempts[i].HealthOnly)
	}

	FinalizeSmartRoutingPoolFailure(ctx, account, failure)
	require.Contains(t, state.FailedAccountIDs, account.ID)
	require.Len(t, collector.attempts, account.GetPoolModeRetryCount()+2)
	final := collector.attempts[len(collector.attempts)-1]
	require.True(t, final.HealthFinal)
	require.True(t, final.HealthOnly)
	require.Equal(t, 429, final.StatusCode)
}

func TestSmartRoutingSeparatesWebsitePoolsFromDirectAccounts(t *testing.T) {
	collector := &routeAttemptCollector{}
	state := &SmartRoutingRequestState{
		SmartRoutingActive: true,
		CurrentGroupID:     2,
		RequestedModel:     "gpt-5",
		EndpointType:       "responses",
		Observer:           collector,
	}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.Background(), &APIKey{ID: 9}), state)
	poolWebsite := &Account{
		ID:          11,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"pool_mode": true},
	}
	directAccount := &Account{ID: 12, Type: AccountTypeOAuth}
	failure := &UpstreamFailoverError{StatusCode: 429, RetryableOnSameAccount: true}

	ObserveSmartRoutingAttempt(ctx, poolWebsite, false, time.Second, nil, failure)
	require.NotContains(t, state.FailedAccountIDs, poolWebsite.ID, "an intermediate failure belongs to the website pool, not its hidden members")

	ObserveSmartRoutingAttempt(ctx, directAccount, false, time.Second, nil, failure)
	require.Contains(t, state.FailedAccountIDs, directAccount.ID, "a directly managed account is its own health unit")

	attemptCount := len(collector.attempts)
	FinalizeSmartRoutingPoolFailure(ctx, directAccount, failure)
	require.Len(t, collector.attempts, attemptCount, "pool finalization must not emit a website-level event for a direct account")
}

func TestSmartRoutingRetryAccountPinIsOneSelectionOnly(t *testing.T) {
	state := &SmartRoutingRequestState{SmartRoutingActive: true}
	ctx := WithSmartRoutingState(context.Background(), state)
	accounts := []Account{{ID: 11}, {ID: 12}}

	SetSmartRoutingRetryAccount(ctx, 11)
	limited := LimitSmartRoutingAccountsForRetry(ctx, accounts)
	require.Equal(t, []Account{{ID: 11}}, limited)
	require.Equal(t, int64(11), state.RetryAccountID)
	CompleteSmartRoutingRetrySelection(ctx, 11)
	require.Zero(t, state.RetryAccountID)
	require.Equal(t, accounts, LimitSmartRoutingAccountsForRetry(ctx, accounts))

	SetSmartRoutingRetryAccount(ctx, 99)
	require.Equal(t, accounts, LimitSmartRoutingAccountsForRetry(ctx, accounts), "a removed website must not hide healthy websites")
	require.Zero(t, state.RetryAccountID)
}

func TestSmartRoutingSpeedDoesNotForceSlowStickyToFront(t *testing.T) {
	s := &defaultOpenAIAccountScheduler{}
	plan := openAIAccountLoadPlan{topK: 2, candidates: []openAIAccountCandidateScore{
		{account: &Account{ID: 12}, score: 100, ttft: 100, hasTTFT: true},
		{account: &Account{ID: 11}, score: 1, ttft: 60000, hasTTFT: true},
	}}
	order := s.buildOpenAISelectionOrder(OpenAIAccountScheduleRequest{SmartRouting: true, SpeedPreferred: true, StickyWeighted: true, StickyAccountID: 11}, plan)
	require.Equal(t, int64(12), order[0].account.ID)
	order = s.buildOpenAISelectionOrder(OpenAIAccountScheduleRequest{StickyWeighted: true, StickyAccountID: 11}, plan)
	require.Equal(t, int64(11), order[0].account.ID, "legacy single-group semantics stay unchanged")
	selection, err := s.tryFallbackToWeightedSticky(context.Background(), OpenAIAccountScheduleRequest{SmartRouting: true, StickyWeighted: true, StickyAccountID: 11})
	require.NoError(t, err)
	require.Nil(t, selection, "smart routing must not resurrect a sticky account in the fallback pass")
}

func TestSmartRoutingGatewaySpeedOrdersActualAccounts(t *testing.T) {
	ctx := WithSmartRoutingState(context.Background(), &SmartRoutingRequestState{SmartRoutingActive: true, Strategy: "speed", AccountMetrics: map[int64]SmartRoutingAccountMetric{
		11: {SpeedMs: 60000, SpeedSamples: 20, Samples: 20}, 12: {SpeedMs: 100, SpeedSamples: 20, Samples: 20},
	}})
	accounts := []accountWithLoad{{account: &Account{ID: 11}}, {account: &Account{ID: 12}}}
	require.Equal(t, int64(12), smartRoutingOrderBySpeed(ctx, accounts)[0].account.ID)
	require.Equal(t, int64(11), smartRoutingOrderBySpeed(context.Background(), accounts)[0].account.ID)
	prefer, decisive := smartRoutingPreferAccountBySpeed(ctx, accounts[1].account, accounts[0].account)
	require.True(t, prefer)
	require.True(t, decisive)
	_, decisive = smartRoutingPreferAccountBySpeed(context.Background(), accounts[1].account, accounts[0].account)
	require.False(t, decisive)
}

func TestSmartRoutingQualityWeightsUseResolvedOverlayValues(t *testing.T) {
	speed, success := smartRoutingQualityWeights(&SmartRoutingRequestState{SpeedWeight: 70, SuccessWeight: 15})
	require.Equal(t, 70.0, speed)
	require.Equal(t, 15.0, success)
	speed, success = smartRoutingQualityWeights(&SmartRoutingRequestState{})
	require.Equal(t, 30.0, speed)
	require.Equal(t, 30.0, success)
}

func TestSmartRoutingAttemptBudgetIsRequestLocal(t *testing.T) {
	state := &SmartRoutingRequestState{SmartRoutingActive: true, SmartRoutingMaxAttempts: 2, SmartRoutingMaxDuration: time.Minute}
	ctx := WithSmartRoutingState(context.Background(), state)
	require.True(t, SmartRoutingAttemptBudgetAvailable(ctx))
	require.True(t, SmartRoutingAttemptBudgetAvailable(ctx))
	require.False(t, SmartRoutingAttemptBudgetAvailable(ctx))
	require.Equal(t, 2, state.TotalAttempts)
}

func TestSmartRoutingSpeedRequiresReliableSamples(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive: true,
		Strategy:           "speed",
		AccountMetrics: map[int64]SmartRoutingAccountMetric{
			11: {SpeedMs: 80, Samples: minSamplesForSmartRoutingMetric - 1},
			12: {SpeedMs: 800, Samples: minSamplesForSmartRoutingMetric - 1},
		},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	accounts := []accountWithLoad{{account: &Account{ID: 11}}, {account: &Account{ID: 12}}}
	ordered := smartRoutingOrderBySpeed(ctx, accounts)
	require.Equal(t, []int64{11, 12}, []int64{ordered[0].account.ID, ordered[1].account.ID}, "cold accounts keep scheduler order")
}

func TestSmartRoutingCustomAccountWeightsAffectSchedulerScore(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive: true,
		Strategy:           "custom",
		PriceWeight:        5,
		SpeedWeight:        90,
		SuccessWeight:      5,
		AccountMetrics: map[int64]SmartRoutingAccountMetric{
			11: {SpeedMs: 900, SpeedSamples: minSamplesForSmartRoutingMetric, ErrorRate: 0, Samples: minSamplesForSmartRoutingMetric},
			12: {SpeedMs: 100, SpeedSamples: minSamplesForSmartRoutingMetric, ErrorRate: 1, Samples: minSamplesForSmartRoutingMetric},
		},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	scheduler := &defaultOpenAIAccountScheduler{service: &OpenAIGatewayService{}}
	plan := scheduler.buildOpenAIAccountLoadPlan(ctx, OpenAIAccountScheduleRequest{SmartRouting: true, SpeedPreferred: true}, []*Account{{ID: 11}, {ID: 12}}, nil)
	require.Greater(t, plan.allCandidates[1].score, plan.allCandidates[0].score, "custom speed weight must favor the lower P50 account")
}

func TestSmartRoutingAccountScoringIgnoresUnrelatedRuntimeTTFT(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	scheduler := &defaultOpenAIAccountScheduler{service: &OpenAIGatewayService{}, stats: stats}
	accounts := []*Account{{ID: 11}, {ID: 12}}
	state := &SmartRoutingRequestState{SmartRoutingActive: true, Strategy: "speed", SpeedWeight: 70, SuccessWeight: 15, AccountMetrics: map[int64]SmartRoutingAccountMetric{
		11: {SpeedMs: 60000, SpeedSamples: 20, Samples: 20},
		12: {SpeedMs: 100, SpeedSamples: 20, Samples: 20},
	}}
	ctx := WithSmartRoutingState(context.Background(), state)
	plan := scheduler.buildOpenAIAccountLoadPlan(ctx, OpenAIAccountScheduleRequest{SmartRouting: true, SpeedPreferred: true}, accounts, nil)
	require.Equal(t, int64(12), plan.selectionOrder[0].account.ID)
	state.ContinuityLocked, state.ContinuityAccountID = true, 11
	plan = scheduler.buildOpenAIAccountLoadPlan(ctx, OpenAIAccountScheduleRequest{SmartRouting: true}, accounts, nil)
	require.Len(t, plan.allCandidates, 1)
	require.Equal(t, int64(11), plan.allCandidates[0].account.ID)
}

type routingCapacityCache struct {
	ConcurrencyCache
	loads map[int64]*AccountLoadInfo
	err   error
}

func (c routingCapacityCache) GetAccountsLoadBatch(context.Context, []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	return c.loads, c.err
}

func TestSmartRoutingCapacityDistinguishesFullFromUnavailable(t *testing.T) {
	accounts := []Account{{ID: 11, Concurrency: 2}, {ID: 12, Concurrency: 2}}
	cache := routingCapacityCache{loads: map[int64]*AccountLoadInfo{11: {LoadRate: 100}, 12: {LoadRate: 50}}}
	free, known := routingCapacity(context.Background(), NewConcurrencyService(cache), accounts)
	require.True(t, known)
	require.Equal(t, 0.25, free)
	cache.loads[12].LoadRate = 100
	free, known = routingCapacity(context.Background(), NewConcurrencyService(cache), accounts)
	require.True(t, known)
	require.Zero(t, free)
	cache.err = errors.New("unavailable")
	_, known = routingCapacity(context.Background(), NewConcurrencyService(cache), accounts)
	require.False(t, known, "Redis failure is not proof of full capacity")
}

type responseRouteResolverStub struct {
	SmartRoutingResolver
	group, account int64
	err            error
}

func (s responseRouteResolverStub) BindRoutingResponse(context.Context, *APIKey, int64, int64, int64, string, time.Duration) error {
	return nil
}
func (s responseRouteResolverStub) LookupRoutingResponse(context.Context, *APIKey, string) (int64, int64, error) {
	return s.group, s.account, s.err
}

func TestSmartRoutingContinuationRestoresActualOwnerWithoutWeakeningTenantCheck(t *testing.T) {
	store := NewOpenAIWSStateStore(nil)
	svc := &OpenAIGatewayService{openaiWSStateStore: store, smartRoutingResolver: responseRouteResolverStub{group: 21, account: 12}}
	state := &SmartRoutingRequestState{MainGroupID: 2, CurrentGroupID: 2, ContinuityLocked: true}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.Background(), &APIKey{ID: 9, UserID: 7}), state)
	require.NoError(t, svc.BindOpenAIHTTPResponseOwner(ctx, 21, "resp_candidate", 7, 9))
	ok, err := svc.ValidateOpenAIHTTPResponseOwner(ctx, 2, "resp_candidate", 7, 9)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(21), state.ContinuityGroupID)
	require.Equal(t, int64(12), state.ContinuityAccountID)
	ok, err = svc.ValidateOpenAIHTTPResponseOwner(ctx, 2, "resp_candidate", 8, 10)
	require.NoError(t, err)
	require.False(t, ok)
	svc.smartRoutingResolver = responseRouteResolverStub{err: errors.New("owner store unavailable")}
	ok, err = svc.ValidateOpenAIHTTPResponseOwner(ctx, 2, "resp_candidate", 7, 9)
	require.Error(t, err)
	require.False(t, ok)
}

func TestSmartRoutingExhaustionStopsRetryEvenAfterPrimaryRestore(t *testing.T) {
	s := &SmartRoutingRequestState{SmartRoutingActive: true, MainGroupID: 2, CurrentGroupID: 2, MaxGroupSwitches: 2}
	ctx := WithSmartRoutingState(context.Background(), s)
	require.True(t, TrySmartRoutingGroupFailover(ctx, nil, false))
	require.Contains(t, s.ExhaustedGroups, int64(2))
	s.NoRoute = true
	require.False(t, TrySmartRoutingGroupFailover(ctx, nil, false))
	require.False(t, SmartRoutingContractValid(ctx))
}

func TestSmartRoutingLegacyResponseRecoveryRequiresActualAccountBinding(t *testing.T) {
	store := NewOpenAIWSStateStore(nil)
	svc := &OpenAIGatewayService{openaiWSStateStore: store}
	ctx := context.Background()
	require.NoError(t, svc.BindOpenAIHTTPResponseOwner(ctx, 2, "resp_legacy", 7, 9))
	require.NoError(t, svc.BindOpenAIHTTPResponseOwner(ctx, 21, "resp_legacy", 7, 9))
	require.NoError(t, store.BindResponseAccount(ctx, 21, "resp_legacy", 12, time.Hour))
	id, err := svc.RoutingResponseAccount(ctx, 2, "resp_legacy", 7, 9)
	require.NoError(t, err)
	require.Zero(t, id, "primary ownership alias must not be confused with an actual serving account")
	id, err = svc.RoutingResponseAccount(ctx, 21, "resp_legacy", 7, 9)
	require.NoError(t, err)
	require.Equal(t, int64(12), id)
	id, err = svc.RoutingResponseAccount(ctx, 21, "resp_legacy", 8, 9)
	require.NoError(t, err)
	require.Zero(t, id)
}
