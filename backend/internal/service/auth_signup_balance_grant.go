package service

import (
	"context"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

const (
	signupBalanceGrantScope     = "user_signup_balance_grant"
	signupBalanceGrantReference = "user_signup_balance_grant"
)

func (s *AuthService) createUserWithSignupBalanceGrant(ctx context.Context, user *User, signupSource string, balance float64) error {
	if s == nil || user == nil {
		return ErrServiceUnavailable
	}
	if s.entClient == nil {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		user.Balance = balance
		return nil
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin signup balance grant transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.userRepo.Create(txCtx, user); err != nil {
		return err
	}
	user.Balance = 0
	if result, err := s.applySignupBalanceGrantInTx(txCtx, tx.Client(), user.ID, signupSource, balance); err != nil {
		return err
	} else if result != nil {
		user.Balance = result.Balance.InexactFloat64()
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit signup balance grant transaction: %w", err)
	}
	return nil
}

func (s *AuthService) applySignupBalanceGrant(ctx context.Context, userID int64, signupSource string, amount float64) (*TransferFundsResult, error) {
	if s == nil || s.entClient == nil {
		return nil, nil
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return s.applySignupBalanceGrantInTx(ctx, tx.Client(), userID, signupSource, amount)
	}
	return s.applySignupBalanceGrantStandalone(ctx, userID, signupSource, amount)
}

func (s *AuthService) applySignupBalanceGrantStandalone(ctx context.Context, userID int64, signupSource string, amount float64) (*TransferFundsResult, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin signup balance grant transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	result, err := s.applySignupBalanceGrantInTx(txCtx, tx.Client(), userID, signupSource, amount)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit signup balance grant transaction: %w", err)
	}
	return result, nil
}

func (s *AuthService) applySignupBalanceGrantInTx(ctx context.Context, client *dbent.Client, userID int64, signupSource string, amount float64) (*TransferFundsResult, error) {
	bankAmount := decimal.NewFromFloat(amount).RoundBank(18)
	if bankAmount.LessThanOrEqual(decimal.Zero) {
		return nil, nil
	}
	normalizedSource := normalizeOAuthSignupSource(signupSource)
	result, err := NewBankService(s.entClient).ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           userID,
		Amount:           bankAmount,
		Type:             BankTxTypeReward,
		BusinessModule:   BankBusinessModuleSystem,
		Description:      signupBalanceGrantDescription(normalizedSource),
		IdempotencyScope: signupBalanceGrantScope,
		IdempotencyKey:   fmt.Sprintf("user-signup-balance-grant:%d:%s", userID, normalizedSource),
		ReferenceType:    signupBalanceGrantReference,
		ReferenceID:      fmt.Sprintf("%d:%s", userID, normalizedSource),
		Metadata: map[string]any{
			"grant_reason":  "signup",
			"signup_source": normalizedSource,
			"user_id":       userID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("credit signup balance through bank ledger: %w", err)
	}
	return result, nil
}

func signupBalanceGrantDescription(signupSource string) string {
	source := strings.TrimSpace(signupSource)
	if source == "" {
		source = "email"
	}
	return fmt.Sprintf("signup default balance grant: %s", source)
}
