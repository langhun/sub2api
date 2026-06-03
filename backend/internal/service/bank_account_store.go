package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/transactionlog"
)

// findBankTransaction 通过幂等唯一键查找既有流水，命中时直接重放结果。
func findBankTransaction(ctx context.Context, client *dbent.Client, req TransferFundsRequest) (*TransferFundsResult, error) {
	log, err := client.TransactionLog.Query().
		Where(
			transactionlog.IdempotencyScopeEQ(req.IdempotencyScope),
			transactionlog.IdempotencyKeyHashEQ(HashIdempotencyKey(req.IdempotencyKey)),
		).
		Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query bank idempotency transaction: %w", err)
	}
	if !bankTransactionMatchesRequest(log, req) {
		return nil, ErrBankIdempotencyConflict
	}
	return &TransferFundsResult{
		TxID:        log.TxID,
		UserID:      log.UserID,
		AccountID:   log.AccountID,
		Type:        log.TxType,
		Module:      log.BusinessModule,
		Amount:      log.Amount,
		Balance:     log.BalanceAfter,
		Frozen:      log.FrozenAfter,
		TotalDebt:   log.DebtSnapshot,
		CreditLimit: log.CreditLimitSnapshot,
		Replayed:    true,
	}, nil
}

// bankTransactionMatchesRequest 防止同一个幂等键被不同金额或类型复用。
func bankTransactionMatchesRequest(log *dbent.TransactionLog, req TransferFundsRequest) bool {
	if log.UserID != req.UserID || log.TxType != req.Type {
		return false
	}
	if log.BusinessModule != req.BusinessModule {
		return false
	}
	if !log.Amount.Abs().Equal(req.Amount) || log.Description != req.Description {
		return false
	}
	if !bankOptionalStringMatches(req.ReferenceType, log.ReferenceType) {
		return false
	}
	if !bankOptionalStringMatches(req.ReferenceID, log.ReferenceID) {
		return false
	}
	return bankOptionalStringMatches(req.RequestID, log.RequestID)
}

// bankOptionalStringMatches 只在新请求显式传值时要求旧流水字段完全一致。
func bankOptionalStringMatches(want string, got *string) bool {
	return want == "" || (got != nil && *got == want)
}

// lockBankAccountForUpdate 以银行账户行为并发边界，避免高并发消费时双花。
func lockBankAccountForUpdate(ctx context.Context, client *dbent.Client, userID int64) (bankAccountSnapshot, error) {
	account, err := queryLockedBankAccount(ctx, client, userID)
	if !errors.Is(err, sql.ErrNoRows) {
		return account, err
	}
	if err := ensureBankAccount(ctx, client, userID); err != nil {
		return bankAccountSnapshot{}, err
	}
	account, err = queryLockedBankAccount(ctx, client, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return bankAccountSnapshot{}, ErrBankAccountNotFound
	}
	return account, err
}

// queryLockedBankAccount 使用 SELECT FOR UPDATE 获取悲观锁，事务提交前其他扣费会等待。
func queryLockedBankAccount(ctx context.Context, client *dbent.Client, userID int64) (bankAccountSnapshot, error) {
	rows, err := client.QueryContext(ctx, `
SELECT id, user_id, balance, frozen_amount, credit_limit, debt_principal, debt_interest, total_debt, version, status
FROM users_bank_account
WHERE user_id = $1
FOR UPDATE
`, userID)
	if err != nil {
		return bankAccountSnapshot{}, fmt.Errorf("lock bank account: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if rowsErr := rows.Err(); rowsErr != nil {
			return bankAccountSnapshot{}, fmt.Errorf("scan bank account lock: %w", rowsErr)
		}
		return bankAccountSnapshot{}, sql.ErrNoRows
	}
	var account bankAccountSnapshot
	if err := rows.Scan(&account.ID, &account.UserID, &account.Balance, &account.FrozenAmount,
		&account.CreditLimit, &account.DebtPrincipal, &account.DebtInterest,
		&account.TotalDebt, &account.Version, &account.Status); err != nil {
		return bankAccountSnapshot{}, fmt.Errorf("scan locked bank account: %w", err)
	}
	return account, rows.Err()
}

