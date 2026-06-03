//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestAuthServiceApplyProviderDefaultSettingsOnFirstBind_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	settingRepo := NewSettingRepository(client)
	withTemporarySettings(t, settingRepo, map[string]string{
		service.SettingKeyAuthSourceDefaultEmailBalance:          "8.5",
		service.SettingKeyAuthSourceDefaultEmailConcurrency:      "4",
		service.SettingKeyAuthSourceDefaultEmailSubscriptions:    "[]",
		service.SettingKeyAuthSourceDefaultEmailGrantOnFirstBind: "true",
	})

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("first-bind-bank-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      2.5,
		Concurrency:  1,
	})
	authService := newFirstBindBankAuthService(client, settingRepo)

	require.NoError(t, authService.ApplyProviderDefaultSettingsOnFirstBind(ctx, user.ID, "email"))
	require.NoError(t, authService.ApplyProviderDefaultSettingsOnFirstBind(ctx, user.ID, "email"))

	requireLegacyUserBalance(t, user.ID, 2.5)
	requireBankAccountSnapshot(t, user.ID, "11", "0")
	requireFirstBindProviderGrantCount(t, user.ID, "email", 1)
	txID := requireFirstBindBankTransaction(t, user.ID, "email", "8.5", 1)
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)

	storedUser, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 5, storedUser.Concurrency)
}

func newFirstBindBankAuthService(client *dbent.Client, settingRepo service.SettingRepository) *service.AuthService {
	cfg := &config.Config{}
	settingService := service.NewSettingService(settingRepo, cfg)
	return service.NewAuthService(
		client,
		NewUserRepository(client, integrationDB),
		nil,
		nil,
		cfg,
		settingService,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func requireFirstBindProviderGrantCount(t *testing.T, userID int64, providerType string, want int) {
	t.Helper()
	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM user_provider_default_grants
WHERE user_id = $1 AND provider_type = $2 AND grant_reason = 'first_bind'
`, userID, providerType).Scan(&count))
	require.Equal(t, want, count)
}

func requireFirstBindBankTransaction(t *testing.T, userID int64, providerType, wantAmount string, wantCount int) string {
	t.Helper()
	referenceID := fmt.Sprintf("%d:%s:first_bind", userID, providerType)

	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM transactions_log
WHERE user_id = $1 AND reference_type = 'user_provider_default_grant' AND reference_id = $2
`, userID, referenceID).Scan(&count))
	require.Equal(t, wantCount, count)

	var (
		txID          string
		amount        decimal.Decimal
		txType        string
		module        string
		description   string
		referenceType sql.NullString
		gotReference  sql.NullString
	)
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT tx_id, amount, tx_type, business_module, description, reference_type, reference_id
FROM transactions_log
WHERE user_id = $1 AND reference_type = 'user_provider_default_grant' AND reference_id = $2
`, userID, referenceID).Scan(
		&txID,
		&amount,
		&txType,
		&module,
		&description,
		&referenceType,
		&gotReference,
	))
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "first bind amount = %s", amount.String())
	require.Equal(t, service.BankTxTypeReward, txType)
	require.Equal(t, service.BankBusinessModuleSystem, module)
	require.Contains(t, description, providerType)
	require.Equal(t, "user_provider_default_grant", referenceType.String)
	require.Equal(t, referenceID, gotReference.String)
	return txID
}
