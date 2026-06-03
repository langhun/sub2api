package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestAPIKeyService_RejectsV11AuthSnapshotWithoutBankAccountPrecheck(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-bank-precheck", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  11,
			APIKeyID: 1,
			UserID:   2,
			GroupID:  &groupID,
			Status:   StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:          2,
				Status:      StatusActive,
				Role:        RoleUser,
				Balance:     10,
				Concurrency: 3,
			},
			Group: &APIKeyAuthGroupSnapshot{
				ID:               groupID,
				Name:             "openai",
				Platform:         PlatformOpenAI,
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
				RateMultiplier:   1,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatalf("expected v11 auth snapshot to be rejected after bank account precheck was added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}

func TestAPIKeyService_RoundTripsBankAccountAuthSnapshot(t *testing.T) {
	svc := &APIKeyService{}
	source := &APIKey{
		ID:     1,
		UserID: 2,
		Status: StatusActive,
		User: &User{
			ID:          2,
			Status:      StatusActive,
			Role:        RoleUser,
			Concurrency: 3,
			BankAccount: &BankAccountView{
				AccountID:    8,
				Balance:      decimal.RequireFromString("1.234567890123456789"),
				FrozenAmount: decimal.RequireFromString("0.500000000000000000"),
				CreditLimit:  decimal.RequireFromString("5.000000000000000000"),
				TotalDebt:    decimal.RequireFromString("1.000000000000000000"),
				Status:       BankAccountStatusActive,
			},
		},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), source)
	if snapshot == nil || snapshot.User.BankAccount == nil {
		t.Fatalf("expected bank account snapshot")
	}
	if snapshot.User.BankAccount.Balance != "1.234567890123456789" {
		t.Fatalf("unexpected bank balance snapshot: %s", snapshot.User.BankAccount.Balance)
	}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-bank", &APIKeyAuthCacheEntry{Snapshot: snapshot})
	if err != nil || !ok {
		t.Fatalf("expected auth snapshot round-trip, ok=%v err=%v", ok, err)
	}
	if apiKey.User == nil || apiKey.User.BankAccount == nil {
		t.Fatalf("expected bank account restored from snapshot")
	}
	if !apiKey.User.BankAccount.AvailableCapacity().Equal(decimal.RequireFromString("5.234567890123456789")) {
		t.Fatalf("unexpected available capacity: %s", apiKey.User.BankAccount.AvailableCapacity())
	}
}
