//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type fixedLotteryProvider struct {
	name  string
	issue *service.Issue
}

func (p *fixedLotteryProvider) Name() string { return p.name }

func (p *fixedLotteryProvider) GetCurrentIssue(ctx context.Context) (*service.Issue, error) {
	_ = ctx
	return p.issue, nil
}

func (p *fixedLotteryProvider) GetLatestResult(ctx context.Context) (*service.Result, error) {
	_ = ctx
	return nil, nil
}

func TestLotteryServiceCreateBetCompletesMVPFlow(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "lottery-mvp", 1000)
	issue := lotteryTestIssue(30 * time.Minute)
	resetLotteryJackpotBalance(t, issue.LotteryType, "10000000")

	svc := service.NewLotteryService(
		client,
		nil,
		nil,
		nil,
		service.NewJackpotService(client),
		map[string]service.LotteryProvider{service.LotteryTypeSSQ: &fixedLotteryProvider{name: "fixed", issue: issue}},
	)

	result, err := svc.CreateBet(ctx, service.LotteryBetInput{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
		RedBalls:    []string{"33", "01", "08", "12", "18", "25"},
		BlueBall:    "09",
		RequestID:   "lottery-mvp-request",
	})
	require.NoError(t, err)
	require.False(t, result.Replayed)
	require.Equal(t, issue.IssueNo, result.IssueNo)
	require.Equal(t, service.LotteryTypeSSQ, result.LotteryType)
	require.Len(t, result.OrderIDs, 1)
	require.True(t, decimal.RequireFromString("100").Equal(result.Cost))
	require.Equal(t, "pending", result.Status)

	requireBankAccountSnapshot(t, user.ID, "900", "0")
	requireBankTransactionCountByType(t, user.ID, service.BankTxTypeLotteryBet, 1)
	requireLotteryJackpotBalance(t, service.LotteryTypeSSQ, "10000070")
	requireLotteryOrderCount(t, user.ID, service.LotteryTypeSSQ, issue.IssueNo, 1)
	requireLotteryOrderRecord(t, result.OrderIDs[0], "01,08,12,18,25,33", "09", "100", "pending")
}

func TestLotteryServiceCreateBetReplaysSameNumbersWithoutDoubleCharge(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "lottery-replay", 1000)
	issue := lotteryTestIssue(30 * time.Minute)
	resetLotteryJackpotBalance(t, issue.LotteryType, "10000000")

	svc := service.NewLotteryService(
		client,
		nil,
		nil,
		nil,
		service.NewJackpotService(client),
		map[string]service.LotteryProvider{service.LotteryTypeSSQ: &fixedLotteryProvider{name: "fixed", issue: issue}},
	)

	first, err := svc.CreateBet(ctx, service.LotteryBetInput{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
		RedBalls:    []string{"01", "08", "12", "18", "25", "33"},
		BlueBall:    "09",
		RequestID:   "lottery-replay-1",
	})
	require.NoError(t, err)

	second, err := svc.CreateBet(ctx, service.LotteryBetInput{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
		RedBalls:    []string{"33", "25", "18", "12", "08", "01"},
		BlueBall:    "09",
		RequestID:   "lottery-replay-2",
	})
	require.NoError(t, err)
	require.False(t, first.Replayed)
	require.True(t, second.Replayed)
	require.Equal(t, first.OrderIDs, second.OrderIDs)
	requireBankAccountSnapshot(t, user.ID, "900", "0")
	requireBankTransactionCountByType(t, user.ID, service.BankTxTypeLotteryBet, 1)
	requireLotteryJackpotBalance(t, service.LotteryTypeSSQ, "10000070")
	requireLotteryOrderCount(t, user.ID, service.LotteryTypeSSQ, issue.IssueNo, 1)
}

func TestLotteryServiceCreateBetRejectsClosedIssue(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "lottery-closed", 1000)
	issue := lotteryTestIssue(5 * time.Minute)
	resetLotteryJackpotBalance(t, issue.LotteryType, "10000000")

	svc := service.NewLotteryService(
		client,
		nil,
		nil,
		nil,
		service.NewJackpotService(client),
		map[string]service.LotteryProvider{service.LotteryTypeSSQ: &fixedLotteryProvider{name: "fixed", issue: issue}},
	)

	_, err := svc.CreateBet(ctx, service.LotteryBetInput{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
		RedBalls:    []string{"01", "08", "12", "18", "25", "33"},
		BlueBall:    "09",
	})
	require.ErrorIs(t, err, service.ErrLotteryIssueClosed)
	requireBankTransactionCountByType(t, user.ID, service.BankTxTypeLotteryBet, 0)
	requireBankLogCount(t, user.ID, 0)
	requireLotteryOrderCount(t, user.ID, service.LotteryTypeSSQ, issue.IssueNo, 0)
	requireLotteryJackpotBalance(t, service.LotteryTypeSSQ, "10000000")
}

