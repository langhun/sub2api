package checkin

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

// NewTransactionRunner adapts Ent transactions to the Activity contract. A
// nested check-in operation reuses its caller's transaction context.
func NewTransactionRunner(client *dbent.Client) contract.TransactionRunner {
	return entTransactionRunner{client: client}
}

type entTransactionRunner struct{ client *dbent.Client }

func (r entTransactionRunner) RunInTransaction(ctx context.Context, operation func(context.Context) error) error {
	if r.client == nil {
		return fmt.Errorf("ent client is required")
	}
	if dbent.TxFromContext(ctx) != nil {
		return operation(ctx)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin checkin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := operation(dbent.NewTxContext(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit()
}
