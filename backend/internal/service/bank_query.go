package service

import (
	"context"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/transactionlog"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/shopspring/decimal"
)

var ErrBankInvalidBusinessModule = infraerrors.BadRequest("BANK_INVALID_BUSINESS_MODULE", "bank business module is invalid")

// BankTransactionListFilter 定义金融中心流水列表的只读筛选条件。
type BankTransactionListFilter struct {
	UserID         int64
	Type           string
	BusinessModule string
	Page           int
	PageSize       int
}

// BankTransactionView 是面向金融中心页面的只读流水快照。
type BankTransactionView struct {
	ID                  int64
	TxID                string
	UserID              int64
	AccountID           int64
	TxType              string
	BusinessModule      string
	Amount              decimal.Decimal
	BalanceBefore       decimal.Decimal
	BalanceAfter        decimal.Decimal
	FrozenBefore        decimal.Decimal
	FrozenAfter         decimal.Decimal
	CreditLimitSnapshot decimal.Decimal
	DebtSnapshot        decimal.Decimal
	Description         string
	ReferenceType       *string
	ReferenceID         *string
	RequestID           *string
	Metadata            map[string]any
	CreatedAt           time.Time
}

// ListTransactionViews 从 PostgreSQL 账本流水表读取当前用户的只读流水。
func (s *BankService) ListTransactionViews(ctx context.Context, filter BankTransactionListFilter) ([]BankTransactionView, int64, error) {
	if s == nil || s.client == nil {
		return nil, 0, ErrBankClientUnavailable
	}
	if filter.UserID <= 0 {
		return nil, 0, ErrBankInvalidUser
	}

	txType, module, err := normalizeBankTransactionListFilter(filter)
	if err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizeBankPagination(filter.Page, filter.PageSize)

	query := s.client.TransactionLog.Query().Where(transactionlog.UserIDEQ(filter.UserID))
	if txType != "" {
		query = query.Where(transactionlog.TxTypeEQ(txType))
	}
	if module != "" {
		query = query.Where(transactionlog.BusinessModuleEQ(module))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	logs, err := query.
		Order(dbent.Desc(transactionlog.FieldCreatedAt), dbent.Desc(transactionlog.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	views := make([]BankTransactionView, 0, len(logs))
	for _, log := range logs {
		views = append(views, bankTransactionViewFromEnt(log))
	}
	return views, int64(total), nil
}

func normalizeBankTransactionListFilter(filter BankTransactionListFilter) (string, string, error) {
	txType := strings.ToUpper(strings.TrimSpace(filter.Type))
	if txType != "" && !isSupportedBankTxType(txType) {
		return "", "", ErrBankInvalidType
	}

	module := strings.ToUpper(strings.TrimSpace(filter.BusinessModule))
	if module != "" && !isSupportedBankBusinessModule(module) {
		return "", "", ErrBankInvalidBusinessModule
	}
	return txType, module, nil
}

func normalizeBankPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func bankTransactionViewFromEnt(log *dbent.TransactionLog) BankTransactionView {
	if log == nil {
		return BankTransactionView{}
	}
	return BankTransactionView{
		ID:                  log.ID,
		TxID:                log.TxID.String(),
		UserID:              log.UserID,
		AccountID:           log.AccountID,
		TxType:              log.TxType,
		BusinessModule:      log.BusinessModule,
		Amount:              log.Amount,
		BalanceBefore:       log.BalanceBefore,
		BalanceAfter:        log.BalanceAfter,
		FrozenBefore:        log.FrozenBefore,
		FrozenAfter:         log.FrozenAfter,
		CreditLimitSnapshot: log.CreditLimitSnapshot,
		DebtSnapshot:        log.DebtSnapshot,
		Description:         log.Description,
		ReferenceType:       log.ReferenceType,
		ReferenceID:         log.ReferenceID,
		RequestID:           log.RequestID,
		Metadata:            log.Metadata,
		CreatedAt:           log.CreatedAt,
	}
}
