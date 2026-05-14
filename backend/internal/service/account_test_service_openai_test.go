//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

// --- shared test helpers ---

type queuedHTTPUpstream struct {
	responses []*http.Response
	requests  []*http.Request
	tlsFlags  []bool
	proxyURLs []string
}

func (u *queuedHTTPUpstream) Do(_ *http.Request, proxyURL string, _ int64, _ int) (*http.Response, error) {
	u.proxyURLs = append(u.proxyURLs, proxyURL)
	return nil, fmt.Errorf("unexpected Do call")
}

func (u *queuedHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, _ int64, _ int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	u.requests = append(u.requests, req)
	u.tlsFlags = append(u.tlsFlags, profile != nil)
	u.proxyURLs = append(u.proxyURLs, proxyURL)
	if len(u.responses) == 0 {
		return nil, fmt.Errorf("no mocked response")
	}
	resp := u.responses[0]
	u.responses = u.responses[1:]
	return resp, nil
}

func newJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// --- test functions ---

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)
	return c, rec
}

type openAIAccountTestRepo struct {
	mockAccountRepoForGemini
	updatedExtra       map[string]any
	bulkUpdatedIDs     []int64
	bulkUpdatedPayload AccountBulkUpdate
	rateLimitedID      int64
	rateLimitedAt      *time.Time
	clearedErrorID     int64
	setErrorID         int64
	setErrorMsg        string
	boundAccountID     int64
	boundGroupIDs      []int64
}

func (r *openAIAccountTestRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updatedExtra = updates
	return nil
}

func (r *openAIAccountTestRepo) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	r.bulkUpdatedIDs = append([]int64(nil), ids...)
	r.bulkUpdatedPayload = updates
	return int64(len(ids)), nil
}

func (r *openAIAccountTestRepo) SetRateLimited(_ context.Context, id int64, resetAt time.Time) error {
	r.rateLimitedID = id
	r.rateLimitedAt = &resetAt
	return nil
}

func (r *openAIAccountTestRepo) ClearError(_ context.Context, id int64) error {
	r.clearedErrorID = id
	if account, ok := r.accountsByID[id]; ok {
		account.Status = StatusActive
		account.ErrorMessage = ""
	}
	return nil
}

func (r *openAIAccountTestRepo) SetError(_ context.Context, id int64, errorMsg string) error {
	r.setErrorID = id
	r.setErrorMsg = errorMsg
	if account, ok := r.accountsByID[id]; ok {
		account.Status = StatusError
		account.ErrorMessage = errorMsg
	}
	return nil
}

func (r *openAIAccountTestRepo) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	r.boundAccountID = accountID
	r.boundGroupIDs = append([]int64(nil), groupIDs...)
	if account, ok := r.accountsByID[accountID]; ok {
		account.GroupIDs = append([]int64(nil), groupIDs...)
	}
	return nil
}

type accountTestGroupRepoStub struct {
	GroupRepository
	groupsByPlatform map[string][]Group
}

func (s *accountTestGroupRepoStub) ListActiveByPlatform(_ context.Context, platform string) ([]Group, error) {
	return append([]Group(nil), s.groupsByPlatform[platform]...), nil
}

