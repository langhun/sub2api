package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListSlowTailCandidatesAppliesRuntimeGuards(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 5, 12, 10, 30, 0, 0, time.UTC)
	rule := OpsSlowTailIsolationSettings{
		WindowMinutes:          10,
		MinRequests:            3,
		TTFTP95MsThreshold:     12000,
		DurationP95MsThreshold: 90000,
		MaxAccountsPerRun:      2,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = 2500")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`(?s)first_token_ms IS NOT NULL OR u\.duration_ms IS NOT NULL.*HAVING COUNT\(\*\) >= 3.*duration_ms.*first_token_ms.*LIMIT 2`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id",
			"platform",
			"model",
			"group_id",
			"reqs",
			"ttft_p95",
			"max_ttft",
			"duration_p95",
			"max_duration",
			"response_latency_p95",
			"max_response_latency",
		}).AddRow(int64(11), "openai", "gpt-5", nil, 4, 13000, 21000, 98000, 120000, nil, nil))
	mock.ExpectCommit()

	collector := &OpsMetricsCollector{db: db}
	candidates, err := collector.listSlowTailCandidates(context.Background(), now, rule)
	if err != nil {
		t.Fatalf("list slow tail candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].AccountID != 11 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestListSlowProxyCandidatesAppliesRuntimeGuards(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 5, 12, 10, 30, 0, 0, time.UTC)
	rule := OpsSlowTailIsolationSettings{
		WindowMinutes:                 10,
		MinRequests:                   5,
		TTFTP95MsThreshold:            15000,
		ResponseLatencyP95MsThreshold: 80000,
		MaxAccountsPerRun:             4,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = 2500")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`(?s)a\.proxy_id IS NOT NULL.*first_token_ms IS NOT NULL OR u\.response_latency_ms IS NOT NULL.*HAVING COUNT\(\*\) >= 5.*response_latency_ms.*LIMIT 4`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"proxy_id",
			"platform",
			"reqs",
			"ttft_p95",
			"max_ttft",
			"duration_p95",
			"max_duration",
			"response_latency_p95",
			"max_response_latency",
		}).AddRow(int64(21), "openai", 8, 16000, 26000, nil, nil, 91000, 110000))
	mock.ExpectCommit()

	collector := &OpsMetricsCollector{db: db}
	candidates, err := collector.listSlowProxyCandidates(context.Background(), now, rule)
	if err != nil {
		t.Fatalf("list slow proxy candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].ProxyID != 21 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestOpsSlowTailRunLimitClampsUnsafeValues(t *testing.T) {
	tests := []struct {
		name string
		rule OpsSlowTailIsolationSettings
		want int
	}{
		{name: "non_positive", rule: OpsSlowTailIsolationSettings{MaxAccountsPerRun: 0}, want: 1},
		{name: "configured", rule: OpsSlowTailIsolationSettings{MaxAccountsPerRun: 12}, want: 12},
		{name: "hard_cap", rule: OpsSlowTailIsolationSettings{MaxAccountsPerRun: 1000}, want: opsMetricsCollectorSlowTailMaxCandidates},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := opsSlowTailRunLimit(tt.rule)
			if got != tt.want {
				t.Fatalf("opsSlowTailRunLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestListSlowTailCandidates_DurationTailDoesNotRequireFirstTokenMs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 5, 12, 10, 30, 0, 0, time.UTC)
	rule := OpsSlowTailIsolationSettings{
		WindowMinutes:          10,
		MinRequests:            3,
		TTFTP95MsThreshold:     12000,
		DurationP95MsThreshold: 90000,
		MaxAccountsPerRun:      2,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = 2500")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`(?s)first_token_ms IS NOT NULL OR u\.duration_ms IS NOT NULL.*duration_ms.*>= 90000`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id",
			"platform",
			"model",
			"group_id",
			"reqs",
			"ttft_p95",
			"max_ttft",
			"duration_p95",
			"max_duration",
			"response_latency_p95",
			"max_response_latency",
		}).AddRow(int64(12), "openai", "gpt-5.5", nil, 3, nil, nil, 95000, 121000, nil, nil))
	mock.ExpectCommit()

	collector := &OpsMetricsCollector{db: db}
	candidates, err := collector.listSlowTailCandidates(context.Background(), now, rule)
	if err != nil {
		t.Fatalf("list slow tail candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].AccountID != 12 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
	if candidates[0].TTFTP95 != nil {
		t.Fatalf("expected nil ttft p95, got %+v", candidates[0].TTFTP95)
	}
	if candidates[0].DurationP95 == nil || *candidates[0].DurationP95 != 95000 {
		t.Fatalf("unexpected duration p95: %+v", candidates[0].DurationP95)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestOpsMetricsCollectorQueryErrorCounts_ExcludesClientDisconnect499(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`(?s)SELECT\s+COALESCE\(COUNT\(\*\) FILTER \(WHERE .*client.*499`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"error_total",
			"business_limited",
			"error_sla",
			"upstream_excl",
			"upstream_429",
			"upstream_529",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), int64(0), int64(0)))

	collector := &OpsMetricsCollector{db: db}
	start := time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	errorTotal, businessLimited, errorSLA, upstreamExcl, upstream429, upstream529, err := collector.queryErrorCounts(context.Background(), start, end)
	if err != nil {
		t.Fatalf("query error counts: %v", err)
	}
	if errorTotal != 0 || businessLimited != 0 || errorSLA != 0 || upstreamExcl != 0 || upstream429 != 0 || upstream529 != 0 {
		t.Fatalf("unexpected counts: total=%d business=%d sla=%d upstream=%d 429=%d 529=%d", errorTotal, businessLimited, errorSLA, upstreamExcl, upstream429, upstream529)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
