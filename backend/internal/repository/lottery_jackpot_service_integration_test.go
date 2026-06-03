//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestJackpotServiceDepositWithdrawAndGetBalance(t *testing.T) {
	tx := testEntTx(t)
	client := tx.Client()
	ctx := context.Background()

	_, err := client.ExecContext(ctx, `
INSERT INTO lottery_jackpot (lottery_type, balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (lottery_type) DO UPDATE SET
    balance = EXCLUDED.balance,
    updated_at = NOW()
`, service.LotteryTypeSSQ, decimal.RequireFromString("10000000"))
	require.NoError(t, err)

	jackpot := service.NewJackpotService(client)

	balance, err := jackpot.GetBalance(ctx, service.LotteryTypeSSQ)
	require.NoError(t, err)
	require.True(t, decimal.RequireFromString("10000000").Equal(balance))

	require.NoError(t, jackpot.Deposit(ctx, service.LotteryTypeSSQ, decimal.RequireFromString("70")))
	balance, err = jackpot.GetBalance(ctx, service.LotteryTypeSSQ)
	require.NoError(t, err)
	require.True(t, decimal.RequireFromString("10000070").Equal(balance))

	require.NoError(t, jackpot.Withdraw(ctx, service.LotteryTypeSSQ, decimal.RequireFromString("50")))
	balance, err = jackpot.GetBalance(ctx, service.LotteryTypeSSQ)
	require.NoError(t, err)
	require.True(t, decimal.RequireFromString("10000020").Equal(balance))
}

func TestJackpotServiceWithdrawRejectsInsufficientBalance(t *testing.T) {
	tx := testEntTx(t)
	client := tx.Client()
	ctx := context.Background()

	_, err := client.ExecContext(ctx, `
INSERT INTO lottery_jackpot (lottery_type, balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (lottery_type) DO UPDATE SET
    balance = EXCLUDED.balance,
    updated_at = NOW()
`, service.LotteryTypeSSQ, decimal.RequireFromString("10"))
	require.NoError(t, err)

	jackpot := service.NewJackpotService(client)
	err = jackpot.Withdraw(ctx, service.LotteryTypeSSQ, decimal.RequireFromString("11"))
	require.ErrorIs(t, err, service.ErrLotteryJackpotInsufficient)
}
