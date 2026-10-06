package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// HTTP POST /v1/responses -> forwardOpenAIWSV2 keeps the canonical outbound
// tier separate from response.completed.service_tier for usage-time billing.
func TestForwardOpenAIWSV2_KeepsOutboundAndObservedServiceTiersSeparate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name        string
		requestTier string
		stream      bool
	}{
		{name: "priority_nonstream", requestTier: "priority", stream: false},
		{name: "fast_stream", requestTier: "fast", stream: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			c.Request.Header.Set("User-Agent", "unit-test-agent/1.0")

			cfg := &config.Config{}
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			cfg.Gateway.OpenAIWS.Enabled = true
			cfg.Gateway.OpenAIWS.APIKeyEnabled = true
			cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
			cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
			cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
			cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
			cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
			cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
			cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 5
			cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

			captureConn := &openAIWSCaptureConn{
				events: [][]byte{
					[]byte(`{"type":"response.completed","response":{"id":"resp_tier_v2","model":"gpt-5.5","status":"completed","service_tier":"default","usage":{"input_tokens":1,"output_tokens":1}}}`),
				},
			}
			captureDialer := &openAIWSCaptureDialer{conn: captureConn}
			pool := newOpenAIWSConnPool(cfg)
			pool.setClientDialerForTest(captureDialer)

			svc := &OpenAIGatewayService{
				cfg:              cfg,
				httpUpstream:     &httpUpstreamRecorder{},
				cache:            &stubGatewayCache{},
				openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
				toolCorrector:    NewCodexToolCorrector(),
				openaiWSPool:     pool,
			}
			account := &Account{
				ID:          5882,
				Name:        "openai-ws-v2-tier",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"},
				Extra:       map[string]any{"responses_websockets_v2_enabled": true},
			}

			body := []byte(fmt.Sprintf(
				`{"model":"gpt-5.5","stream":%t,"service_tier":%q,"input":[{"type":"input_text","text":"hi"}]}`,
				tc.stream, tc.requestTier,
			))
			result, err := svc.Forward(context.Background(), c, account, body)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.OpenAIWSMode, "must take HTTP POST → forwardOpenAIWSV2, not HTTP fallback")
			require.Equal(t, tc.stream, result.Stream)
			require.Equal(t, "resp_tier_v2", result.RequestID)
			require.NotNil(t, result.ServiceTier)
			require.Equal(t, "priority", *result.ServiceTier)
			require.Equal(t, "default", result.UpstreamResponseServiceTier)
			require.Equal(t, "priority", captureConn.lastWrite["service_tier"],
				"outbound WS payload still carries the requested Fast tier")
		})
	}
}

func TestForwardOpenAIWSV2_MarksCyberPolicyForFailureEventShapes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		upstreamEvent []byte
		wantError     bool
		wantInput     int
		wantOutput    int
	}{
		{
			name:          "error_before_return",
			upstreamEvent: []byte(`{"type":"error","error":{"type":"rate_limit_error","code":"cyber_policy","message":"rate limit exceeded by cyber policy"},"usage":{"input_tokens":5,"output_tokens":1}}`),
			wantError:     true,
			wantInput:     5,
			wantOutput:    1,
		},
		{
			name:          "response_failed_terminal",
			upstreamEvent: []byte(`{"type":"response.failed","response":{"id":"resp_cyber","status":"failed","error":{"code":"cyber_policy","message":"blocked by cyber policy"},"usage":{"input_tokens":9,"output_tokens":2}}}`),
			wantInput:     9,
			wantOutput:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)

			cfg := newOpenAIWSV2TestConfig()
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
			captureConn := &openAIWSCaptureConn{events: [][]byte{append([]byte(nil), tt.upstreamEvent...)}}
			pool := newOpenAIWSConnPool(cfg)
			pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
			svc := &OpenAIGatewayService{
				cfg:              cfg,
				httpUpstream:     &httpUpstreamRecorder{},
				cache:            &stubGatewayCache{},
				openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
				toolCorrector:    NewCodexToolCorrector(),
				openaiWSPool:     pool,
			}
			account := &Account{
				ID: 5883, Name: "openai-ws-v2-cyber", Platform: PlatformOpenAI,
				Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"},
				Extra:       map[string]any{"responses_websockets_v2_enabled": true},
			}

			result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.5","stream":false,"input":"hello"}`))
			if tt.wantError {
				require.Error(t, err)
				require.Nil(t, result)
				var failoverErr *UpstreamFailoverError
				require.False(t, errors.As(err, &failoverErr))
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			mark := GetOpsCyberPolicy(c)
			require.NotNil(t, mark)
			require.Equal(t, "cyber_policy", mark.Code)
			require.Equal(t, tt.wantInput, mark.UpstreamInTok)
			require.Equal(t, tt.wantOutput, mark.UpstreamOutTok)
		})
	}
}

