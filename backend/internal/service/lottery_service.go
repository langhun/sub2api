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

type LotteryOrderView struct {
	ID          int64           `json:"id"`
	LotteryType string          `json:"lottery_type"`
	IssueNo     string          `json:"issue_no"`
	RedBalls    []string        `json:"red_balls"`
	BlueBall    string          `json:"blue_ball"`
	Cost        decimal.Decimal `json:"cost"`
	Reward      decimal.Decimal `json:"reward"`
	PrizeLevel  string          `json:"prize_level"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
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
