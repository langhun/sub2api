package service

import (
	"context"
	"errors"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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
