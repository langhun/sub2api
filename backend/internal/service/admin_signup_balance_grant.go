package service

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

func (s *adminServiceImpl) createUserWithInitialBalance(ctx context.Context, user *User, amount float64) error {
	if s == nil || user == nil {
		return ErrServiceUnavailable
	}
	if s.entClient == nil {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		user.Balance = amount
		return nil
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin admin create user transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.userRepo.Create(txCtx, user); err != nil {
		return err
	}
	user.Balance = 0
	if result, err := applyAdminCreateUserBalanceInTx(txCtx, tx.Client(), user.ID, amount); err != nil {
		return err
	} else if result != nil {
		user.Balance = result.Balance.InexactFloat64()
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit admin create user transaction: %w", err)
	}
	return nil
}

func applyAdminCreateUserBalanceInTx(ctx context.Context, client *dbent.Client, userID int64, amount float64) (*TransferFundsResult, error) {
	bankAmount := decimal.NewFromFloat(amount).RoundBank(18)
	if bankAmount.LessThanOrEqual(decimal.Zero) {
		return nil, nil
	}
	result, err := NewBankService(nil).ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           userID,
		Amount:           bankAmount,
		Type:             BankTxTypeReward,
		BusinessModule:   BankBusinessModuleSystem,
		Description:      "admin create user initial balance",
		IdempotencyScope: signupBalanceGrantScope,
		IdempotencyKey:   fmt.Sprintf("user-signup-balance-grant:%d:admin-create", userID),
		ReferenceType:    signupBalanceGrantReference,
		ReferenceID:      fmt.Sprintf("%d:admin-create", userID),
		Metadata: map[string]any{
			"grant_reason":  "admin_create",
			"signup_source": "admin_create",
			"user_id":       userID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("credit admin create initial balance through bank ledger: %w", err)
	}
	return result, nil
}
