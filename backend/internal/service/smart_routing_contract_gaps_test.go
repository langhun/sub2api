package service

import (
	"context"
	"errors"
	"testing"
)

type smartRoutingContractActivationResolver struct {
	groupID int64
	calls   int
}

func (r *smartRoutingContractActivationResolver) Resolve(ctx context.Context, _ *APIKey, _, _ string, _ map[int64]struct{}) (*SmartRouteDecision, error) {
	r.calls++
	if state := SmartRoutingStateFromContext(ctx); state != nil {
		state.SmartRoutingActive = true
	}
	return &SmartRouteDecision{GroupID: r.groupID}, nil
}

type smartRoutingContractCountingAccountRepo struct {
	AccountRepository
	listCalls int
}

func (r *smartRoutingContractCountingAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	r.listCalls++
	return nil, nil
}

func smartRoutingContractTestContext(key *APIKey, state *SmartRoutingRequestState) context.Context {
	ctx := WithAPIKeyContext(context.Background(), key)
	return WithSmartRoutingState(ctx, state)
}

func TestGatewaySmartRoutingMissingGroupRepositoryFailsBeforeAccountList(t *testing.T) {
	const primaryID int64 = 4101
	mainGroupID := primaryID
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 4102, GroupID: &mainGroupID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID:          primaryID,
		CurrentGroupID:       primaryID,
		PrimaryGroupSnapshot: primary,
		EndpointType:         "chat_completions",
	}
	resolver := &smartRoutingContractActivationResolver{groupID: primaryID}
	accounts := &smartRoutingContractCountingAccountRepo{}
	svc := &GatewayService{accountRepo: accounts, smartRoutingResolver: resolver}

	_, err := svc.SelectAccountForModelWithExclusions(
		smartRoutingContractTestContext(key, state),
		key.GroupID,
		"session",
		"gpt-5",
		nil,
	)
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("selector error = %v, want ErrNoAvailableAccounts", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want one profile confirmation", resolver.calls)
	}
	if !state.ContractInvalid {
		t.Fatal("missing group repository did not invalidate the active route")
	}
	if accounts.listCalls != 0 {
		t.Fatalf("account repository list calls = %d, want zero", accounts.listCalls)
	}
}

func TestOpenAISmartRoutingMissingGroupRepositoryFailsBeforeAccountList(t *testing.T) {
	const primaryID int64 = 4201
	mainGroupID := primaryID
	primary := &Group{ID: primaryID, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, Status: StatusActive}
	key := &APIKey{ID: 4202, GroupID: &mainGroupID, Group: primary}
	state := &SmartRoutingRequestState{
		MainGroupID:          primaryID,
		CurrentGroupID:       primaryID,
		PrimaryGroupSnapshot: primary,
		EndpointType:         "responses",
	}
	resolver := &smartRoutingContractActivationResolver{groupID: primaryID}
	accounts := &smartRoutingContractCountingAccountRepo{}
	svc := &OpenAIGatewayService{accountRepo: accounts, smartRoutingResolver: resolver}

	_, _, err := svc.SelectAccountWithSchedulerForCapability(
		smartRoutingContractTestContext(key, state),
		key.GroupID,
		"",
		"session",
		"gpt-5",
		nil,
		OpenAIUpstreamTransportAny,
		"",
		false,
		false,
		true,
	)
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("selector error = %v, want ErrNoAvailableAccounts", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want one profile confirmation", resolver.calls)
	}
	if !state.ContractInvalid {
		t.Fatal("missing group repository did not invalidate the active route")
	}
	if accounts.listCalls != 0 {
		t.Fatalf("account repository list calls = %d, want zero", accounts.listCalls)
	}
}
