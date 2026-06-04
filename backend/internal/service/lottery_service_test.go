package service

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

type lotteryProviderStub struct {
	name         string
	currentIssue *Issue
	latestResult *Result
	err          error
}

func (s *lotteryProviderStub) Name() string { return s.name }

func (s *lotteryProviderStub) GetCurrentIssue(ctx context.Context) (*Issue, error) {
	_ = ctx
	if s.err != nil {
		return nil, s.err
	}
	return s.currentIssue, nil
}

func (s *lotteryProviderStub) GetLatestResult(ctx context.Context) (*Result, error) {
	_ = ctx
	if s.err != nil {
		return nil, s.err
	}
	return s.latestResult, nil
}

type lotteryJackpotStoreStub struct {
	balance decimal.Decimal
	err     error
}

func (s *lotteryJackpotStoreStub) Deposit(ctx context.Context, lotteryType string, amount decimal.Decimal) error {
	_ = ctx
	_ = lotteryType
	_ = amount
	return s.err
}

func (s *lotteryJackpotStoreStub) Withdraw(ctx context.Context, lotteryType string, amount decimal.Decimal) error {
	_ = ctx
	_ = lotteryType
	_ = amount
	return s.err
}

func (s *lotteryJackpotStoreStub) GetBalance(ctx context.Context, lotteryType string) (decimal.Decimal, error) {
	_ = ctx
	_ = lotteryType
	if s.err != nil {
		return decimal.Zero, s.err
	}
	return s.balance, nil
}

func (s *lotteryJackpotStoreStub) depositInTx(ctx context.Context, client lotterySQLClient, lotteryType string, amount decimal.Decimal) error {
	_ = ctx
	_ = client
	_ = lotteryType
	if s.err != nil {
		return s.err
	}
	s.balance = s.balance.Add(amount)
	return nil
}

func newLotterySQLMockClient(t *testing.T) (*dbent.Client, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(drv))
	cleanup := func() {
		_ = client.Close()
		_ = db.Close()
	}
	return client, mock, cleanup
}

func lotteryTestResult() *Result {
	openedAt := time.Date(2026, 6, 2, lotteryOpenHour, lotteryOpenMinute, 0, 0, time.UTC)
	return &Result{
		LotteryType:   LotteryTypeSSQ,
		IssueNo:       "2026062",
		RedBalls:      []string{"29", "02", "14", "07", "28", "04"},
		BlueBall:      "9",
		OpenedAt:      openedAt,
		Source:        "fucai",
		SourceRef:     "https://www.cwl.gov.cn/c/2026/06/02/656270.shtml",
		SourcePayload: []byte(`{"state":0}`),
	}
}

func normalizedLotteryTestResult(t *testing.T, result *Result) *Result {
	t.Helper()

	normalized, err := normalizeLotteryResult(result, LotteryTypeSSQ, "fucai")
	require.NoError(t, err)
	return normalized
}

func newLotterySyncTestService(client *dbent.Client, result *Result) *LotteryService {
	return NewLotteryService(client, nil, nil, nil, &lotteryJackpotStoreStub{}, map[string]LotteryProvider{
		LotteryTypeSSQ: &lotteryProviderStub{
			name:         "fucai",
			latestResult: result,
		},
	})
}

func expectLotterySyncTxBegin(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock($1)")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_xact_lock"}).AddRow(nil))
}

func expectNoExistingLotteryResult(mock sqlmock.Sqlmock, lotteryType, issueNo string) {
	mock.ExpectQuery(regexp.QuoteMeta("FROM lottery_result")).
		WithArgs(lotteryType, issueNo).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "lottery_type", "issue_no", "red_balls", "blue_ball", "opened_at", "source", "source_ref", "created_at",
		}))
}

func expectExistingLotteryResult(mock sqlmock.Sqlmock, view LotteryResultView) {
	mock.ExpectQuery(regexp.QuoteMeta("FROM lottery_result")).
		WithArgs(view.LotteryType, view.IssueNo).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "lottery_type", "issue_no", "red_balls", "blue_ball", "opened_at", "source", "source_ref", "created_at",
		}).AddRow(
			view.ID,
			view.LotteryType,
			view.IssueNo,
			strings.Join(view.RedBalls, ","),
			view.BlueBall,
			view.OpenedAt,
			view.Source,
			view.SourceRef,
			view.CreatedAt,
		))
}

