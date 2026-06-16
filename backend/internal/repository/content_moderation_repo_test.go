package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
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

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesHashBlock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND action <> 'hash_block'")).
		WithArgs(int64(1001), since).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryListLogs_IncludesUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM content_moderation_logs l WHERE l.id IS NOT NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT
    l.id, l.request_id, l.user_id, l.user_email, COALESCE(u.username, ''), l.api_key_id, l.api_key_name, l.group_id, l.group_name,
    l.endpoint, l.provider, l.model, l.mode, l.action, l.flagged, l.highest_category, l.highest_score,
    l.category_scores, l.threshold_snapshot, l.input_excerpt, l.upstream_latency_ms, l.error,
    l.violation_count, l.auto_banned, l.email_sent, COALESCE(u.status, ''), l.queue_delay_ms, l.created_at
FROM content_moderation_logs l
LEFT JOIN users u ON u.id = l.user_id WHERE l.id IS NOT NULL
ORDER BY l.created_at DESC, l.id DESC
LIMIT $1 OFFSET $2`)).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "request_id", "user_id", "user_email", "username", "api_key_id", "api_key_name", "group_id", "group_name",
			"endpoint", "provider", "model", "mode", "action", "flagged", "highest_category", "highest_score",
			"category_scores", "threshold_snapshot", "input_excerpt", "upstream_latency_ms", "error",
			"violation_count", "auto_banned", "email_sent", "user_status", "queue_delay_ms", "created_at",
		}).AddRow(
			int64(1), "req_1", int64(8), "user@example.com", "Alice", int64(2), "key-a", int64(3), "grp",
			"/v1/messages", "openai", "gpt", "input", "block", true, "sexual", 0.98,
			[]byte(`{"sexual":0.98}`), []byte(`{"sexual":0.95}`), "excerpt", 120, "",
			3, true, true, "disabled", 9, time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
		))

	items, page, err := repo.ListLogs(context.Background(), service.ContentModerationLogFilter{})
	require.NoError(t, err)
	require.NotNil(t, page)
	require.Len(t, items, 1)
	require.Equal(t, "Alice", items[0].Username)
	require.Equal(t, "user@example.com", items[0].UserEmail)
	require.NoError(t, mock.ExpectationsWereMet())
}
