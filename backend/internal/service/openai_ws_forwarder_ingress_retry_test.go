package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type openAIWSSlowPongCaptureConn struct {
	*openAIWSCaptureConn
	delay time.Duration
	pings atomic.Int32
}

type openAIWSErrorDialer struct {
	statusCode int
	err        error
	headers    http.Header
	calls      atomic.Int32
}

func (d *openAIWSErrorDialer) Dial(context.Context, string, http.Header, string) (openAIWSClientConn, int, http.Header, error) {
	d.calls.Add(1)
	return nil, d.statusCode, cloneHeader(d.headers), d.err
}

func (c *openAIWSSlowPongCaptureConn) Ping(ctx context.Context) error {
	c.pings.Add(1)
	timer := time.NewTimer(c.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// 轮次间隔后的预检 ping 经代理可能 2 秒多才回 pong：慢 pong 不应被判为连接失效而换连。
func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_PreflightPingToleratesSlowPong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevPreflightPingIdle := openAIWSIngressPreflightPingIdle
	openAIWSIngressPreflightPingIdle = 0
	defer func() {
		openAIWSIngressPreflightPingIdle = prevPreflightPingIdle
	}()

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 2
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 2
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 5
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	slowConn := &openAIWSSlowPongCaptureConn{
		openAIWSCaptureConn: &openAIWSCaptureConn{
			events: [][]byte{
				[]byte(`{"type":"response.completed","response":{"id":"resp_slow_pong_1","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`),
				[]byte(`{"type":"response.completed","response":{"id":"resp_slow_pong_2","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`),
			},
		},
		delay: 2500 * time.Millisecond,
	}
	fallback := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.completed","response":{"id":"resp_slow_pong_fallback","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	dialer := &openAIWSQueueDialer{conns: []openAIWSClientConn{slowConn, fallback}}
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
		ID:          444,
		Name:        "openai-ingress-preflight-slow-pong",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()
		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req
		readCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		msgType, firstMessage, readErr := conn.Read(readCtx)
		cancel()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			serverErrCh <- errors.New("unsupported websocket client message type")
			return
		}
		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(r.Context(), ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	defer wsServer.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	defer func() { _ = clientConn.CloseNow() }()

	writeMessage := func(payload string) {
		writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		require.NoError(t, clientConn.Write(writeCtx, coderws.MessageText, []byte(payload)))
	}
	readMessage := func() []byte {
		readCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		msgType, message, readErr := clientConn.Read(readCtx)
		require.NoError(t, readErr)
		require.Equal(t, coderws.MessageText, msgType)
		return message
	}

	writeMessage(`{"type":"response.create","model":"gpt-5.1","stream":false,"input":[{"type":"input_text","text":"hello"}]}`)
	require.Equal(t, "resp_slow_pong_1", gjson.GetBytes(readMessage(), "response.id").String())
	writeMessage(`{"type":"response.create","model":"gpt-5.1","stream":false,"input":[{"type":"input_text","text":"world"}]}`)
	require.Equal(t, "resp_slow_pong_2", gjson.GetBytes(readMessage(), "response.id").String(), "慢 pong 的健康会话连接应继续使用")

	require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))
	select {
	case serverErr := <-serverErrCh:
		require.NoError(t, serverErr)
	case <-time.After(5 * time.Second):
		t.Fatal("等待 ingress websocket 结束超时")
	}
	require.GreaterOrEqual(t, slowConn.pings.Load(), int32(1), "第二轮前应执行预检 ping")
	require.Equal(t, 1, dialer.DialCount(), "预检 ping 慢于 2 秒不应换连")
}

