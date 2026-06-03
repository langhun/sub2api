package financialhubsdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

const (
	BusinessModuleFinancialHub = "FINANCIAL_HUB"
	BusinessModuleAPIGateway   = "API_GATEWAY"
	BusinessModulePayment      = "PAYMENT"
	BusinessModuleTransfer     = "TRANSFER"
	BusinessModuleGame         = "GAME"
	BusinessModuleLending      = "LENDING"
	BusinessModuleSystem       = "SYSTEM"
)

var ErrNotImplemented = errors.New("financial hub sdk adapter operation is not implemented")

// Client 定义仓库内业务模块统一接入 Financial Hub 的 SDK 契约。
type Client interface {
	GetAccount(ctx context.Context, userID int64) (*AccountSnapshot, error)
	ListTransactions(ctx context.Context, filter TransactionListFilter) ([]TransactionRecord, int64, error)
	ListAuditLogs(ctx context.Context, filter AuditLogListFilter) ([]AuditLogRecord, int64, error)
	ListReconciliationIssues(ctx context.Context, filter ReconciliationIssueListFilter) ([]ReconciliationIssueRecord, int64, error)
	Transfer(ctx context.Context, req TransferRequest) (*TransferResult, error)
	Deposit(ctx context.Context, req TransferRequest) (*TransferResult, error)
	Deduct(ctx context.Context, req TransferRequest) (*TransferResult, error)
	Freeze(ctx context.Context, req TransferRequest) (*TransferResult, error)
	Unfreeze(ctx context.Context, req TransferRequest) (*TransferResult, error)
	Reward(ctx context.Context, req TransferRequest) (*TransferResult, error)
	Loan(ctx context.Context, req TransferRequest) (*TransferResult, error)
	Repayment(ctx context.Context, req TransferRequest) (*TransferResult, error)
	TransferBatch(ctx context.Context, reqs []TransferRequest) ([]*TransferResult, error)
	Adjust(ctx context.Context, req AdminBalanceAdjustmentRequest) (*TransferResult, error)
	ApplyAdminBalanceAdjustment(ctx context.Context, req AdminBalanceAdjustmentRequest) (*TransferResult, error)
	CreateLoanContract(ctx context.Context, req LoanContractRequest) (*LoanContractRecord, error)
	CreateReversal(ctx context.Context, req ReversalRequest) (*ReversalRecord, error)
	StartReconciliation(ctx context.Context, req ReconciliationRunRequest) (*ReconciliationRunRecord, error)
	GetReconciliationRun(ctx context.Context, runID string) (*ReconciliationRunRecord, error)
}

type bankService interface {
	GetAccountView(ctx context.Context, userID int64) (*service.BankAccountView, error)
	ListTransactionViews(ctx context.Context, filter service.BankTransactionListFilter) ([]service.BankTransactionView, int64, error)
	TransferFunds(ctx context.Context, req service.TransferFundsRequest) (*service.TransferFundsResult, error)
	TransferFundsBatch(ctx context.Context, reqs []service.TransferFundsRequest) ([]*service.TransferFundsResult, error)
	ApplyAdminBalanceAdjustment(ctx context.Context, req service.AdminBalanceAdjustmentRequest) (*service.TransferFundsResult, error)
}

type AccountSnapshot struct {
	AccountID         int64
	Balance           string
	FrozenAmount      string
	CreditLimit       string
	DebtPrincipal     string
	DebtInterest      string
	TotalDebt         string
	AvailableCapacity string
	Status            string
	LegacyMissing     bool
}

type TransactionListFilter struct {
	UserID         int64
	Type           string
	BusinessModule string
	Page           int
	PageSize       int
}

type TransactionRecord struct {
	ID                  int64
	TxID                string
	UserID              int64
	AccountID           int64
	TxType              string
	BusinessModule      string
	Amount              string
	BalanceBefore       string
	BalanceAfter        string
	FrozenBefore        string
	FrozenAfter         string
	CreditLimitSnapshot string
	DebtSnapshot        string
	Description         string
	ReferenceType       *string
	ReferenceID         *string
	RequestID           *string
	Metadata            map[string]string
	CreatedAtUnix       int64
}

