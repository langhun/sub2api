package repository

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUserEntityToServiceUsesBankAccountBalance(t *testing.T) {
	user := &dbent.User{
		ID:       1,
		Email:    "bank-balance@example.com",
		Balance:  100,
		Role:     service.RoleUser,
		Status:   service.StatusActive,
		Username: "bank-user",
		Edges: dbent.UserEdges{
			BankAccount: &dbent.UserBankAccount{
				ID:      10,
				UserID:  1,
				Balance: decimal.RequireFromString("7.250000000000000000"),
				Status:  service.BankAccountStatusActive,
			},
		},
	}

	got := userEntityToService(user)

	require.NotNil(t, got)
	require.NotNil(t, got.BankAccount)
	require.Equal(t, 7.25, got.Balance)
}

func TestUserEntityToServiceKeepsLegacyBalanceWithoutBankAccount(t *testing.T) {
	user := &dbent.User{
		ID:       2,
		Email:    "legacy-balance@example.com",
		Balance:  12.5,
		Role:     service.RoleUser,
		Status:   service.StatusActive,
		Username: "legacy-user",
	}

	got := userEntityToService(user)

	require.NotNil(t, got)
	require.Nil(t, got.BankAccount)
	require.Equal(t, 12.5, got.Balance)
}