// 场景：池内两条空闲连接同批陈旧（首读即失败）。首轮拿到其一失败后重试，
// 重试必须新建连接，而不是再从池里拿另一条同样陈旧的连接。
func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_TurnRetryForcesFreshConn(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 2
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 2
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	// 首会话用完 events 后归还池内，再次借出时首读即 EOF，等价于上游已判死的空闲连接。
	staleA := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.completed","response":{"id":"resp_retry_first","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	staleB := &openAIWSCaptureConn{}
	fresh := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.completed","response":{"id":"resp_retry_fresh","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	dialer := &openAIWSQueueDialer{
		conns: []openAIWSClientConn{staleA, fresh},
	}
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
		ID:          443,
		Name:        "openai-ingress-turn-retry-fresh",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
		Extra: map[string]any{
			"responses_websockets_v2_enabled": true,
		},
	}

	serverErrCh := make(chan error, 2)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req

		readCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		msgType, firstMessage, readErr := conn.Read(readCtx)
		cancel()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			serverErrCh <- errors.New("unsupported websocket client message type")
			return
		}

		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(r.Context(), ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	defer wsServer.Close()

	runSingleTurnSession := func(expectedResponseID string) {
		dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
		clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
		cancelDial()
		require.NoError(t, err)
		defer func() {
			_ = clientConn.CloseNow()
		}()

		writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
		err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","stream":false}`))
		cancelWrite()
		require.NoError(t, err)

		readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
		msgType, event, readErr := clientConn.Read(readCtx)
		cancelRead()
		require.NoError(t, readErr)
		require.Equal(t, coderws.MessageText, msgType)
		require.Equal(t, expectedResponseID, gjson.GetBytes(event, "response.id").String())

		require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))

		select {
		case serverErr := <-serverErrCh:
			require.NoError(t, serverErr)
		case <-time.After(5 * time.Second):
			t.Fatal("等待 ingress websocket 结束超时")
		}
	}

	runSingleTurnSession("resp_retry_first")
	require.Equal(t, 1, dialer.DialCount())

	// 用首会话的真实 acquire 参数构造第二条空闲连接，保证握手兼容键与后续请求一致。
	ap := pool.getOrCreateAccountPool(account.ID)
	ap.mu.Lock()
	lastAcquire := ap.lastAcquire
	ap.mu.Unlock()
	require.NotNil(t, lastAcquire)
	staleConnB := newOpenAIWSConn(pool.nextConnID(account.ID), account.ID, staleB, nil)
	staleConnB.handshakeCompatibility = normalizeOpenAIWSHandshakeCompatibility(lastAcquire.Account, lastAcquire.Headers)
	staleConnB.routingAffinity = normalizeOpenAIWSRoutingAffinity(lastAcquire.Headers)
	ap.mu.Lock()
	ap.conns[staleConnB.id] = staleConnB
	idleConns := len(ap.conns)
	ap.mu.Unlock()
	require.Equal(t, 2, idleConns, "第二会话开始前池内应有两条空闲连接")

	runSingleTurnSession("resp_retry_fresh")

	require.Equal(t, 2, dialer.DialCount(), "首读失败后的重试应新建连接")
	fresh.mu.Lock()
	freshWrites := len(fresh.writes)
	fresh.mu.Unlock()
	require.Equal(t, 1, freshWrites)
}

