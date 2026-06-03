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

func TestBalanceTransferServiceTransfer_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	sender := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("transfer-bank-sender-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      20,
	})
	receiver := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("transfer-bank-receiver-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})
	setTransferBankSettings(t, client)
	transferSvc := service.NewBalanceTransferService(
		NewBalanceTransferRepository(client, integrationDB),
		NewBalanceRedPacketRepository(client, integrationDB),
		NewUserRepository(client, integrationDB),
		service.NewSettingService(NewSettingRepository(client), nil),
	)

	record, err := transferSvc.Transfer(ctx, sender.ID, receiver.ID, 5, nil)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "completed", record.Status)
	require.InDelta(t, 0.5, record.Fee, 0.000001)
	require.InDelta(t, 5.5, record.GrossAmount, 0.000001)

	requireBalanceTransferLegacyBalance(t, sender.ID, "20")
	requireBalanceTransferLegacyBalance(t, receiver.ID, "1")
	requireBankAccountSnapshot(t, sender.ID, "14.5", "0")
	requireBankAccountSnapshot(t, receiver.ID, "6", "0")

	requireBalanceTransferBankTransaction(t, sender.ID, service.BankTxTypeTransferOut, "-5", record.ID, 2)
	requireBalanceTransferBankTransaction(t, sender.ID, service.BankTxTypeWithdraw, "-0.5", record.ID, 2)
	requireBalanceTransferBankTransaction(t, receiver.ID, service.BankTxTypeTransferIn, "5", record.ID, 2)
}

func TestBalanceTransferServiceBatchDistribute_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	admin := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("batch-bank-admin-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleAdmin,
		Balance:      0,
	})
	receiver := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("batch-bank-receiver-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})
	transferSvc := service.NewBalanceTransferService(
		NewBalanceTransferRepository(client, integrationDB),
		NewBalanceRedPacketRepository(client, integrationDB),
		NewUserRepository(client, integrationDB),
		service.NewSettingService(NewSettingRepository(client), nil),
	)

	records, err := transferSvc.BatchDistribute(ctx, admin.ID, []service.BatchDistributeTarget{
		{UserID: receiver.ID, Amount: 3.5},
	}, nil)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "batch", records[0].TransferType)
	require.Equal(t, "completed", records[0].Status)

	requireBalanceTransferLegacyBalance(t, receiver.ID, "1")
	requireBankAccountSnapshot(t, receiver.ID, "4.5", "0")
	requireBalanceTransferBankTransaction(t, receiver.ID, service.BankTxTypeReward, "3.5", records[0].ID, 2)
}

func TestBalanceTransferServiceRedPacketLifecycle_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	sender := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("redpacket-bank-sender-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      20,
	})
	claimant := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("redpacket-bank-claimant-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})
	setRedPacketBankSettings(t, client)
	transferSvc := service.NewBalanceTransferService(
		NewBalanceTransferRepository(client, integrationDB),
		NewBalanceRedPacketRepository(client, integrationDB),
		NewUserRepository(client, integrationDB),
		service.NewSettingService(NewSettingRepository(client), nil),
	)

	rp, err := transferSvc.CreateRedPacket(ctx, sender.ID, 6, 2, "equal", nil)
	require.NoError(t, err)
	require.NotNil(t, rp)
	require.Equal(t, "active", rp.Status)
	requireBalanceTransferLegacyBalance(t, sender.ID, "20")
	requireBankAccountSnapshot(t, sender.ID, "14", "0")
	requireRedPacketBankTransaction(t, sender.ID, service.BankTxTypeTransferOut, "-6", rp.ID, 2)

	claim, err := transferSvc.ClaimRedPacket(ctx, claimant.ID, rp.Code)
	require.NoError(t, err)
	require.NotNil(t, claim)
	require.InDelta(t, 3, claim.Amount, 0.000001)
	requireBalanceTransferLegacyBalance(t, claimant.ID, "1")
	requireBankAccountSnapshot(t, claimant.ID, "4", "0")
	requireRedPacketBankTransaction(t, claimant.ID, service.BankTxTypeTransferIn, "3", rp.ID, 2)

	require.NoError(t, transferSvc.ExpireRedPacket(ctx, rp.ID))
	requireBankAccountSnapshot(t, sender.ID, "17", "0")
	requireRedPacketBankTransaction(t, sender.ID, service.BankTxTypeRefund, "3", rp.ID, 2)
	requireRedPacketClearingBalanced(t, rp.ID, "6", "6")
}

