package service

import (
	"context"
	"errors"
	"math"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestApplyBalanceCreditAdjustmentRoundsAndPreservesBaseWithoutHook(t *testing.T) {
	amount, result, err := applyBalanceCreditAdjustment(context.Background(), nil, BalanceCreditAdjustmentInput{BaseCredit: 12.34})
	require.NoError(t, err)
	require.Equal(t, 12.34, amount)
	require.Zero(t, result.ExtraCredit)

	amount, result, err = applyBalanceCreditAdjustment(context.Background(), adjustmentStub{result: BalanceCreditAdjustmentResult{ExtraCredit: 2.345}}, BalanceCreditAdjustmentInput{BaseCredit: 12.34})
	require.NoError(t, err)
	require.Equal(t, 14.69, amount)
	require.Equal(t, 2.345, result.ExtraCredit)
}

func TestApplyBalanceCreditAdjustmentRejectsInvalidResult(t *testing.T) {
	for _, extra := range []float64{-0.01, math.NaN(), math.Inf(1)} {
		_, _, err := applyBalanceCreditAdjustment(context.Background(), adjustmentStub{result: BalanceCreditAdjustmentResult{ExtraCredit: extra}}, BalanceCreditAdjustmentInput{BaseCredit: 10})
		require.Error(t, err)
	}
	_, _, err := applyBalanceCreditAdjustment(context.Background(), adjustmentStub{err: errors.New("settings unavailable")}, BalanceCreditAdjustmentInput{BaseCredit: 10})
	require.ErrorContains(t, err, "settings unavailable")
}

func TestResolveAffiliateRebateBaseUsesExtensionAndFallback(t *testing.T) {
	order := &dbent.PaymentOrder{Amount: 20, OrderType: payment.OrderTypeBalance}
	service := &PaymentService{}
	base, err := service.resolveAffiliateRebateBase(context.Background(), order)
	require.NoError(t, err)
	require.Equal(t, 20.0, base)

	service.SetAffiliateRebateBaseResolver(rebateBaseResolverStub{base: 15})
	base, err = service.resolveAffiliateRebateBase(context.Background(), order)
	require.NoError(t, err)
	require.Equal(t, 15.0, base)
}

type adjustmentStub struct {
	result BalanceCreditAdjustmentResult
	err    error
}

func (s adjustmentStub) CalculateBalanceCredit(context.Context, BalanceCreditAdjustmentInput) (BalanceCreditAdjustmentResult, error) {
	return s.result, s.err
}
func (adjustmentStub) RecordBalanceCreditAdjustment(context.Context, int64, BalanceCreditAdjustmentInput, BalanceCreditAdjustmentResult) error {
	return nil
}

type rebateBaseResolverStub struct{ base float64 }

func (s rebateBaseResolverStub) ResolveAffiliateRebateBase(context.Context, *dbent.PaymentOrder) (float64, error) {
	return s.base, nil
}
