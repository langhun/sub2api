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
		WindowMinutes:      10,
		MinRequests:        3,
		TTFTP95MsThreshold: 12000,
		MaxAccountsPerRun:  2,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = 2500")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`(?s)HAVING COUNT\(\*\) >= 3.*ORDER BY p95 DESC, max_ttft DESC, reqs DESC\s+LIMIT 2`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id",
			"platform",
			"model",
			"group_id",
			"reqs",
			"p95",
			"max_ttft",
		}).AddRow(int64(11), "openai", "gpt-5", nil, 4, 13000, 21000))
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
		WindowMinutes:      10,
		MinRequests:        5,
		TTFTP95MsThreshold: 15000,
		MaxAccountsPerRun:  4,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = 2500")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`(?s)a\.proxy_id IS NOT NULL.*HAVING COUNT\(\*\) >= 5.*ORDER BY p95 DESC, max_ttft DESC, reqs DESC\s+LIMIT 4`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"proxy_id",
			"platform",
			"reqs",
			"p95",
			"max_ttft",
		}).AddRow(int64(21), "openai", 8, 16000, 26000))
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