type TransferRequest struct {
	UserID           int64
	Amount           string
	Type             string
	BusinessModule   string
	Description      string
	IdempotencyScope string
	IdempotencyKey   string
	ReferenceType    string
	ReferenceID      string
	RequestID        string
	Metadata         map[string]string
}

type AuditLogListFilter struct {
	TxID       string
	TargetType string
	TargetID   string
	Page       int
	PageSize   int
}

type AuditLogRecord struct {
	ID            int64
	AuditID       string
	Action        string
	ActorType     string
	ActorUserID   int64
	RequestID     string
	TxID          string
	ReversalID    string
	TargetType    string
	TargetID      string
	Metadata      map[string]string
	CreatedAtUnix int64
}

type ReconciliationRunRequest struct {
	RunDate   string
	RequestID string
	Metadata  map[string]string
}

type ReconciliationRunRecord struct {
	ID                   int64
	RunID                string
	RunDate              string
	Status               string
	CheckedTransactions  int64
	CheckedLedgerEntries int64
	MismatchCount        int64
	Summary              map[string]string
	StartedAtUnix        int64
	FinishedAtUnix       int64
	CreatedAtUnix        int64
}

type ReconciliationIssueListFilter struct {
	RunID    string
	Page     int
	PageSize int
}

type ReconciliationIssueRecord struct {
	ID              int64
	IssueID         string
	IssueType       string
	TxID            string
	LedgerAccountID int64
	ExpectedAmount  string
	ActualAmount    string
	Detail          string
	Metadata        map[string]string
	CreatedAtUnix   int64
}

type ReversalRequest struct {
	OriginalTxID      string
	RequestedByUserID int64
	ApprovedByUserID  int64
	RequestID         string
	Reason            string
	Metadata          map[string]string
}

type ReversalRecord struct {
	ID                int64
	ReversalID        string
	OriginalTxID      string
	ReversalTxID      string
	RequestedByUserID int64
	ApprovedByUserID  int64
	RequestID         string
	Reason            string
	Status            string
	Metadata          map[string]string
	CreatedAtUnix     int64
	AppliedAtUnix     int64
}

type LoanContractRequest struct {
	BorrowerID   int64
	LenderType   string
	LenderID     int64
	Principal    string
	InterestRate string
	DueDateUnix  int64
	RequestID    string
	Metadata     map[string]string
}

type LoanContractRecord struct {
	ID              int64
	LoanID          string
	BorrowerID      int64
	LenderType      string
	LenderID        int64
	Principal       string
	InterestRate    string
	AccruedInterest string
	RepaidPrincipal string
	RepaidInterest  string
	Status          string
	DueDateUnix     int64
	CreatedAtUnix   int64
	UpdatedAtUnix   int64
}

type TransferResult struct {
	TxID           string
	UserID         int64
	AccountID      int64
	Type           string
	BusinessModule string
	Amount         string
	Balance        string
	Frozen         string
	DebtPrincipal  string
	DebtInterest   string
	TotalDebt      string
	CreditLimit    string
	Replayed       bool
}

type AdminBalanceAdjustmentRequest struct {
	UserID         int64
	Operation      string
	Amount         string
	Description    string
	IdempotencyKey string
	Metadata       map[string]string
}

type serviceClient struct {
	bank bankService
}

func New(bank *service.BankService) Client {
	return &serviceClient{bank: bank}
}

