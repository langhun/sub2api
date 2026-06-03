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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestCheckinServiceCheckin_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	settingRepo := NewSettingRepository(client)
	withTemporarySettings(t, settingRepo, map[string]string{
		service.SettingKeyCheckinEnabled:    "true",
		service.SettingKeyCheckinMinBalance: "2.50",
		service.SettingKeyCheckinMaxBalance: "2.50",
	})
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("checkin-bank-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      10,
	})
	checkinService := newBankCheckinService(client, settingRepo)

	result, err := checkinService.Checkin(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 2.5, result.RewardAmount, 0.000001)

	requireLegacyUserBalance(t, user.ID, 10)
	requireBankAccountSnapshot(t, user.ID, "12.5", "0")
	txID := requireCheckinBankTransaction(t, user.ID, service.BankTxTypeReward, "2.5")
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)

	status, err := checkinService.GetStatus(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 12.5, status.Balance, 0.000001)
}

func TestCheckinServiceLuckCheckin_UsesBankBalanceForBet(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	settingRepo := NewSettingRepository(client)
	withTemporarySettings(t, settingRepo, map[string]string{
		service.SettingKeyCheckinLuckEnabled:       "true",
		service.SettingKeyCheckinLuckMinMultiplier: "0.50",
		service.SettingKeyCheckinLuckMaxMultiplier: "0.50",
	})
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("luck-checkin-bank-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0,
	})
	_, err := client.UserBankAccount.Create().
		SetUserID(user.ID).
		SetBalance(decimal.NewFromInt(5)).
		SetStatus(service.BankAccountStatusActive).
		Save(ctx)
	require.NoError(t, err)
	checkinService := newBankCheckinService(client, settingRepo)

	result, err := checkinService.LuckCheckin(ctx, user.ID, 2)
	require.NoError(t, err)
	require.InDelta(t, -1, result.RewardAmount, 0.000001)

	requireLegacyUserBalance(t, user.ID, 0)
	requireBankAccountSnapshot(t, user.ID, "4", "0")
	txID := requireCheckinBankTransaction(t, user.ID, service.BankTxTypeWithdraw, "-1")
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)
}

func newBankCheckinService(client *dbent.Client, settingRepo service.SettingRepository) *service.CheckinService {
	return service.NewCheckinService(
		client,
		NewUserRepository(client, integrationDB),
		NewRedeemCodeRepository(client),
		service.NewSettingService(settingRepo, nil),
		nil,
		nil,
		nil,
	)
}

func withTemporarySettings(t *testing.T, repo service.SettingRepository, values map[string]string) {
	t.Helper()
	ctx := context.Background()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	previous := make(map[string]sql.NullString, len(keys))
	for _, key := range keys {
		value, err := repo.GetValue(ctx, key)
		if err == nil {
			previous[key] = sql.NullString{String: value, Valid: true}
		} else {
			previous[key] = sql.NullString{}
		}
	}
	require.NoError(t, repo.SetMultiple(ctx, values))
	t.Cleanup(func() {
		for _, key := range keys {
			if old := previous[key]; old.Valid {
				require.NoError(t, repo.Set(ctx, key, old.String))
				continue
			}
			require.NoError(t, repo.Delete(ctx, key))
		}
	})
}

func requireLegacyUserBalance(t *testing.T, userID int64, want float64) {
	t.Helper()
	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT balance
FROM users
WHERE id = $1
`, userID).Scan(&balance))
	require.InDelta(t, want, balance, 0.000001)
}

func requireCheckinBankTransaction(t *testing.T, userID int64, txType string, wantAmount string) string {
	t.Helper()
	var txID string
	var amount decimal.Decimal
	var referenceType, referenceID sql.NullString
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT tx_id, amount, reference_type, reference_id
FROM transactions_log
WHERE user_id = $1 AND tx_type = $2
`, userID, txType).Scan(&txID, &amount, &referenceType, &referenceID))
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "checkin amount = %s", amount.String())
	require.Equal(t, "checkin", referenceType.String)
	require.NotEmpty(t, referenceID.String)
	return txID
}
