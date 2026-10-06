package smartrouting

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// platformCapturingAccountRepo records the platform argument passed to the
// group-1 lookup so tests can verify the hydrated group platform is used.
type platformCapturingAccountRepo struct {
	routingAccountRepoStub
	platformsByID map[int64]string
}

func (s *platformCapturingAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]service.Account, error) {
	if _, seen := s.platformsByID[groupID]; !seen && s.platformsByID == nil {
		s.platformsByID = map[int64]string{}
	}
	if s.platformsByID != nil {
		s.platformsByID[groupID] = platform
	}
	if platform == "" {
		// The real repository filters by platform; an empty platform can only
		// ever match nothing.
		return nil, nil
	}
	return s.routingAccountRepoStub.ListSchedulableByGroupIDAndPlatform(ctx, groupID, platform)
}

func (s *platformCapturingAccountRepo) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]service.Account, error) {
	platform := ""
	if len(platforms) == 1 {
		platform = platforms[0]
	}
	return s.ListSchedulableByGroupIDAndPlatform(ctx, groupID, platform)
}

// TestRankFailsClosedWhenStatePrimaryLookupFails guards the failover baseline:
// during a failover the gateway rewrites key.GroupID to the last candidate, so
// a failed MainGroupID lookup must surface as an error instead of silently
// scoring candidates against the candidate group.
func TestRankFailsClosedWhenStatePrimaryLookupFails(t *testing.T) {
	// mainID only exists in the request state; the repository no longer has it.
	mainID, candidateID := int64(3101), int64(3102)
	candidate := &service.Group{ID: candidateID, Name: "expensive", Platform: service.PlatformOpenAI, RateMultiplier: 2.0, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(mainID, candidateID),
		// The groups stub deliberately omits mainID: the lookup for state.MainGroupID
		// must fail (as it would if the primary group were deleted mid-failover).
		groups:   &routingGroupRepoStub{groups: map[int64]*service.Group{candidateID: candidate}},
		accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{candidateID: {{ID: 31, Status: service.StatusActive, Schedulable: true}}}},
	}
	// Simulate the mid-failover key: GroupID/Group point at the candidate.
	key := &service.APIKey{ID: 9, GroupID: &candidateID, Group: candidate}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: mainID, Enabled: true}, {GroupID: candidateID, Enabled: true}}}
	state := &service.SmartRoutingRequestState{MainGroupID: mainID, CurrentGroupID: candidateID}
	ctx := service.WithSmartRoutingState(context.Background(), state)

	if _, err := m.rank(ctx, key, profile, "", nil); err == nil {
		t.Fatal("rank must fail closed when the state primary group cannot be loaded")
	}
}

// TestResolveKeepsCurrentGroupWhenKeyGroupNil verifies the current-group
// account precheck uses the hydrated candidate platform rather than the
// possibly-nil key.Group pointer, so a healthy current group is not mistaken
// for an exhausted one and force-switched away from.
func TestResolveKeepsCurrentGroupWhenKeyGroupNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	const keyID int64 = 92
	mainID, candidateID := int64(1), int64(2)
	mock.ExpectQuery("SELECT api_key_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(keyID, true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.0, 1))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}).AddRow(int64(2), 0, true, "expensive", service.PlatformOpenAI))
	zeroStats := sqlmock.NewRows([]string{"samples", "success", "speed_samples", "speed", "speed_p90", "fallback_samples", "fallback_success", "fallback_speed_samples", "fallback_speed", "fallback_speed_p90"}).AddRow(0, 0, 0, sql.NullInt64{}, sql.NullInt64{}, 0, 0, 0, sql.NullInt64{}, sql.NullInt64{})
	mock.ExpectQuery("FROM custom_route_attempts").WithArgs(mainID, "gpt-5", "chat_completions", false).WillReturnRows(zeroStats)
	mock.ExpectQuery("FROM custom_route_attempts").WithArgs(candidateID, "gpt-5", "chat_completions", false).WillReturnRows(sqlmock.NewRows([]string{"samples", "success", "speed_samples", "speed", "speed_p90", "fallback_samples", "fallback_success", "fallback_speed_samples", "fallback_speed", "fallback_speed_p90"}).AddRow(0, 0, 0, sql.NullInt64{}, sql.NullInt64{}, 0, 0, 0, sql.NullInt64{}, sql.NullInt64{}))

	primary := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: candidateID, Name: "expensive", Platform: service.PlatformOpenAI, RateMultiplier: 5, Status: service.StatusActive, SubscriptionType: "standard"}
	accounts := &platformCapturingAccountRepo{
		routingAccountRepoStub: routingAccountRepoStub{groups: map[int64][]service.Account{
			mainID:      {{ID: 11, Status: service.StatusActive, Schedulable: true}},
			candidateID: {{ID: 21, Status: service.StatusActive, Schedulable: true}},
		}},
	}
	m := &Module{
		repo:            NewRepository(db),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: primary, candidateID: candidate}},
		availableGroups: allowRoutingGroups(mainID, candidateID),
		accounts:        accounts,
	}
	// key.Group is nil mid-failover (RestoreSmartRoutingPrimarySnapshot may nil it).
	key := &service.APIKey{ID: keyID, UserID: 7, GroupID: &mainID, Group: nil}
	state := &service.SmartRoutingRequestState{
		MainGroupID:    mainID,
		CurrentGroupID: mainID,
		EndpointType:   "chat_completions",
	}
	ctx := service.WithSmartRoutingState(context.Background(), state)

	decision, err := m.Resolve(ctx, key, "gpt-5", "", map[int64]struct{}{100: {}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if accounts.platformsByID[mainID] != service.PlatformOpenAI {
		t.Fatalf("current-group precheck used platform %q, want %q", accounts.platformsByID[mainID], service.PlatformOpenAI)
	}
	if decision == nil || decision.GroupID != mainID {
		t.Fatalf("decision = %#v, want the healthy current group %d", decision, mainID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
