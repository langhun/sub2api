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
	BankTxTypeConsume      = "CONSUME"
	BankTxTypeDeposit      = "DEPOSIT"
	BankTxTypeWithdraw     = "WITHDRAW"
	BankTxTypeTransferOut  = "TRANSFER_OUT"
	BankTxTypeTransferIn   = "TRANSFER_IN"
	BankTxTypeSlotBet      = "SLOT_BET"
	BankTxTypeSlotWin      = "SLOT_WIN"
	BankTxTypeLotteryBet   = "LOTTERY_BET"
	BankTxTypeLotteryWin   = "LOTTERY_WIN"
	BankTxTypeLoanBorrow   = "LOAN_BORROW"
	BankTxTypeLoanRepay    = "LOAN_REPAY"
	BankTxTypeLoanInterest = "LOAN_INTEREST"
	BankTxTypeLendInvest   = "LEND_INVEST"
	BankTxTypeLendProfit   = "LEND_PROFIT"
	BankTxTypeReward       = "REWARD"
	BankTxTypeRefund       = "REFUND"
	BankTxTypeFreeze       = "FREEZE"
	BankTxTypeUnfreeze     = "UNFREEZE"

	BankBusinessModuleAPIGateway   = "API_GATEWAY"
	BankBusinessModulePayment      = "PAYMENT"
	BankBusinessModuleTransfer     = "TRANSFER"
	BankBusinessModuleGame         = "GAME"
	BankBusinessModuleLending      = "LENDING"
	BankBusinessModuleSystem       = "SYSTEM"
	BankBusinessModuleFinancialHub = "FINANCIAL_HUB"

	BankAccountStatusActive = "ACTIVE"
	BankAccountStatusFrozen = "FROZEN"
	BankAccountStatusClosed = "CLOSED"

	bankMinimumConsumeProbe = "0.000000000000000001"
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
	BusinessModule   string
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
	TxID          uuid.UUID
	UserID        int64
	AccountID     int64
	Type          string
	Module        string
	Amount        decimal.Decimal
	Balance       decimal.Decimal
	Frozen        decimal.Decimal
	DebtPrincipal decimal.Decimal
	DebtInterest  decimal.Decimal
	TotalDebt     decimal.Decimal
	CreditLimit   decimal.Decimal
	Replayed      bool
}

// BankAccountView 是认证和风控层读取的银行账户快照，不承担资金写入职责。
type BankAccountView struct {
	AccountID     int64
	Balance       decimal.Decimal
	FrozenAmount  decimal.Decimal
	CreditLimit   decimal.Decimal
	DebtPrincipal decimal.Decimal
	DebtInterest  decimal.Decimal
	TotalDebt     decimal.Decimal
	Status        string
	LegacyMissing bool
}

// AvailableCapacity 返回可用于 API 消费的容量：可用余额 + 未占用信用额度。
func (v *BankAccountView) AvailableCapacity() decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	cashAvailable := v.Balance
	if cashAvailable.IsNegative() {
		cashAvailable = decimal.Zero
	}
	creditAvailable := v.CreditLimit.Sub(effectiveBankDebt(v.DebtPrincipal, v.DebtInterest, v.TotalDebt))
	if creditAvailable.IsNegative() {
		creditAvailable = decimal.Zero
	}
	return cashAvailable.Add(creditAvailable)
}

// CanConsume 判断账户是否允许继续进入 API 扣费链路。
func (v *BankAccountView) CanConsume(minimum decimal.Decimal) bool {
	if v == nil || v.Status != BankAccountStatusActive {
		return false
	}
	if minimum.LessThanOrEqual(decimal.Zero) {
		minimum = decimal.RequireFromString(bankMinimumConsumeProbe)
	}
	return v.AvailableCapacity().GreaterThanOrEqual(minimum)
}

// bankAccountSnapshot 是行级锁拿到的账户状态，只在事务内流转。
type bankAccountSnapshot struct {
	ID            int64
	UserID        int64
	Balance       decimal.Decimal
	FrozenAmount  decimal.Decimal
	CreditLimit   decimal.Decimal
	DebtPrincipal decimal.Decimal
	DebtInterest  decimal.Decimal
	TotalDebt     decimal.Decimal
	Version       int64
	Status        string
}

// bankMutation 描述一次资金操作后的新快照，避免在计算阶段写数据库。
type bankMutation struct {
	signedAmount       decimal.Decimal
	balanceAfter       decimal.Decimal
	frozenAfter        decimal.Decimal
	debtPrincipalAfter decimal.Decimal
	debtInterestAfter  decimal.Decimal
	debtAfter          decimal.Decimal
}

// effectiveBankDebt 优先使用本金+利息作为真实负债，同时兼容只写入 total_debt 的旧快照。
func effectiveBankDebt(principal, interest, total decimal.Decimal) decimal.Decimal {
	combined := principal.Add(interest)
	if !combined.IsZero() || total.IsZero() {
		return combined
	}
	return total
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
	req.BusinessModule = normalizeBankBusinessModule(req.BusinessModule, req.Type)
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
	case BankTxTypeConsume, BankTxTypeDeposit, BankTxTypeWithdraw, BankTxTypeTransferOut,
		BankTxTypeTransferIn, BankTxTypeSlotBet, BankTxTypeSlotWin, BankTxTypeLotteryBet,
		BankTxTypeLotteryWin, BankTxTypeLoanBorrow,
		BankTxTypeLoanRepay, BankTxTypeLoanInterest, BankTxTypeLendInvest, BankTxTypeLendProfit,
		BankTxTypeReward, BankTxTypeRefund, BankTxTypeFreeze, BankTxTypeUnfreeze:
		return true
	default:
		return false
	}
}

// normalizeBankBusinessModule 将业务入口归一到统一模块名，便于账单大厅和总账审计筛选。
func normalizeBankBusinessModule(module string, txType string) string {
	module = strings.ToUpper(strings.TrimSpace(module))
	if isSupportedBankBusinessModule(module) {
		return module
	}
	switch txType {
	case BankTxTypeConsume:
		return BankBusinessModuleAPIGateway
	case BankTxTypeDeposit, BankTxTypeWithdraw, BankTxTypeRefund:
		return BankBusinessModulePayment
	case BankTxTypeTransferOut, BankTxTypeTransferIn:
		return BankBusinessModuleTransfer
	case BankTxTypeSlotBet, BankTxTypeSlotWin, BankTxTypeLotteryBet, BankTxTypeLotteryWin:
		return BankBusinessModuleGame
	case BankTxTypeLoanBorrow, BankTxTypeLoanRepay, BankTxTypeLoanInterest,
		BankTxTypeLendInvest, BankTxTypeLendProfit:
		return BankBusinessModuleLending
	case BankTxTypeReward:
		return BankBusinessModuleSystem
	default:
		return BankBusinessModuleFinancialHub
	}
}

// isSupportedBankBusinessModule 白名单化业务模块，避免任意字符串污染财务审计维度。
func isSupportedBankBusinessModule(module string) bool {
	switch module {
	case BankBusinessModuleAPIGateway, BankBusinessModulePayment, BankBusinessModuleTransfer,
		BankBusinessModuleGame, BankBusinessModuleLending, BankBusinessModuleSystem,
		BankBusinessModuleFinancialHub:
		return true
	default:
		return false
	}
}
