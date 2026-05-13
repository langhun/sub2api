package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryListRequestDetails_SuccessRowsIncludeLatencyBreakdown(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	filter := &service.OpsRequestDetailFilter{
		StartTime: &start,
		EndTime:   &end,
		Page:      1,
		PageSize:  10,
	}

	mock.ExpectQuery("WITH combined AS \\(").
		WithArgs(start.UTC(), end.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery("FROM combined").
		WithArgs(start.UTC(), end.UTC(), 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"kind",
			"created_at",
			"request_id",
			"platform",
			"model",
			"duration_ms",
			"first_token_ms",
			"auth_latency_ms",
			"routing_latency_ms",
			"upstream_latency_ms",
			"response_latency_ms",
			"status_code",
			"error_id",
			"phase",
			"severity",
			"message",
			"user_id",
			"api_key_id",
			"account_id",
			"group_id",
			"stream",
		}).AddRow(
			"success",
			start.Add(5*time.Minute),
			"req-success-1",
			"openai",
			"gpt-5.4",
			int64(1200),
			int64(300),
			int64(40),
			int64(50),
			int64(60),
			int64(70),
			nil,
			nil,
			nil,
			nil,
			nil,
			int64(1),
			int64(2),
			int64(3),
			int64(4),
			true,
		))

	items, total, err := repo.ListRequestDetails(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].AuthLatencyMs)
	require.NotNil(t, items[0].RoutingLatencyMs)
	require.NotNil(t, items[0].UpstreamLatencyMs)
	require.NotNil(t, items[0].ResponseLatencyMs)
	require.Equal(t, 40, *items[0].AuthLatencyMs)
	require.Equal(t, 50, *items[0].RoutingLatencyMs)
	require.Equal(t, 60, *items[0].UpstreamLatencyMs)
	require.Equal(t, 70, *items[0].ResponseLatencyMs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryListRequestDetails_ErrorRowsInclude499ClientDisconnected(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	filter := &service.OpsRequestDetailFilter{
		StartTime: &start,
		EndTime:   &end,
		Page:      1,
		PageSize:  10,
		Kind:      string(service.OpsRequestKindError),
	}

	mock.ExpectQuery("WITH combined AS \\(").
		WithArgs(start.UTC(), end.UTC(), string(service.OpsRequestKindError)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery("FROM combined").
		WithArgs(start.UTC(), end.UTC(), string(service.OpsRequestKindError), 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"kind",
			"created_at",
			"request_id",
			"platform",
			"model",
			"duration_ms",
			"first_token_ms",
			"auth_latency_ms",
			"routing_latency_ms",
			"upstream_latency_ms",
			"response_latency_ms",
			"status_code",
			"error_id",
			"phase",
			"severity",
			"message",
			"user_id",
			"api_key_id",
			"account_id",
			"group_id",
			"stream",
		}).AddRow(
			"error",
			start.Add(10*time.Minute),
			"req-error-499",
			"openai",
			"gpt-5.5",
			int64(101000),
			nil,
			nil,
			nil,
			nil,
			int64(99000),
			int64(499),
			int64(77),
			"network",
			"P2",
			"client disconnected during final flush",
			int64(1),
			int64(2),
			int64(3),
			int64(4),
			false,
		))

	items, total, err := repo.ListRequestDetails(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, service.OpsRequestKindError, items[0].Kind)
	require.NotNil(t, items[0].StatusCode)
	require.Equal(t, 499, *items[0].StatusCode)
	require.Equal(t, "network", items[0].Phase)
	require.NotNil(t, items[0].ResponseLatencyMs)
	require.Equal(t, 99000, *items[0].ResponseLatencyMs)
	require.Nil(t, items[0].FirstTokenMs)
	require.NoError(t, mock.ExpectationsWereMet())
}
