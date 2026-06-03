//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestBillingEligibility_LoadsMissingBankAccountFromPostgres(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:   fmt.Sprintf("billing-bank-%d@example.com", time.Now().UnixNano()),
		Balance: 12,
	})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})

	svc := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, &config.Config{}, nil)
	svc.SetBankAccountLoader(service.NewBankService(client))
	t.Cleanup(svc.Stop)

	authSnapshot := &service.User{ID: user.ID, Balance: 0}
	err := svc.CheckBillingEligibility(ctx, authSnapshot, nil, nil, nil, "")
	require.NoError(t, err)

	require.NotNil(t, authSnapshot.BankAccount)
	require.True(t, decimal.NewFromInt(12).Equal(authSnapshot.BankAccount.Balance))
	require.Equal(t, 12.0, authSnapshot.Balance)

	var bankBalance decimal.Decimal
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT balance
FROM users_bank_account
WHERE user_id = $1
`, user.ID).Scan(&bankBalance))
	require.True(t, decimal.NewFromInt(12).Equal(bankBalance), "bank balance = %s", bankBalance.String())
}
