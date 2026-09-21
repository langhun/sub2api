package smartrouting

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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
		"samples", "success", "speed", "fallback_samples", "fallback_success", "fallback_speed",
	}).AddRow(samples, success, speed, samples, success, speed))
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
		sqlmock.NewRows([]string{"account_id", "recent_samples", "recent_speed", "recent_error", "fallback_samples", "fallback_speed", "fallback_error"}).
			AddRow(int64(12), int64(3), 120.0, 0.0, int64(minSamples), 640.0, 0.2),
	)
	metrics := m.RoutingAccountMetrics(context.Background(), 2, "gpt-5", "responses", true)
	require.Equal(t, service.SmartRoutingAccountMetric{Samples: minSamples, SpeedMs: 640, ErrorRate: 0.2}, metrics[12])
	require.NoError(t, mock.ExpectationsWereMet())
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
	p := &Profile{Version: 1, ExplorationRate: 0.05}
	groups := []Group{{GroupID: 2, Eligible: true, Samples: 100}, {GroupID: 21, Eligible: true}}
	key := &service.APIKey{ID: 9}
	explored := 0
	for i := 0; i < 10000; i++ {
		if m.explorationIndex(key, p, groups, 2, fmt.Sprintf("request-%d", i), "gpt-5") == 1 {
			explored++
		}
	}
	require.Greater(t, explored, 400)
	require.Less(t, explored, 600)
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