func expectInsertLotteryResult(mock sqlmock.Sqlmock, result *Result, id int64, createdAt time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO lottery_result")).
		WithArgs(
			result.LotteryType,
			result.IssueNo,
			strings.Join(result.RedBalls, ","),
			result.BlueBall,
			result.Source,
			result.SourceRef,
			driver.Value(string(result.SourcePayload)),
			result.OpenedAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "lottery_type", "issue_no", "red_balls", "blue_ball", "opened_at", "source", "source_ref", "created_at",
		}).AddRow(
			id,
			result.LotteryType,
			result.IssueNo,
			strings.Join(result.RedBalls, ","),
			result.BlueBall,
			result.OpenedAt,
			result.Source,
			result.SourceRef,
			createdAt,
		))
}

func expectMarkLotteryIssueOpened(mock sqlmock.Sqlmock, result *Result) {
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO lottery_issue")).
		WithArgs(result.LotteryType, result.IssueNo, result.OpenedAt, lotteryIssueStatusOpened).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestLotteryServiceGetCurrentIssueUsesProvider(t *testing.T) {
	require.NoError(t, timezone.Init("UTC"))
	now := time.Date(2099, 6, 9, lotteryOpenHour, lotteryOpenMinute, 0, 0, time.UTC)
	svc := NewLotteryService(nil, nil, nil, nil, nil, map[string]LotteryProvider{
		LotteryTypeSSQ: &lotteryProviderStub{
			name: "fucai",
			currentIssue: &Issue{
				LotteryType: LotteryTypeSSQ,
				IssueNo:     "2099001",
				OpenTime:    now,
				Status:      lotteryIssueStatusPending,
				Source:      "fucai",
			},
		},
	})

	issue, err := svc.GetCurrentIssue(context.Background(), "SSQ")
	require.NoError(t, err)
	require.Equal(t, "2099001", issue.IssueNo)
	require.Equal(t, LotteryTypeSSQ, issue.LotteryType)
	require.Equal(t, "fucai", issue.Source)
	require.Equal(t, now.Add(-lotteryCutoffLead), issue.CutoffTime)
	require.False(t, issue.IsClosed)
}

func TestLotteryServiceGetCurrentIssueRejectsUnknownProvider(t *testing.T) {
	svc := NewLotteryService(nil, nil, nil, nil, nil, nil)

	_, err := svc.GetCurrentIssue(context.Background(), LotteryTypeSSQ)
	require.Error(t, err)
	require.Equal(t, "LOTTERY_PROVIDER_NOT_FOUND", infraerrors.Reason(err))
}

func TestLotteryServiceGetJackpotUsesJackpotService(t *testing.T) {
	svc := NewLotteryService(nil, nil, nil, nil, &lotteryJackpotStoreStub{
		balance: decimal.RequireFromString("10000000"),
	}, nil)

	view, err := svc.GetJackpot(context.Background(), "ssq")
	require.NoError(t, err)
	require.Equal(t, LotteryTypeSSQ, view.LotteryType)
	require.True(t, decimal.RequireFromString("10000000").Equal(view.Balance))
}

func TestLotteryServiceGetJackpotReturnsNotFound(t *testing.T) {
	svc := NewLotteryService(nil, nil, nil, nil, &lotteryJackpotStoreStub{
		err: ErrLotteryJackpotNotFound,
	}, nil)

	_, err := svc.GetJackpot(context.Background(), "ssq")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrLotteryJackpotNotFound))
}

func TestLotteryServiceCreateBetRejectsInvalidPayload(t *testing.T) {
	svc := NewLotteryService(nil, nil, nil, nil, &lotteryJackpotStoreStub{}, map[string]LotteryProvider{
		LotteryTypeSSQ: &lotteryProviderStub{name: "fucai"},
	})

	_, err := svc.CreateBet(context.Background(), LotteryBetInput{
		UserID:   1,
		RedBalls: []string{"01", "02"},
		BlueBall: "03",
	})
	require.Error(t, err)
	require.Equal(t, "LOTTERY_NUMBERS_INVALID", infraerrors.Reason(err))
}

func TestLotteryServiceGetMyOrdersRejectsInvalidQuery(t *testing.T) {
	svc := NewLotteryService(nil, nil, nil, nil, &lotteryJackpotStoreStub{}, nil)

	_, err := svc.GetMyOrders(context.Background(), LotteryOrderQuery{})
	require.Error(t, err)
	require.Equal(t, "LOTTERY_ORDER_QUERY_INVALID", infraerrors.Reason(err))
}

