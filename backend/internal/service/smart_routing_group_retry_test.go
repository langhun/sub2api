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

func (r smartRoutingRetryResolver) Resolve(context.Context, *APIKey, string, string, map[int64]struct{}) (*SmartRouteDecision, error) {
	return &SmartRouteDecision{GroupID: r.groupID}, nil
}

type smartRoutingRetryGroupRepo struct {
	GroupRepository
	group *Group
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
	primaryID  int64
	candidate  *Group
	primaryErr error
}

func (r smartRoutingRestoreGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if id == r.primaryID {
		if r.primaryErr != nil {
			return nil, r.primaryErr
		}
		return &Group{ID: id, Platform: PlatformOpenAI, Status: StatusActive}, nil
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
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, Status: StatusActive}
	key := &APIKey{ID: 903, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{CurrentGroupID: primaryID, EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, nil}}
	service := &GatewayService{
		groupRepo:            smartRoutingRestoreGroupRepo{primaryID: primaryID, candidate: candidate, primaryErr: errors.New("primary temporarily unavailable")},
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
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, Status: StatusActive}
	candidate := &Group{ID: candidateID, Platform: PlatformOpenAI, Status: StatusActive}
	key := &APIKey{ID: 913, GroupID: &primaryID, Group: primary}
	state := &SmartRoutingRequestState{CurrentGroupID: primaryID, EndpointType: "chat_completions", TriedGroups: map[int64]struct{}{}}
	resolver := &smartRoutingSequenceResolver{decisions: []*SmartRouteDecision{{GroupID: candidateID}, nil}}
	groupRepo := smartRoutingRestoreGroupRepo{primaryID: primaryID, candidate: candidate, primaryErr: errors.New("primary temporarily unavailable")}
	service := &OpenAIGatewayService{
		channelService:       &ChannelService{groupRepo: groupRepo},
		smartRoutingResolver: resolver,
		billingCacheService:  &BillingCacheService{cfg: &config.Config{RunMode: config.RunModeSimple}},
	}

	ctx := smartRoutingRestoreContext(key, primaryID, state)
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil); got == nil || *got != candidateID {
		t.Fatalf("first selection group = %v, want candidate %d", got, candidateID)
	}
	if got := service.resolveSmartRoutingGroup(ctx, key.GroupID, "session", "model", nil); got == nil || *got != primaryID {
		t.Fatalf("restore group = %v, want primary %d", got, primaryID)
	}
	if key.GroupID == nil || *key.GroupID != primaryID || key.Group == nil || key.Group.ID != primaryID {
		t.Fatalf("restore left mismatched request key: group_id=%v group=%+v", key.GroupID, key.Group)
	}
}

func (r smartRoutingRetryGroupRepo) GetByIDLite(context.Context, int64) (*Group, error) {
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

type smartRoutingNilResolver struct{}

func (smartRoutingNilResolver) Resolve(context.Context, *APIKey, string, string, map[int64]struct{}) (*SmartRouteDecision, error) {
	return nil, nil
}

func smartRoutingRetryContext(key *APIKey, mainGroupID, currentGroupID int64) context.Context {
	ctx := WithAPIKeyContext(context.Background(), key)
	return WithSmartRoutingState(ctx, &SmartRoutingRequestState{
		MainGroupID:    mainGroupID,
		CurrentGroupID: currentGroupID,
		EndpointType:   "chat_completions",
		TriedGroups:    map[int64]struct{}{currentGroupID: {}},
	})
}

func TestGatewaySmartRoutingKeepsCurrentGroupOnIntraGroupRetry(t *testing.T) {
	primaryID, currentID := int64(101), int64(202)
	key := &APIKey{ID: 303, GroupID: &currentID, Group: &Group{ID: currentID}}
	groupRepo := smartRoutingRetryGroupRepo{group: key.Group}
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

func TestOpenAISmartRoutingKeepsCurrentGroupOnIntraGroupRetry(t *testing.T) {
	primaryID, currentID := int64(111), int64(222)
	key := &APIKey{ID: 333, GroupID: &currentID, Group: &Group{ID: currentID}}
	groupRepo := smartRoutingRetryGroupRepo{group: key.Group}
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
