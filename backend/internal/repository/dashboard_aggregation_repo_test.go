package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCreateUsageLogsPartition_AddsTableComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newDashboardAggregationRepositoryWithSQL(db)
	month := time.Date(2026, 5, 17, 9, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	createQuery := regexp.QuoteMeta(
		`CREATE TABLE IF NOT EXISTS "usage_logs_202605" PARTITION OF usage_logs FOR VALUES FROM ('2026-05-01') TO ('2026-06-01')`,
	)
	commentQuery := regexp.QuoteMeta(
		`COMMENT ON TABLE "usage_logs_202605" IS '用量日志月分区表（按 UTC 月份切分）'`,
	)

	mock.ExpectExec(createQuery).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(commentQuery).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.createUsageLogsPartition(context.Background(), month)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
