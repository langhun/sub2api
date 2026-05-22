package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryCreateSyncRequestTypeAndLegacyFields(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	createdAt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	log := &service.UsageLog{
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		RequestID:      "req-1",
		Model:          "gpt-5",
		RequestedModel: "gpt-5",
		InputTokens:    10,
		OutputTokens:   20,
		TotalCost:      1,
		ActualCost:     1,
		BillingType:    service.BillingTypeBalance,
		RequestType:    service.RequestTypeWSV2,
		Stream:         false,
		OpenAIWSMode:   false,
		CreatedAt:      createdAt,
	}

	mock.ExpectQuery("INSERT INTO usage_logs").
		WithArgs(
			log.UserID,
			log.APIKeyID,
			log.AccountID,
			log.RequestID,
			log.Model,
			log.RequestedModel,
			sqlmock.AnyArg(), // upstream_model
			sqlmock.AnyArg(), // group_id
			sqlmock.AnyArg(), // subscription_id
			log.InputTokens,
			log.OutputTokens,
			log.CacheCreationTokens,
			log.CacheReadTokens,
			log.CacheCreation5mTokens,
			log.CacheCreation1hTokens,
			log.ImageOutputTokens,
			log.ImageOutputCost,
			log.InputCost,
			log.OutputCost,
			log.CacheCreationCost,
			log.CacheReadCost,
			log.TotalCost,
			log.ActualCost,
			log.RateMultiplier,
			log.AccountRateMultiplier,
			log.BillingType,
			int16(service.RequestTypeWSV2),
			true,
			true,
			sqlmock.AnyArg(), // duration_ms
			sqlmock.AnyArg(), // first_token_ms
			sqlmock.AnyArg(), // auth_latency_ms
			sqlmock.AnyArg(), // routing_latency_ms
			sqlmock.AnyArg(), // upstream_latency_ms
			sqlmock.AnyArg(), // response_latency_ms
			sqlmock.AnyArg(), // user_agent
			sqlmock.AnyArg(), // ip_address
			log.ImageCount,
			sqlmock.AnyArg(), // image_size
			sqlmock.AnyArg(), // image_input_size
			sqlmock.AnyArg(), // image_output_size
			sqlmock.AnyArg(), // image_size_source
			sqlmock.AnyArg(), // image_size_breakdown
			sqlmock.AnyArg(), // service_tier
			sqlmock.AnyArg(), // reasoning_effort
			sqlmock.AnyArg(), // inbound_endpoint
			sqlmock.AnyArg(), // upstream_endpoint
			log.CacheTTLOverridden,
			sqlmock.AnyArg(), // channel_id
			sqlmock.AnyArg(), // model_mapping_chain
			sqlmock.AnyArg(), // billing_tier
			sqlmock.AnyArg(), // billing_mode
			sqlmock.AnyArg(), // account_stats_cost
			createdAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(99), createdAt))

	inserted, err := repo.Create(context.Background(), log)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, int64(99), log.ID)
	require.Nil(t, log.ServiceTier)
	require.Equal(t, service.RequestTypeWSV2, log.RequestType)
	require.True(t, log.Stream)
	require.True(t, log.OpenAIWSMode)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryCreate_PersistsServiceTier(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	createdAt := time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)
	serviceTier := "priority"
	log := &service.UsageLog{
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		RequestID:      "req-service-tier",
		Model:          "gpt-5.4",
		RequestedModel: "gpt-5.4",
		ServiceTier:    &serviceTier,
		CreatedAt:      createdAt,
	}

	mock.ExpectQuery("INSERT INTO usage_logs").
		WithArgs(
			log.UserID,
			log.APIKeyID,
			log.AccountID,
			log.RequestID,
			log.Model,
			log.RequestedModel,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			log.InputTokens,
			log.OutputTokens,
			log.CacheCreationTokens,
			log.CacheReadTokens,
			log.CacheCreation5mTokens,
			log.CacheCreation1hTokens,
			log.ImageOutputTokens,
			log.ImageOutputCost,
			log.InputCost,
			log.OutputCost,
			log.CacheCreationCost,
			log.CacheReadCost,
			log.TotalCost,
			log.ActualCost,
			log.RateMultiplier,
			log.AccountRateMultiplier,
			log.BillingType,
			int16(service.RequestTypeSync),
			false,
			false,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			log.ImageCount,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(), // image_input_size
			sqlmock.AnyArg(), // image_output_size
			sqlmock.AnyArg(), // image_size_source
			sqlmock.AnyArg(), // image_size_breakdown
			serviceTier,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			log.CacheTTLOverridden,
			sqlmock.AnyArg(), // channel_id
			sqlmock.AnyArg(), // model_mapping_chain
			sqlmock.AnyArg(), // billing_tier
			sqlmock.AnyArg(), // billing_mode
			sqlmock.AnyArg(), // account_stats_cost
			createdAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(100), createdAt))

	inserted, err := repo.Create(context.Background(), log)
	require.NoError(t, err)
	require.True(t, inserted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildUsageLogBestEffortInsertQuery_IncludesRequestedModelColumn(t *testing.T) {
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		RequestID:      "req-best-effort-query",
		Model:          "gpt-5",
		RequestedModel: "gpt-5",
		CreatedAt:      time.Date(2025, 1, 3, 12, 0, 0, 0, time.UTC),
	})

	query, args := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{prepared})

	require.Contains(t, query, "INSERT INTO usage_logs (")
	require.Contains(t, query, "\n\t\t\tmodel,\n\t\t\trequested_model,\n\t\t\tupstream_model,")
	require.Contains(t, query, "\n\t\t\trequest_id,\n\t\t\tmodel,\n\t\t\trequested_model,\n\t\t\tupstream_model,")
	require.Len(t, args, len(prepared.args))
	require.Equal(t, prepared.args[5], args[5])
}

func TestExecUsageLogInsertNoResult_PersistsRequestedModel(t *testing.T) {
	db, mock := newSQLMock(t)
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		RequestID:      "req-best-effort-exec",
		Model:          "gpt-5",
		RequestedModel: "gpt-5",
		CreatedAt:      time.Date(2025, 1, 4, 12, 0, 0, 0, time.UTC),
	})

	mock.ExpectExec("INSERT INTO usage_logs").
		WithArgs(anySliceToDriverValues(prepared.args)...).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := execUsageLogInsertNoResult(context.Background(), db, prepared)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecUsageLogInsertNoResult_ColumnListMatchesPreparedArgs(t *testing.T) {
	authLatency := 11
	routingLatency := 22
	upstreamLatency := 33
	responseLatency := 44
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:            1,
		APIKeyID:          2,
		AccountID:         3,
		RequestID:         "req-best-effort-column-count",
		Model:             "gpt-5",
		RequestedModel:    "gpt-5",
		AuthLatencyMs:     &authLatency,
		RoutingLatencyMs:  &routingLatency,
		UpstreamLatencyMs: &upstreamLatency,
		ResponseLatencyMs: &responseLatency,
		CreatedAt:         time.Date(2025, 1, 4, 13, 0, 0, 0, time.UTC),
	})
	recorder := &capturingSQLExecutor{}

	err := execUsageLogInsertNoResult(context.Background(), recorder, prepared)
	require.NoError(t, err)

	columns := extractUsageLogInsertColumns(t, recorder.query)
	require.Len(t, columns, len(prepared.args))
	require.Len(t, columns, len(usageLogInsertArgTypes))
	require.Equal(t, len(prepared.args), countUsageLogInsertPlaceholders(recorder.query))
	require.Equal(t, prepared.args, recorder.args)
	require.Subset(t, columns, []string{
		"auth_latency_ms",
		"routing_latency_ms",
		"upstream_latency_ms",
		"response_latency_ms",
	})
}

