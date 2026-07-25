package contract

import "context"

// BalanceOperation describes one wallet mutation within the current transaction.
type BalanceOperation struct {
	AccountID      int64
	Amount         float64
	Reason         string
	ReferenceID    string
	IdempotencyKey string
}

// BalanceWriter applies main-balance changes without exposing the core user repository.
type BalanceWriter interface {
	Credit(ctx context.Context, operation BalanceOperation) error
	DebitIfSufficient(ctx context.Context, operation BalanceOperation) (bool, error)
}

// BalanceCacheInvalidator clears cached balance projections after a committed mutation.
type BalanceCacheInvalidator interface {
	InvalidateBalance(ctx context.Context, accountID int64) error
}
