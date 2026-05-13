package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMonitoringServiceQueryGroupModelStats_ErrorOnlyUsesJoinedGroupName(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	usageRows := sqlmock.NewRows([]string{
		"group_id",
		"group_name",
		"model",
		"cnt",
		"success_cnt",
		"error_cnt",
		"avg_latency_ms",
		"p50_latency_ms",
		"p95_latency_ms",
		"avg_ttft",
	})

	errorRows := sqlmock.NewRows([]string{"group_id", "group_name", "model", "err_cnt"}).
		AddRow(int64(42), "生产分组", "gpt-5", 3)

	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs u")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(usageRows)
	mock.ExpectQuery(regexp.QuoteMeta("JOIN groups g ON e.group_id = g.id AND g.deleted_at IS NULL")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(errorRows)

	overview := &MonitoringOverview{}
	err = NewMonitoringService(db).queryGroupModelStats(context.Background(), overview)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, []GroupModelStats{
		{
			GroupID:      42,
			GroupName:    "生产分组",
			Model:        "gpt-5",
			RequestCount: 3,
			SuccessCount: 0,
			ErrorCount:   3,
		},
	}, overview.GroupModels)
}

func TestMonitoringServiceQueryTodaySummary_ExcludesClientDisconnect499(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count", "success_count", "error_count", "avg_latency_ms"}).
			AddRow(int64(1), int64(1), int64(0), 321.0))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM ops_error_logs .*client.*499`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	overview := &MonitoringOverview{}
	err = NewMonitoringService(db).queryTodaySummary(context.Background(), overview)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.EqualValues(t, 1, overview.TotalRequests)
	require.EqualValues(t, 1, overview.SuccessCount)
	require.EqualValues(t, 0, overview.ErrorCount)
	require.Equal(t, 321.0, overview.AvgLatencyMs)
}

func TestMonitoringServiceQueryHourlyStats_ExcludesClientDisconnect499(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	hour := time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)WITH usage_hours AS .*client.*499`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"hour", "total", "success"}).
			AddRow(hour, 1, 1))

	overview := &MonitoringOverview{}
	err = NewMonitoringService(db).queryHourlyStats(context.Background(), overview)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Len(t, overview.HourlyStats, 1)
	require.Equal(t, hour.Format("2006-01-02T15:04:05Z07:00"), overview.HourlyStats[0].Hour)
	require.Equal(t, 1, overview.HourlyStats[0].Total)
	require.Equal(t, 1, overview.HourlyStats[0].Success)
}
