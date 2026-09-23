package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type smartRoutingRetryResolver struct {
	groupID int64
}

// smartRoutingSingleRetryAccountRepo only implements the account query used by
// IsSingleAntigravityAccountGroup. Embedding the full interface keeps this
// regression test focused on the group-scoped retry flag rather than requiring
// an unrelated repository fixture.
type smartRoutingSingleRetryAccountRepo struct {
	AccountRepository
	accounts map[int64][]Account
}

func (r smartRoutingSingleRetryAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]Account, error) {
	accounts := r.accounts[groupID]
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == platform && account.IsSchedulable() {
			filtered = append(filtered, account)
		}
	}
	return filtered, nil
}

func (r smartRoutingRetryResolver) Resolve(context.Context, *APIKey, string, string, map[int64]struct{}) (*SmartRouteDecision, error) {
	return &SmartRouteDecision{GroupID: r.groupID}, nil
}

type smartRoutingRetryGroupRepo struct {
	GroupRepository
	group  *Group
	groups map[int64]*Group
}

type smartRoutingSequenceResolver struct {
	decisions []*SmartRouteDecision
	index     int
}

func (r *smartRoutingSequenceResolver) Resolve(context.Context, *APIKey, string, string, map[int64]struct{}) (*SmartRouteDecision, error) {
	if r.index >= len(r.decisions) {
		return nil, nil
	}
	decision := r.decisions[r.index]
	r.index++
	return decision, nil
}

type smartRoutingRestoreGroupRepo struct {
	GroupRepository
	primaryID    int64
	candidate    *Group
	primaryErr   error
	primaryCalls *int
	failAfter    int
}

func (r smartRoutingRestoreGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if id == r.primaryID {
		if r.primaryCalls != nil {
			*r.primaryCalls++
			if r.failAfter > 0 && *r.primaryCalls > r.failAfter && r.primaryErr != nil {
				return nil, r.primaryErr
			}
		} else if r.primaryErr != nil {
			return nil, r.primaryErr
		}
		if r.primaryErr != nil {
			return &Group{ID: id, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}, nil
		}
		return &Group{ID: id, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}, nil
	}
	return r.candidate, nil
}

func smartRoutingRestoreContext(key *APIKey, primaryID int64, state *SmartRoutingRequestState) context.Context {
	ctx := WithAPIKeyContext(context.Background(), key)
	if state == nil {
		state = &SmartRoutingRequestState{EndpointType: "chat_completions"}
	}
	state.MainGroupID = primaryID
	ctx = WithSmartRoutingState(ctx, state)
	return ctx
}

func TestGatewaySmartRoutingRestoreUsesPrimarySnapshotWhenReloadFails(t *testing.T) {
	primaryID, candidateID := int64(901), int64(902)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	primaryCalls := 0
	key := &APIKey{ID: 903, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{CurrentGroupID: primaryID, EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, nil}}
	service := &GatewayService{
		groupRepo:            smartRoutingRestoreGroupRepo{primaryID: primaryID, candidate: candidate, primaryErr: errors.New("primary temporarily unavailable"), primaryCalls: &primaryCalls, failAfter: 4},
		smartRoutingResolver: resolver,
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}

	ctx := smartRoutingRestoreContext(key, primaryID, state)
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil); got == nil || *got != candidateID {
		t.Fatalf("first selection group = %v, want candidate %d", got, candidateID)
	}
	if key.Group == nil || key.Group.ID != candidateID {
		t.Fatalf("first selection did not publish candidate group: %+v", key.Group)
	}

	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil); got == nil || *got != primaryID {
		t.Fatalf("restore group = %v, want primary %d", got, primaryID)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("restore left mismatched request key: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func TestOpenAISmartRoutingRestoreUsesPrimarySnapshotWhenReloadFails(t *testing.T) {
	primaryID, candidateID := int64(911), int64(912)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	primaryCalls := 0
	key := &APIKey{ID: 913, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{CurrentGroupID: primaryID, EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, nil}}
	groupRepo := smartRoutingRestoreGroupRepo{primaryID: primaryID, candidate: candidate, primaryErr: errors.New("primary temporarily unavailable"), primaryCalls: &primaryCalls, failAfter: 4}
	service := &OpenAIGatewayService{
		channelService:       &ChannelService{groupRepo: groupRepo},
		smartRoutingResolver: resolver,
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}

	ctx := smartRoutingRestoreContext(key, primaryID, state)
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil); got == nil || *got != candidateID {
		t.Fatalf("first selection group = %v, want candidate %d", got, candidateID)
	}
	if key.Group == nil || key.Group.ID != candidateID {
		t.Fatalf("first selection did not publish candidate group: %+v", key.Group)
	}

	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil); got == nil || *got != primaryID {
		t.Fatalf("restore group = %v, want primary %d", got, primaryID)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("restore left mismatched request key: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func (r smartRoutingRetryGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if r.groups != nil {
		return r.groups[id], nil
	}
	return r.group, nil
}

type smartRoutingSnapshotRepo struct {
	GroupRepository
	groups map[int64]*Group
	err    error
}

func (r smartRoutingSnapshotRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.groups[id], nil
}

// smartRoutingPrimarySequenceRepo returns successive primary rows so tests can
// model an administrator changing the primary contract between the ingress
// check and candidate admission.
type smartRoutingPrimarySequenceRepo struct {
	GroupRepository
	primaryID   int64
	primaryRows []*Group
	candidates  map[int64]*Group
	calls       int
}

func (r *smartRoutingPrimarySequenceRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if id == r.primaryID {
		index := r.calls
		r.calls++
		if len(r.primaryRows) == 0 {
			return nil, nil
		}
		if index >= len(r.primaryRows) {
			index = len(r.primaryRows) - 1
		}
		return r.primaryRows[index], nil
	}
	return r.candidates[id], nil
}

type smartRoutingNilResolver struct{}

func (smartRoutingNilResolver) Resolve(context.Context, *APIKey, string, string, map[int64]struct{}) (*SmartRouteDecision, error) {
	return nil, nil
}

func smartRoutingRetryContext(key *APIKey, mainGroupID, currentGroupID int64) context.Context {
	ctx := WithAPIKeyContext(context.Background(), key)
	return WithSmartRoutingState(ctx, &SmartRoutingRequestState{
		MainGroupID:          mainGroupID,
		CurrentGroupID:       currentGroupID,
		PrimaryGroupSnapshot: &Group{ID: mainGroupID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard},
		EndpointType:         "chat_completions",
		TriedGroups:          map[int64]struct{}{currentGroupID: {}},
	})
}

func TestSmartRoutingGroupsCompatibleRequiresConcreteSamePlatformAndSubscription(t *testing.T) {
	primary := &Group{ID: 1, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	tests := []struct {
		name      string
		candidate *Group
		want      bool
	}{
		{name: "same platform and billing type", candidate: &Group{ID: 2, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}, want: true},
		{name: "inactive candidate", candidate: &Group{ID: 8, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusDisabled}},
		{name: "gemini candidate", candidate: &Group{ID: 3, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}},
		{name: "anthropic candidate", candidate: &Group{ID: 4, Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}},
		{name: "composite candidate", candidate: &Group{ID: 5, Platform: PlatformComposite, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}},
		{name: "subscription mismatch", candidate: &Group{ID: 6, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeSubscription, Status: StatusActive}},
		{name: "unknown subscription type", candidate: &Group{ID: 9, Platform: PlatformOpenAI, SubscriptionType: "future", Status: StatusActive}},
		{name: "missing platform", candidate: &Group{ID: 7, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SmartRoutingGroupsCompatible(primary, tt.candidate); got != tt.want {
				t.Fatalf("SmartRoutingGroupsCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSmartRoutingGroupsCompatibleRejectsUnknownPrimarySubscriptionType(t *testing.T) {
	primary := &Group{ID: 1, Platform: PlatformOpenAI, SubscriptionType: "", Status: StatusActive}
	candidate := &Group{ID: 2, Platform: PlatformOpenAI, SubscriptionType: "", Status: StatusActive}
	if SmartRoutingGroupsCompatible(primary, candidate) {
		t.Fatal("unknown primary and candidate subscription types must fail closed")
	}
}

func TestSmartRoutingPrimaryRestoreGroupPrefersFrozenContract(t *testing.T) {
	primaryID := int64(51)
	snapshot := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	tests := []struct {
		name    string
		current *Group
		want    *Group
	}{
		{
			name:    "matching fresh row",
			current: &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive},
		},
		{
			name:    "platform changed",
			current: &Group{ID: primaryID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive},
			want:    snapshot,
		},
		{
			name:    "subscription type changed",
			current: &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeSubscription, Status: StatusActive},
			want:    snapshot,
		},
		{
			name:    "primary disabled",
			current: &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusDisabled},
			want:    snapshot,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SmartRoutingPrimaryRestoreGroup(snapshot, tt.current, primaryID)
			if tt.want == nil {
				if got != tt.current {
					t.Fatalf("restore group = %p, want fresh row %p", got, tt.current)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("restore group = %p, want frozen snapshot %p", got, tt.want)
			}
		})
	}

	currentOnly := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	if got := SmartRoutingPrimaryRestoreGroup(nil, currentOnly, primaryID); got != currentOnly {
		t.Fatalf("legacy restore without snapshot = %p, want fresh row %p", got, currentOnly)
	}
}

func TestSmartRoutingResolvedGroupAllowedRequiresFrozenSameContract(t *testing.T) {
	primaryID, candidateID := int64(61), int64(62)
	snapshot := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	state := &SmartRoutingRequestState{MainGroupID: primaryID, PrimaryGroupSnapshot: snapshot, SmartRoutingActive: true}
	ctx := WithSmartRoutingState(context.Background(), state)

	matchingPrimary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	if !SmartRoutingResolvedGroupAllowed(ctx, matchingPrimary, &primaryID) {
		t.Fatal("matching primary contract was rejected")
	}
	changedPrimary := &Group{ID: primaryID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	if SmartRoutingResolvedGroupAllowed(ctx, changedPrimary, &primaryID) {
		t.Fatal("primary platform change was accepted")
	}
	matchingCandidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	if !SmartRoutingResolvedGroupAllowed(ctx, matchingCandidate, &candidateID) {
		t.Fatal("matching candidate contract was rejected")
	}
	geminiCandidate := &Group{ID: candidateID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	if SmartRoutingResolvedGroupAllowed(ctx, geminiCandidate, &candidateID) {
		t.Fatal("Gemini candidate was accepted for an OpenAI request")
	}
	wrongID := candidateID + 1
	if SmartRoutingResolvedGroupAllowed(ctx, matchingCandidate, &wrongID) {
		t.Fatal("hydrated group whose ID differs from the selected ID was accepted")
	}
	state.PrimaryGroupSnapshot = nil
	if SmartRoutingResolvedGroupAllowed(ctx, matchingPrimary, &primaryID) {
		t.Fatal("active smart route without an ingress snapshot must fail closed")
	}
	state.SmartRoutingActive = false
	if !SmartRoutingResolvedGroupAllowed(ctx, changedPrimary, &primaryID) {
		t.Fatal("ordinary single-group request behavior must remain unchanged")
	}
}

func TestSmartRoutingGroupResolutionChangedRejectsImplicitFallbackOnlyWhenActive(t *testing.T) {
	primaryID, candidateID, fallbackID := int64(1), int64(2), int64(3)
	ctx := WithSmartRoutingState(context.Background(), &SmartRoutingRequestState{SmartRoutingActive: true})
	if !SmartRoutingGroupResolutionChanged(ctx, &candidateID, &fallbackID) {
		t.Fatal("active smart routing must reject an implicit fallback-group redirect")
	}
	if SmartRoutingGroupResolutionChanged(ctx, &candidateID, &candidateID) {
		t.Fatal("same selected group must remain allowed")
	}
	state := SmartRoutingStateFromContext(ctx)
	state.SmartRoutingActive = false
	if SmartRoutingGroupResolutionChanged(ctx, &primaryID, &fallbackID) {
		t.Fatal("legacy single-group fallback must remain unchanged when smart routing is inactive")
	}
	if !SmartRoutingGroupResolutionChanged(WithSmartRoutingState(context.Background(), &SmartRoutingRequestState{SmartRoutingActive: true}), &candidateID, nil) {
		t.Fatal("missing resolved group must fail closed for an active smart route")
	}
}

func TestGatewaySmartRoutingKeepsCurrentGroupOnIntraGroupRetry(t *testing.T) {
	primaryID, currentID := int64(101), int64(202)
	key := &APIKey{ID: 303, GroupID: &currentID, Group: &Group{ID: currentID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}}
	groupRepo := smartRoutingRetryGroupRepo{groups: map[int64]*Group{
		primaryID: {ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive},
		currentID: key.Group,
	}}
	service := &GatewayService{
		groupRepo:            groupRepo,
		smartRoutingResolver: smartRoutingRetryResolver{groupID: currentID},
	}

	got := service.resolveSmartRoutingGroup(smartRoutingRetryContext(key, primaryID, currentID), key.GroupID, "session", "model", map[int64]struct{}{404: {}})
	if got == nil || *got != currentID {
		t.Fatalf("current group id = %v, want %d", got, currentID)
	}
	if key.GroupID == nil || *key.GroupID != currentID || key.Group == nil || key.Group.ID != currentID {
		t.Fatalf("request key moved away from current group: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func TestGatewaySmartRoutingRejectsClaudeCodeFallbackRedirect(t *testing.T) {
	primaryID, candidateID, fallbackID := int64(361), int64(362), int64(363)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &fallbackID}
	fallback := &Group{ID: fallbackID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 364, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: primary,
		EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}, SmartRoutingActive: true,
	}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}}}
	svc := &GatewayService{
		groupRepo:            smartRoutingSnapshotRepo{groups: map[int64]*Group{primaryID: primary, candidateID: candidate, fallbackID: fallback}},
		smartRoutingResolver: resolver,
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}
	ctx := smartRoutingRestoreContext(key, primaryID, state)
	_, err := svc.SelectAccountForModelWithExclusions(ctx, key.GroupID, "session", "gpt-5", nil)
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("fallback redirect error = %v, want ErrNoAvailableAccounts", err)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("fallback guard did not restore the primary request key: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func TestGatewaySmartRoutingRestoresPrimaryWhenClaudeCodeCandidateIsRejected(t *testing.T) {
	primaryID, candidateID := int64(371), int64(372)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive, ClaudeCodeOnly: true}
	key := &APIKey{ID: 373, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: primary, EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}, SmartRoutingActive: true}
	service := &GatewayService{
		groupRepo:            smartRoutingSnapshotRepo{groups: map[int64]*Group{primaryID: primary, candidateID: candidate}},
		smartRoutingResolver: &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}}},
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}

	_, err := service.SelectAccountForModelWithExclusions(smartRoutingRestoreContext(key, primaryID, state), key.GroupID, "session", "gpt-5", nil)
	if !errors.Is(err, ErrClaudeCodeOnly) {
		t.Fatalf("Claude-Code candidate error = %v, want ErrClaudeCodeOnly", err)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID || key.Group.Platform != PlatformOpenAI {
		t.Fatalf("candidate pointer survived Claude-Code rejection: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func TestGatewaySmartRoutingRejectsCandidateThatChangesPlatformAfterRanking(t *testing.T) {
	primaryID, candidateID := int64(341), int64(342)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 343, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: primary,
		EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}, SmartRoutingActive: true,
	}
	ctx := smartRoutingRestoreContext(key, primaryID, state)
	service := &GatewayService{
		groupRepo:            smartRoutingSnapshotRepo{groups: map[int64]*Group{primaryID: primary, candidateID: candidate}},
		smartRoutingResolver: &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, nil}},
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}

	got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "gpt-5", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("resolved group = %v, want primary %d", got, primaryID)
	}
	if key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("incompatible candidate remained on request key: %+v", key.Group)
	}
}