func TestLotteryServiceSyncLatestResultInsertsResultAndMarksIssueOpened(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	providerResult := lotteryTestResult()
	normalized := normalizedLotteryTestResult(t, providerResult)
	createdAt := time.Date(2026, 6, 2, 22, 0, 0, 0, time.UTC)
	svc := newLotterySyncTestService(client, providerResult)

	expectLotterySyncTxBegin(mock)
	expectNoExistingLotteryResult(mock, LotteryTypeSSQ, "2026062")
	expectInsertLotteryResult(mock, normalized, 101, createdAt)
	expectMarkLotteryIssueOpened(mock, normalized)
	mock.ExpectCommit()

	got, err := svc.SyncLatestResult(context.Background(), "SSQ")
	require.NoError(t, err)
	require.False(t, got.Replayed)
	require.Equal(t, LotteryTypeSSQ, got.LotteryType)
	require.Equal(t, "2026062", got.IssueNo)
	require.Equal(t, int64(101), got.Result.ID)
	require.Equal(t, []string{"02", "04", "07", "14", "28", "29"}, got.Result.RedBalls)
	require.Equal(t, "09", got.Result.BlueBall)
	require.Equal(t, "fucai", got.Result.Source)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceSyncLatestResultReplaysExistingSameResult(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	providerResult := lotteryTestResult()
	normalized := normalizedLotteryTestResult(t, providerResult)
	createdAt := time.Date(2026, 6, 2, 22, 0, 0, 0, time.UTC)
	existing := LotteryResultView{
		ID:          102,
		LotteryType: LotteryTypeSSQ,
		IssueNo:     "2026062",
		RedBalls:    []string{"02", "04", "07", "14", "28", "29"},
		BlueBall:    "09",
		OpenedAt:    normalized.OpenedAt,
		Source:      "fucai",
		SourceRef:   normalized.SourceRef,
		CreatedAt:   createdAt,
	}
	svc := newLotterySyncTestService(client, providerResult)

	expectLotterySyncTxBegin(mock)
	expectExistingLotteryResult(mock, existing)
	expectMarkLotteryIssueOpened(mock, normalized)
	mock.ExpectCommit()

	got, err := svc.SyncLatestResult(context.Background(), "ssq")
	require.NoError(t, err)
	require.True(t, got.Replayed)
	require.Equal(t, int64(102), got.Result.ID)
	require.Equal(t, []string{"02", "04", "07", "14", "28", "29"}, got.Result.RedBalls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceSyncLatestResultRejectsConflictingResult(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	providerResult := lotteryTestResult()
	existing := LotteryResultView{
		ID:          103,
		LotteryType: LotteryTypeSSQ,
		IssueNo:     "2026062",
		RedBalls:    []string{"01", "04", "07", "14", "28", "29"},
		BlueBall:    "09",
		OpenedAt:    providerResult.OpenedAt,
		Source:      "fucai",
		SourceRef:   providerResult.SourceRef,
		CreatedAt:   providerResult.OpenedAt,
	}
	svc := newLotterySyncTestService(client, providerResult)

	expectLotterySyncTxBegin(mock)
	expectExistingLotteryResult(mock, existing)
	mock.ExpectRollback()

	_, err := svc.SyncLatestResult(context.Background(), "ssq")
	require.Error(t, err)
	require.Equal(t, "LOTTERY_RESULT_CONFLICT", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceSyncLatestResultRejectsInvalidProviderResult(t *testing.T) {
	client, _, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	invalid := lotteryTestResult()
	invalid.RedBalls = []string{"01", "02", "03", "04", "05", "05"}
	svc := newLotterySyncTestService(client, invalid)

	_, err := svc.SyncLatestResult(context.Background(), "ssq")
	require.Error(t, err)
	require.Equal(t, "LOTTERY_DATA_INVALID", infraerrors.Reason(err))
}

func TestLotteryServiceSyncLatestResultRequiresStorage(t *testing.T) {
	svc := newLotterySyncTestService(nil, lotteryTestResult())

	_, err := svc.SyncLatestResult(context.Background(), "ssq")
	require.Error(t, err)
	require.Equal(t, "LOTTERY_STORAGE_UNAVAILABLE", infraerrors.Reason(err))
}
