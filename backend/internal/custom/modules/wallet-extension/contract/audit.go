package contract

import (
	"context"
	"time"
)

// BalanceAuditEntry records a wallet balance change without exposing a core persistence model.
type BalanceAuditEntry struct {
	AccountID      int64
	Operation      string
	Amount         float64
	BalanceBefore  float64
	BalanceAfter   float64
	ReferenceID    string
	IdempotencyKey string
	OccurredAt     time.Time
}

// AuditWriter writes immutable wallet audit entries inside the caller's transaction.
type AuditWriter interface {
	WriteBalanceAudit(ctx context.Context, entry BalanceAuditEntry) error
}
