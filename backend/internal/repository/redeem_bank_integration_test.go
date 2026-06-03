//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRedeemServiceRedeem_PositiveBalanceCodeWritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("redeem-bank-positive-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      10,
	})
	code := mustCreateRedeemCode(t, client, &service.RedeemCode{
		Code:   "REDEEM-BANK-POSITIVE-" + uuid.NewString(),
		Type:   service.RedeemTypeBalance,
		Value:  4.25,
		Status: service.StatusUnused,
	})
	redeemSvc := newRedeemBankIntegrationService(client)

	redeemed, err := redeemSvc.Redeem(ctx, user.ID, code.Code)
	require.NoError(t, err)
	require.Equal(t, service.StatusUsed, redeemed.Status)
	require.NotNil(t, redeemed.UsedBy)
	require.Equal(t, user.ID, *redeemed.UsedBy)

	requireRedeemLegacyBalance(t, user.ID, "10")
	requireBankAccountSnapshot(t, user.ID, "14.25", "0")

	txID := requireRedeemBankTransaction(t, user.ID, service.BankTxTypeReward, "4.25", code.ID, code.Code, 1)
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)
}

func TestRedeemServiceRedeem_NegativeBalanceCodeClampsToCashBalance(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("redeem-bank-negative-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      3,
	})
	code := mustCreateRedeemCode(t, client, &service.RedeemCode{
		Code:   "REDEEM-BANK-NEGATIVE-" + uuid.NewString(),
		Type:   service.RedeemTypeBalance,
		Value:  -5,
		Status: service.StatusUnused,
	})
	redeemSvc := newRedeemBankIntegrationService(client)

	redeemed, err := redeemSvc.Redeem(ctx, user.ID, code.Code)
	require.NoError(t, err)
	require.Equal(t, service.StatusUsed, redeemed.Status)

	requireRedeemLegacyBalance(t, user.ID, "3")
	requireBankAccountSnapshot(t, user.ID, "0", "0")

	txID := requireRedeemBankTransaction(t, user.ID, service.BankTxTypeWithdraw, "-3", code.ID, code.Code, 1)
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)
}

func TestRedeemServiceRedeem_ReusedBalanceCodeDoesNotDuplicateBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("redeem-bank-reuse-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      8,
	})
	code := mustCreateRedeemCode(t, client, &service.RedeemCode{
		Code:   "REDEEM-BANK-REUSE-" + uuid.NewString(),
		Type:   service.RedeemTypeBalance,
		Value:  2,
		Status: service.StatusUnused,
	})
	redeemSvc := newRedeemBankIntegrationService(client)

	_, err := redeemSvc.Redeem(ctx, user.ID, code.Code)
	require.NoError(t, err)

	_, err = redeemSvc.Redeem(ctx, user.ID, code.Code)
	require.ErrorIs(t, err, service.ErrRedeemCodeUsed)

	requireBankAccountSnapshot(t, user.ID, "10", "0")
	requireRedeemBankTransaction(t, user.ID, service.BankTxTypeReward, "2", code.ID, code.Code, 1)
	requireBankLedgerEntryCountForUser(t, user.ID, 2)
}

func newRedeemBankIntegrationService(client *dbent.Client) *service.RedeemService {
	return service.NewRedeemService(
		NewRedeemCodeRepository(client),
		NewUserRepository(client, integrationDB),
		nil,
		nil,
		nil,
		nil,
		client,
		nil,
		nil,
	)
}

func requireRedeemLegacyBalance(t *testing.T, userID int64, want string) {
	t.Helper()
	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT balance
FROM users
WHERE id = $1
`, userID).Scan(&balance))
	wantBalance := decimal.RequireFromString(want).InexactFloat64()
	require.InDelta(t, wantBalance, balance, 0.000001)
}

func requireRedeemBankTransaction(
	t *testing.T,
	userID int64,
	txType string,
	wantAmount string,
	redeemCodeID int64,
	redeemCode string,
	wantCount int,
) string {
	t.Helper()
	var (
		count         int
		txID          string
		amount        decimal.Decimal
		module        string
		referenceType sql.NullString
		referenceID   sql.NullString
	)
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT
    COUNT(*),
    COALESCE(MAX(tx_id::text), ''),
    COALESCE(SUM(amount), 0),
    COALESCE(MAX(business_module), ''),
    MAX(reference_type),
    MAX(reference_id)
FROM transactions_log
WHERE user_id = $1 AND tx_type = $2
`, userID, txType).Scan(&count, &txID, &amount, &module, &referenceType, &referenceID))
	require.Equal(t, wantCount, count)
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "redeem amount = %s", amount.String())
	require.Equal(t, service.BankBusinessModuleSystem, module)
	require.Equal(t, "redeem_code", referenceType.String)
	require.Equal(t, fmt.Sprintf("%d", redeemCodeID), referenceID.String)
	require.NotEmpty(t, redeemCode)
	return txID
}
