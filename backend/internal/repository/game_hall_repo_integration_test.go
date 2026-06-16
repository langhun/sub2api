//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestGameHallRepositoryCommitSlotRound_WritesJackpotTransactions(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewGameHallRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("game-hall-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      88,
	})

	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET dg_balance = EXCLUDED.dg_balance, updated_at = NOW()
`, user.ID, 50.0)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
INSERT INTO game_hall_jackpots (code, balance, enabled, created_at, updated_at)
VALUES ($1, $2, TRUE, NOW(), NOW())
ON CONFLICT (code) DO UPDATE SET balance = EXCLUDED.balance, enabled = TRUE, updated_at = NOW()
`, gameHallJackpotCode, 100.0)
	require.NoError(t, err)

	result, err := repo.CommitSlotRound(ctx, service.GameSlotRoundPlan{
		UserID:         user.ID,
		GameType:       service.GameTypeSlots,
		BetAmount:      10,
		PayoutAmount:   30,
		NetAmount:      20,
		Multiplier:     3,
		JackpotBefore:  100,
		JackpotAfter:   80,
		Symbols:        []string{"cherry", "cherry", "cherry"},
		Outcome:        "win",
		Message:        "中奖",
		IdempotencyKey: "slot-round-" + uuid.NewString(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 80.0, result.JackpotBalance)

	rows, err := integrationDB.QueryContext(ctx, `
SELECT tx_type, amount, balance_before, balance_after, reference_type, reference_id, user_id
FROM game_hall_jackpot_transactions
WHERE jackpot_code = $1
ORDER BY id
`, gameHallJackpotCode)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	type jackpotTx struct {
		txType        string
		amount        float64
		balanceBefore float64
		balanceAfter  float64
		referenceType string
		referenceID   string
		userID        int64
	}

	var transactions []jackpotTx
	for rows.Next() {
		var item jackpotTx
		require.NoError(t, rows.Scan(
			&item.txType,
			&item.amount,
			&item.balanceBefore,
			&item.balanceAfter,
			&item.referenceType,
			&item.referenceID,
			&item.userID,
		))
		transactions = append(transactions, item)
	}
	require.NoError(t, rows.Err())

	require.Len(t, transactions, 2)
	require.Equal(t, jackpotTx{
		txType:        "bet_in",
		amount:        10,
		balanceBefore: 100,
		balanceAfter:  110,
		referenceType: gameHallReferenceSlot,
		referenceID:   service.GameTypeSlots,
		userID:        user.ID,
	}, transactions[0])
	require.Equal(t, jackpotTx{
		txType:        "payout_out",
		amount:        30,
		balanceBefore: 110,
		balanceAfter:  80,
		referenceType: gameHallReferenceSlot,
		referenceID:   service.GameTypeSlots,
		userID:        user.ID,
	}, transactions[1])
}

func TestGameHallRepositoryCommitSlotRound_CapsPayoutToAvailableJackpot(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewGameHallRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("game-hall-cap-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      88,
	})

	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET dg_balance = EXCLUDED.dg_balance, updated_at = NOW()
`, user.ID, 50.0)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
INSERT INTO game_hall_jackpots (code, balance, enabled, created_at, updated_at)
VALUES ($1, $2, TRUE, NOW(), NOW())
ON CONFLICT (code) DO UPDATE SET balance = EXCLUDED.balance, enabled = TRUE, updated_at = NOW()
`, gameHallJackpotCode, 5.0)
	require.NoError(t, err)

	result, err := repo.CommitSlotRound(ctx, service.GameSlotRoundPlan{
		UserID:         user.ID,
		GameType:       service.GameTypeSlots,
		BetAmount:      10,
		PayoutAmount:   30,
		NetAmount:      20,
		Multiplier:     3,
		JackpotBefore:  5,
		JackpotAfter:   0,
		Symbols:        []string{"cherry", "cherry", "cherry"},
		Outcome:        "win",
		Message:        "中奖",
		IdempotencyKey: "slot-round-cap-" + uuid.NewString(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 15.0, result.PayoutAmount)
	require.Equal(t, 5.0, result.NetAmount)
	require.Equal(t, 1.5, result.Multiplier)
	require.Equal(t, 55.0, result.DGBalanceAfter)
	require.Equal(t, 0.0, result.JackpotBalance)
	require.Equal(t, "win", result.Outcome)
	require.Equal(t, "奖池不足，已按当前奖池派彩", result.Message)

	rows, err := integrationDB.QueryContext(ctx, `
SELECT tx_type, amount, balance_before, balance_after
FROM game_hall_jackpot_transactions
WHERE jackpot_code = $1 AND user_id = $2
ORDER BY id
`, gameHallJackpotCode, user.ID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var amounts [][4]float64
	for rows.Next() {
		var txType string
		var amount, before, after float64
		require.NoError(t, rows.Scan(&txType, &amount, &before, &after))
		if txType == "bet_in" || txType == "payout_out" {
			amounts = append(amounts, [4]float64{amount, before, after, 0})
		}
	}
	require.NoError(t, rows.Err())
	require.Len(t, amounts, 2)
	require.Equal(t, [4]float64{10, 5, 15, 0}, amounts[0])
	require.Equal(t, [4]float64{15, 15, 0, 0}, amounts[1])
}

func TestGameHallRepositoryGetSnapshot_WorksWithoutLegacySharedTables(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewGameHallRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("game-hall-legacy-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      88,
	})

	var legacyWalletTable sql.NullString
	err := integrationDB.QueryRowContext(ctx, `
SELECT to_regclass('public.game_wallets')
`).Scan(&legacyWalletTable)
	require.NoError(t, err)
	require.False(t, legacyWalletTable.Valid)

	_, err = integrationDB.ExecContext(ctx, `
DELETE FROM game_hall_wallet_transactions
WHERE user_id = $1
`, user.ID)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
DELETE FROM game_hall_wallets
WHERE user_id = $1
`, user.ID)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
DELETE FROM game_hall_jackpot_transactions
WHERE jackpot_code = $1
`, gameHallJackpotCode)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
UPDATE game_hall_jackpots
SET balance = 0,
    updated_at = NOW()
WHERE code = $1
`, gameHallJackpotCode)
	require.NoError(t, err)

	snapshot, err := repo.GetSnapshot(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 88.0, snapshot.MainBalance)
	require.Equal(t, 0.0, snapshot.DGBalance)
	require.Equal(t, 0.0, snapshot.JackpotBalance)
}

func TestGameHallRepositoryGetSnapshot_UsesUsersBalanceEvenWhenBankTableExists(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewGameHallRepository(client, integrationDB)

	require.NoError(t, ensureUsersBankAccountTableForTest(ctx))

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("game-hall-bank-snapshot-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      5,
	})

	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO users_bank_account (user_id, balance, frozen_amount, credit_limit, total_debt, version, status, created_at, updated_at, debt_principal, debt_interest)
VALUES ($1, $2, 0, 0, 0, 1, 'ACTIVE', NOW(), NOW(), 0, 0)
ON CONFLICT (user_id) DO UPDATE
SET balance = EXCLUDED.balance,
    updated_at = NOW()
`, user.ID, 88.0)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET dg_balance = EXCLUDED.dg_balance, updated_at = NOW()
`, user.ID, 12.0)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
INSERT INTO game_hall_jackpots (code, balance, enabled, created_at, updated_at)
VALUES ($1, $2, TRUE, NOW(), NOW())
ON CONFLICT (code) DO UPDATE SET balance = EXCLUDED.balance, enabled = TRUE, updated_at = NOW()
`, gameHallJackpotCode, 345.0)
	require.NoError(t, err)

	snapshot, err := repo.GetSnapshot(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 5.0, snapshot.MainBalance)
	require.Equal(t, 12.0, snapshot.DGBalance)
	require.Equal(t, 345.0, snapshot.JackpotBalance)
}

