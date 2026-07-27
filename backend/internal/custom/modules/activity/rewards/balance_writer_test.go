package rewards

import (
	"context"
	"math"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/stretchr/testify/require"
)

func TestEntBalanceWriterCreditUpdatesOnlySpendableBalanceInsideTransaction(t *testing.T) {
	client, _, mock := newOutboxSQLMock(t)
	writer := NewEntBalanceWriter(client)

	mock.ExpectBegin()
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	mock.ExpectExec(`UPDATE "users" SET "updated_at" = \$1, "balance" = COALESCE\("users"\."balance", 0\) \+ \$2 WHERE "users"\."id" = \$3`).
		WithArgs(sqlmock.AnyArg(), 12.5, int64(91)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	err = writer.Credit(dbent.NewTxContext(context.Background(), tx), contract.BalanceOperation{UserID: 91, Amount: 12.5})
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEntBalanceWriterCreditRejectsInvalidRequests(t *testing.T) {
	writer := NewEntBalanceWriter(nil)
	for _, operation := range []contract.BalanceOperation{
		{UserID: 0, Amount: 1},
		{UserID: 1, Amount: 0},
		{UserID: 1, Amount: -1},
		{UserID: 1, Amount: math.NaN()},
		{UserID: 1, Amount: math.Inf(1)},
	} {
		require.Error(t, writer.Credit(context.Background(), operation))
	}
}
