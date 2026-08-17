//go:build unit

package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestDashboardAggregationRepositorySyncGroupUsageRollupsNoopsAtCurrentDate(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	todayStart := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("2026-08-14", "Asia/Shanghai"))
	mock.ExpectCommit()

	require.NoError(t, repo.SyncGroupUsageRollups(context.Background(), todayStart))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositorySyncGroupUsageRollupsClearsStaleBucketsWhenEmpty(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	todayStart := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("1970-01-01", "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(nil))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups`).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs("2026-08-14", todayStart, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SyncGroupUsageRollups(context.Background(), todayStart))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositorySyncGroupUsageRollupsSkipsEmptyYearsAndCommitsEachDay(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	todayStart := time.Date(2026, 5, 2, 16, 0, 0, 0, time.UTC)
	retainedFrom := time.Date(2026, 5, 1, 3, 0, 0, 0, time.UTC)
	mayFirstStart := time.Date(2026, 4, 30, 16, 0, 0, 0, time.UTC)
	maySecondStart := time.Date(2026, 5, 1, 16, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("1970-01-01", "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(retainedFrom))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups.*bucket_date < \$1::date OR bucket_date >= \$2::date`).
		WithArgs("2026-05-01", "2026-05-03").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups.*bucket_date = \$1::date`).
		WithArgs("2026-05-01").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(mayFirstStart, maySecondStart, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs("2026-05-02", retainedFrom, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("2026-05-02", "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(retainedFrom))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups.*bucket_date = \$1::date`).
		WithArgs("2026-05-02").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(maySecondStart, todayStart, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs("2026-05-03", retainedFrom, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SyncGroupUsageRollups(context.Background(), todayStart))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositorySyncGroupUsageRollupsRollsBackOnlyFailingDay(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	todayStart := time.Date(2026, 5, 2, 16, 0, 0, 0, time.UTC)
	retainedFrom := time.Date(2026, 5, 1, 3, 0, 0, 0, time.UTC)
	mayFirstStart := time.Date(2026, 4, 30, 16, 0, 0, 0, time.UTC)
	maySecondStart := time.Date(2026, 5, 1, 16, 0, 0, 0, time.UTC)
	bucketErr := errors.New("rebuild second bucket")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("2026-05-01", "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(retainedFrom))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups.*bucket_date = \$1::date`).
		WithArgs("2026-05-01").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(mayFirstStart, maySecondStart, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs("2026-05-02", retainedFrom, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("2026-05-02", "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(retainedFrom))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups.*bucket_date = \$1::date`).
		WithArgs("2026-05-02").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(maySecondStart, todayStart, "Asia/Shanghai").
		WillReturnError(bucketErr)
	mock.ExpectRollback()

	err := repo.SyncGroupUsageRollups(context.Background(), todayStart)
	require.ErrorIs(t, err, bucketErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositorySyncGroupUsageRollupsRebuildsTimezoneOneDayPerTransaction(t *testing.T) {
	useGroupUsageRepositoryTestTimezone(t, "America/New_York")

	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	todayStart := time.Date(2026, 3, 9, 4, 0, 0, 0, time.UTC)
	retainedFrom := time.Date(2026, 3, 7, 6, 0, 0, 0, time.UTC)
	marchSeventhStart := time.Date(2026, 3, 7, 5, 0, 0, 0, time.UTC)
	marchEighthStart := time.Date(2026, 3, 8, 5, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("2026-03-09", "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(retainedFrom))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups`).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups.*bucket_date = \$1::date`).
		WithArgs("2026-03-07").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(marchSeventhStart, marchEighthStart, "America/New_York").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs("2026-03-08", retainedFrom, "America/New_York").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("2026-03-08", "America/New_York"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(retainedFrom))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups.*bucket_date = \$1::date`).
		WithArgs("2026-03-08").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(marchEighthStart, todayStart, "America/New_York").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs("2026-03-09", retainedFrom, "America/New_York").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SyncGroupUsageRollups(context.Background(), todayStart))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositorySyncGroupUsageRollupsRejectsFutureWatermark(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	todayStart := time.Date(2026, 8, 13, 16, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow("2026-08-15", "Asia/Shanghai"))
	mock.ExpectRollback()

	err := repo.SyncGroupUsageRollups(context.Background(), todayStart)
	require.ErrorContains(t, err, "未来")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositoryRecomputeRangeInvalidatesGroupRollupsBeforeDashboardRebuild(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	start := time.Date(2026, 8, 1, 3, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM usage_group_rollup_state.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs(start, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM usage_dashboard_hourly`).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := repo.RecomputeRange(context.Background(), start, end)
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositoryRecomputeRangeCommitsInvalidationAndDashboardBeforeDailyRollups(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	start := time.Date(2026, 8, 13, 3, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	fixedNow := time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC)
	repo.clock = func() time.Time { return fixedNow }
	todayStart := service.GroupUsageTodayStart(fixedNow)
	startDate := service.GroupUsageDate(start)
	firstBucketStart, err := service.ParseGroupUsageDate(startDate)
	require.NoError(t, err)
	secondBucketStart := firstBucketStart.AddDate(0, 0, 1)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM usage_group_rollup_state.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs(start, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	for _, query := range []string{
		`DELETE FROM usage_dashboard_hourly WHERE`,
		`DELETE FROM usage_dashboard_hourly_users WHERE`,
		`DELETE FROM usage_dashboard_daily WHERE`,
		`DELETE FROM usage_dashboard_daily_users WHERE`,
		`INSERT INTO usage_dashboard_hourly_users`,
		`INSERT INTO usage_dashboard_daily_users`,
		`INSERT INTO usage_dashboard_hourly`,
		`INSERT INTO usage_dashboard_daily`,
	} {
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow(startDate, "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(start))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups`).
		WithArgs(startDate).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(firstBucketStart.UTC(), secondBucketStart.UTC(), "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs(service.GroupUsageDate(secondBucketStart), start, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow(service.GroupUsageDate(secondBucketStart), "Asia/Shanghai"))
	mock.ExpectQuery(`SELECT MIN\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(start))
	mock.ExpectExec(`DELETE FROM usage_group_daily_rollups`).
		WithArgs(service.GroupUsageDate(secondBucketStart)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO usage_group_daily_rollups`).
		WithArgs(secondBucketStart.UTC(), todayStart.UTC(), "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs(service.GroupUsageDate(todayStart), start, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.RecomputeRange(context.Background(), start, end))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositoryCleanupUsageLogsNonPartitionedInvalidatesEachBatchAndSyncs(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	cutoff := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	earliestDeletedAt := time.Date(2026, 5, 3, 2, 0, 0, 0, time.UTC)
	fixedNow := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	repo.clock = func() time.Time { return fixedNow }
	todayStart := service.GroupUsageTodayStart(fixedNow)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM usage_group_rollup_state.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`(?s)DELETE FROM usage_logs.*RETURNING created_at`).
		WithArgs(cutoff, usageLogsCleanupBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).
			AddRow(earliestDeletedAt.Add(time.Hour)).
			AddRow(earliestDeletedAt))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs(earliestDeletedAt, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow(service.GroupUsageDate(todayStart), "Asia/Shanghai"))
	mock.ExpectCommit()

	require.NoError(t, repo.CleanupUsageLogs(context.Background(), cutoff))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositoryCleanupUsageLogsPartitionedSortsAndInvalidatesEachDropBeforeSync(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	cutoff := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	aprilStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	juneStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	fixedNow := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	repo.clock = func() time.Time { return fixedNow }
	todayStart := service.GroupUsageTodayStart(fixedNow)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT c.relname`).
		WillReturnRows(sqlmock.NewRows([]string{"relname"}).
			AddRow("usage_logs_202606").
			AddRow("usage_logs_invalid").
			AddRow("usage_logs_202604").
			AddRow("usage_logs_202607"))

	for _, partition := range []struct {
		name  string
		start time.Time
	}{
		{name: "usage_logs_202604", start: aprilStart},
		{name: "usage_logs_202606", start: juneStart},
	} {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id FROM usage_group_rollup_state.*FOR UPDATE`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec(`UPDATE usage_group_rollup_state`).
			WithArgs(partition.start, "Asia/Shanghai").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DROP TABLE IF EXISTS "` + partition.name + `"`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT closed_before::text, timezone_name.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"closed_before", "timezone_name"}).
			AddRow(service.GroupUsageDate(todayStart), "Asia/Shanghai"))
	mock.ExpectCommit()

	require.NoError(t, repo.CleanupUsageLogs(context.Background(), cutoff))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositoryCleanupUsageLogsNonPartitionedFailureRollsBackWithoutSync(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	cutoff := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 5, 3, 2, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM usage_group_rollup_state.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT ctid.*ORDER BY created_at ASC, id ASC.*DELETE FROM usage_logs.*RETURNING created_at`).
		WithArgs(cutoff, usageLogsCleanupBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(deletedAt))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs(deletedAt, "Asia/Shanghai").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := repo.CleanupUsageLogs(context.Background(), cutoff)
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositoryCleanupUsageLogsPartitionFailureRollsBackAndStops(t *testing.T) {
	setGroupUsageRollupTestTimezone(t)
	db, mock := newSQLMock(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)
	cutoff := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	aprilStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	dropErr := errors.New("drop partition failed")

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT c.relname`).
		WillReturnRows(sqlmock.NewRows([]string{"relname"}).
			AddRow("usage_logs_202606").
			AddRow("usage_logs_202604"))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM usage_group_rollup_state.*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`UPDATE usage_group_rollup_state`).
		WithArgs(aprilStart, "Asia/Shanghai").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DROP TABLE IF EXISTS "usage_logs_202604"`).
		WillReturnError(dropErr)
	mock.ExpectRollback()

	err := repo.CleanupUsageLogs(context.Background(), cutoff)
	require.ErrorIs(t, err, dropErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func setGroupUsageRollupTestTimezone(t *testing.T) {
	t.Helper()
	useGroupUsageRepositoryTestTimezone(t, "Asia/Shanghai")
}