func TestGameHallRepositoryCommitExchange_UsesUsersBalanceEvenWhenBankTableExists(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewGameHallRepository(client, integrationDB)

	require.NoError(t, ensureUsersBankAccountTableForTest(ctx))

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("game-hall-bank-exchange-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      120,
	})

	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO users_bank_account (user_id, balance, frozen_amount, credit_limit, total_debt, version, status, created_at, updated_at, debt_principal, debt_interest)
VALUES ($1, $2, 0, 0, 0, 1, 'ACTIVE', NOW(), NOW(), 0, 0)
ON CONFLICT (user_id) DO UPDATE
SET balance = EXCLUDED.balance,
    updated_at = NOW()
`, user.ID, 5.0)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET dg_balance = EXCLUDED.dg_balance, updated_at = NOW()
`, user.ID, 10.0)
	require.NoError(t, err)

	result, err := repo.CommitExchange(ctx, service.GameExchangePlan{
		UserID:         user.ID,
		Direction:      service.GameExchangeBalanceToDG,
		Amount:         20,
		IdempotencyKey: "bank-account-exchange-" + uuid.NewString(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 120.0, result.MainBalanceBefore)
	require.Equal(t, 100.0, result.MainBalanceAfter)
	require.Equal(t, 10.0, result.DGBalanceBefore)
	require.Equal(t, 30.0, result.DGBalanceAfter)

	var (
		userBalance        float64
		bankAccountBalance float64
		dgBalance          float64
	)
	err = integrationDB.QueryRowContext(ctx, `
SELECT balance
FROM users
WHERE id = $1
`, user.ID).Scan(&userBalance)
	require.NoError(t, err)
	require.Equal(t, 100.0, userBalance)

	err = integrationDB.QueryRowContext(ctx, `
SELECT balance
FROM users_bank_account
WHERE user_id = $1
`, user.ID).Scan(&bankAccountBalance)
	require.NoError(t, err)
	require.Equal(t, 5.0, bankAccountBalance)

	err = integrationDB.QueryRowContext(ctx, `
SELECT dg_balance
FROM game_hall_wallets
WHERE user_id = $1
`, user.ID).Scan(&dgBalance)
	require.NoError(t, err)
	require.Equal(t, 30.0, dgBalance)
}

func ensureUsersBankAccountTableForTest(ctx context.Context) error {
	_, err := integrationDB.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS users_bank_account (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    frozen_amount DECIMAL(20, 8) NOT NULL DEFAULT 0,
    credit_limit DECIMAL(20, 8) NOT NULL DEFAULT 0,
    total_debt DECIMAL(20, 8) NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    debt_principal DECIMAL(20, 8) NOT NULL DEFAULT 0,
    debt_interest DECIMAL(20, 8) NOT NULL DEFAULT 0
)
`)
	return err
}
