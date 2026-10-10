package smartrouting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type routingGroupRepoStub struct {
	service.GroupRepository
	groups map[int64]*service.Group
	err    error
}

func (s *routingGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*service.Group, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.groups[id], nil
}

type routingAccountRepoStub struct {
	service.AccountRepository
	groups map[int64][]service.Account
	err    error
	calls  int
}

func TestRepositoryGetReturnsEmptyGroupsForLegacyProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRepository(db)
	mock.ExpectQuery("SELECT api_key_id").WithArgs(int64(41)).WillReturnError(sql.ErrNoRows)
	profile, err := repo.Get(context.Background(), 41)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if profile == nil || profile.Groups == nil || len(profile.Groups) != 0 {
		t.Fatalf("legacy profile groups = %#v, want a non-nil empty slice", profile.Groups)
	}
	if profile.Version != 0 {
		t.Fatalf("legacy profile version = %d, want 0 sentinel", profile.Version)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRepositoryGetReturnsEmptyGroupsForProfileWithoutCandidates(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRepository(db)
	mock.ExpectQuery("SELECT api_key_id").WithArgs(int64(42)).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(int64(42), false, "single", "auto", 40.0, 30.0, 30.0, 1800, 0.05, 3))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(int64(42)).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}))
	profile, err := repo.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if profile == nil || profile.Groups == nil || len(profile.Groups) != 0 {
		t.Fatalf("profile groups = %#v, want a non-nil empty slice", profile.Groups)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRoutingProfileUsableRejectsMalformedPersistedValues(t *testing.T) {
	base := &Profile{
		Mode:            "smart",
		Strategy:        "auto",
		PriceWeight:     40,
		SpeedWeight:     30,
		SuccessWeight:   30,
		StickyTTL:       1800,
		ExplorationRate: 0.01,
		Groups:          []Group{{GroupID: 7, Enabled: true}},
	}
	if !routingProfileUsable(base) {
		t.Fatal("valid smart profile must be usable")
	}
	cases := []struct {
		name string
		mut  func(*Profile)
	}{
		{name: "nan weight", mut: func(p *Profile) { p.PriceWeight = math.NaN() }},
		{name: "infinite weight", mut: func(p *Profile) { p.SpeedWeight = math.Inf(1) }},
		{name: "wrong sum", mut: func(p *Profile) { p.SuccessWeight = 29 }},
		{name: "negative ttl", mut: func(p *Profile) { p.StickyTTL = -1 }},
		{name: "infinite exploration", mut: func(p *Profile) { p.ExplorationRate = math.Inf(1) }},
		{name: "invalid group id", mut: func(p *Profile) { p.Groups[0].GroupID = 0 }},
		{name: "duplicate group", mut: func(p *Profile) { p.Groups = append(p.Groups, Group{GroupID: 7, Enabled: true}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile := *base
			profile.Groups = append([]Group(nil), base.Groups...)
			tc.mut(&profile)
			if routingProfileUsable(&profile) {
				t.Fatalf("malformed profile should be rejected: %#v", profile)
			}
		})
	}
}

func TestResolveKeepsActiveMarkerWhenRoutingPreflightFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRepository(db)
	mock.ExpectQuery("SELECT api_key_id").WithArgs(int64(46)).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(int64(46), true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.05, 1))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(int64(46)).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}).AddRow(int64(2), 0, true, "backup", "openai"))

	mainID := int64(1)
	main := &service.Group{ID: mainID, Platform: service.PlatformOpenAI}
	state := &service.SmartRoutingRequestState{MainGroupID: mainID, CurrentGroupID: mainID, EndpointType: "chat_completions"}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	m := &Module{repo: repo}
	key := &service.APIKey{ID: 46, GroupID: &mainID, Group: main}

	if _, err := m.Resolve(ctx, key, "gpt-test", "", nil); err == nil {
		t.Fatal("Resolve should report a routing preflight dependency error")
	}
	if !state.SmartRoutingActive {
		t.Fatal("routing must remain active after preflight failure so the scheduler can fail closed")
	}
	if !state.ContractInvalid {
		t.Fatal("routing preflight failure must invalidate the active contract")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRepositorySavePersistsCandidateOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(int64(43)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT version FROM custom_api_key_route_profiles").WithArgs(int64(43)).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO custom_api_key_route_profiles").WithArgs(
		int64(43), true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.01,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM custom_api_key_route_groups").WithArgs(int64(43)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("INSERT INTO custom_api_key_route_groups").WithArgs(int64(43), int64(9), 0).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO custom_api_key_route_groups").WithArgs(int64(43), int64(7), 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT api_key_id").WithArgs(int64(43)).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(int64(43), true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.01, 2))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(int64(43)).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}).AddRow(int64(9), 0, true, "first", "openai").AddRow(int64(7), 1, true, "second", "openai"))

	profile, err := repo.Save(context.Background(), 43, UpdateRequest{
		Enabled:         true,
		Mode:            "smart",
		Strategy:        "auto",
		PriceWeight:     40,
		SpeedWeight:     30,
		SuccessWeight:   30,
		StickyTTL:       1800,
		ExplorationRate: 0.01,
		GroupIDs:        []int64{9, 7},
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if profile == nil || len(profile.Groups) != 2 || profile.Groups[0].GroupID != 9 || profile.Groups[0].SortOrder != 0 || profile.Groups[1].GroupID != 7 || profile.Groups[1].SortOrder != 1 {
		t.Fatalf("saved candidate order = %#v, want [9@0, 7@1]", profile.Groups)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRepositorySaveRejectsVersionZeroWhenProfileExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(int64(44)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT version FROM custom_api_key_route_profiles").WithArgs(int64(44)).WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow(int64(1)),
	)
	mock.ExpectRollback()

	_, err = repo.Save(context.Background(), 44, UpdateRequest{
		Version:         0,
		Mode:            "single",
		Strategy:        "auto",
		PriceWeight:     40,
		SpeedWeight:     30,
		SuccessWeight:   30,
		StickyTTL:       1800,
		ExplorationRate: 0.01,
	})
	if err == nil || err.Error() != "routing profile version conflict" {
		t.Fatalf("Save error = %v, want version conflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRepositorySaveRejectsStaleCreateAndUpdateVersions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(int64(44)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT version FROM custom_api_key_route_profiles").WithArgs(int64(44)).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(2)))
	if _, err := repo.Save(context.Background(), 44, UpdateRequest{
		Version:         0,
		Enabled:         true,
		Mode:            "smart",
		Strategy:        "auto",
		PriceWeight:     40,
		SpeedWeight:     30,
		SuccessWeight:   30,
		StickyTTL:       1800,
		ExplorationRate: 0.01,
		GroupIDs:        []int64{9},
	}); err == nil || !strings.Contains(err.Error(), "version conflict") {
		t.Fatalf("version 0 against existing profile error = %v, want conflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}

	db2, mock2, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New second db: %v", err)
	}
	defer func() { _ = db2.Close() }()
	repo2 := NewRepository(db2)
	mock2.ExpectBegin()
	mock2.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(int64(45)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock2.ExpectQuery("SELECT version FROM custom_api_key_route_profiles").WithArgs(int64(45)).WillReturnError(sql.ErrNoRows)
	if _, err := repo2.Save(context.Background(), 45, UpdateRequest{
		Version:         2,
		Enabled:         true,
		Mode:            "smart",
		Strategy:        "auto",
		PriceWeight:     40,
		SpeedWeight:     30,
		SuccessWeight:   30,
		StickyTTL:       1800,
		ExplorationRate: 0.01,
		GroupIDs:        []int64{9},
	}); err == nil || !strings.Contains(err.Error(), "version conflict") {
		t.Fatalf("nonzero version against missing profile error = %v, want conflict", err)
	}
	if err := mock2.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations for missing profile: %v", err)
	}
}

func (s *routingAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, _ string) ([]service.Account, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.groups[groupID], nil
}

func (s *routingAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, _ []string) ([]service.Account, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.groups[groupID], nil
}

func allowRoutingGroups(ids ...int64) availableGroupsChecker {
	return func(context.Context, int64) ([]service.Group, error) {
		groups := make([]service.Group, 0, len(ids))
		for _, id := range ids {
			groups = append(groups, service.Group{ID: id})
		}
		return groups, nil
	}
}

func TestValidateUpdateWeightsAndCandidates(t *testing.T) {
	valid := UpdateRequest{Mode: "smart", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, StickyTTL: 1800, ExplorationRate: 0.01, GroupIDs: []int64{2}}
	if err := validateUpdate(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	invalid := valid
	invalid.SuccessWeight = 20
	if err := validateUpdate(invalid); err == nil {
		t.Fatal("expected weight validation error")
	}
	invalid = valid
	invalid.GroupIDs = nil
	if err := validateUpdate(invalid); err == nil {
		t.Fatal("expected candidate validation error")
	}
	invalid = valid
	invalid.GroupIDs = []int64{2, 2}
	if err := validateUpdate(invalid); err == nil {
		t.Fatal("expected duplicate candidate validation error")
	}
}

func TestValidateGroupsRejectsCrossPlatformAndSubscriptionCandidates(t *testing.T) {
	primaryID := int64(501)
	primary := &service.Group{
		ID: primaryID, Platform: service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	candidates := []service.Group{
		{ID: 502, Platform: service.PlatformGemini, SubscriptionType: service.SubscriptionTypeStandard},
		{ID: 503, Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard},
		{ID: 504, Platform: service.PlatformComposite, SubscriptionType: service.SubscriptionTypeStandard},
		{ID: 505, Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeSubscription},
	}
	m := &Module{availableGroups: func(context.Context, int64) ([]service.Group, error) {
		return append([]service.Group{{ID: primaryID, Platform: primary.Platform, SubscriptionType: primary.SubscriptionType}}, candidates...), nil
	}}
	key := &service.APIKey{ID: 506, UserID: 7, GroupID: &primaryID, Group: primary}
	for _, candidate := range candidates {
		req := &UpdateRequest{Mode: "smart", GroupIDs: []int64{candidate.ID}}
		if err := m.validateGroups(&gin.Context{Request: httptest.NewRequest(http.MethodPut, "/", nil)}, key, req); err == nil {
			t.Fatalf("candidate %d (%s/%s) was accepted", candidate.ID, candidate.Platform, candidate.SubscriptionType)
		}
	}
}

func TestValidateGroupsFailsClosedWhenPermissionServiceMissing(t *testing.T) {
	primaryID := int64(507)
	m := &Module{}
	key := &service.APIKey{ID: 508, UserID: 7, GroupID: &primaryID, Group: &service.Group{ID: primaryID, Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard}}
	req := &UpdateRequest{Mode: "smart", GroupIDs: []int64{509}}
	if err := m.validateGroups(&gin.Context{Request: httptest.NewRequest(http.MethodPut, "/", nil)}, key, req); err == nil {
		t.Fatal("missing permission service must reject candidate validation")
	}
}

func TestValidateUpdateRejectsExplorationAboveCap(t *testing.T) {
	req := UpdateRequest{Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, ExplorationRate: 0.06}
	if err := validateUpdate(req); err == nil {
		t.Fatal("expected exploration cap validation error")
	}
}

func TestValidateUpdateRejectsNonFiniteNumbers(t *testing.T) {
	valid := UpdateRequest{Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, ExplorationRate: 0.01}
	for name, mutate := range map[string]func(*UpdateRequest){
		"nan price":        func(req *UpdateRequest) { req.PriceWeight = math.NaN() },
		"infinite speed":   func(req *UpdateRequest) { req.SpeedWeight = math.Inf(1) },
		"nan success":      func(req *UpdateRequest) { req.SuccessWeight = math.NaN() },
		"infinite explore": func(req *UpdateRequest) { req.ExplorationRate = math.Inf(1) },
	} {
		t.Run(name, func(t *testing.T) {
			req := valid
			mutate(&req)
			if err := validateUpdate(req); err == nil {
				t.Fatal("expected non-finite value validation error")
			}
		})
	}
}

func TestRouteStateHardLimitsSwitchesAndContinuity(t *testing.T) {
	state := &service.SmartRoutingRequestState{MainGroupID: 1, CurrentGroupID: 2, GroupSwitches: 2, ContinuityLocked: false}
	if routeStateAllowsGroup(state, 3, 2) {
		t.Fatal("expected third group switch to be blocked")
	}
	if !routeStateAllowsGroup(state, 1, 2) {
		t.Fatal("primary group must remain available as the safe fallback at the switch limit")
	}
	state.GroupSwitches = 1
	if !routeStateAllowsGroup(state, 3, 2) {
		t.Fatal("expected second group switch to remain allowed")
	}
	state.ContinuityLocked = true
	state.GroupSwitches = 2
	if routeStateAllowsGroup(state, 3, 2) {
		t.Fatal("continuity-locked request must not switch to another candidate")
	}
	if !routeStateAllowsGroup(state, 1, 2) {
		t.Fatal("continuity-locked request must keep the primary group as its continuity target")
	}
}

func TestSmartRoutingTransportAllowsHTTPAnyAndHTTPSSE(t *testing.T) {
	tests := []struct {
		name      string
		transport service.OpenAIUpstreamTransport
		allowed   bool
	}{
		{name: "unspecified", transport: "", allowed: true},
		{name: "http any", transport: service.OpenAIUpstreamTransportAny, allowed: true},
		{name: "http sse", transport: service.OpenAIUpstreamTransportHTTPSSE, allowed: true},
		{name: "responses websocket", transport: service.OpenAIUpstreamTransportResponsesWebsocketV2, allowed: false},
		{name: "responses websocket ingress", transport: service.OpenAIUpstreamTransportResponsesWebsocketV2Ingress, allowed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := smartRoutingTransportAllowed(tt.transport); got != tt.allowed {
				t.Fatalf("smartRoutingTransportAllowed(%q) = %v, want %v", tt.transport, got, tt.allowed)
			}
		})
	}
}

func TestSmartRoutingPrimaryPlatformScope(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		endpoint string
		allowed  bool
	}{
		{name: "openai chat", platform: service.PlatformOpenAI, endpoint: "chat_completions", allowed: true},
		{name: "openai responses", platform: service.PlatformOpenAI, endpoint: "responses", allowed: true},
		{name: "anthropic messages", platform: service.PlatformAnthropic, endpoint: "messages", allowed: true},
		{name: "anthropic chat excluded", platform: service.PlatformAnthropic, endpoint: "chat_completions", allowed: false},
		{name: "gemini excluded", platform: service.PlatformGemini, endpoint: "chat_completions", allowed: false},
		{name: "grok excluded", platform: service.PlatformGrok, endpoint: "chat_completions", allowed: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := &service.Group{Platform: tt.platform}
			if got := smartRoutingPrimaryPlatformAllowed(group, tt.endpoint); got != tt.allowed {
				t.Fatalf("smartRoutingPrimaryPlatformAllowed(%s, %s) = %v, want %v", tt.platform, tt.endpoint, got, tt.allowed)
			}
		})
	}
}

func TestRankMarksCompatibleCandidateEligible(t *testing.T) {
	mainID := int64(1)
	main := &service.Group{
		ID:               mainID,
		Name:             "primary",
		Platform:         service.PlatformOpenAI,
		RateMultiplier:   1,
		Status:           service.StatusActive,
		SubscriptionType: "standard",
	}
	candidate := &service.Group{
		ID:               2,
		Name:             "backup",
		Platform:         service.PlatformOpenAI,
		RateMultiplier:   0.8,
		Status:           service.StatusActive,
		SubscriptionType: "standard",
	}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(mainID, candidate.ID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts:        &routingAccountRepoStub{groups: map[int64][]service.Account{candidate.ID: {{ID: 10, Status: service.StatusActive, Schedulable: true}}}},
	}
	key := &service.APIKey{ID: 9, GroupID: &mainID, Group: main}
	profile := &Profile{
		Strategy:      "auto",
		PriceWeight:   40,
		SpeedWeight:   30,
		SuccessWeight: 30,
		Groups:        []Group{{GroupID: candidate.ID, Enabled: true}},
	}

	ranked, err := m.rank(context.Background(), key, profile, "", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == candidate.ID {
			if !group.Eligible {
				t.Fatal("compatible candidate must be marked eligible")
			}
			return
		}
	}
	t.Fatalf("rank did not return candidate group: %#v", ranked)
}

func TestRankAllowsHigherPricedCompatibleFallback(t *testing.T) {
	primaryID, candidateID := int64(2001), int64(2002)
	primary := &service.Group{ID: primaryID, Name: "特惠", Platform: service.PlatformOpenAI, RateMultiplier: 0.25, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: candidateID, Name: "Plus", Platform: service.PlatformOpenAI, RateMultiplier: 1.0, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{repo: &Repository{}, availableGroups: allowRoutingGroups(primaryID, candidateID), groups: &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary, candidateID: candidate}}, accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{candidateID: {{ID: 21, Status: service.StatusActive, Schedulable: true}}}}}
	ranked, err := m.rank(context.Background(), &service.APIKey{ID: 20, GroupID: &primaryID, Group: primary}, &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidateID, Enabled: true}}}, "gpt-5", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range ranked {
		if group.GroupID == candidateID && group.Eligible {
			return
		}
	}
	t.Fatal("higher-priced compatible fallback must remain eligible")
}

