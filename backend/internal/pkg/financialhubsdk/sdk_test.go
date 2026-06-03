package financialhubsdk

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type bankServiceStub struct {
	account      *service.BankAccountView
	accountErr   error
	transactions []service.BankTransactionView
	total        int64
	txErr        error
	transferReq  service.TransferFundsRequest
	transferResp *service.TransferFundsResult
	transferErr  error
	adjustReq    service.AdminBalanceAdjustmentRequest
	adjustResp   *service.TransferFundsResult
	adjustErr    error
}

func (s *bankServiceStub) GetAccountView(ctx context.Context, userID int64) (*service.BankAccountView, error) {
	return s.account, s.accountErr
}

func (s *bankServiceStub) ListTransactionViews(ctx context.Context, filter service.BankTransactionListFilter) ([]service.BankTransactionView, int64, error) {
	return s.transactions, s.total, s.txErr
}

func (s *bankServiceStub) TransferFunds(ctx context.Context, req service.TransferFundsRequest) (*service.TransferFundsResult, error) {
	s.transferReq = req
	return s.transferResp, s.transferErr
}

func (s *bankServiceStub) TransferFundsBatch(ctx context.Context, reqs []service.TransferFundsRequest) ([]*service.TransferFundsResult, error) {
	results := make([]*service.TransferFundsResult, 0, len(reqs))
	for range reqs {
		results = append(results, s.transferResp)
	}
	return results, s.transferErr
}

func (s *bankServiceStub) ApplyAdminBalanceAdjustment(ctx context.Context, req service.AdminBalanceAdjustmentRequest) (*service.TransferFundsResult, error) {
	s.adjustReq = req
	return s.adjustResp, s.adjustErr
}

func TestGetAccount(t *testing.T) {
	t.Parallel()

	stub := &bankServiceStub{
		account: &service.BankAccountView{
			AccountID:     11,
			Balance:       decimal.RequireFromString("12.500000"),
			FrozenAmount:  decimal.RequireFromString("3.000000"),
			CreditLimit:   decimal.RequireFromString("50"),
			DebtPrincipal: decimal.RequireFromString("10"),
			DebtInterest:  decimal.RequireFromString("2"),
			TotalDebt:     decimal.RequireFromString("12"),
			Status:        service.BankAccountStatusActive,
			LegacyMissing: true,
		},
	}

	client := &serviceClient{bank: stub}
	got, err := client.GetAccount(context.Background(), 99)
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("GetAccount returned nil snapshot")
	}
	if got.AccountID != 11 || got.Balance != "12.5" || got.AvailableCapacity != "50.5" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestTransferMapsRequestAndResponse(t *testing.T) {
	t.Parallel()

	txID := uuid.New()
	stub := &bankServiceStub{
		transferResp: &service.TransferFundsResult{
			TxID:          txID,
			UserID:        7,
			AccountID:     9,
			Type:          service.BankTxTypeConsume,
			Module:        service.BankBusinessModuleAPIGateway,
			Amount:        decimal.RequireFromString("-1.250000"),
			Balance:       decimal.RequireFromString("8.750000"),
			Frozen:        decimal.Zero,
			DebtPrincipal: decimal.Zero,
			DebtInterest:  decimal.Zero,
			TotalDebt:     decimal.Zero,
			CreditLimit:   decimal.RequireFromString("100"),
		},
	}

	client := &serviceClient{bank: stub}
	got, err := client.Transfer(context.Background(), TransferRequest{
		UserID:         7,
		Amount:         "1.25",
		Type:           service.BankTxTypeConsume,
		BusinessModule: service.BankBusinessModuleAPIGateway,
		Description:    "api billing",
		IdempotencyKey: "req-1",
		Metadata:       map[string]string{"source": "usage"},
	})
	if err != nil {
		t.Fatalf("Transfer returned error: %v", err)
	}
	if stub.transferReq.Amount.String() != "1.25" || stub.transferReq.UserID != 7 {
		t.Fatalf("unexpected mapped request: %+v", stub.transferReq)
	}
	if got == nil || got.TxID != txID.String() || got.Balance != "8.75" {
		t.Fatalf("unexpected mapped response: %+v", got)
	}
}

func TestListTransactionsMapsRows(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000000, 0)
	stub := &bankServiceStub{
		transactions: []service.BankTransactionView{
			{
				ID:                  1,
				TxID:                uuid.NewString(),
				UserID:              10,
				AccountID:           20,
				TxType:              service.BankTxTypeReward,
				BusinessModule:      service.BankBusinessModuleSystem,
				Amount:              decimal.RequireFromString("5"),
				BalanceBefore:       decimal.RequireFromString("10"),
				BalanceAfter:        decimal.RequireFromString("15"),
				FrozenBefore:        decimal.Zero,
				FrozenAfter:         decimal.Zero,
				CreditLimitSnapshot: decimal.RequireFromString("30"),
				DebtSnapshot:        decimal.Zero,
				Description:         "reward",
				Metadata:            map[string]any{"campaign": "welcome"},
				CreatedAt:           now,
			},
		},
		total: 1,
	}

	client := &serviceClient{bank: stub}
	items, total, err := client.ListTransactions(context.Background(), TransactionListFilter{UserID: 10, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListTransactions returned error: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected pagination: total=%d len=%d", total, len(items))
	}
	if items[0].Metadata["campaign"] != "welcome" || items[0].CreatedAtUnix != now.Unix() {
		t.Fatalf("unexpected transaction record: %+v", items[0])
	}
}

func TestApplyAdminBalanceAdjustmentMapsAmount(t *testing.T) {
	t.Parallel()

	stub := &bankServiceStub{
		adjustResp: &service.TransferFundsResult{
			TxID:        uuid.New(),
			UserID:      5,
			AccountID:   8,
			Type:        service.BankTxTypeDeposit,
			Module:      service.BankBusinessModuleSystem,
			Amount:      decimal.RequireFromString("9"),
			Balance:     decimal.RequireFromString("19"),
			Frozen:      decimal.Zero,
			TotalDebt:   decimal.Zero,
			CreditLimit: decimal.RequireFromString("50"),
		},
	}

	client := &serviceClient{bank: stub}
	_, err := client.ApplyAdminBalanceAdjustment(context.Background(), AdminBalanceAdjustmentRequest{
		UserID:         5,
		Operation:      "add",
		Amount:         "9",
		Description:    "manual adjustment",
		IdempotencyKey: "adj-1",
	})
	if err != nil {
		t.Fatalf("ApplyAdminBalanceAdjustment returned error: %v", err)
	}
	if stub.adjustReq.Amount.String() != "9" || stub.adjustReq.Operation != "add" {
		t.Fatalf("unexpected adjustment request: %+v", stub.adjustReq)
	}
}
