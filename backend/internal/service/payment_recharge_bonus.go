package service

import (
	"context"
	"fmt"
	"math"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

// BalanceCreditAdjustmentInput is the narrow contract exposed to optional
// Overlay modules. BaseCredit already includes the upstream recharge rate.
type BalanceCreditAdjustmentInput struct {
	UserID        int64
	PaymentAmount float64
	BaseCredit    float64
	PayAmount     float64
	Now           time.Time
}

type BalanceCreditAdjustmentResult struct {
	ExtraCredit float64
	// Snapshot is opaque to payment core and is persisted only by the Overlay
	// that produced it. It keeps business rules out of upstream order fields.
	Snapshot map[string]string
}

// BalanceCreditAdjuster can add balance to a balance-recharge order. The
// payment core only knows this generic contract, never the custom feature.
type BalanceCreditAdjuster interface {
	CalculateBalanceCredit(context.Context, BalanceCreditAdjustmentInput) (BalanceCreditAdjustmentResult, error)
	RecordBalanceCreditAdjustment(context.Context, int64, BalanceCreditAdjustmentInput, BalanceCreditAdjustmentResult) error
}

// AffiliateRebateBaseResolver lets an Overlay provide an immutable order
// rebate base while preserving the upstream amount fallback.
type AffiliateRebateBaseResolver interface {
	ResolveAffiliateRebateBase(context.Context, *dbent.PaymentOrder) (float64, error)
}

func validateBalanceCreditAdjustment(result BalanceCreditAdjustmentResult) error {
	if math.IsNaN(result.ExtraCredit) || math.IsInf(result.ExtraCredit, 0) || result.ExtraCredit < 0 {
		return &balanceCreditAdjustmentError{message: "balance credit adjustment must be finite and nonnegative"}
	}
	return nil
}

func applyBalanceCreditAdjustment(ctx context.Context, adjuster BalanceCreditAdjuster, input BalanceCreditAdjustmentInput) (float64, BalanceCreditAdjustmentResult, error) {
	if adjuster == nil {
		return input.BaseCredit, BalanceCreditAdjustmentResult{}, nil
	}
	result, err := adjuster.CalculateBalanceCredit(ctx, input)
	if err != nil {
		return 0, BalanceCreditAdjustmentResult{}, fmt.Errorf("calculate balance credit adjustment: %w", err)
	}
	if err := validateBalanceCreditAdjustment(result); err != nil {
		return 0, BalanceCreditAdjustmentResult{}, err
	}
	return decimal.NewFromFloat(input.BaseCredit).Add(decimal.NewFromFloat(result.ExtraCredit)).Round(2).InexactFloat64(), result, nil
}

type balanceCreditAdjustmentError struct{ message string }

func (e *balanceCreditAdjustmentError) Error() string { return e.message }
