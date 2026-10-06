package smartrouting

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func expectRoutingProfile(mock sqlmock.Sqlmock, keyID int64) {
	mock.ExpectQuery("SELECT api_key_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(keyID, true, "smart", "speed", 15, 70, 15, 0, 0, 1))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}).AddRow(21, 0, true, "candidate", service.PlatformOpenAI))
}

func expectRoutingStats(mock sqlmock.Sqlmock, groupID int64, samples, success, speed int) {
	mock.ExpectQuery("FROM custom_route_attempts").WithArgs(groupID, "gpt-5", "responses", true).WillReturnRows(sqlmock.NewRows([]string{
		"samples", "success", "speed_samples", "speed", "speed_p90",
		"fallback_samples", "fallback_success", "fallback_speed_samples", "fallback_speed", "fallback_speed_p90",
	}).AddRow(samples, success, success, speed, speed, samples, success, success, speed, speed))
}

func TestRoutingOutcomeWriterIsBoundedAndCleansExpiredRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	m := &Module{repo: NewRepository(db)}
	outcome := service.SmartRoutingOutcome{
		RequestID: "request-outcome", UsageRequestID: "local:request-outcome", APIKeyID: 9, FinalGroupID: 21,
		FinalAccountID: 12, FinalSuccess: true, Attempts: 2,
		GroupSwitches: 1, DurationMs: 2500, FirstTokenMs: 120,
	}
	mock.ExpectExec("INSERT INTO custom_route_outcomes").
		WithArgs("request-outcome", int64(9), int64(21), int64(12), true, 2, 1, int64(2500), int64(120), 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO custom_usage_log_routing_marks").
		WithArgs("local:request-outcome", int64(9)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM custom_route_outcomes").WillReturnResult(sqlmock.NewResult(0, 3))
	m.RecordRoutingOutcome(outcome)
	require.Eventually(t, func() bool {
		m.feedback.mu.Lock()
		defer m.feedback.mu.Unlock()
		workersStopped := true
		for i := range m.feedback.outcomeWorkers {
			workersStopped = workersStopped && !m.feedback.outcomeWorkers[i]
		}
		return workersStopped && mock.ExpectationsWereMet() == nil
	}, time.Second, 10*time.Millisecond)

	for i := range m.feedback.outcomeQueues {
		m.feedback.outcomeQueues[i] = make([]service.SmartRoutingOutcome, 512)
		m.feedback.outcomeWorkers[i] = true
	}
	m.RecordRoutingOutcome(service.SmartRoutingOutcome{RequestID: "queue-full", APIKeyID: 9})
	for i := range m.feedback.outcomeQueues {
		require.Len(t, m.feedback.outcomeQueues[i], 512)
	}
}

func TestRoutingAuditSerializesDecisionBeforeSelectedAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	m := &Module{repo: NewRepository(db)}
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "audit-order")
	ctx = service.WithSmartRoutingState(ctx, &service.SmartRoutingRequestState{EndpointType: "responses"})
	key := &service.APIKey{ID: 9}
	profile := &Profile{APIKeyID: 9, Strategy: "speed"}
	decision := &service.SmartRouteDecision{GroupID: 21, FallbackDepth: 1, Reason: "speed score"}
	mock.ExpectExec("INSERT INTO custom_route_decisions").
		WithArgs("audit-order", int64(9), int64(9), "gpt-5", "responses", int64(21), "speed", sqlmock.AnyArg(), 1, "speed score").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE custom_route_decisions SET selected_account_id").
		WithArgs(int64(77), "audit-order", int64(9), int64(21)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	m.recordDecision(ctx, key, profile, "gpt-5", Group{GroupID: 21, Score: 90}, decision)
	m.RecordSelectedAccount(ctx, 9, 21, 77)
	require.Eventually(t, func() bool { return mock.ExpectationsWereMet() == nil }, time.Second, 5*time.Millisecond)
}

func TestRoutingAuditQueueDropsWhenFull(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	m := &Module{repo: NewRepository(db)}
	requestID := "audit-queue-full"
	shard := auditShard(requestID)
	m.feedback.auditQueues[shard] = make([]routeAuditEvent, routeAuditQueueCapacity)
	m.feedback.auditWorkers[shard] = true
	m.enqueueAudit(routeAuditEvent{kind: routeAuditDecision, requestID: requestID})
	require.Len(t, m.feedback.auditQueues[shard], routeAuditQueueCapacity)
}

func TestSameAccountRetryPolicyOnlySuppressesDeterministicFailures(t *testing.T) {
	m := &Module{}
	require.True(t, m.SuppressSameAccountRetry(&service.UpstreamFailoverError{StatusCode: 403, ResponseBody: []byte("Insufficient account balance")}))
	require.True(t, m.SuppressSameAccountRetry(&service.UpstreamFailoverError{StatusCode: 404, ResponseBody: []byte("model is not supported")}))
	require.True(t, m.SuppressSameAccountRetry(&service.UpstreamFailoverError{StatusCode: 429, ResponseBody: []byte("All available accounts are currently rate-limited")}))
	require.False(t, m.SuppressSameAccountRetry(&service.UpstreamFailoverError{StatusCode: 429, ResponseBody: []byte("temporary rate limit")}))
	require.False(t, m.SuppressSameAccountRetry(&service.UpstreamFailoverError{StatusCode: 503, ResponseBody: []byte("service busy")}))
}

func TestResolveNeverReadmitsExhaustedGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	main := &service.Group{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: "standard", RateMultiplier: 1}
	candidate := &service.Group{ID: 21, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: "standard", RateMultiplier: 2}
	m := &Module{repo: NewRepository(db), groups: &routingGroupRepoStub{groups: map[int64]*service.Group{2: main, 21: candidate}}, availableGroups: allowRoutingGroups(2, 21), accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{
		2: {{ID: 11, Status: service.StatusActive, Schedulable: true}}, 21: {{ID: 12, Status: service.StatusActive, Schedulable: true}},
	}}}
	mainID := int64(2)
	key := &service.APIKey{ID: 9, GroupID: &mainID, Group: main}
	s := &service.SmartRoutingRequestState{MainGroupID: 2, CurrentGroupID: 2, EndpointType: "responses", Stream: true, ExhaustedGroups: map[int64]struct{}{2: {}}, TriedGroups: map[int64]struct{}{2: {}}}
	ctx := service.WithSmartRoutingState(context.Background(), s)
	expectRoutingProfile(mock, 9)
	expectRoutingStats(mock, 2, 50, 50, 100)
	expectRoutingStats(mock, 21, 50, 50, 1000)
	decision, err := m.Resolve(ctx, key, "gpt-5", "session", nil)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.Equal(t, int64(21), decision.GroupID, "even a higher-scoring exhausted primary must not be retried")
	s.CurrentGroupID = 21
	s.ExhaustedGroups[21] = struct{}{}
	s.TriedGroups[21] = struct{}{}
	expectRoutingProfile(mock, 9)
	expectRoutingStats(mock, 2, 50, 50, 100)
	expectRoutingStats(mock, 21, 50, 50, 1000)
	decision, err = m.Resolve(ctx, key, "gpt-5", "session", nil)
	require.NoError(t, err)
	require.Nil(t, decision)
	require.True(t, s.NoRoute)
	require.False(t, service.ShouldRetrySmartRoutingGroup(ctx))
	require.False(t, service.SmartRoutingContractValid(ctx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAttemptStatsIncludeFailuresAndSeparateRequestTypes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	m := &Module{repo: NewRepository(db)}
	s := &service.SmartRoutingRequestState{EndpointType: "responses", Stream: true}
	expectRoutingStats(mock, 2, 10, 2, 250)
	st := m.loadStats(service.WithSmartRoutingState(context.Background(), s), 2, "gpt-5")
	require.Equal(t, int64(10), st.Samples)
	require.Equal(t, int64(2), st.Success)
	require.Equal(t, int64(250), st.SpeedMs.Int64)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRoutingAccountMetricsFallsBackToReliableHourlyWindow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	m := &Module{repo: NewRepository(db)}
	mock.ExpectQuery("SELECT account_id").WithArgs(int64(2), "gpt-5", "responses", true).WillReturnRows(
		sqlmock.NewRows([]string{"account_id", "recent_samples", "recent_speed_samples", "recent_speed", "recent_speed_p90", "recent_error", "fallback_samples", "fallback_speed_samples", "fallback_speed", "fallback_speed_p90", "fallback_error"}).
			AddRow(int64(12), int64(3), int64(2), 120.0, 180.0, 0.0, int64(minSamples), int64(minSamples), 640.0, 900.0, 0.2),
	)
	metrics := m.RoutingAccountMetrics(context.Background(), 2, "gpt-5", "responses", true)
	require.Equal(t, service.SmartRoutingAccountMetric{Samples: minSamples, SpeedSamples: minSamples, SpeedMs: 640, SpeedP90Ms: 900, ErrorRate: 0.2}, metrics[12])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRoutingWeightsOwnVisibleStrategyPresets(t *testing.T) {
	tests := []struct {
		name                              string
		profile                           *Profile
		wantPrice, wantSpeed, wantSuccess float64
	}{
		{name: "auto", profile: &Profile{Strategy: "auto", PriceWeight: 1, SpeedWeight: 1, SuccessWeight: 98}, wantPrice: 40, wantSpeed: 30, wantSuccess: 30},
		{name: "price", profile: &Profile{Strategy: "price"}, wantPrice: 100, wantSpeed: 0, wantSuccess: 0},
		{name: "speed", profile: &Profile{Strategy: "speed"}, wantPrice: 15, wantSpeed: 70, wantSuccess: 15},
		{name: "success", profile: &Profile{Strategy: "success"}, wantPrice: 15, wantSpeed: 15, wantSuccess: 70},
		{name: "custom", profile: &Profile{Strategy: "custom", PriceWeight: 60, SpeedWeight: 25, SuccessWeight: 15}, wantPrice: 60, wantSpeed: 25, wantSuccess: 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, speed, success := routingWeights(tt.profile)
			require.Equal(t, tt.wantPrice, price)
			require.Equal(t, tt.wantSpeed, speed)
			require.Equal(t, tt.wantSuccess, success)
		})
	}
}

func TestResolveFailureAfterAttemptCannotReplayPrimary(t *testing.T) {
	for _, attempted := range []bool{false, true} {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		m := &Module{repo: NewRepository(db)}
		state := &service.SmartRoutingRequestState{MainGroupID: 2, CurrentGroupID: 21}
		if attempted {
			state.Attempt = 1
		}
		id := int64(21)
		mock.ExpectQuery("SELECT api_key_id").WithArgs(9).WillReturnError(fmt.Errorf("profile unavailable"))
		_, err = m.Resolve(service.WithSmartRoutingState(context.Background(), state), &service.APIKey{ID: 9, GroupID: &id}, "gpt-5", "", nil)
		require.Error(t, err)
		require.Equal(t, attempted, state.NoRoute)
		require.False(t, state.SmartRoutingActive)
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	}
}

func TestExplorationUsesIndependentRequestBuckets(t *testing.T) {
	m := &Module{}
	p := &Profile{Version: 1, Strategy: "auto", ExplorationRate: 0.01}
	groups := []Group{{GroupID: 2, Eligible: true, Samples: 100}, {GroupID: 21, Eligible: true}}
	key := &service.APIKey{ID: 9}
	explored := 0
	for i := 0; i < 10000; i++ {
		if m.explorationIndex(key, p, groups, 2, fmt.Sprintf("request-%d", i), "gpt-5") == 1 {
			explored++
		}
	}
	require.Greater(t, explored, 50)
	require.Less(t, explored, 150)
	groups[1].Eligible = false
	for i := 0; i < 100; i++ {
		require.Equal(t, -1, m.explorationIndex(key, p, groups, 2, fmt.Sprint(i), "gpt-5"))
	}
}

func TestResponseRouteRemainsScopedAndRejectsRevokedCandidates(t *testing.T) {
	r := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: r.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	main := &service.Group{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: 21, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{repo: NewRepository(db), redis: client, groups: &routingGroupRepoStub{groups: map[int64]*service.Group{2: main, 21: candidate}}, availableGroups: allowRoutingGroups(2, 21)}
	mainID := int64(2)
	key := &service.APIKey{ID: 9, UserID: 7, GroupID: &mainID, Group: main}
	ctx := context.Background()
	require.NoError(t, m.BindRoutingResponse(ctx, key, 2, 21, 12, "resp_owner", time.Hour))
	expectRoutingProfile(mock, 9)
	g, a, err := m.LookupRoutingResponse(ctx, key, "resp_owner")
	require.NoError(t, err)
	require.Equal(t, int64(21), g)
	require.Equal(t, int64(12), a)
	m.availableGroups = allowRoutingGroups(2)
	expectRoutingProfile(mock, 9)
	_, _, err = m.LookupRoutingResponse(ctx, key, "resp_owner")
	require.Error(t, err)
	m.availableGroups = allowRoutingGroups(2, 21)
	key.UserID = 8
	expectRoutingProfile(mock, 9)
	_, _, err = m.LookupRoutingResponse(ctx, key, "resp_owner")
	require.Error(t, err)
	key.UserID = 7
	candidate.Platform = service.PlatformGemini
	expectRoutingProfile(mock, 9)
	_, _, err = m.LookupRoutingResponse(ctx, key, "resp_owner")
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
