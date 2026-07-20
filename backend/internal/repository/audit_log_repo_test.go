package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildAuditLogsWhereMatchesActorUsernameAndEmail(t *testing.T) {
	where, args := buildAuditLogsWhere(&service.AuditLogFilter{Actor: "  alice_100%  "})

	require.Contains(t, where, "u.username ILIKE $1")
	require.Contains(t, where, "u.email ILIKE $1")
	require.Contains(t, where, "l.actor_email ILIKE $1")
	require.Equal(t, []any{`%alice\_100\%%`}, args)
}

func TestBuildAuditLogsWhereKeepsLegacyActorEmailFilter(t *testing.T) {
	where, args := buildAuditLogsWhere(&service.AuditLogFilter{ActorEmail: "legacy@example.com"})

	require.Contains(t, where, "u.username ILIKE $1")
	require.Contains(t, where, "l.actor_email ILIKE $1")
	require.Equal(t, []any{"%legacy@example.com%"}, args)
}

func TestBuildAuditLogsWhereQueryIncludesActorUsername(t *testing.T) {
	where, _ := buildAuditLogsWhere(&service.AuditLogFilter{Query: "alice"})

	require.Contains(t, where, "u.username ILIKE $1")
	require.Contains(t, where, "u.email ILIKE $1")
}

func TestAuditLogRepositoryGetByIDJoinsActorUsername(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "actor_user_id", "actor_username", "actor_email", "actor_role",
		"auth_method", "credential_masked", "action", "method", "path", "request_id",
		"client_ip", "user_agent", "request_body", "status_code", "latency_ms", "extra",
	}).AddRow(
		int64(9), now, int64(42), "alice", "old@example.com", "admin",
		"jwt", "masked", "admin.users.read", "GET", "/api/v1/admin/users", "req-1",
		"127.0.0.1", "test", "", 200, int64(12), `{"source":"test"}`,
	)
	mock.ExpectQuery(`(?s)SELECT.*FROM audit_logs l LEFT JOIN users u ON u\.id = l\.actor_user_id WHERE l\.id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(rows)

	repo := &auditLogRepository{db: db}
	item, err := repo.GetByID(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, "alice", item.ActorUsername)
	require.Equal(t, "old@example.com", item.ActorEmail)
	require.Equal(t, "test", item.Extra["source"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuditLogSelectUsesLeftJoinForHistoricalRows(t *testing.T) {
	require.True(t, strings.Contains(auditLogFromClause, "LEFT JOIN users"))
}

func TestAuditLogJSONIncludesActorUsername(t *testing.T) {
	item := &service.AuditLog{ActorUsername: "alice", ActorEmail: "old@example.com"}
	encoded, err := json.Marshal(item)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"actor_username":"alice"`)
}