func TestPrepareUsageLogInsert_ArgCountMatchesTypes(t *testing.T) {
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		RequestID:      "req-arg-count",
		Model:          "gpt-5",
		RequestedModel: "gpt-5",
		CreatedAt:      time.Date(2025, 1, 5, 12, 0, 0, 0, time.UTC),
	})

	require.Len(t, prepared.args, len(usageLogInsertArgTypes))
}

func TestPrepareUsageLogInsert_PersistsImageSizeMetadata(t *testing.T) {
	imageSize := "4K"
	inputSize := "1024x1024"
	outputSize := "3840x2160"
	source := "output"
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:             1,
		APIKeyID:           2,
		AccountID:          3,
		RequestID:          "req-image-metadata",
		Model:              "gpt-image-2",
		RequestedModel:     "gpt-image-2",
		ImageCount:         2,
		ImageSize:          &imageSize,
		ImageInputSize:     &inputSize,
		ImageOutputSize:    &outputSize,
		ImageSizeSource:    &source,
		ImageSizeBreakdown: map[string]int{"1K": 1, "4K": 1},
		CreatedAt:          time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC),
	})

	require.Equal(t, sql.NullString{String: imageSize, Valid: true}, prepared.args[38])
	require.Equal(t, sql.NullString{String: inputSize, Valid: true}, prepared.args[39])
	require.Equal(t, sql.NullString{String: outputSize, Valid: true}, prepared.args[40])
	require.Equal(t, sql.NullString{String: source, Valid: true}, prepared.args[41])
	breakdownJSON, ok := prepared.args[42].(string)
	require.True(t, ok)
	require.JSONEq(t, `{"1K":1,"4K":1}`, breakdownJSON)
}

func TestCoalesceTrimmedString(t *testing.T) {
	require.Equal(t, "fallback", coalesceTrimmedString(sql.NullString{}, "fallback"))
	require.Equal(t, "fallback", coalesceTrimmedString(sql.NullString{Valid: true, String: "   "}, "fallback"))
	require.Equal(t, "value", coalesceTrimmedString(sql.NullString{Valid: true, String: "value"}, "fallback"))
}

func TestAppendUsageLogBillingModeWhereCondition(t *testing.T) {
	tests := []struct {
		name          string
		billingMode   string
		wantCondition string
	}{
		{
			name:          "image includes legacy image rows",
			billingMode:   string(service.BillingModeImage),
			wantCondition: "(billing_mode = $1 OR COALESCE(image_count, 0) > 0)",
		},
		{
			name:          "token includes legacy non-image rows",
			billingMode:   string(service.BillingModeToken),
			wantCondition: "(billing_mode = $1 OR ((billing_mode IS NULL OR billing_mode = '') AND COALESCE(image_count, 0) <= 0))",
		},
		{
			name:          "per request remains exact",
			billingMode:   string(service.BillingModePerRequest),
			wantCondition: "billing_mode = $1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, args := appendUsageLogBillingModeWhereCondition(nil, nil, tt.billingMode)
			require.Equal(t, []string{tt.wantCondition}, conditions)
			require.Equal(t, []any{tt.billingMode}, args)
		})
	}
}

func anySliceToDriverValues(values []any) []driver.Value {
	out := make([]driver.Value, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

type capturingSQLExecutor struct {
	query string
	args  []any
}

func (c *capturingSQLExecutor) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	c.query = query
	c.args = append([]any(nil), args...)
	return driver.RowsAffected(1), nil
}

func (c *capturingSQLExecutor) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected QueryContext")
}

func extractUsageLogInsertColumns(t *testing.T, query string) []string {
	t.Helper()
	const startMarker = "INSERT INTO usage_logs ("
	const endMarker = ") VALUES"
	start := strings.Index(query, startMarker)
	require.NotEqual(t, -1, start)
	columnBlock := query[start+len(startMarker):]
	end := strings.Index(columnBlock, endMarker)
	require.NotEqual(t, -1, end)

	rawColumns := strings.Split(columnBlock[:end], ",")
	columns := make([]string, 0, len(rawColumns))
	for _, raw := range rawColumns {
		column := strings.TrimSpace(raw)
		if column != "" {
			columns = append(columns, column)
		}
	}
	return columns
}

func countUsageLogInsertPlaceholders(query string) int {
	matches := regexp.MustCompile(`\$(\d+)`).FindAllStringSubmatch(query, -1)
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		seen[match[1]] = struct{}{}
	}
	return len(seen)
}

