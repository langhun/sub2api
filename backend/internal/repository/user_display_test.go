package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestQueryUserDisplayNamesPrefersUsernameThenMaskedEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT id, COALESCE\(username, ''\), COALESCE\(email, ''\) FROM users WHERE id IN \(\$1,\$2\)`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}).
			AddRow(10, "alice", "alice@example.com").
			AddRow(55, "", "fallback@example.com"))

	displays, err := queryUserDisplayNames(context.Background(), db, []int64{10, 55, 10})
	require.NoError(t, err)
	require.Equal(t, map[int64]string{10: "alice", 55: "f***k@example.com"}, displays)
	mock.ExpectClose()
	require.NoError(t, db.Close())
	require.NoError(t, mock.ExpectationsWereMet())
}
