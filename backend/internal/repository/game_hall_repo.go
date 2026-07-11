package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	gameHallJackpotCode       = "game_hall"
	gameHallRepoTxMaxRetries  = 3
	gameHallReferenceExchange = "exchange"
	gameHallReferenceSlot     = "slots"
	gameHallCappedMessage     = "奖池不足，已按当前奖池派彩"
	gameHallWalletTxTable     = "game_hall_wallet_transactions"
	gameHallRoundsTable       = "game_hall_rounds"
)

type gameHallRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewGameHallRepository(client *dbent.Client, db *sql.DB) service.GameHallStore {
	return &gameHallRepository{
		client: client,
		db:     db,
	}
}

func (r *gameHallRepository) GetSnapshot(ctx context.Context, userID int64) (*service.GameWalletSnapshot, error) {
	client := clientFromContext(ctx, r.client)
	if err := ensureGameWalletRow(ctx, client, userID); err != nil {
		return nil, err
	}
	if err := ensureGameJackpotRow(ctx, client); err != nil {
		return nil, err
	}

	query := `
SELECT u.balance,
       gw.dg_balance,
       gj.balance
FROM users u
JOIN game_hall_wallets gw ON gw.user_id = u.id
JOIN game_hall_jackpots gj ON gj.code = $2
WHERE u.id = $1 AND u.deleted_at IS NULL
LIMIT 1
`
	rows, err := client.QueryContext(ctx, query, userID, gameHallJackpotCode)
	if err != nil {
		return nil, fmt.Errorf("query game hall snapshot: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("query game hall snapshot: %w", err)
		}
		return nil, service.ErrUserNotFound
	}

	snapshot := &service.GameWalletSnapshot{UserID: userID}
	if err := rows.Scan(&snapshot.MainBalance, &snapshot.DGBalance, &snapshot.JackpotBalance); err != nil {
		return nil, fmt.Errorf("scan game hall snapshot: %w", err)
	}
	return snapshot, rows.Err()
}

