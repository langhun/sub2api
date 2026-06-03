//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestPromoServiceApplyPromoCode_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("promo-bank-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      10,
	})
	promoRepo := NewPromoCodeRepository(client)
	promoCode := &service.PromoCode{
		Code:        "PROMO-" + uuid.NewString(),
		BonusAmount: 3.5,
		MaxUses:     10,
		Status:      service.PromoCodeStatusActive,
	}
	require.NoError(t, promoRepo.Create(ctx, promoCode))

	promoService := service.NewPromoService(
		promoRepo,
		NewUserRepository(client, integrationDB),
		nil,
		client,
		nil,
	)

	require.NoError(t, promoService.ApplyPromoCode(ctx, user.ID, promoCode.Code))

	var legacyBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT balance
FROM users
WHERE id = $1
`, user.ID).Scan(&legacyBalance))
	require.InDelta(t, 10, legacyBalance, 0.000001)
	requireBankAccountSnapshot(t, user.ID, "13.5", "0")

	var (
		txID          string
		amount        decimal.Decimal
		txType        string
		module        string
		description   string
		referenceType sql.NullString
		referenceID   sql.NullString
	)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT tx_id, amount, tx_type, business_module, description, reference_type, reference_id
FROM transactions_log
WHERE user_id = $1 AND tx_type = $2
`, user.ID, service.BankTxTypeReward).Scan(
		&txID,
		&amount,
		&txType,
		&module,
		&description,
		&referenceType,
		&referenceID,
	))
	require.True(t, decimal.RequireFromString("3.5").Equal(amount), "reward amount = %s", amount.String())
	require.Equal(t, service.BankTxTypeReward, txType)
	require.Equal(t, service.BankBusinessModuleSystem, module)
	require.Contains(t, description, promoCode.Code)
	require.Equal(t, "promo_code", referenceType.String)
	require.Equal(t, fmt.Sprintf("%d", promoCode.ID), referenceID.String)
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)

	var usageCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM promo_code_usages
WHERE promo_code_id = $1 AND user_id = $2
`, promoCode.ID, user.ID).Scan(&usageCount))
	require.Equal(t, 1, usageCount)
}