func TestRobustFastLaneSpeedIgnoresSlowTail(t *testing.T) {
	got := robustFastLaneSpeed([]float64{100, 120, 140, 900})
	if got != 110 {
		t.Fatalf("robust fast-lane speed = %v, want 110", got)
	}
}

func TestAccountAwareGroupSpeedUsesOnlyReliableEligibleAccounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()
	m := &Module{repo: NewRepository(db)}
	state := &service.SmartRoutingRequestState{EndpointType: "responses", Stream: true}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	mock.ExpectQuery("SELECT account_id").WithArgs(int64(2), "gpt-5", "responses", true).WillReturnRows(
		sqlmock.NewRows([]string{"account_id", "recent_samples", "recent_speed_samples", "recent_speed", "recent_speed_p90", "recent_error", "fallback_samples", "fallback_speed_samples", "fallback_speed", "fallback_speed_p90", "fallback_error"}).
			AddRow(int64(11), int64(minSamples), int64(minSamples), 100.0, 120.0, 0.0, int64(minSamples), int64(minSamples), 100.0, 120.0, 0.0).
			AddRow(int64(12), int64(minSamples), int64(minSamples), 200.0, 220.0, 0.0, int64(minSamples), int64(minSamples), 200.0, 220.0, 0.0).
			AddRow(int64(13), int64(1), int64(1), 20.0, 20.0, 0.0, int64(1), int64(1), 20.0, 20.0, 0.0),
	)
	accounts := []service.Account{{ID: 11}, {ID: 12}, {ID: 13}}
	got, samples := m.accountAwareGroupSpeed(ctx, 2, "gpt-5", accounts)
	if got != 108 || samples != minSamples {
		t.Fatalf("account-aware speed = %v samples=%d, want 108 and %d", got, samples, minSamples)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRankScopesLegacyExclusionsToCurrentGroup(t *testing.T) {
	primaryID, candidateID := int64(2101), int64(2102)
	primary := &service.Group{ID: primaryID, Name: "特惠", Platform: service.PlatformOpenAI, RateMultiplier: 0.25, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: candidateID, Name: "Plus", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	sharedAccountID := int64(77)
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(primaryID, candidateID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary, candidateID: candidate}},
		accounts:        &routingAccountRepoStub{groups: map[int64][]service.Account{candidateID: {{ID: sharedAccountID, Status: service.StatusActive, Schedulable: true}}}},
	}
	state := &service.SmartRoutingRequestState{
		SmartRoutingActive:  true,
		MainGroupID:         primaryID,
		CurrentGroupID:      primaryID,
		FailedAccountGroups: map[int64]map[int64]struct{}{sharedAccountID: {primaryID: {}}},
		RequestedModel:      "gpt-5",
		EndpointType:        "responses",
	}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidateID, Enabled: true}}}
	ranked, err := m.rank(ctx, &service.APIKey{ID: 21, UserID: 7, GroupID: &primaryID, Group: primary}, profile, "gpt-5", map[int64]struct{}{sharedAccountID: {}})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == candidateID {
			if !group.Eligible {
				t.Fatalf("candidate with same account ID must remain eligible after primary failure: %#v", group)
			}
			return
		}
	}
	t.Fatalf("rank did not return candidate group: %#v", ranked)
}

