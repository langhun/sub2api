package contract

import (
	"context"
	"time"
)

// TransactionRunner runs an activity operation atomically. Implementations must
// propagate the transaction-bound context to every port called by operation.
type TransactionRunner interface {
	RunInTransaction(ctx context.Context, operation func(ctx context.Context) error) error
}

// Lease is a cross-instance lease held while a periodic activity task runs.
// Release must be safe to call exactly once after a successful acquisition.
type Lease interface {
	Release(ctx context.Context) error
}

// SingletonLeaseCoordinator gates a periodic activity task across instances.
// A false acquired result is a normal skip because another instance owns the
// work. Implementations may fall back to a database advisory lock.
type SingletonLeaseCoordinator interface {
	AcquireSingletonLease(ctx context.Context, key, owner string, ttl time.Duration) (lease Lease, acquired bool, err error)
}
