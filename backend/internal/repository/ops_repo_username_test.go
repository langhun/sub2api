package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildOpsErrorLogsWhereUserQueryIncludesUsernameAndEmail(t *testing.T) {
	where, args := buildOpsErrorLogsWhere(&service.OpsErrorLogFilter{UserQuery: "alice", View: "all"})

	require.Contains(t, where, "u.username ILIKE $1 OR u.email ILIKE $1")
	require.Equal(t, []any{"%alice%"}, args)
}

func TestOpsRepositoryListErrorLogsIncludesUsername(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM ops_error_logs e`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT.*COALESCE\(u\.username, ''\).*FROM ops_error_logs e`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "error_phase", "error_type", "error_owner", "error_source", "severity",
			"status_code", "platform", "model", "resolved", "resolved_at", "resolved_by_user_id",
			"resolved_by_user_name", "client_request_id", "request_id", "error_message", "user_id",
			"user_email", "username", "api_key_id", "account_id", "account_name", "group_id", "group_name",
			"client_ip", "request_path", "stream", "inbound_endpoint", "upstream_endpoint", "requested_model",
			"upstream_model", "user_agent", "request_type", "api_key_name", "api_key_deleted_at",
		}).AddRow(
			int64(1), now, "request", "bad_request", "client", "client_request", "warn",
			400, "openai", "gpt", false, nil, nil, "", "client-1", "req-1", "bad request", int64(7),
			"alice@example.com", "alice", nil, nil, "", nil, "", nil, "/v1/chat", false, "chat", "upstream",
			"gpt", "gpt-upstream", "test", nil, "", nil,
		))

	repo := &opsRepository{db: db}
	result, err := repo.ListErrorLogs(context.Background(), &service.OpsErrorLogFilter{Page: 1, PageSize: 20, View: "all"})

	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	require.Equal(t, "alice", result.Errors[0].Username)
	require.Equal(t, "alice@example.com", result.Errors[0].UserEmail)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetErrorLogByIDIncludesUsername(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT.*COALESCE\(u\.username, ''\).*FROM ops_error_logs e`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "error_phase", "error_type", "error_owner", "error_source", "severity",
			"status_code", "platform", "model", "resolved", "resolved_at", "resolved_by_user_id",
			"client_request_id", "request_id", "error_message", "error_body", "upstream_status_code",
			"upstream_error_message", "upstream_error_detail", "upstream_errors", "is_business_limited",
			"user_id", "user_email", "username", "api_key_id", "account_id", "account_name", "group_id",
			"group_name", "client_ip", "request_path", "stream", "inbound_endpoint", "upstream_endpoint",
			"requested_model", "upstream_model", "request_type", "user_agent", "auth_latency_ms",
			"routing_latency_ms", "upstream_latency_ms", "response_latency_ms", "time_to_first_token_ms",
			"api_key_prefix", "api_key_name", "api_key_deleted_at",
		}).AddRow(
			int64(1), now, "request", "bad_request", "client", "client_request", "warn",
			400, "openai", "gpt", false, nil, nil, "client-1", "req-1", "bad request", "body", nil,
			"", "", "", false, int64(7), "alice@example.com", "alice", nil, nil, "", nil, "", nil,
			"/v1/chat", false, "chat", "upstream", "gpt", "gpt-upstream", nil, "test", nil, nil, nil, nil, nil,
			"", "", nil,
		))

	repo := &opsRepository{db: db}
	result, err := repo.GetErrorLogByID(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, "alice", result.Username)
	require.Equal(t, "alice@example.com", result.UserEmail)
	require.NoError(t, mock.ExpectationsWereMet())
}