func TestGatewaySmartRoutingPrimaryPlatformChangeRestoresFrozenSnapshot(t *testing.T) {
	primaryID, candidateID := int64(1451), int64(1452)
	snapshot := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	changed := &Group{ID: primaryID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 1453, GroupID: &primaryID, Group: snapshot}
	state := &SmartRoutingRequestState{MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: snapshot, SmartRoutingActive: true, TriedGroups: map[int64]struct{}{}}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}}}
	repo := &smartRoutingPrimarySequenceRepo{primaryID: primaryID, primaryRows: []*Group{snapshot, changed, changed}, candidates: map[int64]*Group{candidateID: candidate}}
	service := &GatewayService{groupRepo: repo, smartRoutingResolver: resolver, billingCacheService: &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}}}

	got := service.resolveSmartRoutingGroup(smartRoutingRestoreContext(key, primaryID, state), key.GroupID, "session", "gpt-5", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("resolved group = %v, want primary %d", got, primaryID)
	}
	if resolver.index != 0 {
		t.Fatalf("resolver ran after primary contract changed: calls=%d", resolver.index)
	}
	if key.Group == nil || key.Group.ID != primaryID || key.Group.Platform != PlatformOpenAI {
		t.Fatalf("changed primary platform leaked into request key: %+v", key.Group)
	}
	if !state.ContractInvalid {
		t.Fatal("primary contract change did not invalidate smart routing request")
	}
}