func setTransferBankSettings(t *testing.T, client *dbent.Client) {
	t.Helper()
	repo := NewSettingRepository(client)
	require.NoError(t, repo.SetMultiple(context.Background(), map[string]string{
		service.SettingKeyTransferEnabled:         "true",
		service.SettingKeyTransferFeeRate:         "0.100000",
		service.SettingKeyTransferMinAmount:       "0.01000000",
		service.SettingKeyTransferMaxAmount:       "1000.00000000",
		service.SettingKeyTransferDailyLimit:      "1000.00000000",
		service.SettingKeyTransferDailyCountLimit: "50",
		service.SettingKeyTransferVIPFeeExempt:    "false",
		service.SettingKeyRedPacketEnabled:        "false",
	}))
}

func setRedPacketBankSettings(t *testing.T, client *dbent.Client) {
	t.Helper()
	repo := NewSettingRepository(client)
	require.NoError(t, repo.SetMultiple(context.Background(), map[string]string{
		service.SettingKeyTransferEnabled:         "true",
		service.SettingKeyTransferFeeRate:         "0.000000",
		service.SettingKeyTransferMinAmount:       "0.01000000",
		service.SettingKeyTransferMaxAmount:       "1000.00000000",
		service.SettingKeyTransferDailyLimit:      "1000.00000000",
		service.SettingKeyTransferDailyCountLimit: "50",
		service.SettingKeyTransferVIPFeeExempt:    "false",
		service.SettingKeyRedPacketEnabled:        "true",
		service.SettingKeyRedPacketMaxCount:       "10",
		service.SettingKeyRedPacketExpireHours:    "24",
	}))
}

func requireBalanceTransferLegacyBalance(t *testing.T, userID int64, want string) {
	t.Helper()
	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT balance
FROM users
WHERE id = $1
`, userID).Scan(&balance))
	require.InDelta(t, decimal.RequireFromString(want).InexactFloat64(), balance, 0.000001)
}

func requireBalanceTransferBankTransaction(t *testing.T, userID int64, txType, wantAmount string, transferID int64, wantLedgerEntries int) {
	t.Helper()
	var txID string
	var amount decimal.Decimal
	var module string
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COALESCE(MAX(tx_id::text), ''), COALESCE(SUM(amount), 0), COALESCE(MAX(business_module), '')
FROM transactions_log
WHERE user_id = $1
  AND tx_type = $2
  AND reference_type = 'balance_transfer'
  AND reference_id = $3
`, userID, txType, fmt.Sprintf("%d", transferID)).Scan(&txID, &amount, &module))
	require.NotEmpty(t, txID)
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "amount = %s", amount.String())
	require.Equal(t, service.BankBusinessModuleTransfer, module)
	requireBankLedgerEntryCount(t, txID, wantLedgerEntries)
	requireBankLedgerBalanced(t, txID)
}

func requireRedPacketBankTransaction(t *testing.T, userID int64, txType, wantAmount string, redPacketID int64, wantLedgerEntries int) {
	t.Helper()
	var txID string
	var amount decimal.Decimal
	var module string
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COALESCE(MAX(tx_id::text), ''), COALESCE(SUM(amount), 0), COALESCE(MAX(business_module), '')
FROM transactions_log
WHERE user_id = $1
  AND tx_type = $2
  AND reference_type = 'balance_redpacket'
  AND reference_id = $3
`, userID, txType, fmt.Sprintf("%d", redPacketID)).Scan(&txID, &amount, &module))
	require.NotEmpty(t, txID)
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "amount = %s", amount.String())
	require.Equal(t, service.BankBusinessModuleTransfer, module)
	requireBankLedgerEntryCount(t, txID, wantLedgerEntries)
	requireBankLedgerBalanced(t, txID)
}

func requireRedPacketClearingBalanced(t *testing.T, redPacketID int64, wantDebit, wantCredit string) {
	t.Helper()
	var debit, credit decimal.Decimal
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COALESCE(SUM(CASE WHEN le.entry_side = 'DEBIT' THEN le.amount ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN le.entry_side = 'CREDIT' THEN le.amount ELSE 0 END), 0)
FROM ledger_entries le
JOIN ledger_accounts la ON la.id = le.ledger_account_id
WHERE la.account_code = 'PLATFORM:CLEARING:TRANSFER'
  AND le.reference_type = 'balance_redpacket'
  AND le.reference_id = $1
`, fmt.Sprintf("%d", redPacketID)).Scan(&debit, &credit))
	require.True(t, decimal.RequireFromString(wantDebit).Equal(debit), "debit = %s", debit.String())
	require.True(t, decimal.RequireFromString(wantCredit).Equal(credit), "credit = %s", credit.String())
}
