package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestBalanceTransferRepoGetDailyTransferTotalUsesLocalStartOfDay(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &balanceTransferRepo{db: db}

	expectedStart := timezone.Today()
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(gross_amount\\),0\\), COALESCE\\(COUNT\\(\\*\\),0\\) FROM balance_transfers WHERE sender_id = \\$1 AND status != 'revoked' AND created_at >= \\$2").
		WithArgs(int64(123), expectedStart).
		WillReturnRows(sqlmock.NewRows([]string{"total", "count"}).AddRow(12.5, 3))

	total, count, err := repo.GetDailyTransferTotal(context.Background(), 123)
	require.NoError(t, err)
	require.Equal(t, 12.5, total)
	require.Equal(t, 3, count)
	require.NoError(t, mock.ExpectationsWereMet())
}
