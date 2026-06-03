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
	require.True(t, decimal.NewFromInt(-4).Equal(first.Balance))
	require.True(t, decimal.NewFromInt(4).Equal(first.TotalDebt))
	require.True(t, decimal.NewFromInt(4).Equal(first.DebtPrincipal))
	require.Equal(t, first.TxID, replay.TxID)
	requireBankLogCount(t, user.ID, 1)
	requireBankLedgerEntryCount(t, first.TxID.String(), 2)
	requireBankLedgerBalanced(t, first.TxID.String())
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
	requireBankLedgerEntryCountForUser(t, user.ID, 20)
}

func TestBankServiceTransferFundsBatch_WritesSlotBetAndWinAtomically(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "slot-batch", 10)
	_, err := client.UserBankAccount.Create().
		SetUserID(user.ID).
		SetBalance(decimal.NewFromInt(10)).
		SetStatus(service.BankAccountStatusActive).
		Save(ctx)
	require.NoError(t, err)

	bank := service.NewBankService(client)
	results, err := bank.TransferFundsBatch(ctx, []service.TransferFundsRequest{
		{
			UserID:           user.ID,
			Amount:           decimal.NewFromInt(2),
			Type:             service.BankTxTypeSlotBet,
			BusinessModule:   service.BankBusinessModuleGame,
			Description:      "slot bet",
			IdempotencyScope: "test-slot-batch-bet",
			IdempotencyKey:   "test-slot-batch-key-bet",
			ReferenceType:    "game_round",
			ReferenceID:      "round-1",
		},
		{
			UserID:           user.ID,
			Amount:           decimal.NewFromInt(5),
			Type:             service.BankTxTypeSlotWin,
			BusinessModule:   service.BankBusinessModuleGame,
			Description:      "slot win",
			IdempotencyScope: "test-slot-batch-win",
			IdempotencyKey:   "test-slot-batch-key-win",
			ReferenceType:    "game_round",
			ReferenceID:      "round-1",
		},
	})
	require.NoError(t, err)
	require.Len(t, results, 2)

	require.False(t, results[0].Replayed)
	require.False(t, results[1].Replayed)
	require.True(t, decimal.NewFromInt(8).Equal(results[0].Balance), "bet balance = %s", results[0].Balance.String())
	require.True(t, decimal.NewFromInt(13).Equal(results[1].Balance), "win balance = %s", results[1].Balance.String())
	requireBankAccountSnapshot(t, user.ID, "13", "0")
	requireBankTransactionCountByType(t, user.ID, service.BankTxTypeSlotBet, 1)
	requireBankTransactionCountByType(t, user.ID, service.BankTxTypeSlotWin, 1)
	requireBankLedgerEntryCount(t, results[0].TxID.String(), 2)
	requireBankLedgerEntryCount(t, results[1].TxID.String(), 2)
	requireBankLedgerBalanced(t, results[0].TxID.String())
	requireBankLedgerBalanced(t, results[1].TxID.String())
}

func TestBankServiceTransferFundsBatch_SlotBetCannotUseCredit(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "slot-cash-only", 3)
	_, err := client.UserBankAccount.Create().
		SetUserID(user.ID).
		SetBalance(decimal.NewFromInt(3)).
		SetCreditLimit(decimal.NewFromInt(10)).
		SetStatus(service.BankAccountStatusActive).
		Save(ctx)
	require.NoError(t, err)

	bank := service.NewBankService(client)
	_, err = bank.TransferFundsBatch(ctx, []service.TransferFundsRequest{
		{
			UserID:           user.ID,
			Amount:           decimal.NewFromInt(5),
			Type:             service.BankTxTypeSlotBet,
			BusinessModule:   service.BankBusinessModuleGame,
			Description:      "slot bet",
			IdempotencyScope: "test-slot-credit-blocked",
			IdempotencyKey:   "test-slot-credit-blocked-key",
			ReferenceType:    "game_round",
			ReferenceID:      "round-credit-blocked",
		},
	})
	require.ErrorIs(t, err, service.ErrBankInsufficientFunds)
	requireBankAccountSnapshot(t, user.ID, "3", "0")
	requireBankTransactionCountByType(t, user.ID, service.BankTxTypeSlotBet, 0)
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

func requireBankTransactionCountByType(t *testing.T, userID int64, txType string, want int) {
	t.Helper()
	var count int
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM transactions_log
WHERE user_id = $1 AND tx_type = $2
`, userID, txType).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, want, count)
}

func requireBankLedgerEntryCount(t *testing.T, txID string, want int) {
	t.Helper()
	var count int
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM ledger_entries
WHERE tx_id = $1
`, txID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, want, count)
}

func requireBankLedgerEntryCountForUser(t *testing.T, userID int64, want int) {
	t.Helper()
	var count int
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM ledger_entries
WHERE transaction_log_id IN (
    SELECT id
    FROM transactions_log
    WHERE user_id = $1
)
`, userID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, want, count)
}

func requireBankLedgerBalanced(t *testing.T, txID string) {
	t.Helper()
	var balanced bool
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT balanced
FROM ledger_transaction_balances
WHERE tx_id = $1
`, txID).Scan(&balanced)
	require.NoError(t, err)
	require.True(t, balanced)
}