func TestForwardOpenAIWSV2_SmartRoutingDisconnectBeforeTokenReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		SmartRoutingStartedAt:   time.Now(),
		SmartRoutingMaxAttempts: 4,
		SmartRoutingMaxDuration: time.Minute,
		MainGroupID:             97,
		CurrentGroupID:          97,
	}
	c.Request = request.WithContext(WithSmartRoutingState(request.Context(), state))

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","sequence_number":0,"response":{"id":"resp_preoutput","model":"gpt-5.5","status":"in_progress"}}`),
		},
	}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5884, Name: "openai-ws-v2-pre-output-disconnect", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(c.Request.Context(), c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "a pre-token disconnect must reach the account/group failover contract")
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.True(t, failoverErr.ShouldRetryNextAccount())
	require.Empty(t, recorder.Body.String(), "buffered response.created must not commit a partial stream")
}

func TestForwardOpenAIWSV2_SmartRoutingMalformedPreOutputFailsOverWithoutRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		SmartRoutingStartedAt:   time.Now(),
		SmartRoutingMaxAttempts: 4,
		MainGroupID:             97,
		CurrentGroupID:          97,
	}
	c.Request = request.WithContext(WithSmartRoutingState(request.Context(), state))

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	firstConn := &openAIWSCaptureConn{events: [][]byte{[]byte("not-json")}}
	secondConn := &openAIWSCaptureConn{events: [][]byte{
		[]byte(`{"type":"response.completed","response":{"id":"must_not_retry","usage":{"input_tokens":1,"output_tokens":1}}}`),
	}}
	dialer := &openAIWSQueueDialer{conns: []openAIWSClientConn{firstConn, secondConn}}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(dialer)
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5893, Name: "openai-ws-v2-malformed-smart-route", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(c.Request.Context(), c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, GatewayFailureReason("openai_ws_invalid_event_json"), failoverErr.Reason)
	require.Equal(t, 1, state.TotalAttempts, "malformed protocol output should immediately move to another routed site")
	require.Equal(t, 1, dialer.DialCount())
	require.Empty(t, recorder.Body.String())
}

func TestForwardOpenAIWSV2_SmartRoutingBoundsInternalReconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		SmartRoutingStartedAt:   time.Now(),
		SmartRoutingMaxAttempts: 4,
		SmartRoutingMaxDuration: time.Minute,
		MainGroupID:             97,
		CurrentGroupID:          97,
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	c.Request = c.Request.WithContext(WithSmartRoutingState(c.Request.Context(), state))

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","sequence_number":0,"response":{"id":"resp_smart_preoutput","model":"gpt-5.5","status":"in_progress"}}`),
		},
	}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5886, Name: "openai-ws-v2-smart-routing-bound", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(c.Request.Context(), c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Contains(t, string(failoverErr.Reason), "openai_ws_")
	require.Equal(t, 2, state.TotalAttempts, "one internal WS reconnect must consume exactly one additional request attempt")
}

