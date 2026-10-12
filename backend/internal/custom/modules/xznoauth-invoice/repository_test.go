package xznoauthinvoice

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestOwnedCompletedOrderNos(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	orderNos := []string{"OLD-ORDER", "NEW-ORDER"}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT out_trade_no FROM payment_orders
WHERE user_id=$1 AND status='COMPLETED' AND out_trade_no = ANY($2)`)).
		WithArgs(int64(17), pq.Array(orderNos)).
		WillReturnRows(sqlmock.NewRows([]string{"out_trade_no"}).AddRow("NEW-ORDER"))

	got, err := NewRepository(db).OwnedCompletedOrderNos(context.Background(), 17, orderNos)
	require.NoError(t, err)
	require.Equal(t, map[string]struct{}{"NEW-ORDER": {}}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}
