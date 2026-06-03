//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestBankServiceTransferFunds_IdempotentConsumeWithCredit(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "idempotent", 3)
	_, err := client.UserBankAccount.Create().
		SetUserID(user.ID).
		SetBalance(decimal.NewFromInt(3)).
		SetCreditLimit(decimal.NewFromInt(10)).
		SetStatus("ACTIVE").
		Save(ctx)
	require.NoError(t, err)

	bank := service.NewBankService(client)
	req := service.TransferFundsRequest{
		UserID:         user.ID,
		Amount:         decimal.NewFromInt(7),
		Type:           service.BankTxTypeConsume,
		Description:    "integration consume",
		IdempotencyKey: "consume-idempotent-key",
	}
	first, err := bank.TransferFunds(ctx, req)
	require.NoError(t, err)
	replay, err := bank.TransferFunds(ctx, req)
	require.NoError(t, err)

	require.False(t, first.Replayed)
	require.True(t, replay.Replayed)
	require.True(t, decimal.Zero.Equal(first.Balance))
	require.True(t, decimal.NewFromInt(4).Equal(first.TotalDebt))
	require.Equal(t, first.TxID, replay.TxID)
	requireBankLogCount(t, user.ID, 1)
}

func TestBankServiceTransferFunds_ConcurrentConsumeNeverOverdrafts(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "concurrent", 10)
	_, err := client.UserBankAccount.Create().
		SetUserID(user.ID).
		SetBalance(decimal.NewFromInt(10)).
		SetStatus("ACTIVE").
		Save(ctx)
	require.NoError(t, err)

	bank := service.NewBankService(client)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := bank.TransferFunds(ctx, service.TransferFundsRequest{
				UserID:         user.ID,
				Amount:         decimal.NewFromInt(1),
				Type:           service.BankTxTypeConsume,
				Description:    "integration concurrent consume",
				IdempotencyKey: fmt.Sprintf("consume-concurrent-%02d", i),
			})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)

	var successCount, insufficientCount int
	for err := range errs {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, service.ErrBankInsufficientFunds):
			insufficientCount++
		default:
			require.NoError(t, err)
		}
	}
	require.Equal(t, 10, successCount)
	require.Equal(t, 10, insufficientCount)
	requireBankAccountSnapshot(t, user.ID, "0", "0")
	requireBankLogCount(t, user.ID, 10)
}

func mustCreateBankServiceUser(t *testing.T, label string, balance float64) *service.User {
	t.Helper()
	user := mustCreateUser(t, testEntClient(t), &service.User{
		Email:   fmt.Sprintf("bank-%s-%d@example.com", label, time.Now().UnixNano()),
		Balance: balance,
	})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})
	return user
}

func requireBankAccountSnapshot(t *testing.T, userID int64, wantBalance, wantDebt string) {
	t.Helper()
	var balance, debt decimal.Decimal
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT balance, total_debt
FROM users_bank_account
WHERE user_id = $1
`, userID).Scan(&balance, &debt)
	require.NoError(t, err)
	require.True(t, decimal.RequireFromString(wantBalance).Equal(balance), "balance = %s", balance.String())
	require.True(t, decimal.RequireFromString(wantDebt).Equal(debt), "debt = %s", debt.String())
}

func requireBankLogCount(t *testing.T, userID int64, want int) {
	t.Helper()
	var count int
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM transactions_log
WHERE user_id = $1
`, userID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, want, count)
}
