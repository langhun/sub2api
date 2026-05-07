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
			result, err := svc.GetConsumptionLeaderboard(context.Background(), tc.period, 1, 2)
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
