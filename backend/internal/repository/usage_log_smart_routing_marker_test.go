package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestHydrateSmartRoutingFlagsMatchesRequestAndAPIKey(t *testing.T) {
	db, mock := newSQLMock(t)
	defer func() { _ = db.Close() }()
	// Marker hydration only needs the SQL handle; keep the Ent client absent so
	// this test guards the repository's actual dependency boundary.
	repo := &usageLogRepository{sql: db}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT request_id, api_key_id FROM custom_usage_log_routing_marks WHERE (request_id, api_key_id) IN (($1,$2),($3,$4),($5,$6))")).
		WithArgs("smart-request", int64(7), "ordinary-request", int64(7), "smart-request", int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"request_id", "api_key_id"}).AddRow("smart-request", int64(7)))

	logs := []service.UsageLog{
		{RequestID: "smart-request", APIKeyID: 7},
		{RequestID: "ordinary-request", APIKeyID: 7},
		{RequestID: "smart-request", APIKeyID: 8},
	}
	repo.hydrateSmartRoutingFlags(context.Background(), logs)
	require.True(t, logs[0].SmartRouting)
	require.False(t, logs[1].SmartRouting)
	require.False(t, logs[2].SmartRouting)
	require.NoError(t, mock.ExpectationsWereMet())
}
