package smartrouting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type routingGroupRepoStub struct {
	service.GroupRepository
	groups map[int64]*service.Group
}

func (s *routingGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*service.Group, error) {
	return s.groups[id], nil
}

type routingAccountRepoStub struct {
	service.AccountRepository
	groups map[int64][]service.Account
	err    error
}

func TestRepositoryGetReturnsEmptyGroupsForLegacyProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectQuery("SELECT api_key_id").WithArgs(int64(41)).WillReturnError(sql.ErrNoRows)
	profile, err := repo.Get(context.Background(), 41)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if profile == nil || profile.Groups == nil || len(profile.Groups) != 0 {
		t.Fatalf("legacy profile groups = %#v, want a non-nil empty slice", profile.Groups)
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
	defer db.Close()

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

func TestRepositorySavePersistsCandidateOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(int64(43)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO custom_api_key_route_profiles").WithArgs(
		int64(43), true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.05,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM custom_api_key_route_groups").WithArgs(int64(43)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("INSERT INTO custom_api_key_route_groups").WithArgs(int64(43), int64(9), 0).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO custom_api_key_route_groups").WithArgs(int64(43), int64(7), 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT api_key_id").WithArgs(int64(43)).WillReturnRows(sqlmock.NewRows([]string{
		"api_key_id", "enabled", "mode", "strategy", "price_weight", "speed_weight", "success_weight", "sticky_ttl_seconds", "exploration_rate", "version",
	}).AddRow(int64(43), true, "smart", "auto", 40.0, 30.0, 30.0, 1800, 0.05, 2))
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
		ExplorationRate: 0.05,
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

func (s *routingAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, _ string) ([]service.Account, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.groups[groupID], nil
}

func TestValidateUpdateWeightsAndCandidates(t *testing.T) {
	valid := UpdateRequest{Mode: "smart", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, StickyTTL: 1800, ExplorationRate: 0.05, GroupIDs: []int64{2}}
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

func TestValidateUpdateRejectsExplorationAboveCap(t *testing.T) {
	req := UpdateRequest{Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, ExplorationRate: 0.06}
	if err := validateUpdate(req); err == nil {
		t.Fatal("expected exploration cap validation error")
	}
}

func TestValidateUpdateRejectsNonFiniteNumbers(t *testing.T) {
	valid := UpdateRequest{Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, ExplorationRate: 0.05}
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
	if routeStateAllowsGroup(state, 3, 0, 2) {
		t.Fatal("expected third group switch to be blocked")
	}
	state.GroupSwitches = 1
	if !routeStateAllowsGroup(state, 3, 0, 2) {
		t.Fatal("expected second group switch to remain allowed")
	}
	state.ContinuityLocked = true
	if routeStateAllowsGroup(state, 3, 2, 2) {
		t.Fatal("continuation must stay on sticky group")
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
		{name: "responses websocket ingress", transport: service.OpenAIUpstreamTransportResponsesWebsocketV2Ingress, allowed: false},
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
		repo:     &Repository{},
		groups:   &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{candidate.ID: {{ID: 10, Status: service.StatusActive, Schedulable: true}}}},
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

func TestRankSkipsPrimaryWhenAccountQueryConfirmsNoAccounts(t *testing.T) {
	mainID := int64(1)
	main := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: 2, Name: "backup", Platform: service.PlatformOpenAI, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:     &Repository{},
		groups:   &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{candidate.ID: {{ID: 10, Status: service.StatusActive, Schedulable: true}}}},
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

func TestRankKeepsPrimaryOnAccountRepositoryError(t *testing.T) {
	mainID := int64(1)
	main := &service.Group{ID: mainID, Name: "primary", Platform: service.PlatformOpenAI, RateMultiplier: 1, Status: service.StatusActive, SubscriptionType: "standard"}
	candidate := &service.Group{ID: 2, Name: "backup", Platform: service.PlatformOpenAI, RateMultiplier: 0.8, Status: service.StatusActive, SubscriptionType: "standard"}
	m := &Module{
		repo:     &Repository{},
		groups:   &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts: &routingAccountRepoStub{err: errors.New("account repository unavailable")},
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
		repo:     &Repository{},
		groups:   &routingGroupRepoStub{groups: map[int64]*service.Group{mainID: main, candidate.ID: candidate}},
		accounts: &routingAccountRepoStub{groups: map[int64][]service.Account{candidate.ID: {{ID: 10, Status: service.StatusActive, Schedulable: true}}}},
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
	profile := &Profile{Version: 1, ExplorationRate: 0.05}
	key := &service.APIKey{ID: 17}
	ranked := []Group{
		{GroupID: 1, Eligible: true, Samples: 100},
		{GroupID: 2, Eligible: true, Samples: 0},
		{GroupID: 3, Eligible: false, Samples: 0},
	}
	// Find a deterministic session that falls inside the 5% bucket. The test
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

func TestRankUsesPersistedOrderForEqualScores(t *testing.T) {
	mainID := int64(1)
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
		repo:     &Repository{},
		groups:   &routingGroupRepoStub{groups: groups},
		accounts: &routingAccountRepoStub{groups: accounts},
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
