package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildContentModerationLogWhere_BlockedIncludesAllBlockActions(t *testing.T) {
	where, args := buildContentModerationLogWhere(service.ContentModerationLogFilter{Result: "blocked"})

	require.Empty(t, args)
	sql := strings.Join(where, " AND ")
	require.Contains(t, sql, "l.action IN ('block', 'keyword_block', 'hash_block')")
	require.NotContains(t, sql, "l.action = 'block'")
}

func TestBuildContentModerationLogWhere_SearchIncludesUsername(t *testing.T) {
	where, args := buildContentModerationLogWhere(service.ContentModerationLogFilter{Search: "alice"})

	sql := strings.Join(where, " AND ")
	require.Contains(t, sql, "search_user.username ILIKE $3")
	require.Len(t, args, 6)
	for _, arg := range args {
		require.Equal(t, "%alice%", arg)
	}
}

func TestContentModerationRepositoryListLogsIncludesUsername(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM content_moderation_logs l`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT.*COALESCE\(u\.username, ''\).*FROM content_moderation_logs l.*LEFT JOIN users u`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "request_id", "user_id", "user_email", "username", "api_key_id", "api_key_name",
			"group_id", "group_name", "endpoint", "provider", "model", "mode", "action", "flagged",
			"highest_category", "highest_score", "category_scores", "threshold_snapshot", "input_excerpt",
			"upstream_latency_ms", "error", "violation_count", "auto_banned", "email_sent", "user_status",
			"queue_delay_ms", "matched_keyword", "created_at",
		}).AddRow(
			int64(1), "req-1", int64(7), "snapshot@example.com", "alice", nil, "",
			nil, "", "/v1/chat", "openai", "gpt", "audit", "allow", false,
			"", 0.0, `{}`, `{}`, "excerpt", nil, "", 0, false, false, "active", nil, "", now,
		))

	repo := NewContentModerationRepository(db)
	items, page, err := repo.ListLogs(context.Background(), service.ContentModerationLogFilter{
		Pagination: pagination.PaginationParams{Page: 1, PageSize: 20},
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "alice", items[0].Username)
	require.Equal(t, "snapshot@example.com", items[0].UserEmail)
	require.EqualValues(t, 1, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesHashBlock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND action <> 'hash_block'")).
		WithArgs(int64(1001), since, false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, false)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesCyberPolicyWhenRequested(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND ($3::bool IS FALSE OR action <> 'cyber_policy')")).
		WithArgs(int64(1001), since, true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, true)

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, mock.ExpectationsWereMet())
}
