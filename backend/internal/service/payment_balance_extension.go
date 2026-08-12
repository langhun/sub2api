package service

import (
	"context"
	"fmt"
	"math"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// BalanceOrderExtensionInput contains the immutable values available while a
// balance order is being created. BaseCredit already includes the core
// recharge multiplier; PayAmount remains the external payment amount.
type BalanceOrderExtensionInput struct {
	UserID        int64
	PaymentAmount float64
	BaseCredit    float64
	PayAmount     float64
	Now           time.Time
}

// BalanceOrderExtensionResult lets an optional extension replace the credited
// balance and retain an opaque snapshot for the transaction-local persistence
// callback. Core payment code never interprets Snapshot.
type BalanceOrderExtensionResult struct {
	CreditedAmount float64
	Snapshot       any
}

// BalanceRefundExtensionInput describes the final credited-balance refund
// amount after provider confirmation. It deliberately exposes no feature
// concepts so multiple optional extensions can share the same hook.
type BalanceRefundExtensionInput struct {
	Order        *dbent.PaymentOrder
	RefundAmount float64
	RefundedAt   time.Time
}

// BalanceOrderExtension is the narrow, optional seam for application overlays
// that need an immutable balance-order snapshot. PersistBalanceOrder and
// RecordBalanceRefund receive the transaction-scoped Ent client, so their
// writes commit or roll back with the core order state.
type BalanceOrderExtension interface {
	PrepareBalanceOrder(context.Context, BalanceOrderExtensionInput) (BalanceOrderExtensionResult, error)
	PersistBalanceOrder(context.Context, *dbent.Client, *dbent.PaymentOrder, BalanceOrderExtensionInput, BalanceOrderExtensionResult) error
	ResolveAffiliateRebateBase(context.Context, *dbent.PaymentOrder) (amount float64, handled bool, err error)
	RecordBalanceRefund(context.Context, *dbent.Client, BalanceRefundExtensionInput) error
}

func defaultBalanceOrderExtensionResult(baseCredit float64) BalanceOrderExtensionResult {
	return BalanceOrderExtensionResult{CreditedAmount: baseCredit}
}

func validateBalanceOrderExtensionResult(baseCredit float64, result BalanceOrderExtensionResult) error {
	if !finiteNonnegativeBalanceAmount(baseCredit) || !finiteNonnegativeBalanceAmount(result.CreditedAmount) {
		return fmt.Errorf("balance order extension returned an invalid credited amount")
	}
	if result.CreditedAmount+1e-9 < baseCredit {
		return fmt.Errorf("balance order extension cannot reduce credited amount")
	}
	return nil
}

func finiteNonnegativeBalanceAmount(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func (s *PaymentService) prepareBalanceOrderExtension(ctx context.Context, input BalanceOrderExtensionInput) (BalanceOrderExtensionResult, error) {
	if s == nil || s.balanceOrderExtension == nil {
		return defaultBalanceOrderExtensionResult(input.BaseCredit), nil
	}
	result, err := s.balanceOrderExtension.PrepareBalanceOrder(ctx, input)
	if err != nil {
		return BalanceOrderExtensionResult{}, fmt.Errorf("prepare balance order extension: %w", err)
	}
	if err := validateBalanceOrderExtensionResult(input.BaseCredit, result); err != nil {
		return BalanceOrderExtensionResult{}, err
	}
	return result, nil
}

func (s *PaymentService) resolveAffiliateRebateBase(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	fallback := affiliateRebateBaseAmount(order)
	if s == nil || s.balanceOrderExtension == nil || order == nil || order.OrderType != payment.OrderTypeBalance {
		return fallback, nil
	}
	amount, handled, err := s.balanceOrderExtension.ResolveAffiliateRebateBase(ctx, order)
	if err != nil {
		return 0, err
	}
	if !handled {
		return fallback, nil
	}
	if !finiteNonnegativeBalanceAmount(amount) {
		return 0, fmt.Errorf("balance order extension returned an invalid affiliate rebate base")
	}
	return amount, nil
}

func (s *PaymentService) recordBalanceRefundExtension(ctx context.Context, client *dbent.Client, input BalanceRefundExtensionInput) error {
	if s == nil || s.balanceOrderExtension == nil || input.Order == nil || input.Order.OrderType != payment.OrderTypeBalance {
		return nil
	}
	if client == nil {
		return fmt.Errorf("balance order extension transaction client is required")
	}
	if !finiteNonnegativeBalanceAmount(input.RefundAmount) {
		return fmt.Errorf("invalid balance refund extension amount")
	}
	if err := s.balanceOrderExtension.RecordBalanceRefund(ctx, client, input); err != nil {
		return fmt.Errorf("record balance refund extension: %w", err)
	}
	return nil
}
