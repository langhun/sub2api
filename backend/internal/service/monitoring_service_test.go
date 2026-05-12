package service

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMonitoringServiceQueryGroupModelStats_ErrorOnlyUsesJoinedGroupName(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	usageRows := sqlmock.NewRows([]string{
		"group_id",
		"group_name",
		"model",
		"cnt",
		"success_cnt",
		"error_cnt",
		"avg_latency_ms",
		"p50_latency_ms",
		"p95_latency_ms",
		"avg_ttft",
	})

	errorRows := sqlmock.NewRows([]string{"group_id", "group_name", "model", "err_cnt"}).
		AddRow(int64(42), "生产分组", "gpt-5", 3)

	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs u")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(usageRows)
	mock.ExpectQuery(regexp.QuoteMeta("JOIN groups g ON e.group_id = g.id AND g.deleted_at IS NULL")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(errorRows)

	overview := &MonitoringOverview{}
	err = NewMonitoringService(db).queryGroupModelStats(context.Background(), overview)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, []GroupModelStats{
		{
			GroupID:      42,
			GroupName:    "生产分组",
			Model:        "gpt-5",
			RequestCount: 3,
			SuccessCount: 0,
			ErrorCount:   3,
		},
	}, overview.GroupModels)
}