func (c *serviceClient) GetAccount(ctx context.Context, userID int64) (*AccountSnapshot, error) {
	view, err := c.bank.GetAccountView(ctx, userID)
	if err != nil {
		return nil, err
	}
	if view == nil {
		return nil, nil
	}
	return &AccountSnapshot{
		AccountID:         view.AccountID,
		Balance:           view.Balance.String(),
		FrozenAmount:      view.FrozenAmount.String(),
		CreditLimit:       view.CreditLimit.String(),
		DebtPrincipal:     view.DebtPrincipal.String(),
		DebtInterest:      view.DebtInterest.String(),
		TotalDebt:         view.TotalDebt.String(),
		AvailableCapacity: view.AvailableCapacity().String(),
		Status:            view.Status,
		LegacyMissing:     view.LegacyMissing,
	}, nil
}

func (c *serviceClient) ListTransactions(ctx context.Context, filter TransactionListFilter) ([]TransactionRecord, int64, error) {
	items, total, err := c.bank.ListTransactionViews(ctx, service.BankTransactionListFilter{
		UserID:         filter.UserID,
		Type:           filter.Type,
		BusinessModule: filter.BusinessModule,
		Page:           filter.Page,
		PageSize:       filter.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]TransactionRecord, 0, len(items))
	for _, item := range items {
		out = append(out, TransactionRecord{
			ID:                  item.ID,
			TxID:                item.TxID,
			UserID:              item.UserID,
			AccountID:           item.AccountID,
			TxType:              item.TxType,
			BusinessModule:      item.BusinessModule,
			Amount:              item.Amount.String(),
			BalanceBefore:       item.BalanceBefore.String(),
			BalanceAfter:        item.BalanceAfter.String(),
			FrozenBefore:        item.FrozenBefore.String(),
			FrozenAfter:         item.FrozenAfter.String(),
			CreditLimitSnapshot: item.CreditLimitSnapshot.String(),
			DebtSnapshot:        item.DebtSnapshot.String(),
			Description:         item.Description,
			ReferenceType:       item.ReferenceType,
			ReferenceID:         item.ReferenceID,
			RequestID:           item.RequestID,
			Metadata:            stringifyMetadata(item.Metadata),
			CreatedAtUnix:       item.CreatedAt.Unix(),
		})
	}
	return out, total, nil
}

func (c *serviceClient) ListAuditLogs(ctx context.Context, filter AuditLogListFilter) ([]AuditLogRecord, int64, error) {
	return nil, 0, fmt.Errorf("%w: list audit logs", ErrNotImplemented)
}

func (c *serviceClient) ListReconciliationIssues(ctx context.Context, filter ReconciliationIssueListFilter) ([]ReconciliationIssueRecord, int64, error) {
	return nil, 0, fmt.Errorf("%w: list reconciliation issues", ErrNotImplemented)
}

func (c *serviceClient) Transfer(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	serviceReq, err := toServiceTransferRequest(req)
	if err != nil {
		return nil, err
	}
	result, err := c.bank.TransferFunds(ctx, serviceReq)
	if err != nil {
		return nil, err
	}
	return fromServiceTransferResult(result), nil
}

func (c *serviceClient) Deposit(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	return c.transferWithType(ctx, req, service.BankTxTypeDeposit)
}

func (c *serviceClient) Deduct(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	return c.transferWithType(ctx, req, service.BankTxTypeWithdraw)
}

func (c *serviceClient) Freeze(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	return c.transferWithType(ctx, req, service.BankTxTypeFreeze)
}

func (c *serviceClient) Unfreeze(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	return c.transferWithType(ctx, req, service.BankTxTypeUnfreeze)
}

func (c *serviceClient) Reward(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	return c.transferWithType(ctx, req, service.BankTxTypeReward)
}

func (c *serviceClient) Loan(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	return c.transferWithType(ctx, req, service.BankTxTypeLoanBorrow)
}

func (c *serviceClient) Repayment(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	return c.transferWithType(ctx, req, service.BankTxTypeLoanRepay)
}