func TestRankSkipsPermissionRejectedCandidate(t *testing.T) {
	primaryID, candidateID := int64(2111), int64(2112)
	primary := &service.Group{ID: primaryID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: candidateID, Name: "candidate", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(primaryID, candidateID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary, candidateID: candidate}},
		accounts:        &routingAccountRepoStub{groups: map[int64][]service.Account{candidateID: {{ID: 78, Status: service.StatusActive, Schedulable: true}}}},
	}
	state := &service.SmartRoutingRequestState{
		SmartRoutingActive:       true,
		MainGroupID:              primaryID,
		CurrentGroupID:           primaryID,
		PermissionRejectedGroups: map[int64]struct{}{candidateID: {}},
	}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	ranked, err := m.rank(ctx, &service.APIKey{ID: 22, UserID: 7, GroupID: &primaryID, Group: primary}, &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidateID, Enabled: true}}}, "gpt-5", nil)
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == candidateID && group.Eligible {
			// Rank may retain the row for diagnostics; Resolve must filter it
			// before returning a decision. Verify the request-local rejection is
			// visible to that gate as well.
			if _, rejected := state.PermissionRejectedGroups[group.GroupID]; !rejected {
				t.Fatal("permission-rejected candidate lost its request-local rejection")
			}
			return
		}
	}
	t.Fatal("permission-rejected candidate was unexpectedly removed from diagnostics")
}

