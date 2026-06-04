package service

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
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
	balance   decimal.Decimal
	withdrawn decimal.Decimal
	deposited decimal.Decimal
	err       error
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
	s.deposited = s.deposited.Add(amount)
	s.balance = s.balance.Add(amount)
	return nil
}

func (s *lotteryJackpotStoreStub) withdrawInTx(ctx context.Context, client lotterySQLClient, lotteryType string, amount decimal.Decimal) error {
	_ = ctx
	_ = client
	_ = lotteryType
	if s.err != nil {
		return s.err
	}
	if s.balance.LessThan(amount) {
		return ErrLotteryJackpotInsufficient
	}
	s.withdrawn = s.withdrawn.Add(amount)
	s.balance = s.balance.Sub(amount)
	return nil
}

type lotteryBankApplierStub struct {
	requests []TransferFundsRequest
	err      error
}

func (s *lotteryBankApplierStub) ApplyTransferInTx(ctx context.Context, client *dbent.Client, req TransferFundsRequest) (*TransferFundsResult, error) {
	_ = ctx
	_ = client
	if s.err != nil {
		return nil, s.err
	}
	s.requests = append(s.requests, req)
	return &TransferFundsResult{
		UserID:   req.UserID,
		Type:     req.Type,
		Module:   req.BusinessModule,
		Amount:   req.Amount,
		Replayed: false,
	}, nil
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

func expectLotteryResultList(mock sqlmock.Sqlmock, args []driver.Value, views []LotteryResultView) {
	rows := sqlmock.NewRows([]string{
		"id", "lottery_type", "issue_no", "red_balls", "blue_ball", "opened_at", "source", "source_ref", "created_at",
	})
	for _, view := range views {
		rows.AddRow(
			view.ID,
			view.LotteryType,
			view.IssueNo,
			strings.Join(view.RedBalls, ","),
			view.BlueBall,
			view.OpenedAt,
			view.Source,
			view.SourceRef,
			view.CreatedAt,
		)
	}
	mock.ExpectQuery(regexp.QuoteMeta("FROM lottery_result")).
		WithArgs(args...).
		WillReturnRows(rows)
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

func lotteryTestResultView() LotteryResultView {
	openedAt := time.Date(2026, 6, 2, lotteryOpenHour, lotteryOpenMinute, 0, 0, time.UTC)
	return LotteryResultView{
		ID:          201,
		LotteryType: LotteryTypeSSQ,
		IssueNo:     "2026062",
		RedBalls:    []string{"02", "04", "07", "14", "28", "29"},
		BlueBall:    "09",
		OpenedAt:    openedAt,
		Source:      "fucai",
		SourceRef:   "https://www.cwl.gov.cn/c/2026/06/02/656270.shtml",
		CreatedAt:   openedAt.Add(30 * time.Minute),
	}
}

func expectLotteryIssueStatus(mock sqlmock.Sqlmock, lotteryType, issueNo, status string) {
	mock.ExpectQuery(regexp.QuoteMeta("FROM lottery_issue")).
		WithArgs(lotteryType, issueNo).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(status))
}

func expectPendingSettlementOrders(mock sqlmock.Sqlmock, lotteryType, issueNo string, orders []lotterySettlementOrder) {
	rows := sqlmock.NewRows([]string{
		"id", "lottery_type", "issue_no", "user_id", "red_balls", "blue_ball", "cost",
	})
	for _, order := range orders {
		rows.AddRow(
			order.ID,
			order.LotteryType,
			order.IssueNo,
			order.UserID,
			strings.Join(order.RedBalls, ","),
			order.BlueBall,
			order.Cost,
		)
	}
	mock.ExpectQuery(regexp.QuoteMeta("FROM lottery_order")).
		WithArgs(lotteryType, issueNo, lotteryOrderStatusPending).
		WillReturnRows(rows)
}

func expectUpdateLotterySettlementOrder(mock sqlmock.Sqlmock, order lotterySettlementOrder, prize lotteryPrize) {
	status := lotteryOrderStatusLose
	if prize.reward.GreaterThan(decimal.Zero) {
		status = lotteryOrderStatusWin
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE lottery_order")).
		WithArgs(order.ID, prize.reward, prize.level, prize.redHits, prize.blueHit, status, lotteryOrderStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectLotteryRewardLog(mock sqlmock.Sqlmock, order lotterySettlementOrder, prize lotteryPrize) {
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO lottery_reward_log")).
		WithArgs(
			order.LotteryType,
			order.UserID,
			order.IssueNo,
			order.ID,
			prize.reward,
			fmt.Sprintf("lottery %s prize for issue %s", prize.level, order.IssueNo),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectLotteryIssueSettled(mock sqlmock.Sqlmock, lotteryType, issueNo string) {
	mock.ExpectExec(regexp.QuoteMeta("UPDATE lottery_issue")).
		WithArgs(lotteryType, issueNo, lotteryIssueStatusSettled).
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

func TestLotteryServiceGetCurrentIssueFallsBackToLatestResultWhenProviderUnavailable(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	openedAt := time.Date(2099, 6, 2, lotteryOpenHour, lotteryOpenMinute, 0, 0, timezone.Location())
	latest := LotteryResultView{
		ID:          301,
		LotteryType: LotteryTypeSSQ,
		IssueNo:     "2099001",
		RedBalls:    []string{"01", "02", "03", "04", "05", "06"},
		BlueBall:    "07",
		OpenedAt:    openedAt,
		Source:      "fucai",
		CreatedAt:   openedAt.Add(30 * time.Minute),
	}
	svc := NewLotteryService(client, nil, nil, nil, nil, map[string]LotteryProvider{
		LotteryTypeSSQ: &lotteryProviderStub{
			name: "fucai",
			err:  ErrLotteryProviderUnavailable.WithMetadata(map[string]string{"status_code": "403"}),
		},
	})

	expectLotteryResultList(mock, []driver.Value{LotteryTypeSSQ, int64(1)}, []LotteryResultView{latest})
	issue, err := svc.GetCurrentIssue(context.Background(), "ssq")
	require.NoError(t, err)
	require.Equal(t, LotteryTypeSSQ, issue.LotteryType)
	require.Equal(t, "2099002", issue.IssueNo)
	require.Equal(t, "fucai-cache", issue.Source)
	require.True(t, issue.OpenTime.After(openedAt))
	require.False(t, issue.IsClosed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurrentSSQIssueFromLatestResultAdvancesAcrossMissedDraws(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))

	latest := lotteryTestResultView()
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, timezone.Location())

	issue, err := currentSSQIssueFromLatestResult(latest, "fucai", now)
	require.NoError(t, err)
	require.Equal(t, "2026065", issue.IssueNo)
	require.Equal(t, "fucai-cache", issue.Source)
	require.Equal(t, time.Date(2026, 6, 9, lotteryOpenHour, lotteryOpenMinute, 0, 0, timezone.Location()), issue.OpenTime)
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

func TestLotteryServiceGetResultsDefaultsAndCapsLimit(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	view := lotteryTestResultView()
	svc := NewLotteryService(client, nil, nil, nil, &lotteryJackpotStoreStub{}, nil)

	expectLotteryResultList(mock, []driver.Value{LotteryTypeSSQ, int64(100)}, []LotteryResultView{view})
	got, err := svc.GetResults(context.Background(), LotteryResultQuery{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "2026062", got[0].IssueNo)

	expectLotteryResultList(mock, []driver.Value{LotteryTypeSSQ, int64(100)}, []LotteryResultView{view})
	got, err = svc.GetResults(context.Background(), LotteryResultQuery{LotteryType: "SSQ", Limit: 500})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceGetResultReturnsSingleIssue(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	view := lotteryTestResultView()
	svc := NewLotteryService(client, nil, nil, nil, &lotteryJackpotStoreStub{}, nil)

	expectLotteryResultList(mock, []driver.Value{LotteryTypeSSQ, "2026062", int64(1)}, []LotteryResultView{view})
	got, err := svc.GetResult(context.Background(), "ssq", "2026062")
	require.NoError(t, err)
	require.Equal(t, "2026062", got.IssueNo)
	require.Equal(t, []string{"02", "04", "07", "14", "28", "29"}, got.RedBalls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceGetResultReturnsNotFound(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	svc := NewLotteryService(client, nil, nil, nil, &lotteryJackpotStoreStub{}, nil)

	expectLotteryResultList(mock, []driver.Value{LotteryTypeSSQ, "2026062", int64(1)}, nil)
	_, err := svc.GetResult(context.Background(), "ssq", "2026062")
	require.Error(t, err)
	require.Equal(t, "LOTTERY_RESULT_NOT_FOUND", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceSettleIssueSettlesWinAndLoseOrders(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	drawResult := lotteryTestResultView()
	winningOrder := lotterySettlementOrder{
		ID:          501,
		LotteryType: LotteryTypeSSQ,
		IssueNo:     "2026062",
		UserID:      42,
		RedBalls:    []string{"01", "03", "05", "08", "10", "12"},
		BlueBall:    "09",
		Cost:        lotterySingleBetCost,
	}
	losingOrder := lotterySettlementOrder{
		ID:          502,
		LotteryType: LotteryTypeSSQ,
		IssueNo:     "2026062",
		UserID:      42,
		RedBalls:    []string{"01", "03", "05", "08", "10", "12"},
		BlueBall:    "16",
		Cost:        lotterySingleBetCost,
	}
	winningPrize := calculateSSQPrize(winningOrder, drawResult)
	losingPrize := calculateSSQPrize(losingOrder, drawResult)
	require.Equal(t, lotteryPrizeLevelSixth, winningPrize.level)
	require.True(t, lotteryPrizeSixth.Equal(winningPrize.reward))
	require.True(t, losingPrize.reward.IsZero())

	jackpot := &lotteryJackpotStoreStub{balance: decimal.NewFromInt(1000)}
	bank := &lotteryBankApplierStub{}
	svc := NewLotteryService(client, nil, nil, nil, jackpot, nil)
	svc.bankApplier = bank

	expectLotterySyncTxBegin(mock)
	expectExistingLotteryResult(mock, drawResult)
	expectLotteryIssueStatus(mock, LotteryTypeSSQ, "2026062", lotteryIssueStatusOpened)
	expectPendingSettlementOrders(mock, LotteryTypeSSQ, "2026062", []lotterySettlementOrder{winningOrder, losingOrder})
	expectUpdateLotterySettlementOrder(mock, winningOrder, winningPrize)
	expectLotteryRewardLog(mock, winningOrder, winningPrize)
	expectUpdateLotterySettlementOrder(mock, losingOrder, losingPrize)
	expectLotteryIssueSettled(mock, LotteryTypeSSQ, "2026062")
	mock.ExpectCommit()

	got, err := svc.SettleIssue(context.Background(), "ssq", "2026062")
	require.NoError(t, err)
	require.False(t, got.Replayed)
	require.Equal(t, 2, got.TotalOrders)
	require.Equal(t, 1, got.WinOrders)
	require.Equal(t, 1, got.LoseOrders)
	require.True(t, lotteryPrizeSixth.Equal(got.RewardTotal))
	require.True(t, lotteryPrizeSixth.Equal(jackpot.withdrawn))
	require.True(t, decimal.NewFromInt(995).Equal(jackpot.balance))
	require.Len(t, bank.requests, 1)
	require.Equal(t, BankTxTypeLotteryWin, bank.requests[0].Type)
	require.Equal(t, BankBusinessModuleGame, bank.requests[0].BusinessModule)
	require.Equal(t, "lottery:win:2026062:501", bank.requests[0].IdempotencyKey)
	require.Equal(t, "lottery_order", bank.requests[0].ReferenceType)
	require.Equal(t, "501", bank.requests[0].ReferenceID)
	require.True(t, lotteryPrizeSixth.Equal(bank.requests[0].Amount))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceSettleIssueIsIdempotentWhenAlreadySettled(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	drawResult := lotteryTestResultView()
	jackpot := &lotteryJackpotStoreStub{balance: decimal.NewFromInt(1000)}
	bank := &lotteryBankApplierStub{}
	svc := NewLotteryService(client, nil, nil, nil, jackpot, nil)
	svc.bankApplier = bank

	expectLotterySyncTxBegin(mock)
	expectExistingLotteryResult(mock, drawResult)
	expectLotteryIssueStatus(mock, LotteryTypeSSQ, "2026062", lotteryIssueStatusSettled)
	mock.ExpectCommit()

	got, err := svc.SettleIssue(context.Background(), "ssq", "2026062")
	require.NoError(t, err)
	require.True(t, got.Replayed)
	require.Equal(t, 0, got.TotalOrders)
	require.True(t, jackpot.withdrawn.IsZero())
	require.Empty(t, bank.requests)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceSettleIssueRequiresResult(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	svc := NewLotteryService(client, nil, nil, nil, &lotteryJackpotStoreStub{balance: decimal.NewFromInt(1000)}, nil)

	expectLotterySyncTxBegin(mock)
	expectNoExistingLotteryResult(mock, LotteryTypeSSQ, "2026062")
	mock.ExpectRollback()

	_, err := svc.SettleIssue(context.Background(), "ssq", "2026062")
	require.Error(t, err)
	require.Equal(t, "LOTTERY_RESULT_NOT_FOUND", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryServiceSettleIssueRollsBackWhenJackpotInsufficient(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	drawResult := lotteryTestResultView()
	winningOrder := lotterySettlementOrder{
		ID:          503,
		LotteryType: LotteryTypeSSQ,
		IssueNo:     "2026062",
		UserID:      43,
		RedBalls:    []string{"01", "03", "05", "08", "10", "12"},
		BlueBall:    "09",
		Cost:        lotterySingleBetCost,
	}
	jackpot := &lotteryJackpotStoreStub{balance: decimal.NewFromInt(4)}
	bank := &lotteryBankApplierStub{}
	svc := NewLotteryService(client, nil, nil, nil, jackpot, nil)
	svc.bankApplier = bank

	expectLotterySyncTxBegin(mock)
	expectExistingLotteryResult(mock, drawResult)
	expectLotteryIssueStatus(mock, LotteryTypeSSQ, "2026062", lotteryIssueStatusOpened)
	expectPendingSettlementOrders(mock, LotteryTypeSSQ, "2026062", []lotterySettlementOrder{winningOrder})
	mock.ExpectRollback()

	_, err := svc.SettleIssue(context.Background(), "ssq", "2026062")
	require.ErrorIs(t, err, ErrLotteryJackpotInsufficient)
	require.True(t, jackpot.withdrawn.IsZero())
	require.Empty(t, bank.requests)
	require.NoError(t, mock.ExpectationsWereMet())
}