// ensureBankAccount 仅在旧用户缺少银行账户时读取 users.balance 初始化，不做任何余额扣改。
func ensureBankAccount(ctx context.Context, client *dbent.Client, userID int64) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO users_bank_account (user_id, balance, created_at, updated_at)
SELECT id, COALESCE(balance, 0), NOW(), NOW()
FROM users
WHERE id = $1 AND deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING
`, userID)
	if err != nil {
		return fmt.Errorf("ensure bank account: %w", err)
	}
	return nil
}

// updateBankAccount 只更新银行账户表，禁止直接修改 users.balance。
func updateBankAccount(ctx context.Context, client *dbent.Client, account bankAccountSnapshot, mutation bankMutation) error {
	_, err := client.UserBankAccount.UpdateOneID(account.ID).
		SetBalance(mutation.balanceAfter).
		SetFrozenAmount(mutation.frozenAfter).
		SetDebtPrincipal(mutation.debtPrincipalAfter).
		SetDebtInterest(mutation.debtInterestAfter).
		SetTotalDebt(mutation.debtAfter).
		AddVersion(1).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("update bank account: %w", err)
	}
	return nil
}

// createBankTransaction 在账户快照更新前先写入不可变流水，保证没有流水就不会变更余额。
func createBankTransaction(
	ctx context.Context,
	client *dbent.Client,
	req TransferFundsRequest,
	account bankAccountSnapshot,
	mutation bankMutation,
) (*dbent.TransactionLog, error) {
	create := client.TransactionLog.Create().
		SetUserID(req.UserID).
		SetAccountID(account.ID).
		SetTxType(req.Type).
		SetBusinessModule(req.BusinessModule).
		SetAmount(mutation.signedAmount).
		SetBalanceBefore(account.Balance).
		SetBalanceAfter(mutation.balanceAfter).
		SetFrozenBefore(account.FrozenAmount).
		SetFrozenAfter(mutation.frozenAfter).
		SetCreditLimitSnapshot(account.CreditLimit).
		SetDebtSnapshot(mutation.debtAfter).
		SetDescription(req.Description).
		SetIdempotencyScope(req.IdempotencyScope).
		SetIdempotencyKeyHash(HashIdempotencyKey(req.IdempotencyKey)).
		SetMetadata(req.Metadata)
	if req.ReferenceType != "" {
		create.SetReferenceType(req.ReferenceType)
	}
	if req.ReferenceID != "" {
		create.SetReferenceID(req.ReferenceID)
	}
	if req.RequestID != "" {
		create.SetRequestID(req.RequestID)
	}
	log, err := create.Save(ctx)
	if isUniqueBankTxErr(err) {
		return nil, ErrBankIdempotencyConflict.WithCause(err)
	}
	if err != nil {
		return nil, fmt.Errorf("create bank transaction log: %w", err)
	}
	return log, nil
}

func bankTransferResultFromMutation(
	req TransferFundsRequest,
	account bankAccountSnapshot,
	mutation bankMutation,
	log *dbent.TransactionLog,
) *TransferFundsResult {
	return &TransferFundsResult{
		TxID:          log.TxID,
		UserID:        req.UserID,
		AccountID:     account.ID,
		Type:          req.Type,
		Module:        req.BusinessModule,
		Amount:        mutation.signedAmount,
		Balance:       mutation.balanceAfter,
		Frozen:        mutation.frozenAfter,
		DebtPrincipal: mutation.debtPrincipalAfter,
		DebtInterest:  mutation.debtInterestAfter,
		TotalDebt:     mutation.debtAfter,
		CreditLimit:   account.CreditLimit,
	}
}
