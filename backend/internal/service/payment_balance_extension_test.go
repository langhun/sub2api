package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestPrepareBalanceOrderExtensionFallsBackToBaseCredit(t *testing.T) {
	svc := &PaymentService{}
	result, err := svc.prepareBalanceOrderExtension(context.Background(), BalanceOrderExtensionInput{BaseCredit: 12})
	require.NoError(t, err)
	require.Equal(t, 12.0, result.CreditedAmount)
}

func TestPrepareBalanceOrderExtensionRejectsCreditReduction(t *testing.T) {
	svc := &PaymentService{balanceOrderExtension: balanceOrderExtensionStub{result: BalanceOrderExtensionResult{CreditedAmount: 9}}}
	_, err := svc.prepareBalanceOrderExtension(context.Background(), BalanceOrderExtensionInput{BaseCredit: 10})
	require.ErrorContains(t, err, "cannot reduce")
}

func TestResolveAffiliateRebateBaseFallsBackForUnhandledOrder(t *testing.T) {
	svc := &PaymentService{balanceOrderExtension: balanceOrderExtensionStub{}}
	amount, err := svc.resolveAffiliateRebateBase(context.Background(), &dbent.PaymentOrder{Amount: 30, OrderType: payment.OrderTypeBalance})
	require.NoError(t, err)
	require.Equal(t, 30.0, amount)
}

type balanceOrderExtensionStub struct {
	result     BalanceOrderExtensionResult
	handled    bool
	base       float64
	persistErr error
}

func (s balanceOrderExtensionStub) PrepareBalanceOrder(context.Context, BalanceOrderExtensionInput) (BalanceOrderExtensionResult, error) {
	return s.result, nil
}

func (s balanceOrderExtensionStub) PersistBalanceOrder(context.Context, *dbent.Client, *dbent.PaymentOrder, BalanceOrderExtensionInput, BalanceOrderExtensionResult) error {
	return s.persistErr
}

func (s balanceOrderExtensionStub) ResolveAffiliateRebateBase(context.Context, *dbent.PaymentOrder) (float64, bool, error) {
	return s.base, s.handled, nil
}

func (balanceOrderExtensionStub) RecordBalanceRefund(context.Context, *dbent.Client, BalanceRefundExtensionInput) error {
	return nil
}