func (r *gameHallRepository) CommitExchange(ctx context.Context, plan service.GameExchangePlan) (*service.GameExchangeResult, error) {
	if existing, err := r.findExchangeResult(ctx, plan.UserID, plan.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	var result *service.GameExchangeResult
	err := r.runInTx(ctx, func(txCtx context.Context) error {
		client := clientFromContext(txCtx, r.client)
		if err := ensureGameWalletRow(txCtx, client, plan.UserID); err != nil {
			return err
		}

		mainBalance, err := lockUserMainBalance(txCtx, client, plan.UserID)
		if err != nil {
			return err
		}
		dgBalance, err := lockUserDGBalance(txCtx, client, plan.UserID)
		if err != nil {
			return err
		}

		mainAfter := mainBalance
		dgAfter := dgBalance

		switch plan.Direction {
		case service.GameExchangeBalanceToDG:
			if mainBalance < plan.Amount {
				return service.ErrGameInsufficientMainBalance
			}
			mainAfter = roundDBAmount(mainBalance - plan.Amount)
			dgAfter = roundDBAmount(dgBalance + plan.Amount)
		case service.GameExchangeDGToBalance:
			if dgBalance < plan.Amount {
				return service.ErrGameInsufficientDGBalance
			}
			mainAfter = roundDBAmount(mainBalance + plan.Amount)
			dgAfter = roundDBAmount(dgBalance - plan.Amount)
		default:
			return service.ErrGameExchangeDirectionInvalid
		}

		if err := updateUserMainBalance(txCtx, client, plan.UserID, mainAfter); err != nil {
			return err
		}
		if err := updateUserDGBalance(txCtx, client, plan.UserID, dgAfter); err != nil {
			return err
		}
		if err := insertGameWalletTransaction(txCtx, client, gameWalletTransactionParams{
			UserID:         plan.UserID,
			TxType:         resolveExchangeTxType(plan.Direction),
			Amount:         plan.Amount,
			BalanceBefore:  dgBalance,
			BalanceAfter:   dgAfter,
			ReferenceType:  gameHallReferenceExchange,
			ReferenceID:    plan.Direction,
			IdempotencyKey: plan.IdempotencyKey,
			Metadata: map[string]any{
				"direction":           plan.Direction,
				"main_balance_before": mainBalance,
				"main_balance_after":  mainAfter,
			},
		}); err != nil {
			return err
		}
		if err := insertMainBalanceAudit(txCtx, client, plan, mainBalance, mainAfter); err != nil {
			return err
		}

		result = &service.GameExchangeResult{
			Direction:         plan.Direction,
			Amount:            roundDBAmount(plan.Amount),
			MainBalanceBefore: roundDBAmount(mainBalance),
			MainBalanceAfter:  mainAfter,
			DGBalanceBefore:   roundDBAmount(dgBalance),
			DGBalanceAfter:    dgAfter,
		}
		return nil
	})
	if err != nil {
		// A concurrent request with the same idempotency key can win after the
		// optimistic lookup but before our unique insert. Its committed result is
		// the authoritative replay response; do not leak a unique violation.
		if existing, lookupErr := r.findExchangeResult(ctx, plan.UserID, plan.IdempotencyKey); lookupErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return result, nil
}

func insertMainBalanceAudit(ctx context.Context, client *dbent.Client, plan service.GameExchangePlan, before, after float64) error {
	metadata, err := json.Marshal(map[string]any{"direction": plan.Direction, "dg_amount": plan.Amount})
	if err != nil {
		return fmt.Errorf("marshal main balance audit metadata: %w", err)
	}
	_, err = client.ExecContext(ctx, `INSERT INTO game_hall_main_balance_transactions
(user_id, direction, amount, balance_before, balance_after, idempotency_key, metadata, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8)`, plan.UserID, plan.Direction, roundDBAmount(plan.Amount), roundDBAmount(before), roundDBAmount(after), plan.IdempotencyKey, string(metadata), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert game hall main balance audit: %w", err)
	}
	return nil
}

func (r *gameHallRepository) findExchangeResult(ctx context.Context, userID int64, idempotencyKey string) (*service.GameExchangeResult, error) {
	row := r.db.QueryRowContext(ctx, `SELECT tx_type, amount, balance_before, balance_after, metadata
FROM game_hall_wallet_transactions WHERE user_id = $1 AND idempotency_key = $2`, userID, idempotencyKey)
	var txType string
	var amount, dgBefore, dgAfter float64
	var metadata []byte
	if err := row.Scan(&txType, &amount, &dgBefore, &dgAfter, &metadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup idempotent game exchange: %w", err)
	}
	var values struct {
		Direction  string  `json:"direction"`
		MainBefore float64 `json:"main_balance_before"`
		MainAfter  float64 `json:"main_balance_after"`
	}
	if err := json.Unmarshal(metadata, &values); err != nil {
		return nil, fmt.Errorf("decode idempotent game exchange: %w", err)
	}
	if values.Direction == "" {
		values.Direction = service.GameExchangeBalanceToDG
		if txType == "exchange_out" {
			values.Direction = service.GameExchangeDGToBalance
		}
	}
	return &service.GameExchangeResult{
		Direction: values.Direction, Amount: amount,
		MainBalanceBefore: values.MainBefore, MainBalanceAfter: values.MainAfter,
		DGBalanceBefore: dgBefore, DGBalanceAfter: dgAfter,
	}, nil
}

func (r *gameHallRepository) CommitSlotRound(ctx context.Context, plan service.GameSlotRoundPlan) (*service.GamePlayResult, error) {
	if existing, err := r.findSlotRoundResult(ctx, plan.UserID, plan.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	var result *service.GamePlayResult
	err := r.runInTx(ctx, func(txCtx context.Context) error {
		client := clientFromContext(txCtx, r.client)
		if err := ensureGameWalletRow(txCtx, client, plan.UserID); err != nil {
			return err
		}
		if err := ensureGameJackpotRow(txCtx, client); err != nil {
			return err
		}

		dgBalance, err := lockUserDGBalance(txCtx, client, plan.UserID)
		if err != nil {
			return err
		}
		if dgBalance < plan.BetAmount {
			return service.ErrGameInsufficientDGBalance
		}
		jackpotBalance, err := lockGameJackpot(txCtx, client)
		if err != nil {
			return err
		}

		actualPayout := roundDBAmount(plan.PayoutAmount)
		payoutLimit := roundDBAmount(jackpotBalance + plan.BetAmount)
		payoutCapped := actualPayout > payoutLimit
		if payoutCapped {
			actualPayout = payoutLimit
		}
		actualNet := roundDBAmount(actualPayout - plan.BetAmount)
		actualMultiplier := plan.Multiplier
		if payoutCapped {
			actualMultiplier = resolveActualDBMultiplier(actualPayout, plan.BetAmount)
		}
		actualOutcome := resolveGameOutcomeFromNet(actualNet)
		actualMessage := plan.Message
		if payoutCapped {
			actualMessage = gameHallCappedMessage
		}
		dgAfter := roundDBAmount(dgBalance - plan.BetAmount + actualPayout)
		jackpotAfter := roundDBAmount(jackpotBalance + plan.BetAmount - actualPayout)
		roundID, err := insertGameRound(txCtx, client, gameRoundParams{
			UserID: plan.UserID, GameType: plan.GameType, BetAmount: plan.BetAmount,
			PayoutAmount: actualPayout, NetAmount: actualNet, Multiplier: actualMultiplier,
			BalanceBefore: dgBalance, BalanceAfter: dgAfter, JackpotBefore: jackpotBalance,
			JackpotAfter: jackpotAfter, Outcome: actualOutcome, Symbols: plan.Symbols,
			IdempotencyKey: plan.IdempotencyKey, Metadata: map[string]any{"message": actualMessage, "payout_capped": payoutCapped, "rule_version": service.GameSlotsRuleVersion},
		})
		if err != nil {
			return err
		}

		if err := updateUserDGBalance(txCtx, client, plan.UserID, dgAfter); err != nil {
			return err
		}
		if err := updateGameJackpot(txCtx, client, jackpotAfter); err != nil {
			return err
		}
		if err := insertGameJackpotTransaction(txCtx, client, gameJackpotTransactionParams{
			JackpotCode:    gameHallJackpotCode,
			UserID:         &plan.UserID,
			TxType:         "bet_in",
			Amount:         plan.BetAmount,
			BalanceBefore:  jackpotBalance,
			BalanceAfter:   roundDBAmount(jackpotBalance + plan.BetAmount),
			ReferenceType:  gameHallReferenceSlot,
			ReferenceID:    fmt.Sprint(roundID),
			IdempotencyKey: plan.IdempotencyKey + ":jackpot:bet",
			Metadata: map[string]any{
				"game_type": plan.GameType,
				"symbols":   plan.Symbols,
				"outcome":   actualOutcome,
			},
		}); err != nil {
			return err
		}
		if actualPayout > 0 {
			betInAfter := roundDBAmount(jackpotBalance + plan.BetAmount)
			if err := insertGameJackpotTransaction(txCtx, client, gameJackpotTransactionParams{
				JackpotCode:    gameHallJackpotCode,
				UserID:         &plan.UserID,
				TxType:         "payout_out",
				Amount:         actualPayout,
				BalanceBefore:  betInAfter,
				BalanceAfter:   jackpotAfter,
				ReferenceType:  gameHallReferenceSlot,
				ReferenceID:    fmt.Sprint(roundID),
				IdempotencyKey: plan.IdempotencyKey + ":jackpot:payout",
				Metadata: map[string]any{
					"game_type":  plan.GameType,
					"symbols":    plan.Symbols,
					"multiplier": actualMultiplier,
					"outcome":    actualOutcome,
				},
			}); err != nil {
				return err
			}
		}

		if err := insertGameWalletTransaction(txCtx, client, gameWalletTransactionParams{
			UserID:         plan.UserID,
			TxType:         "slot_bet",
			Amount:         plan.BetAmount,
			BalanceBefore:  dgBalance,
			BalanceAfter:   roundDBAmount(dgBalance - plan.BetAmount),
			ReferenceType:  gameHallReferenceSlot,
			ReferenceID:    fmt.Sprint(roundID),
			IdempotencyKey: plan.IdempotencyKey + ":bet",
			Metadata: map[string]any{
				"game_type": plan.GameType,
				"symbols":   plan.Symbols,
				"outcome":   actualOutcome,
			},
		}); err != nil {
			return err
		}

		if actualPayout > 0 {
			if err := insertGameWalletTransaction(txCtx, client, gameWalletTransactionParams{
				UserID:         plan.UserID,
				TxType:         "slot_payout",
				Amount:         actualPayout,
				BalanceBefore:  roundDBAmount(dgBalance - plan.BetAmount),
				BalanceAfter:   dgAfter,
				ReferenceType:  gameHallReferenceSlot,
				ReferenceID:    fmt.Sprint(roundID),
				IdempotencyKey: plan.IdempotencyKey + ":payout",
				Metadata: map[string]any{
					"game_type":  plan.GameType,
					"symbols":    plan.Symbols,
					"multiplier": actualMultiplier,
					"outcome":    actualOutcome,
				},
			}); err != nil {
				return err
			}
		}

		result = &service.GamePlayResult{
			RoundID:         roundID,
			GameType:        plan.GameType,
			BetAmount:       roundDBAmount(plan.BetAmount),
			PayoutAmount:    actualPayout,
			NetAmount:       actualNet,
			Multiplier:      actualMultiplier,
			DGBalanceBefore: roundDBAmount(dgBalance),
			DGBalanceAfter:  dgAfter,
			JackpotBalance:  jackpotAfter,
			Outcome:         actualOutcome,
			Symbols:         append([]string(nil), plan.Symbols...),
			Message:         actualMessage,
		}
		return nil
	})
	if err != nil {
		if existing, lookupErr := r.findSlotRoundResult(ctx, plan.UserID, plan.IdempotencyKey); lookupErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return result, nil
}

func (r *gameHallRepository) findSlotRoundResult(ctx context.Context, userID int64, idempotencyKey string) (*service.GamePlayResult, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, game_type, bet_amount, payout_amount, net_amount, multiplier,
balance_before, balance_after, jackpot_after, outcome, symbols, metadata
FROM game_hall_rounds WHERE user_id = $1 AND idempotency_key = $2`, userID, idempotencyKey)
	var result service.GamePlayResult
	var symbols, metadata []byte
	if err := row.Scan(&result.RoundID, &result.GameType, &result.BetAmount, &result.PayoutAmount, &result.NetAmount,
		&result.Multiplier, &result.DGBalanceBefore, &result.DGBalanceAfter, &result.JackpotBalance, &result.Outcome, &symbols, &metadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup idempotent game round: %w", err)
	}
	if err := json.Unmarshal(symbols, &result.Symbols); err != nil {
		return nil, fmt.Errorf("decode idempotent game round symbols: %w", err)
	}
	var values struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(metadata, &values); err != nil {
		return nil, fmt.Errorf("decode idempotent game round metadata: %w", err)
	}
	result.Message = values.Message
	return &result, nil
}

func (r *gameHallRepository) ListWalletTransactions(ctx context.Context, userID *int64, page, pageSize int) ([]service.GameWalletTransaction, int64, error) {
	where, args := "", []any{}
	if userID != nil {
		where, args = " WHERE user_id = $1", append(args, *userID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+gameHallWalletTxTable+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count game wallet transactions: %w", err)
	}
	args = append(args, pageSize, (page-1)*pageSize)
	limitPos := len(args) - 1
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, user_id, tx_type, amount, balance_before, balance_after,
reference_type, reference_id, metadata, created_at FROM %s%s ORDER BY id DESC LIMIT $%d OFFSET $%d`, gameHallWalletTxTable, where, limitPos, limitPos+1), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list game wallet transactions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.GameWalletTransaction, 0)
	for rows.Next() {
		var item service.GameWalletTransaction
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.TxType, &item.Amount, &item.BalanceBefore, &item.BalanceAfter, &item.ReferenceType, &item.ReferenceID, &metadata, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan game wallet transaction: %w", err)
		}
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, 0, fmt.Errorf("decode game wallet transaction metadata: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *gameHallRepository) ListRounds(ctx context.Context, userID *int64, page, pageSize int) ([]service.GameRound, int64, error) {
	where, args := "", []any{}
	if userID != nil {
		where, args = " WHERE user_id = $1", append(args, *userID)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+gameHallRoundsTable+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count game rounds: %w", err)
	}
	args = append(args, pageSize, (page-1)*pageSize)
	limitPos := len(args) - 1
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, user_id, game_type, bet_amount, payout_amount, net_amount, multiplier,
balance_before, balance_after, jackpot_before, jackpot_after, outcome, symbols, metadata, created_at
FROM %s%s ORDER BY id DESC LIMIT $%d OFFSET $%d`, gameHallRoundsTable, where, limitPos, limitPos+1), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list game rounds: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.GameRound, 0)
	for rows.Next() {
		var item service.GameRound
		var symbols, metadata []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.GameType, &item.BetAmount, &item.PayoutAmount, &item.NetAmount, &item.Multiplier, &item.BalanceBefore, &item.BalanceAfter, &item.JackpotBefore, &item.JackpotAfter, &item.Outcome, &symbols, &metadata, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan game round: %w", err)
		}
		if err := json.Unmarshal(symbols, &item.Symbols); err != nil {
			return nil, 0, fmt.Errorf("decode game round symbols: %w", err)
		}
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, 0, fmt.Errorf("decode game round metadata: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *gameHallRepository) runInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if dbent.TxFromContext(ctx) != nil {
		return fn(ctx)
	}

	var lastErr error
	for attempt := 0; attempt < gameHallRepoTxMaxRetries; attempt++ {
		tx, err := r.client.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return fmt.Errorf("begin game hall tx: %w", err)
		}

		txCtx := dbent.NewTxContext(ctx, tx)
		if err := fn(txCtx); err != nil {
			_ = tx.Rollback()
			if isRetryableGameHallTxErr(err) {
				lastErr = err
				continue
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			if isRetryableGameHallTxErr(err) {
				lastErr = fmt.Errorf("commit game hall tx: %w", err)
				continue
			}
			return err
		}
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("begin game hall tx: exhausted retries")
}

type gameWalletTransactionParams struct {
	UserID         int64
	TxType         string
	Amount         float64
	BalanceBefore  float64
	BalanceAfter   float64
	ReferenceType  string
	ReferenceID    string
	IdempotencyKey string
	Metadata       map[string]any
}

type gameJackpotTransactionParams struct {
	JackpotCode    string
	UserID         *int64
	TxType         string
	Amount         float64
	BalanceBefore  float64
	BalanceAfter   float64
	ReferenceType  string
	ReferenceID    string
	IdempotencyKey string
	Metadata       map[string]any
}

type gameRoundParams struct {
	UserID                                                   int64
	GameType                                                 string
	BetAmount, PayoutAmount, NetAmount, Multiplier           float64
	BalanceBefore, BalanceAfter, JackpotBefore, JackpotAfter float64
	Outcome                                                  string
	Symbols                                                  []string
	IdempotencyKey                                           string
	Metadata                                                 map[string]any
}

func insertGameRound(ctx context.Context, client *dbent.Client, params gameRoundParams) (int64, error) {
	symbolsJSON, err := json.Marshal(params.Symbols)
	if err != nil {
		return 0, fmt.Errorf("marshal game round symbols: %w", err)
	}
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return 0, fmt.Errorf("marshal game round metadata: %w", err)
	}
	rows, err := client.QueryContext(ctx, `INSERT INTO game_hall_rounds (user_id, game_type, bet_amount, payout_amount, net_amount, multiplier,
balance_before, balance_after, jackpot_before, jackpot_after, outcome, symbols, idempotency_key, metadata, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14::jsonb,$15) RETURNING id`, params.UserID, params.GameType,
		roundDBAmount(params.BetAmount), roundDBAmount(params.PayoutAmount), roundDBAmount(params.NetAmount), roundDBAmount(params.Multiplier),
		roundDBAmount(params.BalanceBefore), roundDBAmount(params.BalanceAfter), roundDBAmount(params.JackpotBefore), roundDBAmount(params.JackpotAfter),
		params.Outcome, string(symbolsJSON), params.IdempotencyKey, string(metadataJSON), time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("insert game round: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, sql.ErrNoRows
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		return 0, fmt.Errorf("scan game round id: %w", err)
	}
	return id, nil
}

func insertGameWalletTransaction(ctx context.Context, client *dbent.Client, params gameWalletTransactionParams) error {
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("marshal game wallet transaction metadata: %w", err)
	}
	_, err = client.ExecContext(ctx, `
INSERT INTO game_hall_wallet_transactions (
    user_id, tx_type, amount, balance_before, balance_after,
    reference_type, reference_id, idempotency_key, metadata, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
`,
		params.UserID,
		params.TxType,
		roundDBAmount(params.Amount),
		roundDBAmount(params.BalanceBefore),
		roundDBAmount(params.BalanceAfter),
		params.ReferenceType,
		params.ReferenceID,
		params.IdempotencyKey,
		string(metadataJSON),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert game wallet transaction: %w", err)
	}
	return nil
}

func insertGameJackpotTransaction(ctx context.Context, client *dbent.Client, params gameJackpotTransactionParams) error {
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("marshal game jackpot transaction metadata: %w", err)
	}
	_, err = client.ExecContext(ctx, `
INSERT INTO game_hall_jackpot_transactions (
    jackpot_code, tx_type, amount, balance_before, balance_after,
    reference_type, reference_id, user_id, idempotency_key, metadata, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)
`,
		params.JackpotCode,
		params.TxType,
		roundDBAmount(params.Amount),
		roundDBAmount(params.BalanceBefore),
		roundDBAmount(params.BalanceAfter),
		params.ReferenceType,
		params.ReferenceID,
		params.UserID,
		params.IdempotencyKey,
		string(metadataJSON),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert game jackpot transaction: %w", err)
	}
	return nil
}

func ensureGameWalletRow(ctx context.Context, client *dbent.Client, userID int64) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)
VALUES ($1, 0, NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING
`, userID)
	if err != nil {
		return fmt.Errorf("ensure game wallet row: %w", err)
	}
	return nil
}

func ensureGameJackpotRow(ctx context.Context, client *dbent.Client) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO game_hall_jackpots (code, balance, enabled, created_at, updated_at)
VALUES ($1, 0, TRUE, NOW(), NOW())
ON CONFLICT (code) DO NOTHING
`, gameHallJackpotCode)
	if err != nil {
		return fmt.Errorf("ensure game jackpot row: %w", err)
	}
	return nil
}

func lockUserMainBalance(ctx context.Context, client *dbent.Client, userID int64) (float64, error) {
	return lockLegacyUserMainBalance(ctx, client, userID)
}

func lockLegacyUserMainBalance(ctx context.Context, client *dbent.Client, userID int64) (float64, error) {
	rows, err := client.QueryContext(ctx, `
SELECT balance
FROM users
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE
`, userID)
	if err != nil {
		return 0, fmt.Errorf("lock legacy user main balance: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, fmt.Errorf("lock legacy user main balance: %w", err)
		}
		return 0, service.ErrUserNotFound
	}

	var balance float64
	if err := rows.Scan(&balance); err != nil {
		return 0, fmt.Errorf("scan legacy user main balance: %w", err)
	}
	return balance, rows.Err()
}

func lockUserDGBalance(ctx context.Context, client *dbent.Client, userID int64) (float64, error) {
	rows, err := client.QueryContext(ctx, `
SELECT dg_balance
FROM game_hall_wallets
WHERE user_id = $1
FOR UPDATE
`, userID)
	if err != nil {
		return 0, fmt.Errorf("lock user DG balance: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, fmt.Errorf("lock user DG balance: %w", err)
		}
		return 0, fmt.Errorf("lock user DG balance: %w", sql.ErrNoRows)
	}

	var balance float64
	if err := rows.Scan(&balance); err != nil {
		return 0, fmt.Errorf("scan user DG balance: %w", err)
	}
	return balance, rows.Err()
}

func lockGameJackpot(ctx context.Context, client *dbent.Client) (float64, error) {
	rows, err := client.QueryContext(ctx, `
SELECT balance
FROM game_hall_jackpots
WHERE code = $1
FOR UPDATE
`, gameHallJackpotCode)
	if err != nil {
		return 0, fmt.Errorf("lock game jackpot: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, fmt.Errorf("lock game jackpot: %w", err)
		}
		return 0, fmt.Errorf("lock game jackpot: %w", sql.ErrNoRows)
	}

	var balance float64
	if err := rows.Scan(&balance); err != nil {
		return 0, fmt.Errorf("scan game jackpot: %w", err)
	}
	return balance, rows.Err()
}

func updateUserMainBalance(ctx context.Context, client *dbent.Client, userID int64, balance float64) error {
	_, err := client.ExecContext(ctx, `
UPDATE users
SET balance = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
`, userID, roundDBAmount(balance))
	if err != nil {
		return fmt.Errorf("update user main balance: %w", err)
	}
	return nil
}

func updateUserDGBalance(ctx context.Context, client *dbent.Client, userID int64, balance float64) error {
	_, err := client.ExecContext(ctx, `
UPDATE game_hall_wallets
SET dg_balance = $2,
    updated_at = NOW()
WHERE user_id = $1
`, userID, roundDBAmount(balance))
	if err != nil {
		return fmt.Errorf("update user DG balance: %w", err)
	}
	return nil
}

func updateGameJackpot(ctx context.Context, client *dbent.Client, balance float64) error {
	_, err := client.ExecContext(ctx, `
UPDATE game_hall_jackpots
SET balance = $2,
    updated_at = NOW()
WHERE code = $1
`, gameHallJackpotCode, roundDBAmount(balance))
	if err != nil {
		return fmt.Errorf("update game jackpot: %w", err)
	}
	return nil
}

func resolveExchangeTxType(direction string) string {
	switch direction {
	case service.GameExchangeDGToBalance:
		return "exchange_out"
	default:
		return "exchange_in"
	}
}

func isRetryableGameHallTxErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "serialization") ||
		strings.Contains(message, "deadlock") ||
		strings.Contains(message, "could not serialize")
}

func roundDBAmount(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

func resolveActualDBMultiplier(payoutAmount float64, betAmount float64) float64 {
	if betAmount <= 0 {
		return 0
	}
	return roundDBAmount(payoutAmount / betAmount)
}

func resolveGameOutcomeFromNet(netAmount float64) string {
	switch {
	case netAmount > 0:
		return "win"
	case netAmount < 0:
		return "lose"
	default:
		return "draw"
	}
}