func TestGatewaySmartRoutingPrimaryContractReadFailureFailsClosedAfterActivation(t *testing.T) {
	primaryID := int64(1481)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 1482, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID:          primaryID,
		CurrentGroupID:       primaryID,
		PrimaryGroupSnapshot: primary,
		SmartRoutingActive:   true,
		TriedGroups:          map[int64]struct{}{},
	}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: primaryID}}}
	svc := &GatewayService{groupRepo: smartRoutingSnapshotRepo{err: errors.New("primary lookup unavailable")}, smartRoutingResolver: resolver}

	got := svc.resolveSmartRoutingGroup(smartRoutingRestoreContext(key, primaryID, state), key.GroupID, "session", "gpt-5", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("resolved group = %v, want primary %d", got, primaryID)
	}
	if resolver.index != 0 {
		t.Fatalf("resolver ran despite unavailable primary contract: calls=%d", resolver.index)
	}
	if !state.ContractInvalid {
		t.Fatal("primary contract lookup failure did not fail closed")
	}
}

func TestGatewaySmartRoutingRejectsChangedPrimaryAfterLegacyRehydration(t *testing.T) {
	primaryID, candidateID := int64(1471), int64(1472)
	snapshot := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	changed := &Group{ID: primaryID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 1473, GroupID: &primaryID, Group: snapshot}
	state := &SmartRoutingRequestState{MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: snapshot, SmartRoutingActive: true, TriedGroups: map[int64]struct{}{}}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}}}
	// The first primary read matches ingress, the second detects the edit, and
	// subsequent reads expose the changed row to legacy group hydration.
	repo := &smartRoutingPrimarySequenceRepo{primaryID: primaryID, primaryRows: []*Group{snapshot, changed, changed, changed}, candidates: map[int64]*Group{candidateID: candidate}}
	service := &GatewayService{groupRepo: repo, smartRoutingResolver: resolver, billingCacheService: &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}}}

	_, err := service.SelectAccountForModelWithExclusions(smartRoutingRestoreContext(key, primaryID, state), key.GroupID, "session", "gpt-5", nil)
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("changed primary error = %v, want ErrNoAvailableAccounts", err)
	}
	if resolver.index != 0 {
		t.Fatalf("resolver ran after primary contract changed: calls=%d", resolver.index)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.Platform != PlatformOpenAI {
		t.Fatalf("changed primary leaked through legacy hydration: group_id=%v group=%+v", key.GroupID, key.Group)
	}
	if !state.ContractInvalid {
		t.Fatal("full gateway path did not retain contract-invalid marker")
	}
}

func TestOpenAISmartRoutingPrimaryPlatformChangeRestoresFrozenSnapshot(t *testing.T) {
	primaryID, candidateID := int64(1461), int64(1462)
	snapshot := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	changed := &Group{ID: primaryID, Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 1463, GroupID: &primaryID, Group: snapshot}
	state := &SmartRoutingRequestState{MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: snapshot, SmartRoutingActive: true, TriedGroups: map[int64]struct{}{}}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}}}
	repo := &smartRoutingPrimarySequenceRepo{primaryID: primaryID, primaryRows: []*Group{snapshot, changed, changed}, candidates: map[int64]*Group{candidateID: candidate}}
	service := &OpenAIGatewayService{channelService: &ChannelService{groupRepo: repo}, smartRoutingResolver: resolver, billingCacheService: &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}}}

	got := service.resolveSmartRoutingGroup(smartRoutingRestoreContext(key, primaryID, state), key.GroupID, "session", "gpt-5", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("resolved group = %v, want primary %d", got, primaryID)
	}
	if resolver.index != 0 {
		t.Fatalf("resolver ran after primary contract changed: calls=%d", resolver.index)
	}
	if key.Group == nil || key.Group.ID != primaryID || key.Group.Platform != PlatformOpenAI {
		t.Fatalf("changed primary platform leaked into request key: %+v", key.Group)
	}
	if !state.ContractInvalid {
		t.Fatal("OpenAI resolver did not invalidate changed primary contract")
	}
}