func TestForwardOpenAIWSV2_DisconnectAfterTokenWritesSingleTerminalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","sequence_number":0,"response":{"id":"resp_partial","model":"gpt-5.5","status":"in_progress"}}`),
			[]byte(`{"type":"response.output_text.delta","sequence_number":1,"response_id":"resp_partial","delta":"partial"}`),
			[]byte(`{"type":"response.output_text.done","sequence_number":2,"response_id":"resp_partial","text":"partial","usage":{"input_tokens":7,"output_tokens":3}}`),
		},
	}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5885, Name: "openai-ws-v2-post-output-disconnect", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.Error(t, err)
	require.NotNil(t, result, "partial stream failures must retain collected usage")
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 3, result.Usage.OutputTokens)
	body := recorder.Body.String()
	require.Contains(t, body, `"type":"response.created"`)
	require.Contains(t, body, `"type":"response.output_text.delta"`)
	require.Equal(t, 1, strings.Count(body, `"type":"response.failed"`), "the service must synthesize exactly one Responses terminal event")
	require.NotContains(t, body, "event: error", "Responses clients must not receive a non-terminal error event")
	require.Contains(t, err.Error(), "upstream response failed:")
}

func TestForwardOpenAIWSV2_MalformedEventAfterOutputRetainsUsageAndFailsTerminally(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","sequence_number":0,"response":{"id":"resp_malformed_partial","model":"gpt-5.5","status":"in_progress"}}`),
			[]byte(`{"type":"response.output_text.delta","sequence_number":1,"response_id":"resp_malformed_partial","delta":"partial"}`),
			[]byte(`{"type":"response.output_text.done","sequence_number":2,"response_id":"resp_malformed_partial","text":"partial","usage":{"input_tokens":9,"output_tokens":4}}`),
			[]byte(`not-json`),
		},
	}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5890, Name: "openai-ws-v2-malformed-after-output", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, 9, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)
	body := recorder.Body.String()
	require.Contains(t, body, `"delta":"partial"`)
	require.Equal(t, 1, strings.Count(body, `"type":"response.failed"`))
	require.NotContains(t, body, "not-json")
}

func TestForwardOpenAIWSV2_ErrorAfterTokenWritesSyntheticTerminalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","sequence_number":0,"response":{"id":"resp_error_after_output","model":"gpt-5.5","status":"in_progress"}}`),
			[]byte(`{"type":"response.output_text.delta","sequence_number":1,"response_id":"resp_error_after_output","delta":"partial"}`),
			[]byte(`{"type":"error","sequence_number":2,"error":{"type":"server_error","code":"upstream_failed","message":"upstream failed after output"}}`),
		},
	}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5887, Name: "openai-ws-v2-error-after-output", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.Error(t, err)
	require.Nil(t, result)
	body := recorder.Body.String()
	require.Contains(t, body, `"type":"response.created"`)
	require.Contains(t, body, `"type":"response.output_text.delta"`)
	require.Equal(t, 1, strings.Count(body, `"type":"response.failed"`))
	require.NotContains(t, body, `"type":"error"`, "a post-output Responses turn must not expose a bare non-terminal error event")
	require.Contains(t, body, `"code":"upstream_failed"`)
	require.Contains(t, body, `"status":"failed"`)
	require.Contains(t, err.Error(), "openai ws error event:")
}

func TestForwardOpenAIWSV2_ResponsesPreOutputErrorWritesTerminalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","sequence_number":0,"response":{"id":"resp_preoutput_error","model":"gpt-5.5","status":"in_progress"}}`),
			[]byte(`{"type":"error","sequence_number":1,"error":{"type":"rate_limit_error","code":"cyber_policy","message":"blocked by cyber policy"}}`),
		},
	}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5888, Name: "openai-ws-v2-pre-output-error", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.Error(t, err)
	require.Nil(t, result)
	body := recorder.Body.String()
	require.Contains(t, body, `"type":"response.created"`)
	require.Equal(t, 1, strings.Count(body, `"type":"response.failed"`))
	require.NotContains(t, body, `"type":"error"`, "Responses clients must not receive a bare non-terminal error event")
	require.Contains(t, body, `"code":"cyber_policy"`)
	require.Contains(t, body, `"status":"failed"`)
	require.Contains(t, err.Error(), "openai ws error event:")
}