func (c *serviceClient) TransferBatch(ctx context.Context, reqs []TransferRequest) ([]*TransferResult, error) {
	serviceReqs := make([]service.TransferFundsRequest, 0, len(reqs))
	for _, req := range reqs {
		item, err := toServiceTransferRequest(req)
		if err != nil {
			return nil, err
		}
		serviceReqs = append(serviceReqs, item)
	}

	results, err := c.bank.TransferFundsBatch(ctx, serviceReqs)
	if err != nil {
		return nil, err
	}

	out := make([]*TransferResult, 0, len(results))
	for _, result := range results {
		out = append(out, fromServiceTransferResult(result))
	}
	return out, nil
}

func (c *serviceClient) Adjust(ctx context.Context, req AdminBalanceAdjustmentRequest) (*TransferResult, error) {
	return c.ApplyAdminBalanceAdjustment(ctx, req)
}

func (c *serviceClient) ApplyAdminBalanceAdjustment(ctx context.Context, req AdminBalanceAdjustmentRequest) (*TransferResult, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("parse adjustment amount: %w", err)
	}
	result, err := c.bank.ApplyAdminBalanceAdjustment(ctx, service.AdminBalanceAdjustmentRequest{
		UserID:         req.UserID,
		Operation:      req.Operation,
		Amount:         amount,
		Description:    req.Description,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       metadataToAny(req.Metadata),
	})
	if err != nil {
		return nil, err
	}
	return fromServiceTransferResult(result), nil
}

func (c *serviceClient) CreateLoanContract(ctx context.Context, req LoanContractRequest) (*LoanContractRecord, error) {
	return nil, fmt.Errorf("%w: create loan contract", ErrNotImplemented)
}

func (c *serviceClient) CreateReversal(ctx context.Context, req ReversalRequest) (*ReversalRecord, error) {
	return nil, fmt.Errorf("%w: create reversal", ErrNotImplemented)
}

func (c *serviceClient) StartReconciliation(ctx context.Context, req ReconciliationRunRequest) (*ReconciliationRunRecord, error) {
	return nil, fmt.Errorf("%w: start reconciliation", ErrNotImplemented)
}

func (c *serviceClient) GetReconciliationRun(ctx context.Context, runID string) (*ReconciliationRunRecord, error) {
	return nil, fmt.Errorf("%w: get reconciliation run", ErrNotImplemented)
}

func toServiceTransferRequest(req TransferRequest) (service.TransferFundsRequest, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return service.TransferFundsRequest{}, fmt.Errorf("parse transfer amount: %w", err)
	}
	return service.TransferFundsRequest{
		UserID:           req.UserID,
		Amount:           amount,
		Type:             req.Type,
		BusinessModule:   req.BusinessModule,
		Description:      req.Description,
		IdempotencyScope: req.IdempotencyScope,
		IdempotencyKey:   req.IdempotencyKey,
		ReferenceType:    req.ReferenceType,
		ReferenceID:      req.ReferenceID,
		RequestID:        req.RequestID,
		Metadata:         metadataToAny(req.Metadata),
	}, nil
}

func (c *serviceClient) transferWithType(ctx context.Context, req TransferRequest, txType string) (*TransferResult, error) {
	req.Type = txType
	return c.Transfer(ctx, req)
}

func fromServiceTransferResult(result *service.TransferFundsResult) *TransferResult {
	if result == nil {
		return nil
	}
	return &TransferResult{
		TxID:           result.TxID.String(),
		UserID:         result.UserID,
		AccountID:      result.AccountID,
		Type:           result.Type,
		BusinessModule: result.Module,
		Amount:         result.Amount.String(),
		Balance:        result.Balance.String(),
		Frozen:         result.Frozen.String(),
		DebtPrincipal:  result.DebtPrincipal.String(),
		DebtInterest:   result.DebtInterest.String(),
		TotalDebt:      result.TotalDebt.String(),
		CreditLimit:    result.CreditLimit.String(),
		Replayed:       result.Replayed,
	}
}

func metadataToAny(in map[string]string) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func stringifyMetadata(in map[string]any) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = fmt.Sprint(value)
	}
	return out
}