func TestOpenAISmartRoutingPrimaryContractReadFailureFailsClosedAfterActivation(t *testing.T) {
	primaryID := int64(1491)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 1492, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID:          primaryID,
		CurrentGroupID:       primaryID,
		PrimaryGroupSnapshot: primary,
		SmartRoutingActive:   true,
		TriedGroups:          map[int64]struct{}{},
	}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: primaryID}}}
	svc := &OpenAIGatewayService{
		channelService:       &ChannelService{groupRepo: smartRoutingSnapshotRepo{err: errors.New("primary lookup unavailable")}},
		smartRoutingResolver: resolver,
	}

	got := svc.resolveSmartRoutingGroup(smartRoutingRestoreContext(key, primaryID, state), key.GroupID, "session", "gpt-5", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("resolved group = %v, want primary %d", got, primaryID)
	}
	if resolver.index != 0 {
		t.Fatalf("resolver ran despite unavailable primary contract: calls=%d", resolver.index)
	}
	if !state.ContractInvalid {
		t.Fatal("primary contract lookup failure did not fail closed")
	}
}

func TestOpenAISchedulerFailsClosedBeforeAccountListingWhenContractInvalid(t *testing.T) {
	primaryID := int64(1491)
	snapshot := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 1492, GroupID: &primaryID, Group: snapshot}
	state := &SmartRoutingRequestState{
		MainGroupID: primaryID, PrimaryGroupSnapshot: snapshot,
		SmartRoutingActive: true, ContractInvalid: true,
	}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.Background(), key), state)

	selection, _, err := (&OpenAIGatewayService{}).selectAccountWithSchedulerOnce(
		ctx, &primaryID, "", "session", "gpt-5", nil,
		OpenAIUpstreamTransportAny, "", "", false, PlatformOpenAI, false, false,
	)
	if selection != nil || !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("selection=%v error=%v, want fail-closed ErrNoAvailableAccounts", selection, err)
	}
	if key.Group != snapshot || key.GroupID == nil || *key.GroupID != primaryID {
		t.Fatalf("failed scheduler did not preserve primary snapshot: %+v", key)
	}
}

func TestRefreshSingleAccountRetryAfterRoutingUsesCandidatePool(t *testing.T) {
	primaryID, candidateID := int64(1311), int64(1312)
	service := &GatewayService{
		accountRepo: smartRoutingSingleRetryAccountRepo{
			accounts: map[int64][]Account{
				primaryID: {
					{ID: 1, Platform: PlatformAntigravity, Status: StatusActive, Schedulable: true},
				},
				candidateID: {
					{ID: 2, Platform: PlatformAntigravity, Status: StatusActive, Schedulable: true},
					{ID: 3, Platform: PlatformAntigravity, Status: StatusActive, Schedulable: true},
				},
			},
		},
	}

	primaryCtx := WithSingleAccountRetry(context.Background(), true, false)
	candidateCtx := service.refreshSingleAccountRetryAfterRouting(primaryCtx, &primaryID, &candidateID)
	if got, ok := SingleAccountRetryFromContext(candidateCtx); !ok || got {
		t.Fatalf("candidate with multiple antigravity accounts inherited single-account retry: value=%v present=%v", got, ok)
	}

	service.accountRepo = smartRoutingSingleRetryAccountRepo{
		accounts: map[int64][]Account{
			candidateID: {
				{ID: 4, Platform: PlatformAntigravity, Status: StatusActive, Schedulable: true},
			},
		},
	}
	singleCandidateCtx := service.refreshSingleAccountRetryAfterRouting(primaryCtx, &primaryID, &candidateID)
	if got, ok := SingleAccountRetryFromContext(singleCandidateCtx); !ok || !got {
		t.Fatalf("single-account candidate lost group-specific retry mode: value=%v present=%v", got, ok)
	}
}

func TestOpenAISmartRoutingKeepsCurrentGroupOnIntraGroupRetry(t *testing.T) {
	primaryID, currentID := int64(111), int64(222)
	key := &APIKey{ID: 333, GroupID: &currentID, Group: &Group{ID: currentID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}}
	groupRepo := smartRoutingRetryGroupRepo{groups: map[int64]*Group{
		primaryID: {ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive},
		currentID: key.Group,
	}}
	service := &OpenAIGatewayService{
		channelService:       &ChannelService{groupRepo: groupRepo},
		smartRoutingResolver: smartRoutingRetryResolver{groupID: currentID},
	}

	got := service.resolveSmartRoutingGroup(smartRoutingRetryContext(key, primaryID, currentID), key.GroupID, "session", "model", map[int64]struct{}{505: {}})
	if got == nil || *got != currentID {
		t.Fatalf("current group id = %v, want %d", got, currentID)
	}
	if key.GroupID == nil || *key.GroupID != currentID || key.Group == nil || key.Group.ID != currentID {
		t.Fatalf("request key moved away from current group: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func TestGatewaySmartRoutingRechecksPermissionOnSameGroupRetry(t *testing.T) {
	primaryID, candidateID := int64(1511), int64(1512)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive, IsExclusive: true}
	user := &User{ID: 1513, Status: StatusActive, AllowedGroups: []int64{candidateID}}
	key := &APIKey{ID: 1514, UserID: user.ID, User: user, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: primary,
		EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}, SmartRoutingActive: true,
	}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, {GroupID: candidateID}, nil}}
	service := &GatewayService{
		groupRepo:            smartRoutingRetryGroupRepo{groups: map[int64]*Group{primaryID: primary, candidateID: candidate}},
		smartRoutingResolver: resolver,
		userRepo:             &smartRoutingPermissionUserRepoStub{user: user},
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}
	ctx := smartRoutingRestoreContext(key, primaryID, state)
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "gpt-5", nil); got == nil || *got != candidateID {
		t.Fatalf("initial route = %v, want candidate %d", got, candidateID)
	}
	// Revoke access after the candidate has been selected. The next resolver
	// pass must not use the same-group fast path to bypass authorization.
	user.AllowedGroups = nil
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "gpt-5", nil); got == nil || *got != primaryID {
		t.Fatalf("revoked candidate route = %v, want primary %d", got, primaryID)
	}
	if _, ok := state.TriedGroups[candidateID]; !ok {
		t.Fatal("revoked candidate was not marked tried")
	}
	if key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("revoked candidate remained on request key: %+v", key.Group)
	}
}