func TestForwardOpenAIWSV2_SmartRoutingPreOutputResponseFailedFailoverContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		event      string
		wantReason GatewayFailureReason
	}{
		{
			name:       "rate_limit_switches_immediately",
			event:      `{"type":"response.failed","response":{"id":"resp_failed_429","model":"gpt-5.5","status":"failed","error":{"type":"rate_limit_error","code":"rate_limit_exceeded","status_code":429,"message":"temporarily rate limited"}}}`,
			wantReason: "",
		},
		{
			name:       "server_error_is_converted_after_bounded_same_account_retry",
			event:      `{"type":"response.failed","response":{"id":"resp_failed_503","model":"gpt-5.5","status":"failed","error":{"type":"server_error","code":"upstream_error","status_code":503,"message":"upstream unavailable"}}}`,
			wantReason: "openai_ws_upstream_5xx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			state := &SmartRoutingRequestState{
				SmartRoutingActive:      true,
				SmartRoutingStartedAt:   time.Now(),
				SmartRoutingMaxAttempts: 4,
				SmartRoutingMaxDuration: time.Minute,
				MainGroupID:             97,
				CurrentGroupID:          97,
			}
			c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
			c.Request = c.Request.WithContext(WithSmartRoutingState(c.Request.Context(), state))

			cfg := newOpenAIWSV2TestConfig()
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
			firstConn := &openAIWSCaptureConn{events: [][]byte{[]byte(tt.event)}}
			secondConn := &openAIWSCaptureConn{events: [][]byte{[]byte(tt.event)}}
			pool := newOpenAIWSConnPool(cfg)
			pool.setClientDialerForTest(&openAIWSQueueDialer{conns: []openAIWSClientConn{firstConn, secondConn}})
			svc := &OpenAIGatewayService{
				cfg:              cfg,
				httpUpstream:     &httpUpstreamRecorder{},
				cache:            &stubGatewayCache{},
				openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
				toolCorrector:    NewCodexToolCorrector(),
				openaiWSPool:     pool,
			}
			account := &Account{
				ID: 5891, Name: "openai-ws-v2-smart-response-failed", Platform: PlatformOpenAI,
				Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"},
				Extra:       map[string]any{"responses_websockets_v2_enabled": true},
			}

			result, err := svc.Forward(
				c.Request.Context(),
				c,
				account,
				[]byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`),
			)
			require.Error(t, err)
			require.Nil(t, result, "pre-output failover must not produce a billable/visible result")

			var failoverErr *UpstreamFailoverError
			require.ErrorAs(t, err, &failoverErr, "smart routing must receive a typed account failover error")
			require.True(t, failoverErr.ShouldRetryNextAccount())
			if tt.wantReason == "" {
				require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
			} else {
				require.Equal(t, tt.wantReason, failoverErr.Reason)
				require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
				require.Equal(t, 2, state.TotalAttempts, "5xx may consume only one bounded same-account retry")
			}
			require.Empty(t, recorder.Body.String(), "pre-output failure must not expose a partial stream")
		})
	}
}

func TestForwardOpenAIWSV2_SmartRoutingPreOutputResponseFailedOrdinary4xxRemainsTerminal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		SmartRoutingStartedAt:   time.Now(),
		SmartRoutingMaxAttempts: 4,
		SmartRoutingMaxDuration: time.Minute,
		MainGroupID:             97,
		CurrentGroupID:          97,
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	c.Request = c.Request.WithContext(WithSmartRoutingState(c.Request.Context(), state))

	cfg := newOpenAIWSV2TestConfig()
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	captureConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.failed","response":{"id":"resp_failed_400","model":"gpt-5.5","status":"failed","error":{"type":"invalid_request_error","code":"invalid_request","status_code":400,"message":"invalid request"}}}`),
		},
	}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSCaptureDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID: 5892, Name: "openai-ws-v2-smart-response-failed-400", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	result, err := svc.Forward(
		c.Request.Context(),
		c,
		account,
		[]byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`),
	)
	require.NoError(t, err, "ordinary client 4xx must remain terminal instead of switching upstream")
	require.NotNil(t, result)
	var failoverErr *UpstreamFailoverError
	require.NotErrorAs(t, err, &failoverErr)
	require.Contains(t, recorder.Body.String(), `"type":"response.failed"`)
	require.Equal(t, 1, state.TotalAttempts, "ordinary 4xx must not spend a retry attempt")
}
