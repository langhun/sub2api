package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCreateUsageLogsPartition_AddsTableComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newDashboardAggregationRepositoryWithSQL(db)
	month := time.Date(2026, 5, 17, 9, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	createQuery := regexp.QuoteMeta(
		`CREATE TABLE IF NOT EXISTS "usage_logs_202605" PARTITION OF usage_logs FOR VALUES FROM ('2026-05-01') TO ('2026-06-01')`,
	)
	commentQuery := regexp.QuoteMeta(
		`COMMENT ON TABLE "usage_logs_202605" IS '用量日志月分区表（按 UTC 月份切分）'`,
	)

	mock.ExpectExec(createQuery).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(commentQuery).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.createUsageLogsPartition(context.Background(), month)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecomputeRangeRebuildsDailyUsersForFullDay(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newDashboardAggregationRepositoryWithSQL(db)
	loc := time.FixedZone("UTC+8", 8*60*60)
	dayStart := time.Date(2026, 5, 12, 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)
	hourStart := dayStart.Add(15 * time.Hour)
	hourEnd := hourStart.Add(time.Hour)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM usage_dashboard_hourly WHERE bucket_start >= $1 AND bucket_start < $2")).
		WithArgs(hourStart, hourEnd).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM usage_dashboard_hourly_users WHERE bucket_start >= $1 AND bucket_start < $2")).
		WithArgs(hourStart, hourEnd).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM usage_dashboard_daily WHERE bucket_date >= $1::date AND bucket_date < $2::date")).
		WithArgs(dayStart, dayEnd).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM usage_dashboard_daily_users WHERE bucket_date >= $1::date AND bucket_date < $2::date")).
		WithArgs(dayStart, dayEnd).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("(?s)INSERT INTO usage_dashboard_hourly_users.*FROM usage_logs").
		WithArgs(hourStart, hourEnd, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("(?s)INSERT INTO usage_dashboard_daily_users.*FROM usage_dashboard_hourly_users").
		WithArgs(dayStart, dayEnd, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("(?s)INSERT INTO usage_dashboard_hourly .*FROM hourly").
		WithArgs(hourStart, hourEnd, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("(?s)INSERT INTO usage_dashboard_daily .*FROM daily").
		WithArgs(dayStart, dayEnd, dayStart, dayEnd, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.recomputeRangeInTx(context.Background(), hourStart, hourEnd, dayStart, dayEnd)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