func TestAccountTestService_OpenAISuccessPersistsSnapshotFromHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	resp.Header.Set("x-codex-primary-used-percent", "88")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	resp.Header.Set("x-codex-secondary-used-percent", "42")
	resp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	resp.Header.Set("x-codex-secondary-window-minutes", "300")

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          89,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, repo.updatedExtra)
	require.Equal(t, 42.0, repo.updatedExtra["codex_5h_used_percent"])
	require.Equal(t, 88.0, repo.updatedExtra["codex_7d_used_percent"])
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAISuccessBindsPlatformDefaultGroupForUngroupedAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))

	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				89: {
					ID:          89,
					Platform:    PlatformOpenAI,
					Type:        AccountTypeOAuth,
					Concurrency: 1,
					Credentials: map[string]any{"access_token": "test-token"},
				},
			},
		},
	}
	groupRepo := &accountTestGroupRepoStub{
		groupsByPlatform: map[string][]Group{
			PlatformOpenAI: {
				{ID: 501, Name: "openai-default", Platform: PlatformOpenAI},
			},
		},
	}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, groupRepo: groupRepo, httpUpstream: upstream}

	account := &Account{
		ID:          89,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(89), repo.boundAccountID)
	require.Equal(t, []int64{501}, repo.boundGroupIDs)
	require.Equal(t, []int64{501}, account.GroupIDs)
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAISuccessDoesNotOverrideExplicitGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))

	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				90: {
					ID:          90,
					Platform:    PlatformOpenAI,
					Type:        AccountTypeOAuth,
					Concurrency: 1,
					GroupIDs:    []int64{700},
					Credentials: map[string]any{"access_token": "test-token"},
				},
			},
		},
	}
	groupRepo := &accountTestGroupRepoStub{
		groupsByPlatform: map[string][]Group{
			PlatformOpenAI: {
				{ID: 501, Name: "openai-default", Platform: PlatformOpenAI},
			},
		},
	}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, groupRepo: groupRepo, httpUpstream: upstream}

	account := &Account{
		ID:          90,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		GroupIDs:    []int64{700},
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.Zero(t, repo.boundAccountID)
	require.Nil(t, repo.boundGroupIDs)
	require.Equal(t, []int64{700}, account.GroupIDs)
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAIUsesAutoFailoverProxyPoolForTesting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))

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
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{
		httpUpstream: upstream,
		proxyPool:    poolSvc,
	}
	account := &Account{
		ID:          91,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
		Extra: map[string]any{
			"proxy_mode": AccountProxyModePool,
		},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.Len(t, upstream.proxyURLs, 1)
	require.Equal(t, "http://pool.example:8080", upstream.proxyURLs[0])
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAIExplicitPoolWithoutCandidateFailsClearly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	poolSvc := NewAutoFailoverProxyPoolService(&proxyRepoStubForPool{}, nil, settingSvc, &proxyLatencyCacheStubForPool{}, nil)
	upstream := &queuedHTTPUpstream{}
	svc := &AccountTestService{
		httpUpstream: upstream,
		proxyPool:    poolSvc,
	}
	account := &Account{
		ID:          92,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
		Extra: map[string]any{
			"proxy_mode": AccountProxyModePool,
		},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Empty(t, upstream.proxyURLs)
	require.Contains(t, err.Error(), "no available proxy in auto failover proxy pool")
	require.Contains(t, recorder.Body.String(), "no available proxy in auto failover proxy pool")
}

func TestAccountTestService_OpenAIStreamEOFBeforeCompletedFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.output_text.delta","delta":"hi"}

`))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID:          90,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Contains(t, recorder.Body.String(), "response.completed")
	require.NotContains(t, recorder.Body.String(), `"success":true`)
}

func TestAccountTestService_OpenAI429PersistsSnapshotAndRateLimitState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached","resets_at":1777283883}}`)
	resp.Header.Set("x-codex-primary-used-percent", "100")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	resp.Header.Set("x-codex-secondary-used-percent", "100")
	resp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	resp.Header.Set("x-codex-secondary-window-minutes", "300")

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          88,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusError,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.NotEmpty(t, repo.updatedExtra)
	require.Equal(t, 100.0, repo.updatedExtra["codex_5h_used_percent"])
	require.Equal(t, account.ID, repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
	require.Equal(t, account.ID, repo.clearedErrorID)
	require.Equal(t, StatusActive, account.Status)
	require.Empty(t, account.ErrorMessage)
	require.NotNil(t, account.RateLimitResetAt)
}

func TestAccountTestService_OpenAI429BodyOnlyPersistsRateLimitAndClearsStaleError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached","resets_at":"1777283883"}}`)

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:           77,
		Platform:     PlatformOpenAI,
		Type:         AccountTypeOAuth,
		Status:       StatusError,
		ErrorMessage: "Access forbidden (403): account may be suspended or lack permissions",
		Concurrency:  1,
		Credentials:  map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Equal(t, account.ID, repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
	require.Equal(t, account.ID, repo.clearedErrorID)
	require.Equal(t, StatusActive, account.Status)
	require.Empty(t, account.ErrorMessage)
	require.NotNil(t, account.RateLimitResetAt)
	require.Empty(t, repo.updatedExtra)
}

