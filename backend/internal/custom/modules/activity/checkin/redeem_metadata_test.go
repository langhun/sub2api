package checkin

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestRedeemMetadataStoreUpsertsModuleMetadata(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectExec(`INSERT INTO custom_activity_redeem_metadata`).
		WithArgs(int64(19), 2.5, 7.25).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = NewRedeemMetadataStore(client).Store(context.Background(), 19, 2.5, 7.25)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedeemMetadataStoreRejectsInvalidInput(t *testing.T) {
	require.Error(t, redeemMetadataStore{}.Store(context.Background(), 0, 1, 1))
}