func TestOpenAISmartRoutingRechecksPermissionOnSameGroupRetry(t *testing.T) {
	primaryID, candidateID := int64(1521), int64(1522)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive, IsExclusive: true}
	user := &User{ID: 1523, Status: StatusActive, AllowedGroups: []int64{candidateID}}
	key := &APIKey{ID: 1524, UserID: user.ID, User: user, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: primary,
		EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}, SmartRoutingActive: true,
	}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, {GroupID: candidateID}, nil}}
	service := &OpenAIGatewayService{
		channelService:       &ChannelService{groupRepo: smartRoutingRetryGroupRepo{groups: map[int64]*Group{primaryID: primary, candidateID: candidate}}},
		smartRoutingResolver: resolver,
		userRepo:             &smartRoutingPermissionUserRepoStub{user: user},
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}
	ctx := smartRoutingRestoreContext(key, primaryID, state)
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "gpt-5", nil); got == nil || *got != candidateID {
		t.Fatalf("initial route = %v, want candidate %d", got, candidateID)
	}
	user.AllowedGroups = nil
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "gpt-5", nil); got == nil || *got != primaryID {
		t.Fatalf("revoked candidate route = %v, want primary %d", got, primaryID)
	}
	if _, ok := state.TriedGroups[candidateID]; !ok {
		t.Fatal("revoked candidate was not marked tried")
	}
	if key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("revoked candidate remained on request key: %+v", key.Group)
	}
}

func TestOpenAISmartRoutingRejectsCompositeCandidateAfterHydration(t *testing.T) {
	primaryID, candidateID := int64(351), int64(352)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformComposite, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 353, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID: primaryID, CurrentGroupID: primaryID, PrimaryGroupSnapshot: primary,
		EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}, SmartRoutingActive: true,
	}
	ctx := smartRoutingRestoreContext(key, primaryID, state)
	service := &OpenAIGatewayService{
		channelService:       &ChannelService{groupRepo: smartRoutingSnapshotRepo{groups: map[int64]*Group{primaryID: primary, candidateID: candidate}}},
		smartRoutingResolver: &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, nil}},
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}

	got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "gpt-5", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("resolved group = %v, want primary %d", got, primaryID)
	}
	if key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("composite candidate remained on request key: %+v", key.Group)
	}
}

func TestGatewaySmartRoutingTOCTOUIncompatibleCandidateFailsClosed(t *testing.T) {
	primaryID, badID, laterID := int64(1401), int64(1402), int64(1403)
	primary := &Group{
		ID:               primaryID,
		Platform:         PlatformOpenAI,
		SubscriptionType: SubscriptionTypeStandard,
		Status:           StatusActive,
	}
	cases := []struct {
		name     string
		badGroup *Group
	}{
		{
			name: "platform changed",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformAnthropic,
				SubscriptionType: SubscriptionTypeStandard,
				Status:           StatusActive,
			},
		},
		{
			name: "subscription type changed",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformOpenAI,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
			},
		},
		{
			name: "composite candidate",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformComposite,
				SubscriptionType: SubscriptionTypeStandard,
				Status:           StatusActive,
			},
		},
		{
			name: "inactive candidate",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformOpenAI,
				SubscriptionType: SubscriptionTypeStandard,
				Status:           StatusDisabled,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := &APIKey{ID: 1404, GroupID: &primaryID, Group: primary}
			state := &SmartRoutingRequestState{
				MainGroupID:    primaryID,
				CurrentGroupID: primaryID,
				TriedGroups:    map[int64]struct{}{},
			}
			resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{
				{GroupID: badID},
				{GroupID: laterID},
			}}
			repo := smartRoutingSnapshotRepo{groups: map[int64]*Group{
				primaryID: primary,
				badID:     tc.badGroup,
				laterID: {
					ID:               laterID,
					Platform:         PlatformOpenAI,
					SubscriptionType: SubscriptionTypeStandard,
					Status:           StatusActive,
				},
			}}
			service := &GatewayService{
				groupRepo:            repo,
				smartRoutingResolver: resolver,
				billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
			}
			ctx := smartRoutingRestoreContext(key, primaryID, state)

			got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil)
			if got == nil || *got != primaryID {
				t.Fatalf("TOCTOU rejection selected group %v, want primary %d", got, primaryID)
			}
			if resolver.index != 1 {
				t.Fatalf("resolver was asked for another candidate after TOCTOU rejection: calls=%d", resolver.index)
			}
			if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
				t.Fatalf("request key was not restored to primary: group_id=%v group=%+v", key.GroupID, key.Group)
			}
		})
	}
}