func TestRankExcludedAccountsPreservesGroupScope(t *testing.T) {
	const (
		primaryID     int64 = 101
		candidateID   int64 = 202
		flatID        int64 = 303
		primaryOnly   int64 = 404
		candidateOnly int64 = 505
	)
	flat := map[int64]struct{}{flatID: {}}

	// Legacy callers and ordinary single-group requests must retain the flat
	// exclusion set unchanged.
	ordinaryCtx := context.Background()
	if got := rankExcludedAccounts(ordinaryCtx, primaryID, flat); !reflect.DeepEqual(got, flat) {
		t.Fatalf("ordinary exclusions = %#v, want %#v", got, flat)
	}

	state := &service.SmartRoutingRequestState{
		SmartRoutingActive: true,
		MainGroupID:        primaryID,
		CurrentGroupID:     primaryID,
		FailedAccountGroups: map[int64]map[int64]struct{}{
			primaryOnly:   {primaryID: {}},
			candidateOnly: {candidateID: {}},
		},
	}
	ctx := service.WithSmartRoutingState(context.Background(), state)

	// A primary-group failure supplied by the legacy scheduler merges with the
	// primary-scoped request failures.
	got := rankExcludedAccounts(ctx, primaryID, flat)
	for _, id := range []int64{flatID, primaryOnly} {
		if _, ok := got[id]; !ok {
			t.Fatalf("primary exclusions missing account %d: %#v", id, got)
		}
	}
	if _, ok := got[candidateOnly]; ok {
		t.Fatalf("candidate-scoped account leaked into primary exclusions: %#v", got)
	}

	// A candidate rank pass must discard the flat primary exclusion while still
	// applying failures recorded specifically in that candidate group.
	got = rankExcludedAccounts(ctx, candidateID, flat)
	if _, ok := got[flatID]; ok {
		t.Fatalf("legacy primary exclusion leaked into candidate exclusions: %#v", got)
	}
	if _, ok := got[candidateOnly]; !ok {
		t.Fatalf("candidate-scoped account missing from candidate exclusions: %#v", got)
	}
}

