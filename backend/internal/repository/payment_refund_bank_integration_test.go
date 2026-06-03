//go:build integration

package repository

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestPaymentRefundDeductBalance_WritesBankLedger(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{
		Email:   fmt.Sprintf("refund-bank-%d@example.com", time.Now().UnixNano()),
		Balance: 0,
	})
	_, err := client.UserBankAccount.Create().
		SetUserID(user.ID).
		SetBalance(decimal.NewFromInt(20)).
		SetStatus(service.BankAccountStatusActive).
		Save(ctx)
	require.NoError(t, err)

	inst, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("refund-bank-provider").
		SetConfig("{}").
		SetSupportedTypes(payment.TypeAlipay).
		SetEnabled(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(8).
		SetPayAmount(8).
		SetFeeRate(0).
		SetRechargeCode("REFUND-BANK-ORDER").
		SetOutTradeNo("sub2_refund_bank_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(service.OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderInstanceID(strconv.FormatInt(inst.ID, 10)).
		SetProviderKey(payment.TypeAlipay).
		SetProviderSnapshot(map[string]any{
			"schema_version":       2,
			"provider_instance_id": strconv.FormatInt(inst.ID, 10),
			"provider_key":         payment.TypeAlipay,
		}).
		Save(ctx)
	require.NoError(t, err)

	paymentSvc := service.NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, nil, NewUserRepository(client, integrationDB), nil, nil)
	plan, earlyResult, err := paymentSvc.PrepareRefund(ctx, order.ID, 0, "bank refund", false, true, "refund-bank-deduct")
	require.NoError(t, err)
	require.Nil(t, earlyResult)
	result, err := paymentSvc.ExecuteRefund(ctx, plan)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, 8.0, result.BalanceDeducted)

	requireLegacyUserBalance(t, user.ID, 0)
	requireBankAccountSnapshot(t, user.ID, "12", "0")
	txID := requirePaymentRefundBankTransaction(t, user.ID, service.BankTxTypeWithdraw, "-8", "refund-bank-deduct", "deduct", 1)
	requireBankLedgerEntryCount(t, txID, 2)
	requireBankLedgerBalanced(t, txID)
}

func requirePaymentRefundBankTransaction(t *testing.T, userID int64, txType, wantAmount, requestID, phase string, wantCount int) string {
	t.Helper()
	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM transactions_log
WHERE user_id = $1 AND idempotency_scope = 'payment_refund_balance' AND request_id = $2 AND reference_id LIKE '%' || $3
`, userID, requestID, phase).Scan(&count))
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
WHERE user_id = $1 AND idempotency_scope = 'payment_refund_balance' AND request_id = $2 AND reference_id LIKE '%' || $3
`, userID, requestID, phase).Scan(&txID, &amount, &gotTxType, &module, &referenceType))
	require.True(t, decimal.RequireFromString(wantAmount).Equal(amount), "refund bank amount = %s", amount.String())
	require.Equal(t, txType, gotTxType)
	require.Equal(t, service.BankBusinessModulePayment, module)
	require.Equal(t, "payment_refund", referenceType)
	return txID
}
