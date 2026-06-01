//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type snapshotHydrationCache struct {
	snapshot []*Account
	accounts map[int64]*Account
}

func (c *snapshotHydrationCache) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	if c.snapshot == nil {
		return nil, false, nil
	}
	return c.snapshot, true, nil
}

func (c *snapshotHydrationCache) SetSnapshot(ctx context.Context, bucket SchedulerBucket, accounts []Account) error {
	return nil
}

func (c *snapshotHydrationCache) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if c.accounts == nil {
		return nil, nil
	}
	return c.accounts[accountID], nil
}

func (c *snapshotHydrationCache) SetAccount(ctx context.Context, account *Account) error {
	return nil
}

func (c *snapshotHydrationCache) DeleteAccount(ctx context.Context, accountID int64) error {
	return nil
}

func (c *snapshotHydrationCache) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	return nil
}

func (c *snapshotHydrationCache) TryLockBucket(ctx context.Context, bucket SchedulerBucket, ttl time.Duration) (bool, error) {
	return true, nil
}

func (c *snapshotHydrationCache) UnlockBucket(ctx context.Context, bucket SchedulerBucket) error {
	return nil
}

func (c *snapshotHydrationCache) ListBuckets(ctx context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (c *snapshotHydrationCache) GetOutboxWatermark(ctx context.Context) (int64, error) {
	return 0, nil
}

func (c *snapshotHydrationCache) SetOutboxWatermark(ctx context.Context, id int64) error {
	return nil
}

type snapshotFallbackAccountRepo struct {
	AccountRepository
	getByIDFn                    func(ctx context.Context, id int64) (*Account, error)
	listUngroupedByPlatformFn    func(ctx context.Context, platform string) ([]Account, error)
	getByIDCalls                 int
	listUngroupedByPlatformCalls int
}

func (r *snapshotFallbackAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	r.getByIDCalls++
	if r.getByIDFn != nil {
		return r.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (r *snapshotFallbackAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	r.listUngroupedByPlatformCalls++
	if r.listUngroupedByPlatformFn != nil {
		return r.listUngroupedByPlatformFn(ctx, platform)
	}
	return nil, nil
}

func TestSchedulerSnapshotService_GetAccount_CacheMissReturnsFallbackLimitedWithoutDBCall(t *testing.T) {
	repo := &snapshotFallbackAccountRepo{
		getByIDFn: func(ctx context.Context, id int64) (*Account, error) {
			return &Account{ID: id}, nil
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.DbFallbackEnabled = true
	cfg.Gateway.Scheduling.DbFallbackMaxQPS = 1

	schedulerSnapshot := NewSchedulerSnapshotService(&snapshotHydrationCache{}, nil, repo, nil, cfg)
	schedulerSnapshot.fallbackLimit = &fallbackLimiter{
		maxQPS: 1,
		window: time.Now(),
		count:  1,
	}

	account, err := schedulerSnapshot.GetAccount(context.Background(), 42)
	if !errors.Is(err, ErrSchedulerFallbackLimited) {
		t.Fatalf("expected ErrSchedulerFallbackLimited, got %v", err)
	}
	if account != nil {
		t.Fatalf("expected nil account when fallback is limited")
	}
	if repo.getByIDCalls != 0 {
		t.Fatalf("expected no DB fallback calls when limiter rejects, got %d", repo.getByIDCalls)
	}
}

func TestSchedulerSnapshotService_ListSchedulableAccounts_CacheMissWithExpiredContextPropagatesFallbackError(t *testing.T) {
	var repoCtxErr error
	repo := &snapshotFallbackAccountRepo{
		listUngroupedByPlatformFn: func(ctx context.Context, platform string) ([]Account, error) {
			<-ctx.Done()
			repoCtxErr = ctx.Err()
			return nil, ctx.Err()
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.DbFallbackEnabled = true
	cfg.Gateway.Scheduling.DbFallbackTimeoutSeconds = 5

	schedulerSnapshot := NewSchedulerSnapshotService(&snapshotHydrationCache{}, nil, repo, nil, cfg)
	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	accounts, useMixed, err := schedulerSnapshot.ListSchedulableAccounts(expiredCtx, nil, PlatformOpenAI, false)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded from DB fallback, got %v", err)
	}
	if repoCtxErr == nil || !errors.Is(repoCtxErr, context.DeadlineExceeded) {
		t.Fatalf("expected repo to observe deadline exceeded context, got %v", repoCtxErr)
	}
	if accounts != nil {
		t.Fatalf("expected nil accounts on fallback error, got %v", accounts)
	}
	if useMixed {
		t.Fatalf("expected single-platform scheduling for openai cache miss")
	}
	if repo.listUngroupedByPlatformCalls != 1 {
		t.Fatalf("expected one DB fallback call, got %d", repo.listUngroupedByPlatformCalls)
	}
}

func TestOpenAISelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot(t *testing.T) {
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				GroupIDs:    []int64{2},
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-4": "gpt-4",
					},
				},
			},
		},
		accounts: map[int64]*Account{
			1: {
				ID:          1,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				GroupIDs:    []int64{2},
				Credentials: map[string]any{
					"api_key":       "sk-live",
					"model_mapping": map[string]any{"gpt-4": "gpt-4"},
				},
			},
		},
	}

	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, nil, nil, nil)
	groupID := int64(2)
	svc := &OpenAIGatewayService{
		schedulerSnapshot: schedulerSnapshot,
		cache:             &stubGatewayCache{},
	}

	selection, err := svc.SelectAccountWithLoadAwareness(context.Background(), &groupID, "", "gpt-4", nil)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if selection == nil || selection.Account == nil {
		t.Fatalf("expected selected account")
	}
	if got := selection.Account.GetOpenAIApiKey(); got != "sk-live" {
		t.Fatalf("expected hydrated api key, got %q", got)
	}
}

func TestGatewaySelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot(t *testing.T) {
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          9,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
			},
		},
		accounts: map[int64]*Account{
			9: {
				ID:          9,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Credentials: map[string]any{
					"api_key": "anthropic-live-key",
				},
			},
		},
	}

	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, nil, nil, nil)
	svc := &GatewayService{
		schedulerSnapshot: schedulerSnapshot,
		cache:             &mockGatewayCacheForPlatform{},
		cfg:               testConfig(),
	}

	result, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "claude-3-5-sonnet-20241022", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatalf("expected selected account")
	}
	if got := result.Account.GetCredential("api_key"); got != "anthropic-live-key" {
		t.Fatalf("expected hydrated api key, got %q", got)
	}
}