func TestRankRejectsCrossPlatformAndCompositeCandidates(t *testing.T) {
	mainID := int64(101)
	main := &service.Group{
		ID:               mainID,
		Name:             "openai-primary",
		Platform:         service.PlatformOpenAI,
		RateMultiplier:   1,
		Status:           service.StatusActive,
		SubscriptionType: "standard",
	}
	ids := []int64{102, 103, 104, 105}
	groups := map[int64]*service.Group{mainID: main}
	accounts := map[int64][]service.Account{}
	profileGroups := make([]Group, 0, len(ids))
	for i, id := range ids {
		platform := service.PlatformAnthropic
		switch i {
		case 1:
			platform = service.PlatformGemini
		case 2:
			platform = service.PlatformComposite
		}
		subscriptionType := "standard"
		if i == 3 {
			subscriptionType = "future"
		}
		groups[id] = &service.Group{
			ID:               id,
			Name:             "incompatible",
			Platform:         platform,
			RateMultiplier:   0.5,
			Status:           service.StatusActive,
			SubscriptionType: subscriptionType,
		}
		accounts[id] = []service.Account{{ID: id, Status: service.StatusActive, Schedulable: true}}
		profileGroups = append(profileGroups, Group{GroupID: id, Enabled: true})
	}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(append([]int64{mainID}, ids...)...),
		groups:          &routingGroupRepoStub{groups: groups},
		accounts:        &routingAccountRepoStub{groups: accounts},
	}
	key := &service.APIKey{ID: 1, UserID: 7, GroupID: &mainID, Group: main}
	ranked, err := m.rank(context.Background(), key, &Profile{
		Strategy:      "auto",
		PriceWeight:   40,
		SpeedWeight:   30,
		SuccessWeight: 30,
		Groups:        profileGroups,
	}, "gpt-5", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, candidate := range ranked {
		if candidate.GroupID == mainID {
			continue
		}
		if candidate.Eligible {
			t.Fatalf("cross-platform/composite candidate %d unexpectedly eligible: %#v", candidate.GroupID, candidate)
		}
		if candidate.Reason != "incompatible group" {
			t.Fatalf("candidate %d reason = %q, want incompatible group", candidate.GroupID, candidate.Reason)
		}
	}
}

func TestRankFailsClosedWhenPermissionServiceUnavailable(t *testing.T) {
	mainID := int64(1)
	candidateID := int64(2)
	main := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: candidateID, Name: "backup", Platform: service.PlatformOpenAI, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:     &Repository{},
		groups:   &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidateID: candidate}},
		accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{mainID: {{ID: 1, Status: service.StatusActive, Schedulable: true}}, candidateID: {{ID: 2, Status: service.StatusActive, Schedulable: true}}}},
	}
	key := &service.APIKey{ID: 9, UserID: 42, GroupID: &mainID, Group: main}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidateID, Enabled: true}}}

	ranked, err := m.rank(context.Background(), key, profile, "", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		switch group.GroupID {
		case mainID:
			if !group.Eligible {
				t.Fatalf("primary must remain eligible during permission dependency outage: %#v", group)
			}
		case candidateID:
			if group.Eligible {
				t.Fatal("candidate must be ineligible when permission dependency is unavailable")
			}
			if group.Reason != "permission service unavailable" {
				t.Fatalf("candidate reason = %q, want permission service unavailable", group.Reason)
			}
		}
	}
}

func TestRankSkipsPrimaryWhenAccountQueryConfirmsNoAccounts(t *testing.T) {
	mainID := int64(1)
	main := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: 2, Name: "backup", Platform: service.PlatformOpenAI, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(mainID, candidate.ID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts:        &routingAccountRepoStub{groups: map[int64][]service.Account{candidate.ID: {{ID: 10, Status: service.StatusActive, Schedulable: true}}}},
	}
	key := &service.APIKey{ID: 9, GroupID: &mainID, Group: main}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidate.ID, Enabled: true}}}

	ranked, err := m.rank(context.Background(), key, profile, "", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		switch group.GroupID {
		case mainID:
			if group.Eligible {
				t.Fatal("primary with no schedulable accounts must be ineligible")
			}
		case candidate.ID:
			if !group.Eligible {
				t.Fatalf("candidate with an account must remain eligible: %#v", group)
			}
		}
	}
}

func TestRankReturnsToRecoveredDiscountPrimaryWithoutGroupStickiness(t *testing.T) {
	primaryID, plusID, proID := int64(2101), int64(2102), int64(2103)
	primary := &service.Group{ID: primaryID, Name: "GPT-特惠", Platform: service.PlatformOpenAI, RateMultiplier: 0.08, Status: service.StatusActive, SubscriptionType: "standard"}
	plus := &service.Group{ID: plusID, Name: "GPT-Plus", Platform: service.PlatformOpenAI, RateMultiplier: 0.15, Status: service.StatusActive, SubscriptionType: "standard"}
	pro := &service.Group{ID: proID, Name: "GPT-Pro", Platform: service.PlatformOpenAI, RateMultiplier: 0.25, Status: service.StatusActive, SubscriptionType: "standard"}
	accountRepo := &routingAccountRepoStub{groups: map[int64][]service.Account{
		plusID: {{ID: 22, Status: service.StatusActive, Schedulable: true}},
		proID:  {{ID: 23, Status: service.StatusActive, Schedulable: true}},
	}}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(primaryID, plusID, proID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary, plusID: plus, proID: pro}},
		accounts:        accountRepo,
	}
	key := &service.APIKey{ID: 21, UserID: 7, GroupID: &primaryID, Group: primary}
	profile := &Profile{
		Strategy:      "custom",
		PriceWeight:   60,
		SpeedWeight:   20,
		SuccessWeight: 20,
		Groups: []Group{
			{GroupID: plusID, SortOrder: 0, Enabled: true},
			{GroupID: proID, SortOrder: 1, Enabled: true},
		},
	}

	firstEligible := func() int64 {
		ranked, err := m.rank(context.Background(), key, profile, "gpt-5", map[int64]struct{}{})
		if err != nil {
			t.Fatalf("rank returned error: %v", err)
		}
		for _, group := range ranked {
			if group.Eligible {
				return group.GroupID
			}
		}
		t.Fatal("rank returned no eligible group")
		return 0
	}

	if got := firstEligible(); got != plusID {
		t.Fatalf("route with unavailable primary = %d, want Plus %d", got, plusID)
	}

	// Simulate the primary account becoming schedulable before the next request.
	// The same key/profile is reused: a cross-group sticky binding must not keep
	// the request on Plus after the lower-priced primary has recovered.
	accountRepo.groups[primaryID] = []service.Account{{ID: 21, Status: service.StatusActive, Schedulable: true}}
	if got := firstEligible(); got != primaryID {
		t.Fatalf("route after primary recovery = %d, want discounted primary %d", got, primaryID)
	}
}

func TestRankKeepsPrimaryOnAccountRepositoryError(t *testing.T) {
	mainID := int64(1)
	main := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: 2, Name: "backup", Platform: service.PlatformOpenAI, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(mainID, candidate.ID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts:        &routingAccountRepoStub{err: errors.New("account repository unavailable")},
	}
	key := &service.APIKey{ID: 9, GroupID: &mainID, Group: main}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidate.ID, Enabled: true}}}

	ranked, err := m.rank(context.Background(), key, profile, "", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == mainID && !group.Eligible {
			t.Fatalf("primary must remain eligible when account discovery fails: %#v", group)
		}
		if group.GroupID == candidate.ID && group.Eligible {
			t.Fatal("candidate must not be selected when account discovery fails")
		}
	}
}