func TestUsageLogRepositoryListWithFiltersRequestTypePriority(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	requestType := int16(service.RequestTypeWSV2)
	stream := false
	filters := usagestats.UsageLogFilters{
		RequestType: &requestType,
		Stream:      &stream,
		ExactTotal:  true,
	}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM usage_logs WHERE \\(request_type = \\$1 OR \\(request_type = 0 AND openai_ws_mode = TRUE\\)\\)").
		WithArgs(requestType).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery("SELECT .* FROM usage_logs WHERE \\(request_type = \\$1 OR \\(request_type = 0 AND openai_ws_mode = TRUE\\)\\) ORDER BY id DESC LIMIT \\$2 OFFSET \\$3").
		WithArgs(requestType, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	logs, page, err := repo.ListWithFilters(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, filters)
	require.NoError(t, err)
	require.Empty(t, logs)
	require.NotNil(t, page)
	require.Equal(t, int64(0), page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetUsageTrendWithFiltersRequestTypePriority(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	requestType := int16(service.RequestTypeStream)
	stream := true

	mock.ExpectQuery("AND \\(request_type = \\$3 OR \\(request_type = 0 AND stream = TRUE AND openai_ws_mode = FALSE\\)\\)").
		WithArgs(start, end, requestType).
		WillReturnRows(sqlmock.NewRows([]string{"date", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost"}))

	trend, err := repo.GetUsageTrendWithFilters(context.Background(), start, end, "day", 0, 0, 0, 0, "", &requestType, &stream, nil)
	require.NoError(t, err)
	require.Empty(t, trend)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetModelStatsWithFiltersRequestTypePriority(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	requestType := int16(service.RequestTypeWSV2)
	stream := false

	mock.ExpectQuery("AND \\(request_type = \\$3 OR \\(request_type = 0 AND openai_ws_mode = TRUE\\)\\)").
		WithArgs(start, end, requestType).
		WillReturnRows(sqlmock.NewRows([]string{"model", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost", "account_cost"}))

	stats, err := repo.GetModelStatsWithFilters(context.Background(), start, end, 0, 0, 0, 0, &requestType, &stream, nil, 0)
	require.NoError(t, err)
	require.Empty(t, stats)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetModelStatsWithFiltersLimit(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	mock.ExpectQuery("FROM usage_logs").
		WithArgs(start, end, 25).
		WillReturnRows(sqlmock.NewRows([]string{
			"model", "requests", "input_tokens", "output_tokens",
			"cache_creation_tokens", "cache_read_tokens", "total_tokens",
			"cost", "actual_cost", "account_cost",
		}))

	stats, err := repo.GetModelStatsWithFilters(context.Background(), start, end, 0, 0, 0, 0, nil, nil, nil, 25)
	require.NoError(t, err)
	require.Empty(t, stats)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetStatsWithFiltersRequestTypePriority(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	requestType := int16(service.RequestTypeSync)
	stream := true
	filters := usagestats.UsageLogFilters{
		RequestType: &requestType,
		Stream:      &stream,
	}

	mock.ExpectQuery("FROM usage_logs\\s+WHERE \\(request_type = \\$1 OR \\(request_type = 0 AND stream = FALSE AND openai_ws_mode = FALSE\\)\\)").
		WithArgs(requestType).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_tokens",
			"total_cost",
			"total_actual_cost",
			"total_account_cost",
			"avg_duration_ms",
		}).AddRow(int64(1), int64(2), int64(3), int64(4), 1.2, 1.0, 1.2, 20.0))
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(inbound_endpoint\\), ''\\), 'unknown'\\) AS endpoint").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), requestType).
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(upstream_endpoint\\), ''\\), 'unknown'\\) AS endpoint").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), requestType).
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))
	mock.ExpectQuery("SELECT CONCAT\\(").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), requestType).
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))

	stats, err := repo.GetStatsWithFilters(context.Background(), filters)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalRequests)
	require.Equal(t, int64(9), stats.TotalTokens)
	require.NotNil(t, stats.TotalAccountCost, "TotalAccountCost should always be returned")
	require.Equal(t, 1.2, *stats.TotalAccountCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetStatsWithFiltersEndpointLimit(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	filters := usagestats.UsageLogFilters{
		StartTime:     &start,
		EndTime:       &end,
		EndpointLimit: 20,
	}

	mock.ExpectQuery("FROM usage_logs\\s+WHERE created_at >= \\$1 AND created_at < \\$2").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_tokens",
			"total_cost",
			"total_actual_cost",
			"total_account_cost",
			"avg_duration_ms",
		}).AddRow(int64(1), int64(2), int64(3), int64(4), 1.2, 1.0, 1.2, 20.0))
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(inbound_endpoint\\), ''\\), 'unknown'\\) AS endpoint").
		WithArgs(start, end, 20).
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(upstream_endpoint\\), ''\\), 'unknown'\\) AS endpoint").
		WithArgs(start, end, 20).
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))
	mock.ExpectQuery("SELECT CONCAT\\(").
		WithArgs(start, end, 20).
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))

	stats, err := repo.GetStatsWithFilters(context.Background(), filters)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalRequests)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetModelStatsAccountCostColumn(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	mock.ExpectQuery("FROM usage_logs").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"model", "requests", "input_tokens", "output_tokens",
			"cache_creation_tokens", "cache_read_tokens", "total_tokens",
			"cost", "actual_cost", "account_cost",
		}).
			AddRow("claude-opus-4-6", int64(10), int64(100), int64(200), int64(5), int64(3), int64(308), 2.5, 2.0, 1.8).
			AddRow("claude-sonnet-4-6", int64(5), int64(50), int64(100), int64(0), int64(0), int64(150), 1.0, 0.8, 0.7))

	results, err := repo.GetModelStatsWithFilters(context.Background(), start, end, 0, 0, 0, 0, nil, nil, nil, 0)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "claude-opus-4-6", results[0].Model)
	require.Equal(t, 2.5, results[0].Cost)
	require.Equal(t, 2.0, results[0].ActualCost)
	require.Equal(t, 1.8, results[0].AccountCost)
	require.Equal(t, "claude-sonnet-4-6", results[1].Model)
	require.Equal(t, 0.7, results[1].AccountCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetGroupStatsAccountCostColumn(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	mock.ExpectQuery("FROM usage_logs").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"group_id", "group_name", "requests", "total_tokens",
			"cost", "actual_cost", "account_cost",
		}).
			AddRow(int64(1), "azure-cc", int64(100), int64(5000), 10.0, 8.5, 7.2).
			AddRow(int64(2), "max", int64(50), int64(2000), 5.0, 4.0, 3.5))

	results, err := repo.GetGroupStatsWithFilters(context.Background(), start, end, 0, 0, 0, 0, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, int64(1), results[0].GroupID)
	require.Equal(t, "azure-cc", results[0].GroupName)
	require.Equal(t, 10.0, results[0].Cost)
	require.Equal(t, 8.5, results[0].ActualCost)
	require.Equal(t, 7.2, results[0].AccountCost)
	require.Equal(t, int64(2), results[1].GroupID)
	require.Equal(t, 3.5, results[1].AccountCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryExcludesDeletedGroups(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(1), 12.5, 2.5))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, int64(1), summaries[0].GroupID)
	require.Equal(t, 12.5, summaries[0].TotalCost)
	require.Equal(t, 2.5, summaries[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryUsesCache(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(1), 12.5, 2.5))

	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, first, 1)

	// 命中缓存后不应再次触发 SQL；若触发将因 sqlmock 无新期望而报错。
	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryTotalCacheHitSkipsRequery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(100), 88.8, 8.8))

	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, 88.8, first[0].TotalCost)

	// 第二次同日请求应命中 total cache，不应再触发 SQL。
	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, 88.8, second[0].TotalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryTodayCacheHit(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	// 本地时区日界线转换为同一 UTC 日界线后，应命中同一缓存键（today cache）。
	todayStartLocal := time.Date(2026, 5, 21, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	todayStartUTC := todayStartLocal.UTC()
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStartUTC).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(200), 66.6, 6.6))

	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStartLocal)
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, 6.6, first[0].TodayCost)

	// 使用等价 UTC 起始时间再次查询：应命中 today cache，不应二次查询数据库。
	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStartUTC)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, 6.6, second[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryExpiredTotalCacheUsesTodayAndIncrementalBeforeRebuildThreshold(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	// total_cost 长缓存过期，但 fullRebuildAt 仍在重建阈值内，应走增量+today 路径而非全量。
	now := time.Now().UTC()
	incrementalFrom := now.Add(-10 * time.Minute)
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       now.Add(-time.Minute),                          // 强制 needsRefresh=true
		fullRebuildAt:   now.Add(-(usageLogGroupTotalCostRebuildAt / 2)), // 未超阈值
		incrementalFrom: incrementalFrom,
		totalCost: map[int64]float64{
			1: 100.0,
			2: 200.0,
		},
	}

	todayStart := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)

	// 1) 先触发增量 total_cost 查询
	mock.ExpectQuery("SELECT\\s+ul.group_id,[\\s\\S]*delta_total_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*AND ul.created_at < \\$2").
		WithArgs(incrementalFrom, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "delta_total_cost"}).
			AddRow(int64(1), 1.25).
			AddRow(int64(2), 2.5))

	// 2) 再触发 today 查询并与增量后的 total_cost 合并
	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "today_cost"}).
			AddRow(int64(1), 10.0).
			AddRow(int64(2), 20.0))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	got := map[int64]usagestats.GroupUsageSummary{}
	for _, item := range summaries {
		got[item.GroupID] = item
	}
	require.Equal(t, 101.25, got[1].TotalCost)
	require.Equal(t, 10.0, got[1].TodayCost)
	require.Equal(t, 202.5, got[2].TotalCost)
	require.Equal(t, 20.0, got[2].TodayCost)

	// 若误走全量查询，这里会因未设置 full-query 期望而失败。
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryExpiredTotalCacheWithoutIncrementalFromFallsBackToFullQuery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	now := time.Now().UTC()
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       now.Add(-time.Minute),                           // 强制 needsRefresh=true
		fullRebuildAt:   now.Add(-(usageLogGroupTotalCostRebuildAt / 2)), // 未超全量重建阈值
		incrementalFrom: time.Time{},                                     // 缺失增量窗口起点
		totalCost: map[int64]float64{
			1: 100.0,
			2: 200.0,
		},
	}

	todayStart := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	// 仅允许 full query；若误走 today+旧 total 路径会因无匹配期望失败。
	mock.ExpectQuery("COALESCE\\(agg.total_cost, 0\\) AS total_cost[\\s\\S]*COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*GROUP BY ul.group_id").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(1), 12.0, 2.0).
			AddRow(int64(2), 0.0, 0.0))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	require.Equal(t, 12.0, summaries[0].TotalCost)
	require.Equal(t, 2.0, summaries[0].TodayCost)
	require.Equal(t, 0.0, summaries[1].TotalCost)
	require.Equal(t, 0.0, summaries[1].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryIncrementalRefreshMergesTotalCost(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	repo.setGroupUsageTotalCostCache(map[int64]float64{
		1: 10.5,
		2: 20.0,
	})

	todayStart := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "today_cost"}).
			AddRow(int64(1), 1.5).
			AddRow(int64(2), 0.0))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	got := map[int64]usagestats.GroupUsageSummary{}
	for _, item := range summaries {
		got[item.GroupID] = item
	}
	require.Equal(t, 10.5, got[1].TotalCost)
	require.Equal(t, 1.5, got[1].TodayCost)
	require.Equal(t, 20.0, got[2].TotalCost)
	require.Equal(t, 0.0, got[2].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryFreshTotalCacheSkipsIncrementalDeltaQuery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	// fresh total_cost 缓存：未过期且 incrementalFrom 刚设置，未达到增量刷新最小间隔。
	repo.setGroupUsageTotalCostCache(map[int64]float64{
		1: 88.0,
	})
	beforeTotalCost, ok, _, beforeNeedsRefresh, _, beforeIncrementalFrom, beforeHasIncrementalFrom := repo.getGroupUsageTotalCostCacheState(true)
	require.True(t, ok)
	require.True(t, beforeHasIncrementalFrom)
	require.False(t, beforeNeedsRefresh)
	require.Equal(t, map[int64]float64{1: 88.0}, beforeTotalCost)
	require.False(t, beforeIncrementalFrom.IsZero())

	todayStart := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	// 仅期望 today 查询；若误触发 delta SQL，会因无匹配期望而失败。
	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "today_cost"}).
			AddRow(int64(1), 8.0))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, int64(1), summaries[0].GroupID)
	require.Equal(t, 88.0, summaries[0].TotalCost)
	require.Equal(t, 8.0, summaries[0].TodayCost)
	afterTotalCost, ok, _, afterNeedsRefresh, _, afterIncrementalFrom, afterHasIncrementalFrom := repo.getGroupUsageTotalCostCacheState(true)
	require.True(t, ok)
	require.True(t, afterHasIncrementalFrom)
	require.False(t, afterNeedsRefresh)
	require.Equal(t, map[int64]float64{1: 88.0}, afterTotalCost)
	require.Equal(t, beforeIncrementalFrom, afterIncrementalFrom)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryFreshTotalCacheWithStaleIncrementalFromTriggersDeltaQuery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	now := time.Now().UTC()
	incrementalFrom := now.Add(-usageLogGroupTotalCostIncrementalMinInterval - time.Second)
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       now.Add(10 * time.Minute), // fresh total cache
		fullRebuildAt:   now.Add(-30 * time.Minute),
		incrementalFrom: incrementalFrom, // 触发 shouldRefreshIncremental=true
		totalCost: map[int64]float64{
			1: 88.0,
		},
	}

	todayStart := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	// fresh 缓存但 incrementalFrom 达到最小刷新间隔，必须触发 delta SQL。
	mock.ExpectQuery("SELECT\\s+ul.group_id,[\\s\\S]*delta_total_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*AND ul.created_at < \\$2").
		WithArgs(incrementalFrom, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "delta_total_cost"}).
			AddRow(int64(1), 2.0))

	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "today_cost"}).
			AddRow(int64(1), 8.0))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, int64(1), summaries[0].GroupID)
	require.Equal(t, 90.0, summaries[0].TotalCost)
	require.Equal(t, 8.0, summaries[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryFreshTotalCacheWithEmptyDeltaKeepsTotalCostAndMergesToday(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	now := time.Now().UTC()
	incrementalFrom := now.Add(-usageLogGroupTotalCostIncrementalMinInterval - time.Second)
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       now.Add(10 * time.Minute), // fresh total cache
		fullRebuildAt:   now.Add(-30 * time.Minute),
		incrementalFrom: incrementalFrom, // 触发 shouldRefreshIncremental=true
		totalCost: map[int64]float64{
			1: 88.0,
			2: 12.0,
		},
	}

	todayStart := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	// 增量查询返回空结果：refresh 仍应成功，不改变 total_cost 语义。
	mock.ExpectQuery("SELECT\\s+ul.group_id,[\\s\\S]*delta_total_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*AND ul.created_at < \\$2").
		WithArgs(incrementalFrom, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "delta_total_cost"}))

	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "today_cost"}).
			AddRow(int64(1), 8.0).
			AddRow(int64(2), 1.2))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	got := map[int64]usagestats.GroupUsageSummary{}
	for _, item := range summaries {
		got[item.GroupID] = item
	}
	require.Equal(t, 88.0, got[1].TotalCost)
	require.Equal(t, 8.0, got[1].TodayCost)
	require.Equal(t, 12.0, got[2].TotalCost)
	require.Equal(t, 1.2, got[2].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryRecalibrationThresholdForcesFullQuery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	// 超过 total_cost 长缓存阈值后，应回到全量汇总查询路径（校准）。
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt: time.Now().Add(-time.Minute),
		totalCost: map[int64]float64{1: 99.9},
	}

	todayStart := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("COALESCE\\(agg.total_cost, 0\\) AS total_cost[\\s\\S]*COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*GROUP BY ul.group_id").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(1), 12.0, 2.0).
			AddRow(int64(2), 0.0, 0.0))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	require.Equal(t, 12.0, summaries[0].TotalCost)
	require.Equal(t, 2.0, summaries[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryIncrementalQueryFailureFallsBackToFull(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	repo.setGroupUsageTotalCostCache(map[int64]float64{1: 30.0})

	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*AND ul.created_at >= \\$1").
		WithArgs(todayStart).
		WillReturnError(fmt.Errorf("today query failed"))

	_, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.Error(t, err)

	// 增量路径失败后，触发一次“全量校准”回退。
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt: time.Now().Add(-time.Second),
		totalCost: map[int64]float64{1: 30.0},
	}
	mock.ExpectQuery("COALESCE\\(agg.total_cost, 0\\) AS total_cost[\\s\\S]*COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*GROUP BY ul.group_id").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(1), 31.5, 1.5))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, 31.5, summaries[0].TotalCost)
	require.Equal(t, 1.5, summaries[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryTodayQueryFailureFallsBackToFull(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	repo.setGroupUsageTotalCostCache(map[int64]float64{1: 30.0})

	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*AND ul.created_at >= \\$1").
		WithArgs(todayStart).
		WillReturnError(fmt.Errorf("today query failed"))

	mock.ExpectQuery("COALESCE\\(agg.total_cost, 0\\) AS total_cost[\\s\\S]*COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*GROUP BY ul.group_id").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(1), 31.5, 1.5))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, 31.5, summaries[0].TotalCost)
	require.Equal(t, 1.5, summaries[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryIncrementalCacheReturnsCopy(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	repo.setGroupUsageTotalCostCache(map[int64]float64{1: 50.0})
	todayStart := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("COALESCE\\(agg.today_cost, 0\\) AS today_cost[\\s\\S]*AND ul.created_at >= \\$1").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "today_cost"}).
			AddRow(int64(1), 5.0))

	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, 50.0, first[0].TotalCost)
	require.Equal(t, 5.0, first[0].TodayCost)

	first[0].TotalCost = 999
	first[0].TodayCost = -10

	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, 50.0, second[0].TotalCost)
	require.Equal(t, 5.0, second[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryRefreshGroupUsageTotalCostIncrementalConflictReturnsLatestSnapshot(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	baseNow := time.Now().UTC()
	baseFrom := baseNow.Add(-15 * time.Minute)
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       baseNow.Add(10 * time.Minute),
		fullRebuildAt:   baseNow.Add(-30 * time.Minute),
		incrementalFrom: baseFrom,
		totalCost: map[int64]float64{
			1: 10.0,
		},
	}

	// 延迟增量查询，给并发 goroutine 留出时间推进 incrementalFrom。
	mock.ExpectQuery("SELECT\\s+ul.group_id,[\\s\\S]*delta_total_cost[\\s\\S]*FROM usage_logs ul[\\s\\S]*AND ul.created_at >= \\$1[\\s\\S]*AND ul.created_at < \\$2").
		WithArgs(baseFrom, sqlmock.AnyArg()).
		WillDelayFor(80 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "delta_total_cost"}).
			AddRow(int64(1), 3.0))

	advancedDone := make(chan struct{})
	go func() {
		time.Sleep(20 * time.Millisecond)
		repo.groupUsageTotalCostCacheMu.Lock()
		repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
			expiresAt:       time.Now().Add(10 * time.Minute),
			fullRebuildAt:   baseNow.Add(-20 * time.Minute),
			incrementalFrom: baseFrom.Add(5 * time.Minute), // 使 expectedFrom 不匹配
			totalCost: map[int64]float64{
				1: 999.0, // 最新快照（应优先返回）
			},
		}
		repo.groupUsageTotalCostCacheMu.Unlock()
		close(advancedDone)
	}()

	refreshed, err := repo.refreshGroupUsageTotalCostIncremental(context.Background(), map[int64]float64{
		1: 10.0, // stale current（若无冲突会变成 13）
	})
	require.NoError(t, err)
	<-advancedDone

	// 并发推进导致 setGroupUsageTotalCostCacheFromIncrement 返回 false，
	// 函数应返回最新缓存快照，而不是 stale refreshed 结果。
	require.Equal(t, 999.0, refreshed[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryRefreshGroupUsageTotalCostIncrementalFreshCacheSkipsWriteSideEffects(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	baseNow := time.Now().UTC()
	entry := usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       baseNow.Add(10 * time.Minute),
		fullRebuildAt:   baseNow.Add(-30 * time.Minute),
		incrementalFrom: baseNow.Add(-time.Minute), // 未达到最小增量刷新间隔
		totalCost: map[int64]float64{
			1: 88.0,
			2: 12.0,
		},
	}
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       entry.expiresAt,
		fullRebuildAt:   entry.fullRebuildAt,
		incrementalFrom: entry.incrementalFrom,
		totalCost:       cloneGroupUsageTotalCostMap(entry.totalCost),
	}

	refreshed, err := repo.refreshGroupUsageTotalCostIncremental(context.Background(), map[int64]float64{
		1: 999.0, // 传入 stale current，不应回写或覆盖 fresh cache
	})
	require.NoError(t, err)
	require.Equal(t, entry.totalCost, refreshed)

	// 返回值应是独立副本；调用方修改不应反向污染缓存。
	refreshed[1] = -1

	repo.groupUsageTotalCostCacheMu.RLock()
	cached := repo.groupUsageTotalCostCache
	repo.groupUsageTotalCostCacheMu.RUnlock()

	require.Equal(t, entry.expiresAt, cached.expiresAt)
	require.Equal(t, entry.fullRebuildAt, cached.fullRebuildAt)
	require.Equal(t, entry.incrementalFrom, cached.incrementalFrom)
	require.Equal(t, entry.totalCost, cached.totalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositorySetGroupUsageTotalCostCacheFromIncrementKeepsAnchorMonotonic(t *testing.T) {
	repo := &usageLogRepository{}

	baseNow := time.Now().UTC()
	expectedFrom := baseNow.Add(-5 * time.Minute)
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       baseNow.Add(-time.Minute),
		fullRebuildAt:   baseNow.Add(-20 * time.Minute),
		incrementalFrom: expectedFrom,
		totalCost: map[int64]float64{
			1: 10.0,
		},
	}

	// 锚点单调性：即使传入更早的 incrementalUpperBound，也不应让缓存 incrementalFrom 倒退。
	backwardUpperBound := expectedFrom.Add(-time.Minute)
	updatedTotalCost := map[int64]float64{
		1: 11.0,
	}
	updated := repo.setGroupUsageTotalCostCacheFromIncrement(updatedTotalCost, expectedFrom, backwardUpperBound)
	require.True(t, updated)

	totalCostByGroup, ok, _, _, _, incrementalFrom, hasIncrementalFrom := repo.getGroupUsageTotalCostCacheState(true)
	require.True(t, ok)
	require.True(t, hasIncrementalFrom)
	require.Equal(t, expectedFrom, incrementalFrom)
	require.Equal(t, updatedTotalCost, totalCostByGroup)

	// expectedFrom 校验：起点不匹配时必须拒绝更新，且不改变已缓存数据。
	rejected := repo.setGroupUsageTotalCostCacheFromIncrement(map[int64]float64{
		1: 999.0,
	}, expectedFrom.Add(-time.Second), expectedFrom.Add(2*time.Minute))
	require.False(t, rejected)
	totalCostAfterRejected, ok, _, _, _, incrementalFromAfterRejected, hasIncrementalFromAfterRejected := repo.getGroupUsageTotalCostCacheState(true)
	require.True(t, ok)
	require.True(t, hasIncrementalFromAfterRejected)
	require.Equal(t, expectedFrom, incrementalFromAfterRejected)
	require.Equal(t, updatedTotalCost, totalCostAfterRejected)
}

func TestUsageLogRepositoryAdvanceGroupUsageTotalCostCacheWindowEmptyDeltaKeepsTotalCostAndAnchorMonotonic(t *testing.T) {
	repo := &usageLogRepository{}

	baseNow := time.Now().UTC()
	expectedFrom := baseNow.Add(-5 * time.Minute)
	originalTotalCost := map[int64]float64{
		1: 88.0,
		2: 12.0,
	}
	repo.groupUsageTotalCostCache = usageLogGroupUsageTotalCostCacheEntry{
		expiresAt:       baseNow.Add(-time.Minute),
		fullRebuildAt:   baseNow.Add(-20 * time.Minute),
		incrementalFrom: expectedFrom,
		totalCost:       cloneGroupUsageTotalCostMap(originalTotalCost),
	}

	// 空增量语义：仅推进窗口；即使 upperBound 回退，也不应让锚点倒退，且 totalCost 保持不变。
	backwardUpperBound := expectedFrom.Add(-2 * time.Minute)
	advanced := repo.advanceGroupUsageTotalCostCacheWindow(expectedFrom, backwardUpperBound)
	require.True(t, advanced)

	totalCostByGroup, ok, _, needsRefresh, _, incrementalFrom, hasIncrementalFrom := repo.getGroupUsageTotalCostCacheState(true)
	require.True(t, ok)
	require.True(t, hasIncrementalFrom)
	require.False(t, needsRefresh)
	require.Equal(t, expectedFrom, incrementalFrom)
	require.Equal(t, originalTotalCost, totalCostByGroup)

	// expectedFrom 校验：窗口起点不匹配时必须拒绝推进，且不改变 totalCost 与锚点。
	rejected := repo.advanceGroupUsageTotalCostCacheWindow(expectedFrom.Add(-time.Second), expectedFrom.Add(2*time.Minute))
	require.False(t, rejected)
	totalCostAfterRejected, ok, _, _, _, incrementalFromAfterRejected, hasIncrementalFromAfterRejected := repo.getGroupUsageTotalCostCacheState(true)
	require.True(t, ok)
	require.True(t, hasIncrementalFromAfterRejected)
	require.Equal(t, expectedFrom, incrementalFromAfterRejected)
	require.Equal(t, originalTotalCost, totalCostAfterRejected)
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryCacheKeyUsesUTCNormalizedDayStart(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	// +08:00 的 08:00 等价于 UTC 当天 00:00；仓库应统一按 UTC 参数查询并命中同一个缓存键。
	todayStartLocal := time.Date(2026, 5, 15, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	todayStartUTC := todayStartLocal.UTC()

	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStartUTC).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(11), 110.0, 11.0))

	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStartLocal)
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, int64(11), first[0].GroupID)

	// 同一 UTC 时刻再次请求应命中缓存，不再触发 SQL。
	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStartUTC)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryExpiredCacheKeyRewriteKeepsSingleNormalizedEntry(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStartLocal := time.Date(2026, 5, 16, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	todayStartUTC := todayStartLocal.UTC()
	cacheKey := todayStartUTC.Unix()
	repo.groupUsageSummaryCache = map[int64]usageLogGroupUsageSummaryCacheEntry{
		cacheKey: {
			expiresAt: time.Now().Add(-time.Minute),
			summaries: []usagestats.GroupUsageSummary{
				{GroupID: 1, TotalCost: 1.0, TodayCost: 0.1},
			},
		},
	}

	// 先确认过期键只会 miss，不会把过期值当成命中返回。
	cached, ok := repo.getGroupUsageSummaryCache(cacheKey)
	require.False(t, ok)
	require.Nil(t, cached)

	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStartUTC).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(11), 110.0, 11.0))

	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStartLocal)
	require.NoError(t, err)
	require.Equal(t, []usagestats.GroupUsageSummary{
		{GroupID: 11, TotalCost: 110.0, TodayCost: 11.0},
	}, first)

	repo.groupUsageSummaryCacheMu.RLock()
	entry, exists := repo.groupUsageSummaryCache[cacheKey]
	cacheLen := len(repo.groupUsageSummaryCache)
	repo.groupUsageSummaryCacheMu.RUnlock()

	require.True(t, exists)
	require.Equal(t, 1, cacheLen)
	require.True(t, entry.expiresAt.After(time.Now()))
	require.Equal(t, []usagestats.GroupUsageSummary{
		{GroupID: 11, TotalCost: 110.0, TodayCost: 11.0},
	}, entry.summaries)

	// 等价 UTC 日起点再次访问应直接命中同一键，不再产生新 SQL 或新增 map 项。
	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStartUTC)
	require.NoError(t, err)
	require.Equal(t, first, second)

	repo.groupUsageSummaryCacheMu.RLock()
	finalCacheLen := len(repo.groupUsageSummaryCache)
	repo.groupUsageSummaryCacheMu.RUnlock()
	require.Equal(t, 1, finalCacheLen)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryKeepsGroupsWithoutLogs(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(1), 10.5, 1.5).
			AddRow(int64(2), 0.0, 0.0))

	summaries, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	got := map[int64]usagestats.GroupUsageSummary{}
	for _, item := range summaries {
		got[item.GroupID] = item
	}
	require.Contains(t, got, int64(1))
	require.Equal(t, 10.5, got[1].TotalCost)
	require.Equal(t, 1.5, got[1].TodayCost)
	require.Contains(t, got, int64(2))
	require.Equal(t, 0.0, got[2].TotalCost)
	require.Equal(t, 0.0, got[2].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryCacheReturnsCopy(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(7), 70.0, 7.0))

	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, first, 1)

	// 篡改第一次返回结果，不应污染缓存中的原始值。
	first[0].TotalCost = 9999

	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, 70.0, second[0].TotalCost)

	// 再次篡改第二次返回值，第三次读取应仍保持原值，证明每次返回都是独立拷贝。
	second[0].TodayCost = -1
	third, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, third, 1)
	require.Equal(t, 70.0, third[0].TotalCost)
	require.Equal(t, 7.0, third[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryCallerMutationDoesNotAffectCachedHit(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(13), 130.0, 13.0))

	// 首次调用走查询路径，主线当前返回 query 结果本体。
	first, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, first, 1)
	first[0].TotalCost = 9999
	first[0].TodayCost = -999

	// 第二次同键调用应命中缓存，且不受调用方篡改影响。
	second, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, int64(13), second[0].GroupID)
	require.Equal(t, 130.0, second[0].TotalCost)
	require.Equal(t, 13.0, second[0].TodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetAllGroupUsageSummaryQueryFailureDoesNotPolluteCache(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	todayStart := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnError(fmt.Errorf("db unavailable"))
	mock.ExpectQuery("FROM groups g\\s+LEFT JOIN \\(\\s*SELECT\\s+ul.group_id[\\s\\S]*FROM usage_logs ul\\s+WHERE ul.group_id IS NOT NULL\\s+GROUP BY ul.group_id\\s*\\) agg ON agg.group_id = g.id\\s+WHERE g.deleted_at IS NULL").
		WithArgs(todayStart).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "total_cost", "today_cost"}).
			AddRow(int64(9), 90.0, 9.0))

	_, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.Error(t, err)

	got, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(9), got[0].GroupID)

	// 成功后应写入缓存，后续同键读取不应再查询数据库。
	cached, err := repo.GetAllGroupUsageSummary(context.Background(), todayStart)
	require.NoError(t, err)
	require.Equal(t, got, cached)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetStatsWithFiltersAlwaysReturnsAccountCost(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	// No AccountID filter set - TotalAccountCost should still be returned
	filters := usagestats.UsageLogFilters{}

	mock.ExpectQuery("FROM usage_logs").
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests", "total_input_tokens", "total_output_tokens",
			"total_cache_tokens", "total_cost", "total_actual_cost",
			"total_account_cost", "avg_duration_ms",
		}).AddRow(int64(50), int64(1000), int64(2000), int64(100), 15.0, 12.5, 11.0, 100.0))
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(inbound_endpoint\\)").
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(upstream_endpoint\\)").
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))
	mock.ExpectQuery("SELECT CONCAT\\(").
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}))

	stats, err := repo.GetStatsWithFilters(context.Background(), filters)
	require.NoError(t, err)
	require.NotNil(t, stats.TotalAccountCost, "TotalAccountCost must always be returned, even without AccountID filter")
	require.Equal(t, 11.0, *stats.TotalAccountCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetUserSpendingRanking(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	rows := sqlmock.NewRows([]string{"user_id", "email", "username", "actual_cost", "requests", "tokens", "total_actual_cost", "total_requests", "total_tokens"}).
		AddRow(int64(2), "beta@example.com", "Beta", 12.5, int64(9), int64(900), 40.0, int64(30), int64(2600)).
		AddRow(int64(1), "alpha@example.com", "", 12.5, int64(8), int64(800), 40.0, int64(30), int64(2600)).
		AddRow(int64(3), "gamma@example.com", "Gamma", 4.25, int64(5), int64(300), 40.0, int64(30), int64(2600))

	mock.ExpectQuery("WITH user_spend AS \\(").
		WithArgs(start, end, 12).
		WillReturnRows(rows)

	got, err := repo.GetUserSpendingRanking(context.Background(), start, end, 12)
	require.NoError(t, err)
	require.Equal(t, &usagestats.UserSpendingRankingResponse{
		Ranking: []usagestats.UserSpendingRankingItem{
			{UserID: 2, Email: "beta@example.com", Username: "Beta", ActualCost: 12.5, Requests: 9, Tokens: 900},
			{UserID: 1, Email: "alpha@example.com", Username: "", ActualCost: 12.5, Requests: 8, Tokens: 800},
			{UserID: 3, Email: "gamma@example.com", Username: "Gamma", ActualCost: 4.25, Requests: 5, Tokens: 300},
		},
		TotalActualCost: 40.0,
		TotalRequests:   30,
		TotalTokens:     2600,
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildRequestTypeFilterConditionLegacyFallback(t *testing.T) {
	tests := []struct {
		name      string
		request   int16
		wantWhere string
		wantArg   int16
	}{
		{
			name:      "sync_with_legacy_fallback",
			request:   int16(service.RequestTypeSync),
			wantWhere: "(request_type = $3 OR (request_type = 0 AND stream = FALSE AND openai_ws_mode = FALSE))",
			wantArg:   int16(service.RequestTypeSync),
		},
		{
			name:      "stream_with_legacy_fallback",
			request:   int16(service.RequestTypeStream),
			wantWhere: "(request_type = $3 OR (request_type = 0 AND stream = TRUE AND openai_ws_mode = FALSE))",
			wantArg:   int16(service.RequestTypeStream),
		},
		{
			name:      "ws_v2_with_legacy_fallback",
			request:   int16(service.RequestTypeWSV2),
			wantWhere: "(request_type = $3 OR (request_type = 0 AND openai_ws_mode = TRUE))",
			wantArg:   int16(service.RequestTypeWSV2),
		},
		{
			name:      "invalid_request_type_normalized_to_unknown",
			request:   int16(99),
			wantWhere: "request_type = $3",
			wantArg:   int16(service.RequestTypeUnknown),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, args := buildRequestTypeFilterCondition(3, tt.request)
			require.Equal(t, tt.wantWhere, where)
			require.Equal(t, []any{tt.wantArg}, args)
		})
	}
}

type usageLogScannerStub struct {
	values []any
}

func (s usageLogScannerStub) Scan(dest ...any) error {
	if len(dest) != len(s.values) {
		return fmt.Errorf("scan arg count mismatch: got %d want %d", len(dest), len(s.values))
	}
	for i := range dest {
		dv := reflect.ValueOf(dest[i])
		if dv.Kind() != reflect.Ptr {
			return fmt.Errorf("dest[%d] is not pointer", i)
		}
		dv.Elem().Set(reflect.ValueOf(s.values[i]))
	}
	return nil
}

func TestScanUsageLogRequestTypeAndLegacyFallback(t *testing.T) {
	t.Run("image_size_metadata_is_scanned", func(t *testing.T) {
		now := time.Now().UTC()
		log, err := scanUsageLog(usageLogScannerStub{values: []any{
			int64(4),
			int64(13),
			int64(23),
			int64(33),
			sql.NullString{Valid: true, String: "req-image-metadata"},
			"gpt-image-2",
			sql.NullString{Valid: true, String: "gpt-image-2"},
			sql.NullString{},
			sql.NullInt64{},
			sql.NullInt64{},
			0, 0, 0, 0, 0, 0,
			0, 0.0, // image_output_tokens, image_output_cost
			0.0, 0.0, 0.0, 0.0, 0.8, 0.8,
			1.0,
			sql.NullFloat64{},
			int16(service.BillingTypeBalance),
			int16(service.RequestTypeSync),
			false,
			false,
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullString{},
			sql.NullString{},
			2,
			sql.NullString{Valid: true, String: "4K"},
			sql.NullString{Valid: true, String: "1024x1024"},
			sql.NullString{Valid: true, String: "3840x2160"},
			sql.NullString{Valid: true, String: "output"},
			sql.NullString{Valid: true, String: `{"4K":2}`},
			sql.NullString{},
			sql.NullString{},
			sql.NullString{},
			sql.NullString{},
			false,
			sql.NullInt64{},
			sql.NullString{},
			sql.NullString{},
			sql.NullString{},
			sql.NullFloat64{},
			now,
		}})
		require.NoError(t, err)
		require.Equal(t, 2, log.ImageCount)
		require.NotNil(t, log.ImageSize)
		require.Equal(t, "4K", *log.ImageSize)
		require.NotNil(t, log.ImageInputSize)
		require.Equal(t, "1024x1024", *log.ImageInputSize)
		require.NotNil(t, log.ImageOutputSize)
		require.Equal(t, "3840x2160", *log.ImageOutputSize)
		require.NotNil(t, log.ImageSizeSource)
		require.Equal(t, "output", *log.ImageSizeSource)
		require.Equal(t, map[string]int{"4K": 2}, log.ImageSizeBreakdown)
	})

	t.Run("request_type_ws_v2_overrides_legacy", func(t *testing.T) {
		now := time.Now().UTC()
		log, err := scanUsageLog(usageLogScannerStub{values: []any{
			int64(1),  // id
			int64(10), // user_id
			int64(20), // api_key_id
			int64(30), // account_id
			sql.NullString{Valid: true, String: "req-1"},
			"gpt-5", // model
			sql.NullString{Valid: true, String: "gpt-5"}, // requested_model
			sql.NullString{},  // upstream_model
			sql.NullInt64{},   // group_id
			sql.NullInt64{},   // subscription_id
			1,                 // input_tokens
			2,                 // output_tokens
			3,                 // cache_creation_tokens
			4,                 // cache_read_tokens
			5,                 // cache_creation_5m_tokens
			6,                 // cache_creation_1h_tokens
			0,                 // image_output_tokens
			0.0,               // image_output_cost
			0.1,               // input_cost
			0.2,               // output_cost
			0.3,               // cache_creation_cost
			0.4,               // cache_read_cost
			1.0,               // total_cost
			0.9,               // actual_cost
			1.0,               // rate_multiplier
			sql.NullFloat64{}, // account_rate_multiplier
			int16(service.BillingTypeBalance),
			int16(service.RequestTypeWSV2),
			false, // legacy stream
			false, // legacy openai ws
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullString{},
			sql.NullString{},
			0,
			sql.NullString{},
			sql.NullString{}, // image_input_size
			sql.NullString{}, // image_output_size
			sql.NullString{}, // image_size_source
			sql.NullString{}, // image_size_breakdown
			sql.NullString{Valid: true, String: "priority"},
			sql.NullString{},
			sql.NullString{},
			sql.NullString{},
			false,
			sql.NullInt64{},   // channel_id
			sql.NullString{},  // model_mapping_chain
			sql.NullString{},  // billing_tier
			sql.NullString{},  // billing_mode
			sql.NullFloat64{}, // account_stats_cost
			now,
		}})
		require.NoError(t, err)
		require.NotNil(t, log.ServiceTier)
		require.Equal(t, "priority", *log.ServiceTier)
		require.Equal(t, service.RequestTypeWSV2, log.RequestType)
		require.True(t, log.Stream)
		require.True(t, log.OpenAIWSMode)
	})

	t.Run("request_type_unknown_falls_back_to_legacy", func(t *testing.T) {
		now := time.Now().UTC()
		log, err := scanUsageLog(usageLogScannerStub{values: []any{
			int64(2),
			int64(11),
			int64(21),
			int64(31),
			sql.NullString{Valid: true, String: "req-2"},
			"gpt-5",
			sql.NullString{Valid: true, String: "gpt-5"},
			sql.NullString{},
			sql.NullInt64{},
			sql.NullInt64{},
			1, 2, 3, 4, 5, 6,
			0, 0.0, // image_output_tokens, image_output_cost
			0.1, 0.2, 0.3, 0.4, 1.0, 0.9,
			1.0,
			sql.NullFloat64{},
			int16(service.BillingTypeBalance),
			int16(service.RequestTypeUnknown),
			true,
			false,
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullString{},
			sql.NullString{},
			0,
			sql.NullString{},
			sql.NullString{}, // image_input_size
			sql.NullString{}, // image_output_size
			sql.NullString{}, // image_size_source
			sql.NullString{}, // image_size_breakdown
			sql.NullString{Valid: true, String: "flex"},
			sql.NullString{},
			sql.NullString{},
			sql.NullString{},
			false,
			sql.NullInt64{},   // channel_id
			sql.NullString{},  // model_mapping_chain
			sql.NullString{},  // billing_tier
			sql.NullString{},  // billing_mode
			sql.NullFloat64{}, // account_stats_cost
			now,
		}})
		require.NoError(t, err)
		require.NotNil(t, log.ServiceTier)
		require.Equal(t, "flex", *log.ServiceTier)
		require.Equal(t, service.RequestTypeStream, log.RequestType)
		require.True(t, log.Stream)
		require.False(t, log.OpenAIWSMode)
	})

	t.Run("service_tier_is_scanned", func(t *testing.T) {
		now := time.Now().UTC()
		log, err := scanUsageLog(usageLogScannerStub{values: []any{
			int64(3),
			int64(12),
			int64(22),
			int64(32),
			sql.NullString{Valid: true, String: "req-3"},
			"gpt-5.4",
			sql.NullString{Valid: true, String: "gpt-5.4"},
			sql.NullString{},
			sql.NullInt64{},
			sql.NullInt64{},
			1, 2, 3, 4, 5, 6,
			0, 0.0, // image_output_tokens, image_output_cost
			0.1, 0.2, 0.3, 0.4, 1.0, 0.9,
			1.0,
			sql.NullFloat64{},
			int16(service.BillingTypeBalance),
			int16(service.RequestTypeSync),
			false,
			false,
			sql.NullInt64{},
			sql.NullInt64{},
			sql.NullInt64{Valid: true, Int64: 11},
			sql.NullInt64{Valid: true, Int64: 22},
			sql.NullInt64{Valid: true, Int64: 33},
			sql.NullInt64{Valid: true, Int64: 44},
			sql.NullString{},
			sql.NullString{},
			0,
			sql.NullString{},
			sql.NullString{}, // image_input_size
			sql.NullString{}, // image_output_size
			sql.NullString{}, // image_size_source
			sql.NullString{}, // image_size_breakdown
			sql.NullString{Valid: true, String: "priority"},
			sql.NullString{},
			sql.NullString{},
			sql.NullString{},
			false,
			sql.NullInt64{},   // channel_id
			sql.NullString{},  // model_mapping_chain
			sql.NullString{},  // billing_tier
			sql.NullString{},  // billing_mode
			sql.NullFloat64{}, // account_stats_cost
			now,
		}})
		require.NoError(t, err)
		require.NotNil(t, log.ServiceTier)
		require.Equal(t, "priority", *log.ServiceTier)
		require.NotNil(t, log.AuthLatencyMs)
		require.NotNil(t, log.RoutingLatencyMs)
		require.NotNil(t, log.UpstreamLatencyMs)
		require.NotNil(t, log.ResponseLatencyMs)
		require.Equal(t, 11, *log.AuthLatencyMs)
		require.Equal(t, 22, *log.RoutingLatencyMs)
		require.Equal(t, 33, *log.UpstreamLatencyMs)
		require.Equal(t, 44, *log.ResponseLatencyMs)
	})

}

