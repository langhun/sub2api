package service

import (
	"context"
	"testing"
)

type preferredOpenAIAccountPolicy map[int64]bool

func (p preferredOpenAIAccountPolicy) PreferOpenAIAccount(_ context.Context, _ *int64, _ string, accountID int64) bool {
	return p[accountID]
}

func TestSelectBestAccountPrefersOptionalSchedulingPolicyAfterEligibility(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.SetOpenAIAccountSchedulingPolicy(preferredOpenAIAccountPolicy{2: true})
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: "active", Schedulable: true, Priority: 1},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: "active", Schedulable: true, Priority: 9},
	}

	selected, compactBlocked := svc.selectBestAccount(context.Background(), nil, PlatformOpenAI, accounts, "gpt-5", nil, false, "", false)
	if compactBlocked {
		t.Fatal("non-compact request must not be compact blocked")
	}
	if selected == nil || selected.ID != 2 {
		t.Fatalf("selected account = %#v, want preferred account 2", selected)
	}
}

func TestOpenAIAccountSchedulingPolicyDoesNotOverrideCompactCapability(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.SetOpenAIAccountSchedulingPolicy(preferredOpenAIAccountPolicy{1: true})
	unsupported := false
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: "active", Schedulable: true, Priority: 1, Extra: map[string]any{"openai_compact_supported": unsupported}},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: "active", Schedulable: true, Priority: 9, Extra: map[string]any{"openai_compact_supported": true}},
	}

	selected, _ := svc.selectBestAccount(context.Background(), nil, PlatformOpenAI, accounts, "gpt-5", nil, true, "", false)
	if selected == nil || selected.ID != 2 {
		t.Fatalf("selected compact account = %#v, want eligible account 2", selected)
	}
}