func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_ReadFailureReturnsFailoverAfterBoundedRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 2
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 2
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	firstConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.completed","response":{"id":"resp_failover_seed","model":"gpt-5.1","status":"completed","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	retryConn := &openAIWSCaptureConn{}
	dialer := &openAIWSQueueDialer{conns: []openAIWSClientConn{firstConn, retryConn}}
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
		ID:          445,
		Name:        "openai-ingress-read-failover",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, firstMessage, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		state := &SmartRoutingRequestState{
			SmartRoutingActive:      true,
			MainGroupID:             74,
			CurrentGroupID:          74,
			SmartRoutingStartedAt:   time.Now(),
			SmartRoutingMaxAttempts: 4,
			SmartRoutingMaxDuration: time.Minute,
		}
		routingCtx := WithSmartRoutingState(r.Context(), state)
		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(routingCtx, ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	defer wsServer.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	defer func() { _ = clientConn.CloseNow() }()

	writeClientMessage := func(payload string) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		require.NoError(t, clientConn.Write(ctx, coderws.MessageText, []byte(payload)))
	}
	readClientMessage := func() []byte {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, payload, readErr := clientConn.Read(ctx)
		require.NoError(t, readErr)
		return payload
	}

	writeClientMessage(`{"type":"response.create","model":"gpt-5.1","stream":false,"input":[{"type":"input_text","text":"first"}]}`)
	require.Equal(t, "response.completed", gjson.GetBytes(readClientMessage(), "type").String())

	writeClientMessage(`{"type":"response.create","model":"gpt-5.1","stream":false,"previous_response_id":"resp_failover_seed","input":[{"type":"input_text","text":"second"}]}`)

	select {
	case proxyErr := <-serverErrCh:
		require.Error(t, proxyErr)
		var failoverErr *UpstreamFailoverError
		require.ErrorAs(t, proxyErr, &failoverErr)
		require.Equal(t, GatewayFailureReason("openai_ws_read_event"), failoverErr.Reason)
		retryPayload, retryCurrentTurn := OpenAIWSCurrentTurnRetryPayload(proxyErr)
		require.True(t, retryCurrentTurn)
		require.NotEmpty(t, retryPayload)
		require.False(t, gjson.GetBytes(retryPayload, "previous_response_id").Exists())
		require.Equal(t, "gpt-5.1", gjson.GetBytes(retryPayload, "model").String())
		input := gjson.GetBytes(retryPayload, "input")
		require.True(t, input.IsArray())
		require.Len(t, input.Array(), 2)
		require.Contains(t, input.Raw, "first")
		require.Contains(t, input.Raw, "second")
	case <-time.After(5 * time.Second):
		t.Fatal("等待 bounded ingress failover 超时")
	}

	require.Equal(t, 2, dialer.DialCount(), "同账号只允许一次新连接重试，随后应返回 failover")
	firstConn.mu.Lock()
	firstWrites := len(firstConn.writes)
	firstConn.mu.Unlock()
	retryConn.mu.Lock()
	retryWrites := len(retryConn.writes)
	retryConn.mu.Unlock()
	require.Equal(t, 2, firstWrites, "首连接应处理两轮请求")
	require.Equal(t, 1, retryWrites, "同账号重试连接只应处理失败的当前轮")
}

func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_SmartRoutingDropsStagedMetadataBeforeFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 2
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 2
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 2
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 2
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 2

	makeMetadataConn := func(responseID string) *openAIWSCaptureConn {
		return &openAIWSCaptureConn{
			events: [][]byte{
				[]byte(`{"type":"response.created","sequence_number":0,"response":{"id":"` + responseID + `","status":"in_progress"}}`),
				[]byte(`{"type":"response.in_progress","sequence_number":1,"response":{"id":"` + responseID + `","status":"in_progress"}}`),
			},
		}
	}
	firstConn := makeMetadataConn("resp_staged_first")
	retryConn := makeMetadataConn("resp_staged_retry")
	dialer := &openAIWSQueueDialer{conns: []openAIWSClientConn{firstConn, retryConn}}
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
		ID: 449, Name: "openai-ingress-smart-metadata-failover", Platform: PlatformOpenAI,
		Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()
		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req
		_, firstMessage, readErr := conn.Read(r.Context())
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		state := &SmartRoutingRequestState{
			SmartRoutingActive:      true,
			MainGroupID:             73,
			CurrentGroupID:          73,
			SmartRoutingStartedAt:   time.Now(),
			SmartRoutingMaxAttempts: 4,
			SmartRoutingMaxDuration: time.Minute,
		}
		routingCtx := WithSmartRoutingState(r.Context(), state)
		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(routingCtx, ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	defer wsServer.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 2*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	defer func() { _ = clientConn.CloseNow() }()

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 2*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","stream":true,"input":"hello"}`))
	cancelWrite()
	require.NoError(t, err)

	select {
	case proxyErr := <-serverErrCh:
		require.Error(t, proxyErr)
		var failoverErr *UpstreamFailoverError
		require.ErrorAs(t, proxyErr, &failoverErr)
	case <-time.After(6 * time.Second):
		t.Fatal("等待 pre-output 智能路由 failover 超时")
	}

	readCtx, cancelRead := context.WithTimeout(context.Background(), 300*time.Millisecond)
	_, leakedMessage, readErr := clientConn.Read(readCtx)
	cancelRead()
	require.Error(t, readErr, "created/in_progress metadata from failed attempts must remain hidden")
	require.Empty(t, leakedMessage)
	require.Equal(t, 2, dialer.DialCount(), "one bounded same-account retry should precede routed failover")
}

