//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/checkinprizeitem"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestBlindBoxServiceDraw_BalanceRewardWritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	enabledPrizeIDs, err := client.CheckinPrizeItem.Query().
		Where(checkinprizeitem.IsEnabled(true), checkinprizeitem.DeletedAtIsNil()).
		IDs(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		for _, id := range enabledPrizeIDs {
			_ = client.CheckinPrizeItem.UpdateOneID(id).SetIsEnabled(true).Exec(context.Background())
		}
	})

	_, err = client.CheckinPrizeItem.Update().SetIsEnabled(false).Save(ctx)
	require.NoError(t, err)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("blindbox-bank-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      10,
	})
	prize, err := client.CheckinPrizeItem.Create().
		SetName("Bank Balance Reward").
		SetRarity(service.RarityRare).
		SetRewardType(service.BlindboxRewardBalance).
		SetRewardValue(4).
		SetRewardValueMax(4).
		SetWeight(100).
		SetIsEnabled(true).
		Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.CheckinPrizeItem.UpdateOneID(prize.ID).SetDeletedAt(time.Now()).Exec(context.Background())
	})
	blindboxService := service.NewBlindBoxService(
		client,
		integrationDB,
		service.NewSettingService(NewSettingRepository(client), nil),
		NewUserRepository(client, integrationDB),
		nil,
		nil,
		NewRedeemCodeRepository(client),
	)

	result, err := blindboxService.Draw(ctx, user.ID, 7)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, service.BlindboxRewardBalance, result.RewardType)
	require.InDelta(t, 4, result.RewardValue, 0.000001)

	requireLegacyUserBalance(t, user.ID, 10)
	requireBankAccountSnapshot(t, user.ID, "14", "0")

	var recordID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
SELECT id
FROM checkin_blindbox_records
WHERE user_id = $1 AND prize_item_id = $2
`, user.ID, prize.ID).Scan(&recordID))

	txID := requireBlindboxBankTransaction(t, user.ID, recordID, "4")
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)
}

func requireBlindboxBankTransaction(t *testing.T, userID int64, recordID int64, wantAmount string) string {
	t.Helper()
	var txID string
	var amount decimal.Decimal
	var referenceType, referenceID sql.NullString
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT tx_id, amount, reference_type, reference_id
FROM transactions_log
WHERE user_id = $1 AND tx_type = $2
`, userID, service.BankTxTypeReward).Scan(&txID, &amount, &referenceType, &referenceID))
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "blindbox amount = %s", amount.String())
	require.Equal(t, "checkin_blindbox_record", referenceType.String)
	require.Equal(t, fmt.Sprintf("%d", recordID), referenceID.String)
	return txID
}