func TestOpenAISmartRoutingTOCTOUIncompatibleCandidateFailsClosed(t *testing.T) {
	primaryID, badID, laterID := int64(1411), int64(1412), int64(1413)
	primary := &Group{
		ID:               primaryID,
		Platform:         PlatformOpenAI,
		SubscriptionType: SubscriptionTypeStandard,
		Status:           StatusActive,
	}
	cases := []struct {
		name     string
		badGroup *Group
	}{
		{
			name: "platform changed",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformGemini,
				SubscriptionType: SubscriptionTypeStandard,
				Status:           StatusActive,
			},
		},
		{
			name: "subscription type changed",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformOpenAI,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
			},
		},
		{
			name: "composite candidate",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformComposite,
				SubscriptionType: SubscriptionTypeStandard,
				Status:           StatusActive,
			},
		},
		{
			name: "inactive candidate",
			badGroup: &Group{
				ID:               badID,
				Platform:         PlatformOpenAI,
				SubscriptionType: SubscriptionTypeStandard,
				Status:           StatusDisabled,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := &APIKey{ID: 1414, GroupID: &primaryID, Group: primary}
			state := &SmartRoutingRequestState{
				MainGroupID:    primaryID,
				CurrentGroupID: primaryID,
				TriedGroups:    map[int64]struct{}{},
			}
			resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{
				{GroupID: badID},
				{GroupID: laterID},
			}}
			repo := smartRoutingSnapshotRepo{groups: map[int64]*Group{
				primaryID: primary,
				badID:     tc.badGroup,
				laterID: {
					ID:               laterID,
					Platform:         PlatformOpenAI,
					SubscriptionType: SubscriptionTypeStandard,
					Status:           StatusActive,
				},
			}}
			service := &OpenAIGatewayService{
				channelService:       &ChannelService{groupRepo: repo},
				smartRoutingResolver: resolver,
				billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
			}
			ctx := smartRoutingRestoreContext(key, primaryID, state)

			got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil)
			if got == nil || *got != primaryID {
				t.Fatalf("TOCTOU rejection selected group %v, want primary %d", got, primaryID)
			}
			if resolver.index != 1 {
				t.Fatalf("resolver was asked for another candidate after TOCTOU rejection: calls=%d", resolver.index)
			}
			if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
				t.Fatalf("request key was not restored to primary: group_id=%v group=%+v", key.GroupID, key.Group)
			}
		})
	}
}

func TestSmartRoutingGroupExhaustionEnablesCandidateRetry(t *testing.T) {
	mainID, candidateID := int64(1001), int64(1002)
	state := &SmartRoutingRequestState{
		MainGroupID:                mainID,
		CurrentGroupID:             candidateID,
		MaxGroupSwitches:           2,
		SmartRoutingActive:         true,
		TriedGroups:                map[int64]struct{}{},
		SmartRoutingSelectedGroups: map[int64]struct{}{candidateID: {}},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	MarkSmartRoutingGroupExhausted(ctx)
	if !ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("candidate exhaustion should allow a bounded resolver retry")
	}
	if _, ok := state.TriedGroups[candidateID]; !ok {
		t.Fatal("candidate group was not marked exhausted")
	}
}

func TestSmartRoutingGroupExhaustionEnablesPrimaryCandidateRetry(t *testing.T) {
	mainID := int64(1041)
	state := &SmartRoutingRequestState{
		MainGroupID:                mainID,
		CurrentGroupID:             mainID,
		MaxGroupSwitches:           2,
		SmartRoutingActive:         true,
		TriedGroups:                map[int64]struct{}{},
		SmartRoutingSelectedGroups: map[int64]struct{}{},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	MarkSmartRoutingGroupExhausted(ctx)
	if !ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("an active smart route should try candidates after primary exhaustion")
	}
	if ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("primary exhaustion must consume only one resolver retry")
	}
	state.GroupSwitches = state.MaxGroupSwitches
	if ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("primary recovery must stop once the candidate switch budget is spent")
	}
}

func TestSmartRoutingGroupExhaustionAllowsPrimaryRecoveryAtSwitchLimit(t *testing.T) {
	mainID, candidateID := int64(1051), int64(1052)
	state := &SmartRoutingRequestState{
		MainGroupID:                mainID,
		CurrentGroupID:             candidateID,
		GroupSwitches:              2,
		MaxGroupSwitches:           2,
		SmartRoutingActive:         true,
		TriedGroups:                map[int64]struct{}{candidateID: {}},
		SmartRoutingSelectedGroups: map[int64]struct{}{candidateID: {}},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	if !ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("switch-limit boundary must allow one resolver pass for primary recovery")
	}
	state.GroupSwitches++
	if ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("resolver retry must stop after the primary-recovery boundary")
	}
}

func TestSmartRoutingGroupExhaustionDoesNotRetrySingleGroup(t *testing.T) {
	mainID := int64(1101)
	state := &SmartRoutingRequestState{
		MainGroupID:        mainID,
		CurrentGroupID:     mainID,
		SmartRoutingActive: false,
		TriedGroups:        map[int64]struct{}{},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	MarkSmartRoutingGroupExhausted(ctx)
	if ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("single-group requests must keep their immediate no-account behavior")
	}
	if len(state.TriedGroups) != 0 {
		t.Fatal("inactive routing state must not mutate tried groups")
	}
}

