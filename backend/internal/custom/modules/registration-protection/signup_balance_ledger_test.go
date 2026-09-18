package registrationprotection

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSignupBalanceLedgerRecordsAndListsImmutableGrant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ledger := NewSignupBalanceLedger(db)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO custom_signup_balance_ledger")).WithArgs(int64(71), 1.0, "email").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(9)))
	require.NoError(t, ledger.RecordSignupBalance(context.Background(), 71, 1, ""))

	now := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM custom_signup_balance_ledger WHERE user_id = $1")).WithArgs(int64(71)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, amount::double precision, signup_source, created_at")).WithArgs(int64(71), 0, 15).WillReturnRows(sqlmock.NewRows([]string{"id", "amount", "signup_source", "created_at"}).AddRow(int64(9), 1.0, "email", now))

	items, total, err := ledger.ListSignupBalanceHistory(context.Background(), 71, 1, 15)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, service.SignupBalanceHistoryType, items[0].Type)
	require.InDelta(t, 1, items[0].Value, 0.00001)
	require.Equal(t, "SIGNUP-9", items[0].Code)
	require.Equal(t, int64(-1_000_000_000_000_009), items[0].ID)
	require.Empty(t, items[0].Notes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSignupBalanceLedgerRejectsConflictingGrant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ledger := NewSignupBalanceLedger(db)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO custom_signup_balance_ledger")).WithArgs(int64(72), 1.0, "email").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	err = ledger.RecordSignupBalance(context.Background(), 72, 1, "email")
	require.ErrorContains(t, err, "conflicting signup balance ledger entry")
	require.NoError(t, mock.ExpectationsWereMet())
}
