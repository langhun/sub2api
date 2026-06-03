//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestAdminServiceUpdateUserBalance_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:   fmt.Sprintf("admin-balance-bank-%d@example.com", time.Now().UnixNano()),
		Balance: 10,
	})
	adminSvc := newAdminBalanceBankService(client)

	result, err := adminSvc.UpdateUserBalance(ctx, user.ID, 5, "add", "integration add", "admin-balance-add")
	require.NoError(t, err)
	require.Equal(t, 15.0, result.Balance)

	replay, err := adminSvc.UpdateUserBalance(ctx, user.ID, 5, "add", "integration add", "admin-balance-add")
	require.NoError(t, err)
	require.Equal(t, 15.0, replay.Balance)

	requireLegacyUserBalance(t, user.ID, 10)
	requireBankAccountSnapshot(t, user.ID, "15", "0")
	addTxID := requireAdminBalanceBankTransaction(t, user.ID, service.BankTxTypeDeposit, "5", "admin-balance-add", 1)
	requireBankLedgerEntryCount(t, addTxID, 2)
	requireBankLedgerBalanced(t, addTxID)

	result, err = adminSvc.UpdateUserBalance(ctx, user.ID, 4, "subtract", "integration subtract", "admin-balance-subtract")
	require.NoError(t, err)
	require.Equal(t, 11.0, result.Balance)
	requireLegacyUserBalance(t, user.ID, 10)
	requireBankAccountSnapshot(t, user.ID, "11", "0")
	subtractTxID := requireAdminBalanceBankTransaction(t, user.ID, service.BankTxTypeWithdraw, "-4", "admin-balance-subtract", 1)
	requireBankLedgerEntryCount(t, subtractTxID, 2)
	requireBankLedgerBalanced(t, subtractTxID)

	result, err = adminSvc.UpdateUserBalance(ctx, user.ID, 7, "set", "integration set", "admin-balance-set")
	require.NoError(t, err)
	require.Equal(t, 7.0, result.Balance)
	requireLegacyUserBalance(t, user.ID, 10)
	requireBankAccountSnapshot(t, user.ID, "7", "0")
	setTxID := requireAdminBalanceBankTransaction(t, user.ID, service.BankTxTypeWithdraw, "-4", "admin-balance-set", 1)
	requireBankLedgerEntryCount(t, setTxID, 2)
	requireBankLedgerBalanced(t, setTxID)
}

func newAdminBalanceBankService(client *dbent.Client) service.AdminService {
	return service.NewAdminService(
		NewUserRepository(client, integrationDB),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		NewRedeemCodeRepository(client),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		client,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func requireAdminBalanceBankTransaction(t *testing.T, userID int64, txType, wantAmount, requestID string, wantCount int) string {
	t.Helper()
	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM transactions_log
WHERE user_id = $1 AND idempotency_scope = 'admin_balance_adjustment' AND request_id = $2
`, userID, requestID).Scan(&count))
	require.Equal(t, wantCount, count)

	var (
		txID          string
		amount        decimal.Decimal
		gotTxType     string
		module        string
		referenceType string
	)
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT tx_id, amount, tx_type, business_module, reference_type
FROM transactions_log
WHERE user_id = $1 AND idempotency_scope = 'admin_balance_adjustment' AND request_id = $2
`, userID, requestID).Scan(&txID, &amount, &gotTxType, &module, &referenceType))
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "admin balance amount = %s", amount.String())
	require.Equal(t, txType, gotTxType)
	require.Equal(t, service.BankBusinessModuleSystem, module)
	require.Equal(t, "admin_balance_adjustment", referenceType)
	return txID
}