func TestTrySmartRoutingGroupFailoverRequiresPreOutputRetryableFailure(t *testing.T) {
	mainID, candidateID := int64(1201), int64(1202)
	state := &SmartRoutingRequestState{
		MainGroupID:                mainID,
		CurrentGroupID:             candidateID,
		MaxGroupSwitches:           2,
		SmartRoutingActive:         true,
		TriedGroups:                map[int64]struct{}{},
		SmartRoutingSelectedGroups: map[int64]struct{}{candidateID: {}},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	retryable := &UpstreamFailoverError{NextAccountAction: NextAccountRetry}

	if !TrySmartRoutingGroupFailover(ctx, retryable, false) {
		t.Fatal("pre-output retryable failure should advance smart routing")
	}
	if _, ok := state.TriedGroups[candidateID]; !ok {
		t.Fatal("failed candidate should be marked exhausted")
	}

	state.CurrentGroupID = candidateID
	state.TriedGroups = map[int64]struct{}{}
	if TrySmartRoutingGroupFailover(ctx, retryable, true) {
		t.Fatal("visible output must prohibit cross-group replay")
	}
	if len(state.TriedGroups) != 0 {
		t.Fatal("visible output must not mutate routing exhaustion state")
	}

	state.TriedGroups = map[int64]struct{}{}
	nonRetryable := &UpstreamFailoverError{NextAccountAction: NextAccountStop}
	if TrySmartRoutingGroupFailover(ctx, nonRetryable, false) {
		t.Fatal("non-retryable failures must not cross groups")
	}
}

func TestTrySmartRoutingGroupFailoverStopsOnCancelledOrContinuityLockedRequest(t *testing.T) {
	mainID, candidateID := int64(1211), int64(1212)
	state := &SmartRoutingRequestState{
		MainGroupID:                mainID,
		CurrentGroupID:             candidateID,
		MaxGroupSwitches:           2,
		SmartRoutingActive:         true,
		ContinuityLocked:           true,
		TriedGroups:                map[int64]struct{}{},
		SmartRoutingSelectedGroups: map[int64]struct{}{candidateID: {}},
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	retryable := &UpstreamFailoverError{NextAccountAction: NextAccountRetry}
	if TrySmartRoutingGroupFailover(ctx, retryable, false) {
		t.Fatal("continuity-locked requests must not cross groups")
	}
	if len(state.TriedGroups) != 0 {
		t.Fatal("continuity lock must not mutate exhaustion state")
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if TrySmartRoutingGroupFailover(cancelled, retryable, false) {
		t.Fatal("cancelled requests must not cross groups")
	}
}

func TestGatewaySmartRoutingRestoresFrozenPrimarySnapshotWhenRepositoryFails(t *testing.T) {
	primaryID, candidateID := int64(701), int64(702)
	primary := &Group{ID: primaryID, Platform: PlatformAnthropic, RateMultiplier: 1}
	key := &APIKey{ID: 703, GroupID: &candidateID, Group: &Group{ID: candidateID, Platform: PlatformAnthropic, RateMultiplier: 9}}
	state := &SmartRoutingRequestState{
		MainGroupID:          primaryID,
		CurrentGroupID:       candidateID,
		PrimaryGroupSnapshot: primary,
		TriedGroups:          map[int64]struct{}{candidateID: {}},
		EndpointType:         "messages",
	}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.Background(), key), state)
	service := &GatewayService{
		groupRepo:            smartRoutingSnapshotRepo{err: context.DeadlineExceeded},
		smartRoutingResolver: smartRoutingNilResolver{},
	}

	got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("restored group id = %v, want %d", got, primaryID)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("request key was not restored from snapshot: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func TestOpenAISmartRoutingRestoresPrimaryWhenPrimaryIDArrivesWithCandidatePointer(t *testing.T) {
	primaryID, candidateID := int64(711), int64(712)
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, RateMultiplier: 1}
	key := &APIKey{ID: 713, GroupID: &primaryID, Group: &Group{ID: candidateID, Platform: PlatformOpenAI, RateMultiplier: 9}}
	state := &SmartRoutingRequestState{
		MainGroupID:          primaryID,
		CurrentGroupID:       candidateID,
		PrimaryGroupSnapshot: primary,
		TriedGroups:          map[int64]struct{}{candidateID: {}},
		EndpointType:         "chat_completions",
	}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.Background(), key), state)
	service := &OpenAIGatewayService{
		channelService:       &ChannelService{groupRepo: smartRoutingSnapshotRepo{err: context.DeadlineExceeded}},
		smartRoutingResolver: smartRoutingNilResolver{},
	}

	got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil)
	if got == nil || *got != primaryID {
		t.Fatalf("restored group id = %v, want %d", got, primaryID)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("candidate group pointer survived restore: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func TestShouldRetrySmartRoutingGroupOnlyForExhaustedCandidate(t *testing.T) {
	primaryID, candidateID := int64(801), int64(802)
	key := &APIKey{ID: 803, GroupID: &candidateID, Group: &Group{ID: candidateID}}
	state := &SmartRoutingRequestState{
		MainGroupID:                primaryID,
		CurrentGroupID:             candidateID,
		MaxGroupSwitches:           2,
		SmartRoutingActive:         true,
		TriedGroups:                map[int64]struct{}{candidateID: {}},
		SmartRoutingSelectedGroups: map[int64]struct{}{candidateID: {}},
	}
	ctx := WithSmartRoutingState(WithAPIKeyContext(context.Background(), key), state)
	if !ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("expected exhausted routed candidate to request another route")
	}
	state.CurrentGroupID = primaryID
	if ShouldRetrySmartRoutingGroup(ctx) {
		t.Fatal("primary group must not enter smart-routing retry")
	}
}
