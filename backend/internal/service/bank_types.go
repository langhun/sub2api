package service

import (
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 银行流水类型与 migration 中 transactions_log.tx_type 约束保持一致。
const (
	BankTxTypeConsume    = "CONSUME"
	BankTxTypeDeposit    = "DEPOSIT"
	BankTxTypeLoanBorrow = "LOAN_BORROW"
	BankTxTypeLoanRepay  = "LOAN_REPAY"
	BankTxTypeLendInvest = "LEND_INVEST"
	BankTxTypeLendProfit = "LEND_PROFIT"
	BankTxTypeFreeze     = "FREEZE"
	BankTxTypeUnfreeze   = "UNFREEZE"

	bankAccountStatusActive = "ACTIVE"
	bankTxMaxRetries        = 3
)

// 银行服务错误统一使用项目 ApplicationError，便于后续 Gin handler 直接映射 HTTP 响应。
var (
	ErrBankClientUnavailable       = infraerrors.InternalServer("BANK_CLIENT_UNAVAILABLE", "bank client is unavailable")
	ErrBankInvalidUser             = infraerrors.BadRequest("BANK_INVALID_USER", "bank user id is invalid")
	ErrBankInvalidAmount           = infraerrors.BadRequest("BANK_INVALID_AMOUNT", "bank amount must be greater than zero")
	ErrBankInvalidType             = infraerrors.BadRequest("BANK_INVALID_TYPE", "bank transaction type is invalid")
	ErrBankAccountNotFound         = infraerrors.NotFound("BANK_ACCOUNT_NOT_FOUND", "bank account not found")
	ErrBankAccountNotActive        = infraerrors.Forbidden("BANK_ACCOUNT_NOT_ACTIVE", "bank account is not active")
	ErrBankInsufficientFunds       = infraerrors.BadRequest("BANK_INSUFFICIENT_FUNDS", "bank account has insufficient funds")
	ErrBankCreditLimitExceeded     = infraerrors.BadRequest("BANK_CREDIT_LIMIT_EXCEEDED", "bank credit limit exceeded")
	ErrBankIdempotencyKeyRequired  = infraerrors.BadRequest("BANK_IDEMPOTENCY_KEY_REQUIRED", "bank idempotency key is required")
	ErrBankIdempotencyScopeInvalid = infraerrors.BadRequest("BANK_IDEMPOTENCY_SCOPE_INVALID", "bank idempotency scope is invalid")
	ErrBankIdempotencyConflict     = infraerrors.Conflict("BANK_IDEMPOTENCY_CONFLICT", "bank idempotency key conflicts")
)

// TransferFundsRequest 是所有资金变动的入口参数，调用方必须提供幂等键。
type TransferFundsRequest struct {
	UserID           int64
	Amount           decimal.Decimal
	Type             string
	Description      string
	IdempotencyScope string
	IdempotencyKey   string
	ReferenceType    string
	ReferenceID      string
	RequestID        string
	Metadata         map[string]any
}

// TransferFundsResult 返回本次账本写入后的账户快照。
type TransferFundsResult struct {
	TxID        uuid.UUID
	UserID      int64
	AccountID   int64
	Type        string
	Amount      decimal.Decimal
	Balance     decimal.Decimal
	Frozen      decimal.Decimal
	TotalDebt   decimal.Decimal
	CreditLimit decimal.Decimal
	Replayed    bool
}

// bankAccountSnapshot 是行级锁拿到的账户状态，只在事务内流转。
type bankAccountSnapshot struct {
	ID           int64
	UserID       int64
	Balance      decimal.Decimal
	FrozenAmount decimal.Decimal
	CreditLimit  decimal.Decimal
	TotalDebt    decimal.Decimal
	Version      int64
	Status       string
}

// bankMutation 描述一次资金操作后的新快照，避免在计算阶段写数据库。
type bankMutation struct {
	signedAmount decimal.Decimal
	balanceAfter decimal.Decimal
	frozenAfter  decimal.Decimal
	debtAfter    decimal.Decimal
}

// normalizeTransferFundsRequest 在入库前统一金额精度、流水类型和幂等参数。
func normalizeTransferFundsRequest(req TransferFundsRequest) (TransferFundsRequest, error) {
	req.Type = strings.ToUpper(strings.TrimSpace(req.Type))
	if req.UserID <= 0 {
		return req, ErrBankInvalidUser
	}
	if !isSupportedBankTxType(req.Type) {
		return req, ErrBankInvalidType
	}
	amount := req.Amount.RoundBank(18)
	if amount.LessThanOrEqual(decimal.Zero) {
		return req, ErrBankInvalidAmount
	}
	key, err := NormalizeIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return req, err
	}
	if key == "" {
		return req, ErrBankIdempotencyKeyRequired
	}
	scope, err := normalizeBankIdempotencyScope(req.IdempotencyScope, req.UserID)
	if err != nil {
		return req, err
	}
	req.Amount = amount
	req.IdempotencyKey = key
	req.IdempotencyScope = scope
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	return req, nil
}

// normalizeBankIdempotencyScope 默认按用户隔离幂等作用域，避免不同用户的同名请求互相冲突。
func normalizeBankIdempotencyScope(scope string, userID int64) (string, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fmt.Sprintf("bank:transfer:user:%d", userID), nil
	}
	if len(scope) > 128 {
		return "", ErrBankIdempotencyScopeInvalid
	}
	return scope, nil
}

// isSupportedBankTxType 白名单化流水类型，防止任意字符串进入财务日志。
func isSupportedBankTxType(txType string) bool {
	switch txType {
	case BankTxTypeConsume, BankTxTypeDeposit, BankTxTypeLoanBorrow, BankTxTypeLoanRepay,
		BankTxTypeLendInvest, BankTxTypeLendProfit, BankTxTypeFreeze, BankTxTypeUnfreeze:
		return true
	default:
		return false
	}
}
