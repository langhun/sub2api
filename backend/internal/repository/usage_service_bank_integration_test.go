//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestUsageServiceCreate_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-service-bank-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      20,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-service-bank-" + uuid.NewString(),
		Name:   "usage-service-bank",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-service-bank-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	usageRepo := newUsageLogRepositoryWithSQL(client, integrationDB)
	userRepo := NewUserRepository(client, integrationDB)
	usageSvc := service.NewUsageService(usageRepo, userRepo, client, nil)
	requestID := "usage-service-bank-" + uuid.NewString()

	req := service.CreateUsageLogRequest{
		UserID:      user.ID,
		APIKeyID:    apiKey.ID,
		AccountID:   account.ID,
		RequestID:   requestID,
		Model:       "gpt-5.1",
		InputTokens: 8,
		ActualCost:  1.5,
		TotalCost:   1.5,
	}

	first, err := usageSvc.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotZero(t, first.ID)

	replay, err := usageSvc.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, replay)
	require.Equal(t, first.ID, replay.ID)

	var legacyBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&legacyBalance))
	require.InDelta(t, 20, legacyBalance, 0.000001)

	requireUsageServiceBankAccount(t, user.ID, "18.5", "0")
	requireUsageServiceBankConsumeLog(t, user.ID, first.ID, requestID, "1.5", 1)
	requireUsageServiceBankLedgerEntries(t, user.ID, 2)

	var usageLogCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM usage_logs
WHERE request_id = $1 AND api_key_id = $2
`, requestID, apiKey.ID).Scan(&usageLogCount))
	require.Equal(t, 1, usageLogCount)
}

func requireUsageServiceBankAccount(t *testing.T, userID int64, wantBalance, wantDebt string) {
	t.Helper()
	var balance, debt decimal.Decimal
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT balance, total_debt
FROM users_bank_account
WHERE user_id = $1
`, userID).Scan(&balance, &debt))
	require.True(t, decimal.RequireFromString(wantBalance).Equal(balance), "bank balance = %s", balance.String())
	require.True(t, decimal.RequireFromString(wantDebt).Equal(debt), "bank debt = %s", debt.String())
}

func requireUsageServiceBankConsumeLog(t *testing.T, userID, usageLogID int64, requestID, wantAbsAmount string, wantCount int) {
	t.Helper()
	var count int
	var amount decimal.Decimal
	var referenceType, referenceID, gotRequestID string
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT
    COUNT(*),
    COALESCE(ABS(SUM(amount)), 0),
    COALESCE(MAX(reference_type), ''),
    COALESCE(MAX(reference_id), ''),
    COALESCE(MAX(request_id), '')
FROM transactions_log
WHERE user_id = $1 AND tx_type = $2
`, userID, service.BankTxTypeConsume).Scan(&count, &amount, &referenceType, &referenceID, &gotRequestID))
	require.Equal(t, wantCount, count)
	require.True(t, decimal.RequireFromString(wantAbsAmount).Equal(amount), "consume amount = %s", amount.String())
	require.Equal(t, "usage_log", referenceType)
	require.Equal(t, fmt.Sprint(usageLogID), referenceID)
	require.Equal(t, requestID, gotRequestID)
}

func requireUsageServiceBankLedgerEntries(t *testing.T, userID int64, wantCount int) {
	t.Helper()
	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM ledger_entries le
JOIN transactions_log tl ON tl.id = le.transaction_log_id
WHERE tl.user_id = $1 AND tl.tx_type = $2
`, userID, service.BankTxTypeConsume).Scan(&count))
	require.Equal(t, wantCount, count)
}