func TestRankCountsMixedSchedulingAntigravityAccounts(t *testing.T) {
	mainID := int64(1)
	main := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformAnthropic, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: 2, Name: "backup", Platform: service.PlatformAnthropic, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard"}
	mixedAccount := service.Account{ID: 10, Platform: service.PlatformAntigravity, Status: service.StatusActive, Schedulable: true, Extra: map[string]any{"mixed_scheduling": true}}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(mainID, candidate.ID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts:        &routingAccountRepoStub{groups: map[int64][]service.Account{candidate.ID: {mixedAccount}}},
	}
	key := &service.APIKey{ID: 9, GroupID: &mainID, Group: main}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidate.ID, Enabled: true}}}

	ranked, err := m.rank(context.Background(), key, profile, "", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == candidate.ID {
			if !group.Eligible {
				t.Fatalf("mixed-scheduling candidate must remain eligible: %#v", group)
			}
			return
		}
	}
	t.Fatalf("rank did not return candidate group: %#v", ranked)
}

func TestRankRejectsNonFinitePeakMultiplier(t *testing.T) {
	mainID := int64(1)
	main := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{
		ID:                 2,
		Name:               "backup",
		Platform:           service.PlatformOpenAI,
		RateMultiplier:     0.8,
		Status:             service.StatusActive,
		SubscriptionType:   "standard",
		PeakRateEnabled:    true,
		PeakRateMultiplier: math.NaN(),
	}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(mainID, candidate.ID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts:        &routingAccountRepoStub{groups: map[int64][]service.Account{candidate.ID: {{ID: 10, Status: service.StatusActive, Schedulable: true}}}},
	}
	key := &service.APIKey{ID: 9, GroupID: &mainID, Group: main}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidate.ID, Enabled: true}}}

	ranked, err := m.rank(context.Background(), key, profile, "", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == candidate.ID {
			if group.Eligible {
				t.Fatal("candidate with a non-finite peak multiplier must be ineligible")
			}
			return
		}
	}
	t.Fatalf("rank did not return candidate group: %#v", ranked)
}

func TestRankRejectsLongContextCandidateWhenPrimaryDisabled(t *testing.T) {
	primaryID, candidateID := int64(1), int64(2)
	primary := &service.Group{ID: primaryID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard", LongContextPricingEnabled: false}
	candidate := &service.Group{ID: candidateID, Name: "backup", Platform: service.PlatformOpenAI, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard", LongContextPricingEnabled: true}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(primaryID, candidateID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary, candidateID: candidate}},
		accounts:        &routingAccountRepoStub{groups: map[int64][]service.Account{primaryID: {{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true}}, candidateID: {{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true}}}},
	}
	key := &service.APIKey{ID: 9, GroupID: &primaryID, Group: primary}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidateID, Enabled: true}}}
	ranked, err := m.rank(context.Background(), key, profile, "gpt-5", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == candidateID {
			if group.Eligible || group.Reason != "long-context pricing exceeds primary group" {
				t.Fatalf("candidate = %#v, want long-context price protection rejection", group)
			}
			return
		}
	}
	t.Fatalf("rank did not return candidate group: %#v", ranked)
}

func TestRankRejectsOpenAILongContextAccountOptInWhenPrimaryDisabled(t *testing.T) {
	primaryID, candidateID := int64(1), int64(2)
	primary := &service.Group{ID: primaryID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: candidateID, Name: "backup", Platform: service.PlatformOpenAI, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(primaryID, candidateID),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary, candidateID: candidate}},
		accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{
			primaryID:   {{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true}},
			candidateID: {{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, Extra: map[string]any{"openai_long_context_billing_enabled": true}}},
		}},
	}
	key := &service.APIKey{ID: 9, GroupID: &primaryID, Group: primary}
	profile := &Profile{Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, Groups: []Group{{GroupID: candidateID, Enabled: true}}}
	ranked, err := m.rank(context.Background(), key, profile, "gpt-5", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	for _, group := range ranked {
		if group.GroupID == candidateID {
			if group.Eligible || group.Reason != "long-context pricing exceeds primary group" {
				t.Fatalf("candidate = %#v, want account long-context price protection rejection", group)
			}
			return
		}
	}
	t.Fatalf("rank did not return candidate group: %#v", ranked)
}

func TestLongContextPricingCapabilityRejectsMixedOpenAIAccountFlags(t *testing.T) {
	group := &service.Group{ID: 2, Platform: service.PlatformOpenAI}
	m := &Module{accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{
		group.ID: {
			{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, Extra: map[string]any{"openai_long_context_billing_enabled": true}},
			{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true, Extra: map[string]any{"openai_long_context_billing_enabled": false}},
		},
	}}}
	enabled, known := m.longContextPricingCapability(context.Background(), group)
	if enabled || known {
		t.Fatalf("mixed account flags = enabled:%v known:%v, want unknown disabled", enabled, known)
	}
}

func TestCandidateEndpointAllowedHonorsGroupGates(t *testing.T) {
	state := &service.SmartRoutingRequestState{EndpointType: "responses"}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	if candidateEndpointAllowed(ctx, &service.Group{ClaudeCodeOnly: true}) {
		t.Fatal("Claude-Code-only group must not serve responses")
	}

	state.EndpointType = "messages"
	ctx = service.WithSmartRoutingState(context.Background(), state)
	if candidateEndpointAllowed(ctx, &service.Group{Platform: service.PlatformOpenAI}) {
		t.Fatal("OpenAI group without messages dispatch must be rejected")
	}
	if !candidateEndpointAllowed(ctx, &service.Group{Platform: service.PlatformAnthropic}) {
		t.Fatal("native Anthropic messages group should remain eligible")
	}
}

func TestSmartRoutingDisabledForExplicitForcePlatformRoute(t *testing.T) {
	base := context.Background()
	if smartRoutingForcePlatform(base) {
		t.Fatal("ordinary HTTP context must not be treated as force-platform")
	}
	forced := context.WithValue(base, ctxkey.ForcePlatform, service.PlatformAntigravity)
	if !smartRoutingForcePlatform(forced) {
		t.Fatal("explicit force-platform context must disable smart routing")
	}
	blank := context.WithValue(base, ctxkey.ForcePlatform, "")
	if smartRoutingForcePlatform(blank) {
		t.Fatal("blank force-platform marker must not disable ordinary routing")
	}
}

