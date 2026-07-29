package accountdrain

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestRepositoryEnableAccountCreatesSingleAccountTarget(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock($1)")).
		WithArgs(int64(17)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT platform FROM accounts WHERE id = $1")).
		WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"platform"}).AddRow("openai"))
	mock.ExpectQuery("SELECT EXISTS").WithArgs(int64(17), StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("INSERT INTO custom_account_drain_plans").
		WithArgs("account-target-17", StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(91)))
	mock.ExpectExec("INSERT INTO custom_account_drain_plan_accounts").
		WithArgs(int64(91), int64(17)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	created, err := NewRepository(db).EnableAccount(context.Background(), 17)
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryEnableAccountIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock($1)")).
		WithArgs(int64(17)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT platform FROM accounts WHERE id = $1")).
		WithArgs(int64(17)).WillReturnRows(sqlmock.NewRows([]string{"platform"}).AddRow("openai"))
	mock.ExpectQuery("SELECT EXISTS").WithArgs(int64(17), StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectCommit()

	created, err := NewRepository(db).EnableAccount(context.Background(), 17)
	require.NoError(t, err)
	require.False(t, created)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryDisableAccountRemovesOnlyTheTarget(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock($1)")).
		WithArgs(int64(17)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("WITH removed AS").WithArgs(int64(17), StatusActive, StatusStopped).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = NewRepository(db).DisableAccount(context.Background(), 17)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
