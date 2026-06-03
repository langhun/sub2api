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

func TestAuthServiceRegister_WritesSignupBalanceThroughBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	settingRepo := NewSettingRepository(client)
	withTemporarySettings(t, settingRepo, map[string]string{
		service.SettingKeyRegistrationEnabled:                 "true",
		service.SettingKeyAuthSourceDefaultEmailBalance:       "8.5",
		service.SettingKeyAuthSourceDefaultEmailConcurrency:   "4",
		service.SettingKeyAuthSourceDefaultEmailSubscriptions: "[]",
		service.SettingKeyAuthSourceDefaultEmailGrantOnSignup: "true",
	})

	authService := newSignupBalanceBankAuthService(client, settingRepo)
	email := fmt.Sprintf("signup-bank-%d@example.com", time.Now().UnixNano())
	token, user, err := authService.Register(ctx, email, "strong-password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.Equal(t, 8.5, user.Balance)

	requireLegacyUserBalance(t, user.ID, 0)
	requireBankAccountSnapshot(t, user.ID, "8.5", "0")
	txID := requireSignupBalanceBankTransaction(t, user.ID, "email", "8.5", 1)
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)
}

func newSignupBalanceBankAuthService(client *dbent.Client, settingRepo service.SettingRepository) *service.AuthService {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			ExpireHour: 1,
		},
		Default: config.DefaultConfig{
			UserBalance:     0,
			UserConcurrency: 1,
		},
	}
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

func requireSignupBalanceBankTransaction(t *testing.T, userID int64, signupSource, wantAmount string, wantCount int) string {
	t.Helper()
	referenceID := fmt.Sprintf("%d:%s", userID, signupSource)

	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM transactions_log
WHERE user_id = $1 AND reference_type = 'user_signup_balance_grant' AND reference_id = $2
`, userID, referenceID).Scan(&count))
	require.Equal(t, wantCount, count)

	var (
		txID          string
		amount        decimal.Decimal
		txType        string
		module        string
		referenceType sql.NullString
		gotReference  sql.NullString
	)
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT tx_id, amount, tx_type, business_module, reference_type, reference_id
FROM transactions_log
WHERE user_id = $1 AND reference_type = 'user_signup_balance_grant' AND reference_id = $2
`, userID, referenceID).Scan(
		&txID,
		&amount,
		&txType,
		&module,
		&referenceType,
		&gotReference,
	))
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "signup amount = %s", amount.String())
	require.Equal(t, service.BankTxTypeReward, txType)
	require.Equal(t, service.BankBusinessModuleSystem, module)
	require.Equal(t, "user_signup_balance_grant", referenceType.String)
	require.Equal(t, referenceID, gotReference.String)
	return txID
}
