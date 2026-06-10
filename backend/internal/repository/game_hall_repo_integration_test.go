//go:build integration

package repository

import (
	"context"
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
INSERT INTO game_wallets (user_id, dg_balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET dg_balance = EXCLUDED.dg_balance, updated_at = NOW()
`, user.ID, 50.0)
	require.NoError(t, err)

	_, err = integrationDB.ExecContext(ctx, `
INSERT INTO game_jackpots (code, balance, enabled, created_at, updated_at)
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
FROM game_jackpot_transactions
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
