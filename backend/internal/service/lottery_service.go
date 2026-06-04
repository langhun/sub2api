package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/shopspring/decimal"
)

type LotteryJackpotStore interface {
	Deposit(ctx context.Context, lotteryType string, amount decimal.Decimal) error
	Withdraw(ctx context.Context, lotteryType string, amount decimal.Decimal) error
	GetBalance(ctx context.Context, lotteryType string) (decimal.Decimal, error)
	depositInTx(ctx context.Context, client lotterySQLClient, lotteryType string, amount decimal.Decimal) error
	withdrawInTx(ctx context.Context, client lotterySQLClient, lotteryType string, amount decimal.Decimal) error
}

type lotteryBankApplier interface {
	ApplyTransferInTx(ctx context.Context, client *dbent.Client, req TransferFundsRequest) (*TransferFundsResult, error)
}

type LotteryBetInput struct {
	UserID         int64    `json:"user_id"`
	LotteryType    string   `json:"lottery_type"`
	IssueNo        string   `json:"issue_no"`
	RedBalls       []string `json:"red_balls"`
	BlueBall       string   `json:"blue_ball"`
	BetCount       int      `json:"bet_count"`
	IdempotencyKey string   `json:"idempotency_key"`
	RequestID      string   `json:"request_id"`
}

type LotteryBetResult struct {
	LotteryType string          `json:"lottery_type"`
	IssueNo     string          `json:"issue_no"`
	OrderIDs    []int64         `json:"order_ids"`
	Cost        decimal.Decimal `json:"cost"`
	Status      string          `json:"status"`
	Replayed    bool            `json:"replayed"`
}

type LotteryOrderQuery struct {
	UserID      int64  `json:"user_id"`
	LotteryType string `json:"lottery_type"`
	IssueNo     string `json:"issue_no"`
}

type LotteryResultQuery struct {
	LotteryType string `json:"lottery_type"`
	IssueNo     string `json:"issue_no"`
	Limit       int    `json:"limit"`
}

