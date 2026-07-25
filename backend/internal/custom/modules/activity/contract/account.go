// Package contract defines the core capabilities consumed by the activity module.
package contract

import "context"

// Account is the minimum user state required by activity operations.
type Account struct {
	ID      int64
	Role    string
	Status  string
	Balance float64
}

// AccountReader resolves the current account before an activity operation.
type AccountReader interface {
	GetAccount(ctx context.Context, userID int64) (Account, error)
}

// BalanceOperation identifies an activity balance mutation for core accounting.
type BalanceOperation struct {
	UserID         int64
	Amount         float64
	Reason         string
	IdempotencyKey string
}

// BalanceWriter applies non-recharge balance changes within the caller's transaction context.
type BalanceWriter interface {
	Credit(ctx context.Context, operation BalanceOperation) error
	DebitIfSufficient(ctx context.Context, operation BalanceOperation) (bool, error)
}

// BalanceCacheInvalidator clears user balance views after a committed mutation.
type BalanceCacheInvalidator interface {
	InvalidateBalance(ctx context.Context, userID int64) error
}

// ConcurrencyGrant is the account adjustment awarded by a blind-box prize.
// Implementations must deduplicate the grant using IdempotencyKey in the
// transaction represented by ctx.
type ConcurrencyGrant struct {
	UserID         int64
	Slots          int
	Reason         string
	IdempotencyKey string
}

// ConcurrencyGranter applies a blind-box concurrency reward. It exists apart
// from BalanceWriter so reward delivery never reaches into a user repository.
type ConcurrencyGranter interface {
	GrantConcurrency(ctx context.Context, grant ConcurrencyGrant) error
}
