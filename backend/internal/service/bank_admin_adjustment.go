package service

import (
	"context"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/transactionlog"
	"github.com/shopspring/decimal"
)

const (
	bankAdminBalanceAdjustmentScope     = "admin_balance_adjustment"
	bankAdminBalanceAdjustmentReference = "admin_balance_adjustment"
)

type AdminBalanceAdjustmentRequest struct {
	UserID         int64
	Operation      string
	Amount         decimal.Decimal
	Description    string
	IdempotencyKey string
	Metadata       map[string]any
}

type adminBalanceAdjuster interface {
	ApplyAdminBalanceAdjustment(ctx context.Context, req AdminBalanceAdjustmentRequest) (*TransferFundsResult, error)
}

// ApplyAdminBalanceAdjustment 在银行事务内完成后台余额调整，set 操作会在行锁后计算差额。
func (s *BankService) ApplyAdminBalanceAdjustment(ctx context.Context, req AdminBalanceAdjustmentRequest) (*TransferFundsResult, error) {
	if s == nil || s.client == nil {
		return nil, ErrBankClientUnavailable
	}
	normalized, err := normalizeAdminBalanceAdjustmentRequest(req)
	if err != nil {
		return nil, err
	}
	return s.runSerializableBankTx(ctx, func(txClient *dbent.Client) (*TransferFundsResult, error) {
		return s.applyAdminBalanceAdjustmentInTx(ctx, txClient, normalized)
	})
}

func (s *BankService) applyAdminBalanceAdjustmentInTx(
	ctx context.Context,
	client *dbent.Client,
	req AdminBalanceAdjustmentRequest,
) (*TransferFundsResult, error) {
	if existing, err := findAdminBalanceAdjustmentTransaction(ctx, client, req); err != nil || existing != nil {
		return existing, err
	}
	account, err := lockBankAccountForUpdate(ctx, client, req.UserID)
	if err != nil {
		return nil, err
	}
	if account.Status != BankAccountStatusActive {
		return nil, ErrBankAccountNotActive
	}
	if existing, err := findAdminBalanceAdjustmentTransaction(ctx, client, req); err != nil || existing != nil {
		return existing, err
	}

	txType, transferAmount, err := resolveAdminBalanceAdjustmentTransfer(account, req)
	if err != nil || transferAmount.IsZero() {
		return adminBalanceNoopResult(account), err
	}
	return s.transferFundsInTx(ctx, client, TransferFundsRequest{
		UserID:           req.UserID,
		Amount:           transferAmount,
		Type:             txType,
		BusinessModule:   BankBusinessModuleSystem,
		Description:      req.Description,
		IdempotencyScope: bankAdminBalanceAdjustmentScope,
		IdempotencyKey:   req.IdempotencyKey,
		ReferenceType:    bankAdminBalanceAdjustmentReference,
		ReferenceID:      adminBalanceAdjustmentReferenceID(req),
		RequestID:        req.IdempotencyKey,
		Metadata:         adminBalanceAdjustmentMetadata(req),
	})
}

func normalizeAdminBalanceAdjustmentRequest(req AdminBalanceAdjustmentRequest) (AdminBalanceAdjustmentRequest, error) {
	req.Operation = strings.ToLower(strings.TrimSpace(req.Operation))
	if req.UserID <= 0 {
		return req, ErrBankInvalidUser
	}
	if req.Operation != "set" && req.Operation != "add" && req.Operation != "subtract" {
		return req, ErrBankInvalidType
	}
	req.Amount = req.Amount.RoundBank(18)
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return req, ErrBankInvalidAmount
	}
	key, err := NormalizeIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return req, err
	}
	if key == "" {
		return req, ErrBankIdempotencyKeyRequired
	}
	req.IdempotencyKey = key
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		req.Description = fmt.Sprintf("后台手动调余额：%s", req.Operation)
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	return req, nil
}

func resolveAdminBalanceAdjustmentTransfer(
	account bankAccountSnapshot,
	req AdminBalanceAdjustmentRequest,
) (string, decimal.Decimal, error) {
	switch req.Operation {
	case "add":
		return BankTxTypeDeposit, req.Amount, nil
	case "subtract":
		return BankTxTypeWithdraw, req.Amount, nil
	case "set":
		delta := req.Amount.Sub(account.Balance)
		if delta.IsPositive() {
			return BankTxTypeDeposit, delta, nil
		}
		return BankTxTypeWithdraw, delta.Abs(), nil
	default:
		return "", decimal.Zero, ErrBankInvalidType
	}
}

func findAdminBalanceAdjustmentTransaction(
	ctx context.Context,
	client *dbent.Client,
	req AdminBalanceAdjustmentRequest,
) (*TransferFundsResult, error) {
	log, err := client.TransactionLog.Query().
		Where(
			transactionlog.IdempotencyScopeEQ(bankAdminBalanceAdjustmentScope),
			transactionlog.IdempotencyKeyHashEQ(HashIdempotencyKey(req.IdempotencyKey)),
		).
		Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query admin balance adjustment transaction: %w", err)
	}
	if !adminBalanceAdjustmentLogMatchesRequest(log, req) {
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

func adminBalanceAdjustmentLogMatchesRequest(log *dbent.TransactionLog, req AdminBalanceAdjustmentRequest) bool {
	if log.UserID != req.UserID || log.BusinessModule != BankBusinessModuleSystem {
		return false
	}
	if !adminBalanceAdjustmentAllowsTxType(req.Operation, log.TxType) {
		return false
	}
	if log.Description != req.Description {
		return false
	}
	if !bankOptionalStringMatches(bankAdminBalanceAdjustmentReference, log.ReferenceType) {
		return false
	}
	return bankOptionalStringMatches(adminBalanceAdjustmentReferenceID(req), log.ReferenceID)
}

func adminBalanceAdjustmentAllowsTxType(operation string, txType string) bool {
	switch operation {
	case "add":
		return txType == BankTxTypeDeposit
	case "subtract":
		return txType == BankTxTypeWithdraw
	case "set":
		return txType == BankTxTypeDeposit || txType == BankTxTypeWithdraw
	default:
		return false
	}
}

func adminBalanceAdjustmentReferenceID(req AdminBalanceAdjustmentRequest) string {
	return fmt.Sprintf("%d:%s:%s", req.UserID, req.Operation, req.Amount.String())
}

func adminBalanceAdjustmentMetadata(req AdminBalanceAdjustmentRequest) map[string]any {
	metadata := map[string]any{
		"operation":        req.Operation,
		"requested_amount": req.Amount.String(),
		"user_id":          req.UserID,
	}
	for key, value := range req.Metadata {
		metadata[key] = value
	}
	return metadata
}

func adminBalanceNoopResult(account bankAccountSnapshot) *TransferFundsResult {
	return &TransferFundsResult{
		UserID:        account.UserID,
		AccountID:     account.ID,
		Module:        BankBusinessModuleSystem,
		Amount:        decimal.Zero,
		Balance:       account.Balance,
		Frozen:        account.FrozenAmount,
		DebtPrincipal: account.DebtPrincipal,
		DebtInterest:  account.DebtInterest,
		TotalDebt:     account.TotalDebt,
		CreditLimit:   account.CreditLimit,
	}
}