func TestLotteryServiceCreateBetRejectsIssueLimit(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "lottery-limit", 1000)
	issue := lotteryTestIssue(30 * time.Minute)
	resetLotteryJackpotBalance(t, issue.LotteryType, "10000000")

	for i := 0; i < 100; i++ {
		_, err := integrationDB.ExecContext(ctx, `
INSERT INTO lottery_order (
    lottery_type, issue_no, user_id, red_balls, blue_ball, cost, reward, prize_level, red_hits, blue_hit, status, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, 0, '', 0, FALSE, 'pending', NOW(), NOW())
`, service.LotteryTypeSSQ, issue.IssueNo, user.ID, "01,02,03,04,05,06", "01", decimal.RequireFromString("100"))
		require.NoError(t, err)
	}

	svc := service.NewLotteryService(
		client,
		nil,
		nil,
		nil,
		service.NewJackpotService(client),
		map[string]service.LotteryProvider{service.LotteryTypeSSQ: &fixedLotteryProvider{name: "fixed", issue: issue}},
	)

	_, err := svc.CreateBet(ctx, service.LotteryBetInput{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
		RedBalls:    []string{"02", "08", "12", "18", "25", "33"},
		BlueBall:    "09",
	})
	require.ErrorIs(t, err, service.ErrLotteryBetLimitExceeded)
	requireBankTransactionCountByType(t, user.ID, service.BankTxTypeLotteryBet, 0)
	requireBankLogCount(t, user.ID, 0)
	requireLotteryJackpotBalance(t, service.LotteryTypeSSQ, "10000000")
}

func TestLotteryServiceGetMyOrdersReturnsOrdersByIssue(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateBankServiceUser(t, "lottery-query", 1000)
	issue := lotteryTestIssue(30 * time.Minute)
	resetLotteryJackpotBalance(t, issue.LotteryType, "10000000")

	svc := service.NewLotteryService(
		client,
		nil,
		nil,
		nil,
		service.NewJackpotService(client),
		map[string]service.LotteryProvider{service.LotteryTypeSSQ: &fixedLotteryProvider{name: "fixed", issue: issue}},
	)

	_, err := svc.CreateBet(ctx, service.LotteryBetInput{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
		RedBalls:    []string{"01", "08", "12", "18", "25", "33"},
		BlueBall:    "09",
	})
	require.NoError(t, err)
	_, err = svc.CreateBet(ctx, service.LotteryBetInput{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
		RedBalls:    []string{"02", "08", "12", "18", "25", "33"},
		BlueBall:    "10",
	})
	require.NoError(t, err)

	orders, err := svc.GetMyOrders(ctx, service.LotteryOrderQuery{
		UserID:      user.ID,
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     issue.IssueNo,
	})
	require.NoError(t, err)
	require.Len(t, orders, 2)
	require.Equal(t, issue.IssueNo, orders[0].IssueNo)
	require.Equal(t, "pending", orders[0].Status)
	require.Equal(t, issue.IssueNo, orders[1].IssueNo)
	require.Equal(t, "pending", orders[1].Status)
}

func lotteryTestIssue(offset time.Duration) *service.Issue {
	now := time.Now().UTC()
	return &service.Issue{
		LotteryType: service.LotteryTypeSSQ,
		IssueNo:     fmt.Sprintf("2099%05d", now.UnixNano()%100000),
		OpenTime:    now.Add(offset),
		Status:      "pending",
		Source:      "fixed",
	}
}

func resetLotteryJackpotBalance(t *testing.T, lotteryType string, balance string) {
	t.Helper()
	_, err := integrationDB.ExecContext(context.Background(), `
INSERT INTO lottery_jackpot (lottery_type, balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (lottery_type) DO UPDATE SET
    balance = EXCLUDED.balance,
    updated_at = NOW()
`, lotteryType, decimal.RequireFromString(balance))
	require.NoError(t, err)
}

func requireLotteryJackpotBalance(t *testing.T, lotteryType string, want string) {
	t.Helper()
	var balance decimal.Decimal
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT balance
FROM lottery_jackpot
WHERE lottery_type = $1
`, lotteryType).Scan(&balance)
	require.NoError(t, err)
	require.True(t, decimal.RequireFromString(want).Equal(balance), "jackpot = %s", balance.String())
}

func requireLotteryOrderCount(t *testing.T, userID int64, lotteryType, issueNo string, want int) {
	t.Helper()
	var count int
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM lottery_order
WHERE user_id = $1
  AND lottery_type = $2
  AND issue_no = $3
`, userID, lotteryType, issueNo).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, want, count)
}

func requireLotteryOrderRecord(t *testing.T, orderID int64, wantRed, wantBlue, wantCost, wantStatus string) {
	t.Helper()
	var redBalls string
	var blueBall string
	var cost decimal.Decimal
	var status string
	err := integrationDB.QueryRowContext(context.Background(), `
SELECT red_balls, blue_ball, cost, status
FROM lottery_order
WHERE id = $1
`, orderID).Scan(&redBalls, &blueBall, &cost, &status)
	require.NoError(t, err)
	require.Equal(t, wantRed, redBalls)
	require.Equal(t, wantBlue, blueBall)
	require.True(t, decimal.RequireFromString(wantCost).Equal(cost), "cost = %s", cost.String())
	require.Equal(t, wantStatus, status)
}
