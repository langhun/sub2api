package contract

import "context"

// TransactionRunner runs all participating core contracts in one atomic transaction.
type TransactionRunner interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