func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_HandshakeFailureReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	dialer := &openAIWSErrorDialer{
		statusCode: http.StatusBadGateway,
		headers:    http.Header{"X-Request-Id": []string{"handshake-req"}},
		err:        errors.New("upstream handshake failed"),
	}
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
		ID:          446,
		Name:        "openai-ingress-handshake-failover",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	serverErrCh := make(chan error, 2)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, firstMessage, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		ctx := r.Context()
		if r.URL.Query().Get("smart") == "1" {
			state := &SmartRoutingRequestState{
				SmartRoutingActive:      true,
				MainGroupID:             75,
				CurrentGroupID:          75,
				SmartRoutingStartedAt:   time.Now(),
				SmartRoutingMaxAttempts: 4,
				SmartRoutingMaxDuration: time.Minute,
			}
			ctx = WithSmartRoutingState(ctx, state)
		}
		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(ctx, ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	defer wsServer.Close()

	runHandshakeFailure := func(smartRouting bool) error {
		target := "ws" + strings.TrimPrefix(wsServer.URL, "http")
		if smartRouting {
			target += "?smart=1"
		}
		dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
		clientConn, _, dialErr := coderws.Dial(dialCtx, target, nil)
		cancelDial()
		require.NoError(t, dialErr)
		defer func() { _ = clientConn.CloseNow() }()

		writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
		writeErr := clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","stream":false,"input":"hello"}`))
		cancelWrite()
		require.NoError(t, writeErr)

		select {
		case proxyErr := <-serverErrCh:
			return proxyErr
		case <-time.After(5 * time.Second):
			t.Fatal("等待 websocket handshake 处理超时")
			return nil
		}
	}

	smartErr := runHandshakeFailure(true)
	require.Error(t, smartErr)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, smartErr, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Equal(t, GatewayFailureReason("openai_ws_upstream_5xx"), failoverErr.Reason)
	require.True(t, failoverErr.ShouldRetryNextAccount())

	ordinaryErr := runHandshakeFailure(false)
	require.Error(t, ordinaryErr)
	require.NotErrorAs(t, ordinaryErr, &failoverErr, "ordinary requests must preserve the raw handshake error")
	var dialErr *openAIWSDialError
	require.ErrorAs(t, ordinaryErr, &dialErr)
	require.Equal(t, http.StatusBadGateway, dialErr.StatusCode)
	require.EqualValues(t, 2, dialer.calls.Load(), "each handshake attempt should dial once without retrying the same upstream")
}

