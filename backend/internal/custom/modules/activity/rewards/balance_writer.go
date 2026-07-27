package rewards

import (
	"context"
	"fmt"
	"math"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

// NewEntBalanceWriter owns reward balance credits over the compatibility
// users table. It deliberately updates only balance, never total_recharged.
func NewEntBalanceWriter(client *dbent.Client) contract.BalanceWriter {
	return entBalanceWriter{client: client}
}

type entBalanceWriter struct{ client *dbent.Client }

func (w entBalanceWriter) Credit(ctx context.Context, operation contract.BalanceOperation) error {
	if w.client == nil || operation.UserID <= 0 || !validBalanceCredit(operation.Amount) {
		return fmt.Errorf("invalid activity reward balance credit")
	}
	updated, err := w.clientFor(ctx).User.Update().Where(user.IDEQ(operation.UserID)).AddBalance(operation.Amount).Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("activity reward account %d not found", operation.UserID)
	}
	return nil
}

func (w entBalanceWriter) DebitIfSufficient(context.Context, contract.BalanceOperation) (bool, error) {
	return false, fmt.Errorf("activity reward balance writer does not debit")
}

func (w entBalanceWriter) clientFor(ctx context.Context) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return w.client
}

func validBalanceCredit(amount float64) bool {
	return amount > 0 && !math.IsNaN(amount) && !math.IsInf(amount, 0)
}

var _ contract.BalanceWriter = entBalanceWriter{}