func TestDispatchModelForGroupUsesCandidateMessagesMapping(t *testing.T) {
	state := &service.SmartRoutingRequestState{EndpointType: "messages"}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	m := &Module{}
	openAIGroup := &service.Group{Platform: service.PlatformOpenAI}
	if got := m.dispatchModelForGroup(ctx, openAIGroup, "claude-opus-4"); got != "gpt-5.4" {
		t.Fatalf("OpenAI Messages candidate model = %q, want gpt-5.4", got)
	}
	anthropicGroup := &service.Group{Platform: service.PlatformAnthropic}
	if got := m.dispatchModelForGroup(ctx, anthropicGroup, "claude-opus-4"); got != "claude-opus-4" {
		t.Fatalf("native Anthropic Messages model = %q, want requested alias", got)
	}
}

func TestExplorationIndexOnlyTargetsColdEligibleCandidates(t *testing.T) {
	profile := &Profile{Version: 1, Strategy: "auto", ExplorationRate: 0.01}
	key := &service.APIKey{ID: 17}
	ranked := []Group{
		{GroupID: 1, Eligible: true, Samples: 100},
		{GroupID: 2, Eligible: true, Samples: 0},
		{GroupID: 3, Eligible: false, Samples: 0},
	}
	// Find a deterministic session that falls inside the 1% bucket. The test
	// does not depend on a particular hash byte so changes to the digest remain
	// harmless.
	found := -1
	for i := 0; i < 10000; i++ {
		idx := (&Module{}).explorationIndex(key, profile, ranked, 1, fmt.Sprintf("session-%d", i), "gpt-test")
		if idx == 1 {
			found = i
			break
		}
	}
	if found < 0 {
		t.Fatal("expected at least one deterministic exploration bucket")
	}
	ranked[1].Samples = minSamples
	if idx := (&Module{}).explorationIndex(key, profile, ranked, 1, fmt.Sprintf("session-%d", found), "gpt-test"); idx >= 0 {
		t.Fatalf("sample-complete candidate should not be explored, got index %d", idx)
	}
}

func TestExplorationIndexDisabledForExplicitStrategies(t *testing.T) {
	key := &service.APIKey{ID: 17}
	ranked := []Group{{GroupID: 1, Eligible: true, Samples: 100}, {GroupID: 2, Eligible: true, Samples: 0}}
	for _, strategy := range []string{"price", "speed", "success", "custom"} {
		p := &Profile{Version: 1, Strategy: strategy, ExplorationRate: 0.01}
		for i := 0; i < 500; i++ {
			if got := (&Module{}).explorationIndex(key, p, ranked, 1, fmt.Sprintf("%s-%d", strategy, i), "gpt-test"); got >= 0 {
				t.Fatalf("strategy %q unexpectedly explored index %d", strategy, got)
			}
		}
	}
}