func TestAccountTestService_OpenAI429SyncsObservedPlanType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached","plan_type":"free","resets_at":1777283883}}`)

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          81,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token", "plan_type": "plus"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Equal(t, []int64{account.ID}, repo.bulkUpdatedIDs)
	require.Equal(t, "free", repo.bulkUpdatedPayload.Credentials["plan_type"])
	require.Equal(t, "free", account.Credentials["plan_type"])
	require.Equal(t, account.ID, repo.rateLimitedID)
	require.NotNil(t, account.RateLimitResetAt)
}

func TestAccountTestService_OpenAI429ActiveAccountDoesNotClearError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached","resets_in_seconds":3600}}`)

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:          78,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Equal(t, account.ID, repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
	require.Zero(t, repo.clearedErrorID)
	require.Equal(t, StatusActive, account.Status)
	require.NotNil(t, account.RateLimitResetAt)
}

func TestAccountTestService_OpenAI429WithoutResetSignalDoesNotMutateRuntimeState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached"}}`)

	repo := &openAIAccountTestRepo{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	account := &Account{
		ID:           79,
		Platform:     PlatformOpenAI,
		Type:         AccountTypeOAuth,
		Status:       StatusError,
		ErrorMessage: "stale 403",
		Concurrency:  1,
		Credentials:  map[string]any{"access_token": "test-token"},
	}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Zero(t, repo.rateLimitedID)
	require.Nil(t, repo.rateLimitedAt)
	require.Zero(t, repo.clearedErrorID)
	require.Equal(t, StatusError, account.Status)
	require.Equal(t, "stale 403", account.ErrorMessage)
	require.Nil(t, account.RateLimitResetAt)
}

func TestAccountTestService_OpenAI401SetsPermanentErrorOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := newTestContext()

	resp := newJSONResponse(http.StatusUnauthorized, `{"error":"bad token"}`)

	account := &Account{
		ID:          80,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				80: account,
			},
		},
	}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}

	err := svc.testOpenAIAccountConnection(ctx, account, "gpt-5.4", "", "")
	require.Error(t, err)
	require.Equal(t, account.ID, repo.setErrorID)
	require.Contains(t, repo.setErrorMsg, "Authentication failed (401)")
	require.Zero(t, repo.rateLimitedID)
	require.Zero(t, repo.clearedErrorID)
	require.Nil(t, account.RateLimitResetAt)
	require.Equal(t, StatusError, account.Status)
	require.Contains(t, account.ErrorMessage, "Authentication failed (401)")
}

func TestAccountTestService_RunTestBackground_OpenAI401ReturnsFailedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				80: {
					ID:          80,
					Platform:    PlatformOpenAI,
					Type:        AccountTypeOAuth,
					Status:      StatusActive,
					Concurrency: 1,
					Credentials: map[string]any{"access_token": "test-token"},
				},
			},
		},
	}
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusUnauthorized, `{"error":"bad token"}`),
		},
	}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.RunTestBackground(context.Background(), 80, "gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ErrorMessage, "API returned 401")
	require.Empty(t, result.ResponseText)
	require.NotZero(t, result.StartedAt)
	require.NotZero(t, result.FinishedAt)
	require.GreaterOrEqual(t, result.LatencyMs, int64(0))
	require.Equal(t, int64(80), repo.setErrorID)
	require.Equal(t, StatusError, repo.accountsByID[80].Status)
	require.Contains(t, repo.accountsByID[80].ErrorMessage, "Authentication failed (401)")
}

func TestAccountTestService_RunTestBackground_OpenAI429ReturnsFailedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				88: {
					ID:          88,
					Platform:    PlatformOpenAI,
					Type:        AccountTypeOAuth,
					Status:      StatusError,
					Concurrency: 1,
					Credentials: map[string]any{"access_token": "test-token"},
				},
			},
		},
	}
	resp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached","resets_at":1777283883}}`)
	resp.Header.Set("x-codex-primary-used-percent", "100")
	resp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	resp.Header.Set("x-codex-primary-window-minutes", "10080")
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.RunTestBackground(context.Background(), 88, "gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ErrorMessage, "API returned 429")
	require.Empty(t, result.ResponseText)
	require.Equal(t, int64(88), repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
}
