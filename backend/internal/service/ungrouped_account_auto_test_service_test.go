package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type ungroupedAutoTestAccountRepoStub struct {
	accounts              []Account
	listWithFiltersGroup  int64
	listWithFiltersPlatform string
	listWithFiltersStatus string
	listWithFiltersCalls  int
	updatedExtraByID      map[int64]map[string]any
	deletedIDs            []int64
}

func (s *ungroupedAutoTestAccountRepoStub) Create(context.Context, *Account) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return nil, ErrAccountNotFound
}
func (s *ungroupedAutoTestAccountRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	return false, nil
}
func (s *ungroupedAutoTestAccountRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) Update(context.Context, *Account) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) Delete(_ context.Context, id int64) error {
	s.deletedIDs = append(s.deletedIDs, id)
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, platform, _, status, _ string, groupID int64, _, _ string) ([]Account, *pagination.PaginationResult, error) {
	s.listWithFiltersCalls++
	s.listWithFiltersGroup = groupID
	s.listWithFiltersPlatform = platform
	s.listWithFiltersStatus = status
	out := append([]Account(nil), s.accounts...)
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: 1, PageSize: len(out)}, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListActive(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) UpdateLastUsed(context.Context, int64) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetError(context.Context, int64, string) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) ClearError(context.Context, int64) error       { return nil }
func (s *ungroupedAutoTestAccountRepoStub) SetSchedulable(context.Context, int64, bool) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (s *ungroupedAutoTestAccountRepoStub) BindGroups(context.Context, int64, []int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ClearRateLimit(context.Context, int64) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ClearModelRateLimits(context.Context, int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if s.updatedExtraByID == nil {
		s.updatedExtraByID = make(map[int64]map[string]any)
	}
	copied := make(map[string]any, len(updates))
	for k, v := range updates {
		copied[k] = v
	}
	s.updatedExtraByID[id] = copied
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (s *ungroupedAutoTestAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func TestShouldRunUngroupedAutoTest(t *testing.T) {
	now := time.Date(2026, 5, 14, 16, 0, 0, 0, time.UTC)

	require.True(t, shouldRunUngroupedAutoTest(now, nil))
	require.True(t, shouldRunUngroupedAutoTest(now, map[string]any{}))
	require.True(t, shouldRunUngroupedAutoTest(now, map[string]any{
		"auto_test_last_at": now.Add(-ungroupedAccountAutoTestRecencyInterval - time.Minute).Format(time.RFC3339),
	}))
	require.False(t, shouldRunUngroupedAutoTest(now, map[string]any{
		"auto_test_last_at": now.Add(-(ungroupedAccountAutoTestRecencyInterval - time.Minute)).Format(time.RFC3339),
	}))
}

func TestUngroupedAccountAutoTestServiceListCandidatesFiltersByRecencyAndSchedulable(t *testing.T) {
	now := time.Now().UTC()
	repo := &ungroupedAutoTestAccountRepoStub{
		accounts: []Account{
			{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true},
			{ID: 2, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Extra: map[string]any{
				"auto_test_last_at": now.Add(-(ungroupedAccountAutoTestRecencyInterval - 10*time.Minute)).Format(time.RFC3339),
			}},
			{ID: 3, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: false},
			{ID: 4, Platform: PlatformOpenAI, Status: StatusDisabled, Schedulable: true},
			{ID: 5, Platform: PlatformGemini, Status: StatusActive, Schedulable: true},
		},
	}
	svc := NewUngroupedAccountAutoTestService(repo, &AccountTestService{}, nil)

	candidates, err := svc.listCandidates(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(0), repo.listWithFiltersGroup)
	require.Equal(t, PlatformOpenAI, repo.listWithFiltersPlatform)
	require.Equal(t, StatusActive, repo.listWithFiltersStatus)
	require.Equal(t, 1, repo.listWithFiltersCalls)
	require.Len(t, candidates, 1)
	require.Equal(t, int64(1), candidates[0].ID)
}

func TestUngroupedAccountAutoTestServiceListCandidatesPrioritizesUngroupedOpenAI(t *testing.T) {
	repo := &ungroupedAutoTestAccountRepoStub{
		accounts: []Account{
			{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, GroupIDs: []int64{100}},
			{ID: 2, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true},
			{ID: 3, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, GroupIDs: []int64{101}},
			{ID: 4, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true},
		},
	}
	svc := NewUngroupedAccountAutoTestService(repo, &AccountTestService{}, nil)

	candidates, err := svc.listCandidates(context.Background())
	require.NoError(t, err)
	require.Len(t, candidates, 4)
	require.Equal(t, []int64{2, 4, 1, 3}, []int64{candidates[0].ID, candidates[1].ID, candidates[2].ID, candidates[3].ID})
}

func TestBuildUngroupedAutoTestExtraUpdates(t *testing.T) {
	started := time.Date(2026, 5, 14, 16, 30, 0, 0, time.UTC)
	statusCode := 403
	result := &ScheduledTestResult{
		Status:         "failed",
		ErrorMessage:   "forbidden",
		HTTPStatusCode: &statusCode,
		LatencyMs:      1234,
		FinishedAt:     started.Add(5 * time.Second),
	}

	updates := buildUngroupedAutoTestExtraUpdates(started, result)
	require.Equal(t, "failed", updates["auto_test_last_status"])
	require.Equal(t, "forbidden", updates["auto_test_last_error"])
	require.Equal(t, 403, updates["auto_test_http_status"])
	require.Equal(t, int64(1234), updates["auto_test_latency_ms"])
	require.Equal(t, started.Format(time.RFC3339), updates["auto_test_last_at"])
	require.Equal(t, started.Add(5*time.Second).Format(time.RFC3339), updates["auto_test_last_finished_at"])
}

func TestUngroupedAccountAutoTestServiceApplyOpenAIAutoTestActions401DeletesAccount(t *testing.T) {
	repo := &ungroupedAutoTestAccountRepoStub{}
	svc := NewUngroupedAccountAutoTestService(repo, &AccountTestService{}, nil)
	account := &Account{ID: 401, Platform: PlatformOpenAI}
	code := http.StatusUnauthorized
	result := &ScheduledTestResult{
		Status:         "failed",
		HTTPStatusCode: &code,
		ErrorMessage:   "Authentication failed (401)",
	}

	skipPersist := svc.applyOpenAIAutoTestActions(context.Background(), account, result)
	require.True(t, skipPersist)
	require.Equal(t, []int64{401}, repo.deletedIDs)
}

func TestUngroupedAccountAutoTestServiceApplyOpenAIAutoTestActions403SwitchesToPoolAndRetests(t *testing.T) {
	repo := &ungroupedAutoTestAccountRepoStub{}
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[11]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			11: {ID: 11, Name: "pool", Protocol: "http", Host: "pool.example", Port: 8080, Status: StatusActive},
		},
	}
	poolSvc := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, &proxyLatencyCacheStubForPool{}, nil)

	successResp := newJSONResponse(http.StatusOK, "")
	successResp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	upstream := &queuedHTTPUpstream{responses: []*http.Response{successResp}}
	account := &Account{
		ID:          403,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
	accountTestSvc := &AccountTestService{
		accountRepo: &openAIAccountTestRepo{
			mockAccountRepoForGemini: mockAccountRepoForGemini{
				accountsByID: map[int64]*Account{
					403: account,
				},
			},
		},
		httpUpstream: upstream,
		proxyPool:    poolSvc,
	}
	svc := NewUngroupedAccountAutoTestService(repo, accountTestSvc, nil)
	code := http.StatusForbidden
	result := &ScheduledTestResult{
		Status:         "failed",
		HTTPStatusCode: &code,
		ErrorMessage:   "Access forbidden (403)",
	}

	skipPersist := svc.applyOpenAIAutoTestActions(context.Background(), account, result)
	require.False(t, skipPersist)
	require.Equal(t, AccountProxyModePool, account.Extra["proxy_mode"])
	require.Equal(t, AccountProxyModePool, repo.updatedExtraByID[403]["proxy_mode"])
	require.Equal(t, "success", result.Status)
	require.Len(t, upstream.proxyURLs, 1)
	require.Equal(t, "http://pool.example:8080", upstream.proxyURLs[0])
}

func TestRunTestBackgroundOpenAI429WithoutResetStillReturnsRateLimitState(t *testing.T) {
	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				4291: {
					ID:          4291,
					Platform:    PlatformOpenAI,
					Type:        AccountTypeOAuth,
					Status:      StatusActive,
					Concurrency: 1,
					Credentials: map[string]any{"access_token": "test-token"},
				},
			},
		},
	}
	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached"}}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.RunTestBackground(context.Background(), 4291, "gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "failed", result.Status)
	require.Equal(t, int64(4291), repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
	require.NotNil(t, repo.accountsByID[4291].RateLimitResetAt)
}
