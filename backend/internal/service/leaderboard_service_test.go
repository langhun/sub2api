package service

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

type exactTimeArg struct {
	expected time.Time
}

func (a exactTimeArg) Match(v driver.Value) bool {
	actual, ok := v.(time.Time)
	return ok && actual.Equal(a.expected)
}

func TestLeaderboardService_GetConsumptionLeaderboard_ReturnsSummaryAndChartItems(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))

	countQuery := regexp.QuoteMeta(`
		SELECT COUNT(*) FROM (
			SELECT ul.user_id
			FROM usage_logs ul
			INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
			WHERE ul.created_at >= $1 AND u.status = 'active' AND u.role != 'admin'
			GROUP BY ul.user_id
			HAVING SUM(ul.actual_cost) > 0
		) sub
	`)

	dataQuery := regexp.QuoteMeta(`
		SELECT u.username, u.email, COALESCE(SUM(ul.actual_cost), 0) as total_cost, COUNT(*) as request_count
		FROM usage_logs ul
		INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
		WHERE ul.created_at >= $1 AND u.status = 'active' AND u.role != 'admin'
		GROUP BY ul.user_id, u.username, u.email
		HAVING SUM(ul.actual_cost) > 0
		ORDER BY total_cost DESC, ul.user_id ASC
		LIMIT $2 OFFSET $3
	`)

	chartQuery := regexp.QuoteMeta(`
		SELECT u.username, u.email, COALESCE(SUM(ul.actual_cost), 0) as total_cost
		FROM usage_logs ul
		INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
		WHERE ul.created_at >= $1 AND u.status = 'active' AND u.role != 'admin'
		GROUP BY ul.user_id, u.username, u.email
		HAVING SUM(ul.actual_cost) > 0
		ORDER BY total_cost DESC, ul.user_id ASC
	`)

	testCases := []struct {
		name     string
		period   string
		expected time.Time
	}{
		{
			name:     "daily",
			period:   "daily",
			expected: timezone.Today(),
		},
		{
			name:     "weekly",
			period:   "weekly",
			expected: timezone.Today().AddDate(0, 0, -7),
		},
		{
			name:     "monthly",
			period:   "monthly",
			expected: timezone.Today().AddDate(0, 0, -30),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			mock.ExpectQuery(countQuery).
				WithArgs(exactTimeArg{expected: tc.expected}).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

			mock.ExpectQuery(dataQuery).
				WithArgs(exactTimeArg{expected: tc.expected}, 2, 0).
				WillReturnRows(
					sqlmock.NewRows([]string{"username", "email", "total_cost", "request_count"}).
						AddRow("Alpha", "alpha@example.com", 12.345, 3).
						AddRow("", "beta@example.com", 5.4, 2),
				)

			mock.ExpectQuery(chartQuery).
				WithArgs(exactTimeArg{expected: tc.expected}).
				WillReturnRows(
					sqlmock.NewRows([]string{"username", "email", "total_cost"}).
						AddRow("Alpha", "alpha@example.com", 12.345).
						AddRow("", "beta@example.com", 5.4).
						AddRow("Gamma", "gamma@example.com", 1.005),
				)

			svc := &LeaderboardService{db: db}
			result, err := svc.GetConsumptionLeaderboard(context.Background(), tc.period, 1, 2, false)
			require.NoError(t, err)

			require.Equal(t, int64(3), result.Total)
			require.Len(t, result.Entries, 2)
			require.Equal(t, 1, result.Entries[0].Rank)
			require.Equal(t, "Alpha", result.Entries[0].Username)
			require.Equal(t, 12.35, result.Entries[0].Value)
			require.Equal(t, 3, result.Entries[0].ExtraInt)
			require.Equal(t, "bet***", result.Entries[1].Username)
			require.Equal(t, 5.4, result.Entries[1].Value)

			require.NotNil(t, result.Summary)
			require.Equal(t, 18.75, result.Summary.TotalValue)
			require.Equal(t, int64(3), result.Summary.TotalUsers)

			require.Len(t, result.ChartItems, 3)
			require.Equal(t, "Alpha", result.ChartItems[0].Username)
			require.Equal(t, 12.35, result.ChartItems[0].Value)
			require.Equal(t, "bet***", result.ChartItems[1].Username)
			require.Equal(t, 5.4, result.ChartItems[1].Value)
			require.Equal(t, "Gamma", result.ChartItems[2].Username)
			require.Equal(t, 1.0, result.ChartItems[2].Value)

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLeaderboardService_GetConsumptionLeaderboard_CanIncludeAdmins(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))

	countQuery := regexp.QuoteMeta(`
		SELECT COUNT(*) FROM (
			SELECT ul.user_id
			FROM usage_logs ul
			INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
			WHERE ul.created_at >= $1 AND u.status = 'active'
			GROUP BY ul.user_id
			HAVING SUM(ul.actual_cost) > 0
		) sub
	`)

	dataQuery := regexp.QuoteMeta(`
		SELECT u.username, u.email, COALESCE(SUM(ul.actual_cost), 0) as total_cost, COUNT(*) as request_count
		FROM usage_logs ul
		INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
		WHERE ul.created_at >= $1 AND u.status = 'active'
		GROUP BY ul.user_id, u.username, u.email
		HAVING SUM(ul.actual_cost) > 0
		ORDER BY total_cost DESC, ul.user_id ASC
		LIMIT $2 OFFSET $3
	`)

	chartQuery := regexp.QuoteMeta(`
		SELECT u.username, u.email, COALESCE(SUM(ul.actual_cost), 0) as total_cost
		FROM usage_logs ul
		INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
		WHERE ul.created_at >= $1 AND u.status = 'active'
		GROUP BY ul.user_id, u.username, u.email
		HAVING SUM(ul.actual_cost) > 0
		ORDER BY total_cost DESC, ul.user_id ASC
	`)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	startTime := timezone.Today()
	mock.ExpectQuery(countQuery).
		WithArgs(exactTimeArg{expected: startTime}).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(dataQuery).
		WithArgs(exactTimeArg{expected: startTime}, 10, 0).
		WillReturnRows(
			sqlmock.NewRows([]string{"username", "email", "total_cost", "request_count"}).
				AddRow("AdminUser", "admin@example.com", 88.8, 6),
		)

	mock.ExpectQuery(chartQuery).
		WithArgs(exactTimeArg{expected: startTime}).
		WillReturnRows(
			sqlmock.NewRows([]string{"username", "email", "total_cost"}).
				AddRow("AdminUser", "admin@example.com", 88.8),
		)

	svc := &LeaderboardService{db: db}
	result, err := svc.GetConsumptionLeaderboard(context.Background(), "daily", 1, 10, true)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Entries, 1)
	require.Equal(t, "AdminUser", result.Entries[0].Username)
	require.Equal(t, 88.8, result.Entries[0].Value)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardService_GetCheckinLeaderboard_UsesAggregatedSingleQuery(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))

	dataQuery := regexp.QuoteMeta(`
		WITH latest_checkins AS (
			SELECT user_id, MAX(checkin_date) AS last_date, COUNT(*) AS total_checkins
			FROM checkins
			GROUP BY user_id
		),
		eligible_checkins AS (
			SELECT u.username, u.email, c.streak_days, c.reward_amount,
				latest_checkins.total_checkins, latest_checkins.last_date
			FROM latest_checkins
			INNER JOIN checkins c ON c.user_id = latest_checkins.user_id AND c.checkin_date = latest_checkins.last_date
			INNER JOIN users u ON latest_checkins.user_id = u.id AND u.deleted_at IS NULL
			WHERE latest_checkins.last_date >= $1 AND u.status = 'active' AND u.role != 'admin'
		),
		page_checkins AS (
			SELECT *
			FROM eligible_checkins
			ORDER BY streak_days DESC, last_date DESC
			LIMIT $2 OFFSET $3
		)
		SELECT p.username, p.email, p.streak_days, p.reward_amount,
			p.total_checkins, p.last_date, total.total_count
		FROM (SELECT COUNT(*) AS total_count FROM eligible_checkins) total
		LEFT JOIN page_checkins p ON TRUE
		ORDER BY p.streak_days DESC NULLS LAST, p.last_date DESC NULLS LAST
	`)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	yesterday := timezone.Today().AddDate(0, 0, -1)
	lastDate := timezone.Today()
	mock.ExpectQuery(dataQuery).
		WithArgs(exactTimeArg{expected: yesterday}, 2, 0).
		WillReturnRows(
			sqlmock.NewRows([]string{"username", "email", "streak_days", "reward_amount", "total_checkins", "last_date", "total_count"}).
				AddRow("Alpha", "alpha@example.com", 5, 1.25, 8, lastDate, 3).
				AddRow("", "beta@example.com", 3, 0.75, 4, lastDate, 3),
		)

	svc := &LeaderboardService{db: db}
	result, err := svc.GetCheckinLeaderboard(context.Background(), 1, 2, false)
	require.NoError(t, err)

	require.Equal(t, int64(3), result.Total)
	require.Len(t, result.Entries, 2)
	require.Equal(t, 1, result.Entries[0].Rank)
	require.Equal(t, "Alpha", result.Entries[0].Username)
	require.Equal(t, 5.0, result.Entries[0].Value)
	require.Equal(t, 8, result.Entries[0].ExtraInt)
	require.Equal(t, 1.25, result.Entries[0].ExtraFloat)
	require.Equal(t, lastDate.Format("2006-01-02"), result.Entries[0].ExtraDate)
	require.Equal(t, "bet***", result.Entries[1].Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardService_GetCheckinCounts_BatchesUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	batchQuery := regexp.QuoteMeta(`
		SELECT user_id, COUNT(*)
		FROM checkins
		WHERE user_id = ANY($1)
		GROUP BY user_id
	`)
	mock.ExpectQuery(batchQuery).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(
			sqlmock.NewRows([]string{"user_id", "count"}).
				AddRow(1, 3).
				AddRow(2, 7),
		)

	svc := &LeaderboardService{db: db}
	counts, err := svc.getCheckinCounts(context.Background(), []int64{1, 2})
	require.NoError(t, err)
	require.Equal(t, map[int64]int{1: 3, 2: 7}, counts)
	require.NoError(t, mock.ExpectationsWereMet())
}
