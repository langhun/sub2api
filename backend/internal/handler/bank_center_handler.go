package handler

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// bankCenterService 定义金融中心只读页面需要的银行服务能力，方便 handler 单元测试替换。
type bankCenterService interface {
	GetAccountView(ctx context.Context, userID int64) (*service.BankAccountView, error)
	ListTransactionViews(ctx context.Context, filter service.BankTransactionListFilter) ([]service.BankTransactionView, int64, error)
}

// BankCenterHandler 提供用户侧金融中心只读接口。
type BankCenterHandler struct {
	bankService bankCenterService
}

// NewBankCenterHandler 创建金融中心 handler。
func NewBankCenterHandler(bankService *service.BankService) *BankCenterHandler {
	return &BankCenterHandler{bankService: bankService}
}

type bankAccountResponse struct {
	AccountID         int64  `json:"account_id"`
	Balance           string `json:"balance"`
	FrozenAmount      string `json:"frozen_amount"`
	CreditLimit       string `json:"credit_limit"`
	DebtPrincipal     string `json:"debt_principal"`
	DebtInterest      string `json:"debt_interest"`
	TotalDebt         string `json:"total_debt"`
	AvailableCapacity string `json:"available_capacity"`
	Status            string `json:"status"`
	LegacyMissing     bool   `json:"legacy_missing"`
}

type bankTransactionResponse struct {
	ID                  int64          `json:"id"`
	TxID                string         `json:"tx_id"`
	UserID              int64          `json:"user_id"`
	AccountID           int64          `json:"account_id"`
	TxType              string         `json:"tx_type"`
	BusinessModule      string         `json:"business_module"`
	Amount              string         `json:"amount"`
	BalanceBefore       string         `json:"balance_before"`
	BalanceAfter        string         `json:"balance_after"`
	FrozenBefore        string         `json:"frozen_before"`
	FrozenAfter         string         `json:"frozen_after"`
	CreditLimitSnapshot string         `json:"credit_limit_snapshot"`
	DebtSnapshot        string         `json:"debt_snapshot"`
	Description         string         `json:"description"`
	ReferenceType       *string        `json:"reference_type,omitempty"`
	ReferenceID         *string        `json:"reference_id,omitempty"`
	RequestID           *string        `json:"request_id,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
}

// GetAccount 返回当前登录用户的银行账户快照。
func (h *BankCenterHandler) GetAccount(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}

	account, err := h.bankService.GetAccountView(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, bankAccountResponseFromView(account))
}

// ListTransactions 返回当前登录用户的资金流水分页列表。
func (h *BankCenterHandler) ListTransactions(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}

	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}

	items, total, err := h.bankService.ListTransactionViews(c.Request.Context(), service.BankTransactionListFilter{
		UserID:         subject.UserID,
		Type:           c.Query("type"),
		BusinessModule: firstNonEmptyBankFilter(c.Query("business_module"), c.Query("module")),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]bankTransactionResponse, 0, len(items))
	for _, item := range items {
		out = append(out, bankTransactionResponseFromView(item))
	}
	response.Paginated(c, out, total, page, pageSize)
}

func bankAccountResponseFromView(view *service.BankAccountView) bankAccountResponse {
	if view == nil {
		return bankAccountResponse{}
	}
	return bankAccountResponse{
		AccountID:         view.AccountID,
		Balance:           decimalString(view.Balance),
		FrozenAmount:      decimalString(view.FrozenAmount),
		CreditLimit:       decimalString(view.CreditLimit),
		DebtPrincipal:     decimalString(view.DebtPrincipal),
		DebtInterest:      decimalString(view.DebtInterest),
		TotalDebt:         decimalString(view.TotalDebt),
		AvailableCapacity: decimalString(view.AvailableCapacity()),
		Status:            view.Status,
		LegacyMissing:     view.LegacyMissing,
	}
}

func bankTransactionResponseFromView(view service.BankTransactionView) bankTransactionResponse {
	return bankTransactionResponse{
		ID:                  view.ID,
		TxID:                view.TxID,
		UserID:              view.UserID,
		AccountID:           view.AccountID,
		TxType:              view.TxType,
		BusinessModule:      view.BusinessModule,
		Amount:              decimalString(view.Amount),
		BalanceBefore:       decimalString(view.BalanceBefore),
		BalanceAfter:        decimalString(view.BalanceAfter),
		FrozenBefore:        decimalString(view.FrozenBefore),
		FrozenAfter:         decimalString(view.FrozenAfter),
		CreditLimitSnapshot: decimalString(view.CreditLimitSnapshot),
		DebtSnapshot:        decimalString(view.DebtSnapshot),
		Description:         view.Description,
		ReferenceType:       view.ReferenceType,
		ReferenceID:         view.ReferenceID,
		RequestID:           view.RequestID,
		Metadata:            view.Metadata,
		CreatedAt:           view.CreatedAt,
	}
}

func decimalString(value decimal.Decimal) string {
	return value.String()
}

func firstNonEmptyBankFilter(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
