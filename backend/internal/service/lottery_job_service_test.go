package service

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type lotteryJobRunnerStub struct {
	mu          sync.Mutex
	syncCalls   int
	settleCalls int
	syncType    string
	settleType  string
	settleLimit int

	syncResult   *LotterySyncResult
	settleResult *LotteryOpenedSettlementResult
	syncErr      error
	settleErr    error
	syncFn       func(context.Context, string) (*LotterySyncResult, error)
	settleFn     func(context.Context, string, int) (*LotteryOpenedSettlementResult, error)
}

func (s *lotteryJobRunnerStub) SyncLatestResult(ctx context.Context, lotteryType string) (*LotterySyncResult, error) {
	s.mu.Lock()
	s.syncCalls++
	s.syncType = lotteryType
	s.mu.Unlock()
	if s.syncFn != nil {
		return s.syncFn(ctx, lotteryType)
	}
	return s.syncResult, s.syncErr
}

func (s *lotteryJobRunnerStub) SettleOpenedIssues(ctx context.Context, lotteryType string, limit int) (*LotteryOpenedSettlementResult, error) {
	s.mu.Lock()
	s.settleCalls++
	s.settleType = lotteryType
	s.settleLimit = limit
	s.mu.Unlock()
	if s.settleFn != nil {
		return s.settleFn(ctx, lotteryType, limit)
	}
	return s.settleResult, s.settleErr
}

func (s *lotteryJobRunnerStub) stats() (syncCalls, settleCalls int, syncType, settleType string, settleLimit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.syncCalls, s.settleCalls, s.syncType, s.settleType, s.settleLimit
}

func TestLotteryJobServiceRunOnceSyncsAndSettles(t *testing.T) {
	settlement := &LotteryOpenedSettlementResult{
		LotteryType: LotteryTypeSSQ,
		TotalIssues: 1,
		SettledIssues: []LotterySettlementResult{{
			LotteryType: LotteryTypeSSQ,
			IssueNo:     "2026062",
			TotalOrders: 1,
			WinOrders:   1,
			RewardTotal: decimal.NewFromInt(5),
		}},
	}
	runner := &lotteryJobRunnerStub{
		syncResult: &LotterySyncResult{
			LotteryType: LotteryTypeSSQ,
			IssueNo:     "2026062",
			Replayed:    true,
		},
		settleResult: settlement,
	}
	svc := newLotteryJobService(runner, nil, "")

	got, err := svc.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, LotteryTypeSSQ, got.LotteryType)
	require.Equal(t, "2026062", got.SyncedIssueNo)
	require.True(t, got.SyncReplayed)
	require.Same(t, settlement, got.SettlementSummary)

	syncCalls, settleCalls, syncType, settleType, settleLimit := runner.stats()
	require.Equal(t, 1, syncCalls)
	require.Equal(t, 1, settleCalls)
	require.Equal(t, LotteryTypeSSQ, syncType)
	require.Equal(t, LotteryTypeSSQ, settleType)
	require.Equal(t, lotteryOpenedSettlementDefaultLimit, settleLimit)
}

func TestLotteryJobServiceRunOnceSettlesWhenSyncFails(t *testing.T) {
	syncErr := errors.New("provider unavailable")
	settlement := &LotteryOpenedSettlementResult{LotteryType: LotteryTypeSSQ}
	runner := &lotteryJobRunnerStub{
		syncErr:      syncErr,
		settleResult: settlement,
	}
	svc := newLotteryJobService(runner, nil, "")

	got, err := svc.RunOnce(context.Background())
	require.Error(t, err)
	require.ErrorContains(t, err, "sync latest lottery result")
	require.NotNil(t, got)
	require.Same(t, settlement, got.SettlementSummary)

	syncCalls, settleCalls, _, _, _ := runner.stats()
	require.Equal(t, 1, syncCalls)
	require.Equal(t, 1, settleCalls)
}

func TestLotteryJobServiceRunOnceSkipsOverlappingRun(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	runner := &lotteryJobRunnerStub{
		syncFn: func(ctx context.Context, lotteryType string) (*LotterySyncResult, error) {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return &LotterySyncResult{LotteryType: lotteryType, IssueNo: "2026062"}, nil
		},
		settleResult: &LotteryOpenedSettlementResult{LotteryType: LotteryTypeSSQ},
	}
	svc := newLotteryJobService(runner, nil, "")

	firstDone := make(chan error, 1)
	go func() {
		_, err := svc.RunOnce(context.Background())
		firstDone <- err
	}()
	<-entered

	got, err := svc.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, LotteryTypeSSQ, got.LotteryType)
	require.Nil(t, got.SettlementSummary)

	close(release)
	require.NoError(t, <-firstDone)

	syncCalls, settleCalls, _, _, _ := runner.stats()
	require.Equal(t, 1, syncCalls)
	require.Equal(t, 1, settleCalls)
}

func TestListOpenedLotteryIssues(t *testing.T) {
	client, mock, cleanup := newLotterySQLMockClient(t)
	t.Cleanup(cleanup)

	openTime := time.Date(2026, 6, 2, lotteryOpenHour, lotteryOpenMinute, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM lottery_issue")).
		WithArgs(LotteryTypeSSQ, lotteryIssueStatusOpened, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"lottery_type", "issue_no", "open_time", "status",
		}).AddRow(
			LotteryTypeSSQ,
			"2026062",
			openTime,
			lotteryIssueStatusOpened,
		))

	got, err := listOpenedLotteryIssues(context.Background(), client, LotteryTypeSSQ, 20)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "2026062", got[0].IssueNo)
	require.Equal(t, LotteryTypeSSQ, got[0].LotteryType)
	require.Equal(t, lotteryIssueStatusOpened, got[0].Status)
	require.Equal(t, openTime, got[0].OpenTime)
	require.Equal(t, openTime.Add(-lotteryCutoffLead), got[0].CutoffTime)
	require.NoError(t, mock.ExpectationsWereMet())
}
