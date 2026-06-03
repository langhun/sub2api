package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	ErrLotteryJackpotNotFound      = infraerrors.NotFound("LOTTERY_JACKPOT_NOT_FOUND", "lottery jackpot not found")
	ErrLotteryJackpotInsufficient  = infraerrors.Conflict("LOTTERY_JACKPOT_INSUFFICIENT", "lottery jackpot balance is insufficient")
	ErrLotteryJackpotAmountInvalid = ErrBankInvalidAmount
)

type lotterySQLClient interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type JackpotService struct {
	client *dbent.Client
}

func NewJackpotService(client *dbent.Client) *JackpotService {
	return &JackpotService{client: client}
}

func (s *JackpotService) Deposit(ctx context.Context, lotteryType string, amount decimal.Decimal) error {
	if s == nil || s.client == nil {
		return ErrLotteryJackpotUnavailable
	}
	return s.depositInTx(ctx, s.client, lotteryType, amount)
}

func (s *JackpotService) Withdraw(ctx context.Context, lotteryType string, amount decimal.Decimal) error {
	if s == nil || s.client == nil {
		return ErrLotteryJackpotUnavailable
	}
	return s.withdrawInTx(ctx, s.client, lotteryType, amount)
}

func (s *JackpotService) GetBalance(ctx context.Context, lotteryType string) (decimal.Decimal, error) {
	if s == nil || s.client == nil {
		return decimal.Zero, ErrLotteryJackpotUnavailable
	}
	return s.getBalanceInTx(ctx, s.client, lotteryType)
}

func (s *JackpotService) depositInTx(ctx context.Context, client lotterySQLClient, lotteryType string, amount decimal.Decimal) error {
	return s.adjustUpInTx(ctx, client, lotteryType, normalizeLotteryJackpotAmount(amount))
}

func (s *JackpotService) withdrawInTx(ctx context.Context, client lotterySQLClient, lotteryType string, amount decimal.Decimal) error {
	normalizedType, normalizedAmount, err := normalizeLotteryJackpotChange(lotteryType, amount)
	if err != nil {
		return err
	}
	result, err := client.ExecContext(ctx, `
UPDATE lottery_jackpot
SET balance = balance - $2,
    updated_at = NOW()
WHERE lottery_type = $1
  AND balance >= $2
`, normalizedType, normalizedAmount)
	if err != nil {
		return fmt.Errorf("withdraw jackpot: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("withdraw jackpot rows affected: %w", err)
	}
	if affected == 1 {
		return nil
	}

	balance, err := s.getBalanceInTx(ctx, client, normalizedType)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLotteryJackpotNotFound
	}
	if err != nil {
		return err
	}
	if balance.LessThan(normalizedAmount) {
		return ErrLotteryJackpotInsufficient
	}
	return ErrLotteryJackpotUnavailable
}

func (s *JackpotService) getBalanceInTx(ctx context.Context, client lotterySQLClient, lotteryType string) (decimal.Decimal, error) {
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return decimal.Zero, err
	}
	rows, err := client.QueryContext(ctx, `
SELECT balance
FROM lottery_jackpot
WHERE lottery_type = $1
`, normalizedType)
	if err != nil {
		return decimal.Zero, fmt.Errorf("query jackpot balance: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return decimal.Zero, fmt.Errorf("scan jackpot balance: %w", err)
		}
		return decimal.Zero, sql.ErrNoRows
	}
	var balance decimal.Decimal
	if err := rows.Scan(&balance); err != nil {
		return decimal.Zero, fmt.Errorf("scan jackpot balance value: %w", err)
	}
	if err := rows.Err(); err != nil {
		return decimal.Zero, fmt.Errorf("scan jackpot balance rows: %w", err)
	}
	return balance, nil
}

func (s *JackpotService) adjustUpInTx(ctx context.Context, client lotterySQLClient, lotteryType string, amount decimal.Decimal) error {
	normalizedType, normalizedAmount, err := normalizeLotteryJackpotChange(lotteryType, amount)
	if err != nil {
		return err
	}
	if _, err := client.ExecContext(ctx, `
INSERT INTO lottery_jackpot (lottery_type, balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (lottery_type) DO UPDATE
SET balance = lottery_jackpot.balance + EXCLUDED.balance,
    updated_at = NOW()
`, normalizedType, normalizedAmount); err != nil {
		return fmt.Errorf("deposit jackpot: %w", err)
	}
	return nil
}

func normalizeLotteryJackpotChange(lotteryType string, amount decimal.Decimal) (string, decimal.Decimal, error) {
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return "", decimal.Zero, err
	}
	normalizedAmount := normalizeLotteryJackpotAmount(amount)
	if normalizedAmount.LessThanOrEqual(decimal.Zero) {
		return "", decimal.Zero, ErrLotteryJackpotAmountInvalid
	}
	return normalizedType, normalizedAmount, nil
}

func normalizeLotteryJackpotAmount(amount decimal.Decimal) decimal.Decimal {
	return amount.RoundBank(18)
}