type LotteryOrderView struct {
	ID          int64           `json:"id"`
	LotteryType string          `json:"lottery_type"`
	IssueNo     string          `json:"issue_no"`
	RedBalls    []string        `json:"red_balls"`
	BlueBall    string          `json:"blue_ball"`
	Cost        decimal.Decimal `json:"cost"`
	Reward      decimal.Decimal `json:"reward"`
	PrizeLevel  string          `json:"prize_level"`
	RedHits     int             `json:"red_hits"`
	BlueHit     bool            `json:"blue_hit"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
}

type LotteryResultView struct {
	ID          int64     `json:"id"`
	LotteryType string    `json:"lottery_type"`
	IssueNo     string    `json:"issue_no"`
	RedBalls    []string  `json:"red_balls"`
	BlueBall    string    `json:"blue_ball"`
	OpenedAt    time.Time `json:"opened_at"`
	Source      string    `json:"source"`
	SourceRef   string    `json:"source_ref"`
	CreatedAt   time.Time `json:"created_at"`
}

type LotterySyncResult struct {
	LotteryType string            `json:"lottery_type"`
	IssueNo     string            `json:"issue_no"`
	Result      LotteryResultView `json:"result"`
	Replayed    bool              `json:"replayed"`
}

type LotterySettlementResult struct {
	LotteryType string          `json:"lottery_type"`
	IssueNo     string          `json:"issue_no"`
	TotalOrders int             `json:"total_orders"`
	WinOrders   int             `json:"win_orders"`
	LoseOrders  int             `json:"lose_orders"`
	RewardTotal decimal.Decimal `json:"reward_total"`
	Replayed    bool            `json:"replayed"`
}

type LotteryOpenedSettlementResult struct {
	LotteryType   string                    `json:"lottery_type"`
	TotalIssues   int                       `json:"total_issues"`
	SettledIssues []LotterySettlementResult `json:"settled_issues"`
	FailedIssues  []string                  `json:"failed_issues"`
}

type LotteryJobRunResult struct {
	LotteryType       string                         `json:"lottery_type"`
	SyncedIssueNo     string                         `json:"synced_issue_no"`
	SyncReplayed      bool                           `json:"sync_replayed"`
	SettlementSummary *LotteryOpenedSettlementResult `json:"settlement_summary"`
}

type LotteryJackpotView struct {
	LotteryType string          `json:"lottery_type"`
	Balance     decimal.Decimal `json:"balance"`
}

type LotteryService struct {
	entClient            *dbent.Client
	settingRepo          SettingRepository
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	jackpotService       LotteryJackpotStore
	providers            map[string]LotteryProvider
	bankApplier          lotteryBankApplier
}

func NewLotteryService(
	entClient *dbent.Client,
	settingRepo SettingRepository,
	billingCacheService *BillingCacheService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	jackpotService LotteryJackpotStore,
	providers map[string]LotteryProvider,
) *LotteryService {
	return &LotteryService{
		entClient:            entClient,
		settingRepo:          settingRepo,
		billingCacheService:  billingCacheService,
		authCacheInvalidator: authCacheInvalidator,
		jackpotService:       jackpotService,
		providers:            copyLotteryProviders(providers),
		bankApplier:          NewBankService(entClient),
	}
}

func DefaultLotteryProviders() (map[string]LotteryProvider, error) {
	ssqProvider, err := NewSSQProvider()
	if err != nil {
		return nil, err
	}
	return map[string]LotteryProvider{
		LotteryTypeSSQ: ssqProvider,
	}, nil
}

func (s *LotteryService) CreateBet(ctx context.Context, input LotteryBetInput) (*LotteryBetResult, error) {
	return s.createBet(ctx, input)
}

func (s *LotteryService) GetCurrentIssue(ctx context.Context, lotteryType string) (*Issue, error) {
	if strings.TrimSpace(lotteryType) == "" {
		lotteryType = LotteryTypeSSQ
	}
	provider, err := s.providerByType(lotteryType)
	if err != nil {
		return nil, err
	}
	issue, err := provider.GetCurrentIssue(ctx)
	if err != nil {
		return nil, err
	}
	return decorateLotteryIssue(issue, timezone.Now()), nil
}

func (s *LotteryService) GetMyOrders(ctx context.Context, query LotteryOrderQuery) ([]LotteryOrderView, error) {
	return s.getMyOrders(ctx, query)
}

func (s *LotteryService) GetJackpot(ctx context.Context, lotteryType string) (*LotteryJackpotView, error) {
	if s == nil || s.jackpotService == nil {
		return nil, ErrLotteryJackpotUnavailable
	}
	if strings.TrimSpace(lotteryType) == "" {
		lotteryType = LotteryTypeSSQ
	}
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return nil, err
	}
	balance, err := s.jackpotService.GetBalance(ctx, normalizedType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLotteryJackpotNotFound
	}
	if err != nil {
		return nil, err
	}
	return &LotteryJackpotView{
		LotteryType: normalizedType,
		Balance:     balance,
	}, nil
}

func (s *LotteryService) SyncLatestResult(ctx context.Context, lotteryType string) (*LotterySyncResult, error) {
	return s.syncLatestResult(ctx, lotteryType)
}

func (s *LotteryService) GetResults(ctx context.Context, query LotteryResultQuery) ([]LotteryResultView, error) {
	return s.getResults(ctx, query)
}

func (s *LotteryService) GetResult(ctx context.Context, lotteryType, issueNo string) (*LotteryResultView, error) {
	return s.getResult(ctx, lotteryType, issueNo)
}

func (s *LotteryService) SettleIssue(ctx context.Context, lotteryType, issueNo string) (*LotterySettlementResult, error) {
	return s.settleIssue(ctx, lotteryType, issueNo)
}

func (s *LotteryService) SettleOpenedIssues(ctx context.Context, lotteryType string, limit int) (*LotteryOpenedSettlementResult, error) {
	return s.settleOpenedIssues(ctx, lotteryType, limit)
}

func (s *LotteryService) providerByType(lotteryType string) (LotteryProvider, error) {
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ErrLotteryProviderNotFound
	}
	provider, ok := s.providers[normalizedType]
	if !ok || provider == nil {
		return nil, ErrLotteryProviderNotFound.WithMetadata(map[string]string{
			"lottery_type": normalizedType,
		})
	}
	return provider, nil
}

func copyLotteryProviders(providers map[string]LotteryProvider) map[string]LotteryProvider {
	if len(providers) == 0 {
		return map[string]LotteryProvider{}
	}
	cloned := make(map[string]LotteryProvider, len(providers))
	for lotteryType, provider := range providers {
		normalizedType := strings.ToLower(strings.TrimSpace(lotteryType))
		if normalizedType == "" || provider == nil {
			continue
		}
		cloned[normalizedType] = provider
	}
	return cloned
}
