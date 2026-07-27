package platform

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/stretchr/testify/require"
)

func TestRedeemRecordWriterUsesTransactionFromContext(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectBegin()
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	mock.ExpectQuery(`INSERT INTO "redeem_codes"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(19)))
	mock.ExpectCommit()

	userID := int64(7)
	err = redeemRecordWriter{client: client}.CreateRedeemRecord(
		dbent.NewTxContext(context.Background(), tx),
		RedeemRecord{Code: "ACT-19", Type: "checkin_blindbox", Value: 2.5, Status: RedeemRecordStatusUsed, UsedBy: &userID},
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}
