//go:build unit

package service

import (
	"context"
	"errors"
	"regexp"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func newAdminUsageStatsSQLMockClient(t *testing.T) (*dbent.Client, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(drv))

	cleanup := func() {
		_ = client.Close()
		_ = db.Close()
	}
	return client, mock, cleanup
}

func TestAdminService_GetUserUsageStats_TodaySuccess(t *testing.T) {
	client, mock, cleanup := newAdminUsageStatsSQLMockClient(t)
	t.Cleanup(cleanup)

	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_tokens",
			"total_cost",
			"total_actual_cost",
			"avg_duration_ms",
		}).AddRow(
			int64(5),
			int64(100),
			int64(200),
			int64(50),
			1.23,
			0.99,
			int64(321),
		))

	svc := &adminServiceImpl{entClient: client}
	raw, err := svc.GetUserUsageStats(context.Background(), 123, "today")
	require.NoError(t, err)

	stats, ok := raw.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "today", stats["period"])
	require.Equal(t, int64(5), stats["total_requests"])
	require.Equal(t, 1.23, stats["total_cost"])
	require.Equal(t, int64(350), stats["total_tokens"])
	require.Equal(t, int64(321), stats["avg_duration_ms"])
	require.Equal(t, int64(100), stats["total_input_tokens"])
	require.Equal(t, int64(200), stats["total_output_tokens"])
	require.Equal(t, int64(50), stats["total_cache_tokens"])
	require.Equal(t, 0.99, stats["total_actual_cost"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminService_GetUserUsageStats_DefaultsToMonth(t *testing.T) {
	client, mock, cleanup := newAdminUsageStatsSQLMockClient(t)
	t.Cleanup(cleanup)

	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WithArgs(int64(77), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_tokens",
			"total_cost",
			"total_actual_cost",
			"avg_duration_ms",
		}).AddRow(
			int64(0),
			int64(0),
			int64(0),
			int64(0),
			0.0,
			0.0,
			int64(0),
		))

	svc := &adminServiceImpl{entClient: client}
	raw, err := svc.GetUserUsageStats(context.Background(), 77, "unknown-period")
	require.NoError(t, err)

	stats, ok := raw.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "month", stats["period"])
	require.Equal(t, int64(0), stats["total_requests"])
	require.Equal(t, int64(0), stats["total_tokens"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminService_GetUserUsageStats_QueryError(t *testing.T) {
	client, mock, cleanup := newAdminUsageStatsSQLMockClient(t)
	t.Cleanup(cleanup)

	queryErr := errors.New("query failed")
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WithArgs(int64(9), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(queryErr)

	svc := &adminServiceImpl{entClient: client}
	stats, err := svc.GetUserUsageStats(context.Background(), 9, "month")
	require.ErrorIs(t, err, queryErr)
	require.Nil(t, stats)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminService_GetUserUsageStats_EntClientNil(t *testing.T) {
	svc := &adminServiceImpl{}

	stats, err := svc.GetUserUsageStats(context.Background(), 1, "today")
	require.Error(t, err)
	require.Contains(t, err.Error(), "entClient is nil")
	require.Nil(t, stats)
}