func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_ReadFailureAfterOutputWritesTerminalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 2
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 2
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	upstreamConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","sequence_number":4,"response":{"id":"resp_ingress_partial","model":"gpt-5.1","status":"in_progress"}}`),
			[]byte(`{"type":"response.output_text.delta","sequence_number":5,"response_id":"resp_ingress_partial","delta":"partial output"}`),
		},
	}
	dialer := &openAIWSQueueDialer{conns: []openAIWSClientConn{upstreamConn}}
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
		ID:          447,
		Name:        "openai-ingress-read-failure-after-output",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, firstMessage, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(r.Context(), ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	defer wsServer.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	defer func() { _ = clientConn.CloseNow() }()

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","stream":true,"input":[{"type":"input_text","text":"hello"}]}`))
	cancelWrite()
	require.NoError(t, err)

	readMessage := func() []byte {
		readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelRead()
		messageType, payload, readErr := clientConn.Read(readCtx)
		require.NoError(t, readErr)
		require.Equal(t, coderws.MessageText, messageType)
		return payload
	}
	require.Equal(t, "response.created", gjson.GetBytes(readMessage(), "type").String())
	require.Equal(t, "response.output_text.delta", gjson.GetBytes(readMessage(), "type").String())
	terminal := readMessage()
	require.Equal(t, "response.failed", gjson.GetBytes(terminal, "type").String())
	require.EqualValues(t, 6, gjson.GetBytes(terminal, "sequence_number").Int())
	require.Equal(t, "resp_ingress_partial", gjson.GetBytes(terminal, "response.id").String())
	require.Equal(t, "gpt-5.1", gjson.GetBytes(terminal, "response.model").String())
	require.Equal(t, "upstream_stream_read_error", gjson.GetBytes(terminal, "response.error.code").String())

	select {
	case proxyErr := <-serverErrCh:
		require.Error(t, proxyErr)
		var failoverErr *UpstreamFailoverError
		require.False(t, errors.As(proxyErr, &failoverErr), "partial output must not be replayed through account failover")
	case <-time.After(5 * time.Second):
		t.Fatal("等待 ingress response.failed 终态超时")
	}
	require.Equal(t, 1, dialer.DialCount(), "已写出内容后不得重拨或切换上游")
	upstreamConn.mu.Lock()
	upstreamWrites := len(upstreamConn.writes)
	upstreamClosed := upstreamConn.closed
	upstreamConn.mu.Unlock()
	require.Equal(t, 1, upstreamWrites, "当前轮只应提交一次上游请求")
	require.True(t, upstreamClosed, "断开的上游连接必须从连接池淘汰")
}

func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_TurnBudgetRejectionReleasesLease(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 2
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 2
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	const accountID = int64(448)
	const groupID = int64(72)
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		CurrentGroupID:          groupID,
		SmartRoutingMaxAttempts: 5,
		siteAttempts: map[smartRoutingSiteAttemptKey]int{
			{groupID: groupID, accountID: accountID}: SmartRoutingMaxSiteAttempts,
		},
	}
	upstreamConn := &openAIWSCaptureConn{}
	dialer := &openAIWSQueueDialer{conns: []openAIWSClientConn{upstreamConn}}
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
		ID:          accountID,
		Name:        "openai-ingress-budget-rejection",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, firstMessage, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		smartCtx := WithSmartRoutingState(r.Context(), state)
		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(smartCtx, ginCtx, conn, account, "sk-test", firstMessage, nil)
	}))
	defer wsServer.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	defer func() { _ = clientConn.CloseNow() }()

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","stream":true,"input":"hello"}`))
	cancelWrite()
	require.NoError(t, err)

	select {
	case proxyErr := <-serverErrCh:
		require.Error(t, proxyErr)
		var failoverErr *UpstreamFailoverError
		require.ErrorAs(t, proxyErr, &failoverErr)
		require.Equal(t, SmartRoutingSiteAttemptBudgetExceededReason, failoverErr.Reason)
	case <-time.After(5 * time.Second):
		t.Fatal("等待智能路由预算拒绝超时")
	}

	require.Equal(t, 1, dialer.DialCount(), "预算拒绝时不应因新 lease 再发起重拨")
	upstreamConn.mu.Lock()
	upstreamWrites := len(upstreamConn.writes)
	upstreamClosed := upstreamConn.closed
	upstreamConn.mu.Unlock()
	require.Zero(t, upstreamWrites, "预算耗尽后不得发送当前轮请求")
	require.True(t, upstreamClosed, "预算拒绝后已取得的 lease 必须从连接池淘汰并释放")
}