func TestNormalizeRoutingEndpoint(t *testing.T) {
	tests := map[string]string{
		"chat_completions":     "chat_completions",
		"/v1/chat/completions": "chat_completions",
		"/v1/responses":        "responses",
		"/messages":            "messages",
		"embeddings":           "",
	}
	for input, want := range tests {
		if got := normalizeRoutingEndpoint(input); got != want {
			t.Errorf("normalizeRoutingEndpoint(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSortRankedGroupsHonorsPersistedOrderOnTies(t *testing.T) {
	groups := []Group{
		{GroupID: 9, Eligible: true, Score: 50, SuccessRate: 0.99, SortOrder: 2},
		{GroupID: 8, Eligible: true, Score: 50, SuccessRate: 0.10, SortOrder: 0},
		{GroupID: 7, Eligible: true, Score: 50, SuccessRate: 0.80, SortOrder: 1},
		{GroupID: 6, Eligible: false, Score: 100, SortOrder: 0},
	}
	sortRankedGroups(groups)
	if got := []int64{groups[0].GroupID, groups[1].GroupID, groups[2].GroupID, groups[3].GroupID}; !reflect.DeepEqual(got, []int64{8, 7, 9, 6}) {
		t.Fatalf("sorted group order = %v, want [8 7 9 6]", got)
	}
}

func TestRankSpeedScoreUsesRelativeLatency(t *testing.T) {
	minSpeed, maxSpeed := int64(100), int64(900)
	fast := relativeSpeedScore(100, minSpeed, maxSpeed)
	slow := relativeSpeedScore(900, minSpeed, maxSpeed)
	if fast <= slow || fast != 100 || slow != 0 {
		t.Fatalf("relative speed scores = %.1f/%.1f, want fast=100 slow=0", fast, slow)
	}
}

func TestRankUsesPersistedOrderForEqualScores(t *testing.T) {
	// Use a primary ID greater than every candidate so this test proves the
	// primary's implicit priority instead of passing through the group_id
	// stability fallback.
	mainID := int64(10)
	main := &service.Group{
		ID:               mainID,
		Name:             "primary",
		Platform:         service.PlatformOpenAI,
		RateMultiplier:   1,
		Status:           service.StatusActive,
		SubscriptionType: "standard",
	}
	candidateIDs := []int64{9, 8, 7}
	groups := map[int64]*service.Group{mainID: main}
	accounts := map[int64][]service.Account{mainID: {{ID: 1, Status: service.StatusActive, Schedulable: true}}}
	for _, id := range candidateIDs {
		groups[id] = &service.Group{
			ID:               id,
			Name:             "candidate",
			Platform:         service.PlatformOpenAI,
			RateMultiplier:   1,
			Status:           service.StatusActive,
			SubscriptionType: "standard",
		}
		accounts[id] = []service.Account{{ID: id, Status: service.StatusActive, Schedulable: true}}
	}
	m := &Module{
		repo:            &Repository{},
		availableGroups: allowRoutingGroups(mainID, 7, 8, 9),
		groups:          &routingGroupRepoStub{groups: groups},
		accounts:        &routingAccountRepoStub{groups: accounts},
	}
	key := &service.APIKey{ID: 9, GroupID: &mainID, Group: main}
	profile := &Profile{
		Strategy:      "auto",
		PriceWeight:   40,
		SpeedWeight:   30,
		SuccessWeight: 30,
		Groups: []Group{
			{GroupID: 9, SortOrder: 2, Enabled: true},
			{GroupID: 8, SortOrder: 0, Enabled: true},
			{GroupID: 7, SortOrder: 1, Enabled: true},
		},
	}

	ranked, err := m.rank(context.Background(), key, profile, "", map[int64]struct{}{})
	if err != nil {
		t.Fatalf("rank returned error: %v", err)
	}
	got := make([]int64, 0, len(ranked))
	for _, group := range ranked {
		if group.Eligible {
			got = append(got, group.GroupID)
		}
	}
	if want := []int64{mainID, 8, 7, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("eligible ranked groups = %v, want %v", got, want)
	}
}

func TestResolveFailsClosedAfterRankFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	const keyID int64 = 91
	mock.ExpectQuery("SELECT api_key_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(keyID, true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.05, 1))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}).AddRow(int64(2), 0, true, "backup", service.PlatformOpenAI))

	primaryID := int64(1)
	primary := &service.Group{
		ID:               primaryID,
		Name:             "primary",
		Platform:         service.PlatformOpenAI,
		RateMultiplier:   1,
		Status:           service.StatusActive,
		SubscriptionType: "standard",
	}
	m := &Module{
		repo:   NewRepository(db),
		groups: &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary}},
		availableGroups: func(context.Context, int64) ([]service.Group, error) {
			return nil, errors.New("permission backend unavailable")
		},
	}
	key := &service.APIKey{ID: keyID, UserID: 7, GroupID: &primaryID, Group: primary}
	state := &service.SmartRoutingRequestState{
		MainGroupID:    primaryID,
		CurrentGroupID: primaryID,
		EndpointType:   "chat_completions",
	}
	ctx := service.WithSmartRoutingState(context.Background(), state)

	decision, err := m.Resolve(ctx, key, "gpt-5", "", nil)
	if err == nil {
		t.Fatal("expected rank failure")
	}
	if decision != nil {
		t.Fatalf("decision = %#v, want nil on rank failure", decision)
	}
	if !state.SmartRoutingActive {
		t.Fatal("SmartRoutingActive must remain set until scheduler observes the fail-closed contract marker")
	}
	if !state.ContractInvalid {
		t.Fatal("rank failure after profile activation must invalidate the routing contract")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestResolvePrimaryHydrateFailureMarksActiveContractInvalid(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	const keyID int64 = 92
	mock.ExpectQuery("SELECT api_key_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(keyID, true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.05, 1))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}).AddRow(int64(3), 0, true, "backup", service.PlatformOpenAI))

	primaryID := int64(1)
	key := &service.APIKey{ID: keyID, UserID: 7, GroupID: &primaryID}
	state := &service.SmartRoutingRequestState{
		MainGroupID:    primaryID,
		CurrentGroupID: primaryID,
		EndpointType:   "chat_completions",
	}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	module := &Module{
		repo:            NewRepository(db),
		groups:          &routingGroupRepoStub{err: errors.New("primary lookup failed")},
		availableGroups: allowRoutingGroups(primaryID, 3),
	}

	decision, err := module.Resolve(ctx, key, "gpt-5", "", nil)
	if err == nil {
		t.Fatal("expected primary group hydration error")
	}
	if decision != nil {
		t.Fatalf("decision = %#v, want nil", decision)
	}
	if !state.SmartRoutingActive {
		t.Fatal("profile activation must remain set after primary hydrate failure")
	}
	if !state.ContractInvalid {
		t.Fatal("primary hydrate failure must invalidate the active routing contract")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestResolveMissingSharedGroupAdapterFailsBeforeRanking(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	const keyID int64 = 93
	mock.ExpectQuery("SELECT api_key_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(keyID, true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.05, 1))
	mock.ExpectQuery("SELECT rg.group_id").WithArgs(keyID).WillReturnRows(sqlmock.NewRows([]string{
		"group_id", "sort_order", "enabled", "name", "platform",
	}).AddRow(int64(4), 0, true, "backup", service.PlatformOpenAI))

	primaryID := int64(1)
	primary := &service.Group{
		ID:               primaryID,
		Platform:         service.PlatformOpenAI,
		RateMultiplier:   1,
		Status:           service.StatusActive,
		SubscriptionType: "standard",
	}
	accounts := &routingAccountRepoStub{groups: map[int64][]service.Account{
		primaryID: {{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive, Schedulable: true}},
	}}
	module := &Module{
		repo:            NewRepository(db),
		groups:          &routingGroupRepoStub{groups: map[int64]*service.Group{primaryID: primary}},
		accounts:        accounts,
		availableGroups: allowRoutingGroups(primaryID, 4),
	}
	key := &service.APIKey{ID: keyID, UserID: 7, GroupID: &primaryID, Group: primary}
	state := &service.SmartRoutingRequestState{
		MainGroupID:                         primaryID,
		CurrentGroupID:                      primaryID,
		EndpointType:                        "chat_completions",
		SmartRoutingGroupAdapterUnavailable: true,
	}
	ctx := service.WithSmartRoutingState(context.Background(), state)

	decision, err := module.Resolve(ctx, key, "gpt-5", "", nil)
	if err == nil {
		t.Fatal("expected unavailable group adapter error")
	}
	if decision != nil {
		t.Fatalf("decision = %#v, want nil", decision)
	}
	if !state.SmartRoutingActive || !state.ContractInvalid {
		t.Fatalf("routing state after missing adapter = active:%v invalid:%v, want both true", state.SmartRoutingActive, state.ContractInvalid)
	}
	if accounts.calls != 0 {
		t.Fatalf("account repository calls = %d, want zero before ranking", accounts.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestResolveDisablesSmartRoutingWhenRepositoryUnavailable(t *testing.T) {
	state := &service.SmartRoutingRequestState{SmartRoutingActive: true}
	ctx := service.WithSmartRoutingState(context.Background(), state)
	_, err := (&Module{}).Resolve(ctx, &service.APIKey{}, "gpt-5", "", nil)
	if err == nil {
		t.Fatal("expected repository-unavailable error")
	}
	if state.SmartRoutingActive {
		t.Fatal("SmartRoutingActive must be cleared when repository is unavailable")
	}
}
