package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type gatewaySmartRetryBudgetUpstream struct {
	responses []*http.Response
	calls     int
}

func (u *gatewaySmartRetryBudgetUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls++
	if req != nil && req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(nil))
	}
	if len(u.responses) == 0 {
		return nil, errors.New("unexpected upstream request")
	}
	resp := u.responses[0]
	u.responses = u.responses[1:]
	return resp, nil
}

func (u *gatewaySmartRetryBudgetUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func gatewayRetryBudgetTestResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"upstream temporarily unavailable"}}`)),
	}
}

func gatewayRetryBudgetTestService(upstream HTTPUpstream) *GatewayService {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}
	return &GatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
		deferredService:  &DeferredService{},
	}
}

func gatewayRetryBudgetTestRequest(t *testing.T) (*gin.Context, *ParsedRequest) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.5","stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	return c, &ParsedRequest{
		Body:  NewRequestBodyRef(body),
		Model: "gpt-5.5",
	}
}

func gatewayRetryBudgetTestAccount() *Account {
	return &Account{
		ID:          47110,
		Name:        "smart-routing-openai",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":   "test-key",
			"base_url":  "https://upstream.example",
			"pool_mode": true,
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func gatewayRetryBudgetTestNonOpenAIRequest(t *testing.T) (*gin.Context, *ParsedRequest) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"deepseek-chat","stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	return c, &ParsedRequest{
		Body:  NewRequestBodyRef(body),
		Model: "deepseek-chat",
	}
}

func gatewayRetryBudgetTestNonOpenAIAccount() *Account {
	return &Account{
		ID:          47111,
		Name:        "smart-routing-non-openai",
		Platform:    PlatformDeepseek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":   "test-key",
			"base_url":  "https://upstream.example",
			"pool_mode": true,
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func TestGatewayService_Forward_SmartRoutedOpenAIExposesFailoverWithoutInternal5xxRetries(t *testing.T) {
	c, parsed := gatewayRetryBudgetTestRequest(t)
	upstream := &gatewaySmartRetryBudgetUpstream{responses: []*http.Response{
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
	}}
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          20,
		SmartRoutingMaxAttempts: 12,
		SmartRoutingMaxDuration: 30 * time.Second,
		SmartRoutingStartedAt:   time.Now(),
	}
	ctx := WithSmartRoutingState(context.Background(), state)

	_, err := gatewayRetryBudgetTestService(upstream).Forward(ctx, c, gatewayRetryBudgetTestAccount(), parsed)

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.True(t, failoverErr.ShouldRetryNextAccount(), "the handler must be able to select another upstream")
	require.Equal(t, 1, upstream.calls, "smart OpenAI should expose a failover status after the first provider send")
	require.Equal(t, 1, state.TotalAttempts, "the actual provider send is reserved immediately before DoWithTLS")
	require.Equal(t, -1, c.Writer.Size(), "the service must leave the response for the handler")
}

func TestGatewayService_Forward_NonSmartOpenAIRetainsInternal5xxRetryBehavior(t *testing.T) {
	c, parsed := gatewayRetryBudgetTestRequest(t)
	upstream := &gatewaySmartRetryBudgetUpstream{responses: []*http.Response{
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
	}}

	account := gatewayRetryBudgetTestAccount()
	account.Credentials["custom_error_codes_enabled"] = true
	account.Credentials["custom_error_codes"] = []any{float64(http.StatusBadRequest)}
	_, err := gatewayRetryBudgetTestService(upstream).Forward(context.Background(), c, account, parsed)

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.Equal(t, maxRetryAttempts, upstream.calls, "ordinary requests retain the existing bounded retry count")
}

func TestGatewayService_Forward_SmartRoutedNonOpenAIExposesFailoverWithoutInternal5xxRetries(t *testing.T) {
	c, parsed := gatewayRetryBudgetTestNonOpenAIRequest(t)
	upstream := &gatewaySmartRetryBudgetUpstream{responses: []*http.Response{
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
		gatewayRetryBudgetTestResponse(http.StatusServiceUnavailable),
	}}
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          20,
		SmartRoutingMaxAttempts: 12,
		SmartRoutingMaxDuration: 30 * time.Second,
		SmartRoutingStartedAt:   time.Now(),
	}
	ctx := WithSmartRoutingState(context.Background(), state)

	_, err := gatewayRetryBudgetTestService(upstream).Forward(ctx, c, gatewayRetryBudgetTestNonOpenAIAccount(), parsed)

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.True(t, failoverErr.ShouldRetryNextAccount(), "the handler must be able to select another upstream")
	require.Equal(t, 1, upstream.calls, "smart-routed non-OpenAI requests should expose a failover status after the first provider send")
	require.Equal(t, 1, state.TotalAttempts, "every real non-OpenAI provider send must consume the shared routing budget")
	require.Equal(t, -1, c.Writer.Size(), "the service must leave the response for the handler")
}
