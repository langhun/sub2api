package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// openAIWSSmartRoutingAccountRepoStub intentionally scopes every scheduler
// query by group ID. The older WS fixtures returned the same account for every
// group, which could hide a primary-empty/candidate-available regression.
type openAIWSSmartRoutingAccountRepoStub struct {
	service.AccountRepository

	mu         sync.Mutex
	accounts   []service.Account
	groupCalls []int64
}

func (r *openAIWSSmartRoutingAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]service.Account, error) {
	r.mu.Lock()
	r.groupCalls = append(r.groupCalls, groupID)
	r.mu.Unlock()

	out := make([]service.Account, 0)
	for _, account := range r.accounts {
		if account.Platform != platform || !account.IsSchedulable() {
			continue
		}
		for _, boundGroupID := range account.GroupIDs {
			if boundGroupID == groupID {
				out = append(out, account)
				break
			}
		}
	}
	return out, nil
}

func (r *openAIWSSmartRoutingAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]service.Account, error) {
	out := make([]service.Account, 0)
	for _, platform := range platforms {
		accounts, err := r.ListSchedulableByGroupIDAndPlatform(ctx, groupID, platform)
		if err != nil {
			return nil, err
		}
		out = append(out, accounts...)
	}
	return out, nil
}

func (r *openAIWSSmartRoutingAccountRepoStub) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	out := make([]service.Account, 0)
	for _, account := range r.accounts {
		if account.Platform == platform && account.IsSchedulable() {
			out = append(out, account)
		}
	}
	return out, nil
}

func (r *openAIWSSmartRoutingAccountRepoStub) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for _, account := range r.accounts {
		if account.ID == id {
			copy := account
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *openAIWSSmartRoutingAccountRepoStub) calls() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int64(nil), r.groupCalls...)
}

type openAIWSSmartRoutingGroupRepoStub struct {
	service.GroupRepository
	groups map[int64]*service.Group
}

func (r *openAIWSSmartRoutingGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*service.Group, error) {
	return r.groups[id], nil
}

func (r *openAIWSSmartRoutingGroupRepoStub) GetByID(ctx context.Context, id int64) (*service.Group, error) {
	return r.GetByIDLite(ctx, id)
}

type openAIWSSmartRoutingChannelRepoStub struct {
	service.ChannelRepository
}

func (r *openAIWSSmartRoutingChannelRepoStub) ListAll(context.Context) ([]service.Channel, error) {
	return nil, nil
}

func (r *openAIWSSmartRoutingChannelRepoStub) GetGroupPlatforms(context.Context, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}

type openAIWSSmartRoutingUserRepoStub struct {
	service.UserRepository
}

func (r *openAIWSSmartRoutingUserRepoStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	return &service.User{ID: id, Status: service.StatusActive}, nil
}

type openAIWSSmartRoutingResolver struct {
	primaryGroupID   int64
	candidateGroupID int64
	calls            atomic.Int32
}

func (r *openAIWSSmartRoutingResolver) Resolve(ctx context.Context, _ *service.APIKey, model, _ string, _ map[int64]struct{}) (*service.SmartRouteDecision, error) {
	r.calls.Add(1)
	if state := service.SmartRoutingStateFromContext(ctx); state != nil {
		state.SmartRoutingActive = true
		state.RequestedModel = model
		state.Strategy = "speed"
		if _, exhausted := state.ExhaustedGroups[r.primaryGroupID]; !exhausted {
			return &service.SmartRouteDecision{
				GroupID: r.primaryGroupID,
				Reason:  "try primary group first",
			}, nil
		}
	}
	return &service.SmartRouteDecision{
		GroupID: r.candidateGroupID,
		Reason:  "candidate has available Responses WebSocket account",
	}, nil
}

type openAIWSSmartRoutingUpstream struct {
	url          string
	firstPayload chan []byte
	connections  atomic.Int32
}

func newOpenAIWSSmartRoutingUpstream(t *testing.T) *openAIWSSmartRoutingUpstream {
	t.Helper()
	upstream := &openAIWSSmartRoutingUpstream{
		firstPayload: make(chan []byte, 1),
	}
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream.connections.Add(1)
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, payload, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			return
		}
		select {
		case upstream.firstPayload <- append([]byte(nil), payload...):
		default:
		}

		writeCtx, cancelWrite := context.WithTimeout(r.Context(), 3*time.Second)
		_ = conn.Write(writeCtx, coderws.MessageText, []byte(
			`{"type":"response.completed","response":{"id":"resp_ws_smart_routing_candidate","model":"gpt-5.1","status":"completed","usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}}`,
		))
		cancelWrite()
		_, _, _ = conn.Read(r.Context())
	}))
	t.Cleanup(upstreamServer.Close)
	upstream.url = upstreamServer.URL
	return upstream
}

func TestOpenAIResponsesWebSocketSmartRoutingPrimaryEmptyUsesCandidateGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		primaryGroupID   int64 = 5101
		candidateGroupID int64 = 5102
		userID           int64 = 6101
		apiKeyID         int64 = 7101
		accountID        int64 = 8101
	)

	primaryGroup := &service.Group{
		ID:               primaryGroupID,
		Platform:         service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeStandard,
		Status:           service.StatusActive,
	}
	candidateGroup := &service.Group{
		ID:               candidateGroupID,
		Platform:         service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeStandard,
		Status:           service.StatusActive,
	}
	upstream := newOpenAIWSSmartRoutingUpstream(t)
	accountRepo := &openAIWSSmartRoutingAccountRepoStub{
		accounts: []service.Account{{
			ID:          accountID,
			Name:        "candidate-ws-account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{candidateGroupID},
			Credentials: map[string]any{"api_key": "sk-candidate", "base_url": upstream.url},
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
				"openai_apikey_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
			},
		}},
	}
	groupRepo := &openAIWSSmartRoutingGroupRepoStub{
		groups: map[int64]*service.Group{
			primaryGroupID:   primaryGroup,
			candidateGroupID: candidateGroup,
		},
	}
	channelService := service.NewChannelService(
		&openAIWSSmartRoutingChannelRepoStub{},
		groupRepo,
		nil,
		nil,
		nil,
	)
	resolver := &openAIWSSmartRoutingResolver{
		primaryGroupID:   primaryGroupID,
		candidateGroupID: candidateGroupID,
	}

	cfg := &config.Config{RunMode: config.RunModeSimple}
	cfg.Default.RateMultiplier = 1
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3
	cfg.Gateway.MaxAccountSwitches = 2

	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		nil,
		nil,
		&openAIWSSmartRoutingUserRepoStub{},
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		service.NewBillingService(cfg, nil),
		nil,
		billingCache,
		nil,
		&service.DeferredService{},
		nil,
		nil,
		nil,
		channelService,
		nil,
		nil,
		nil,
	)
	gatewayService.SetSmartRoutingResolver(resolver)

	concurrencyCache := &concurrencyCacheMock{
		acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) {
			return true, nil
		},
		acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) {
			return true, nil
		},
	}
	handler := &OpenAIGatewayHandler{
		cfg:                 cfg,
		gatewayService:      gatewayService,
		billingCacheService: billingCache,
		apiKeyService:       &service.APIKeyService{},
		concurrencyHelper:   NewConcurrencyHelper(service.NewConcurrencyService(concurrencyCache), SSEPingFormatNone, time.Second),
		maxAccountSwitches:  2,
	}

	apiKey := &service.APIKey{
		ID:      apiKeyID,
		UserID:  userID,
		GroupID: func() *int64 { id := primaryGroupID; return &id }(),
		User:    &service.User{ID: userID, Status: service.StatusActive},
		Group:   primaryGroup,
	}
	state := &service.SmartRoutingRequestState{
		MainGroupID:          primaryGroupID,
		CurrentGroupID:       primaryGroupID,
		EndpointType:         "responses",
		MaxGroupSwitches:     2,
		TriedGroups:          map[int64]struct{}{},
		PrimaryGroupSnapshot: primaryGroup,
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID, Concurrency: 1})
		ctx := service.WithAPIKeyContext(c.Request.Context(), apiKey)
		ctx = service.WithSmartRoutingState(ctx, state)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.GET("/openai/v1/responses", handler.ResponsesWebSocket)
	server := httptest.NewServer(router)
	defer server.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(
		dialCtx,
		"ws"+strings.TrimPrefix(server.URL, "http")+"/openai/v1/responses",
		&coderws.DialOptions{CompressionMode: coderws.CompressionContextTakeover},
	)
	cancelDial()
	require.NoError(t, err)
	defer func() { _ = clientConn.CloseNow() }()

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(
		`{"type":"response.create","model":"gpt-5.1","stream":false}`,
	))
	cancelWrite()
	require.NoError(t, err)

	readCtx, cancelRead := context.WithTimeout(context.Background(), 5*time.Second)
	_, event, err := clientConn.Read(readCtx)
	cancelRead()
	require.NoError(t, err)
	require.Equal(t, "response.completed", gjson.GetBytes(event, "type").String())
	require.Equal(t, "resp_ws_smart_routing_candidate", gjson.GetBytes(event, "response.id").String())

	select {
	case payload := <-upstream.firstPayload:
		require.Equal(t, "response.create", gjson.GetBytes(payload, "type").String())
		require.Equal(t, "gpt-5.1", gjson.GetBytes(payload, "model").String())
	case <-time.After(3 * time.Second):
		t.Fatal("candidate upstream did not receive the first response.create frame")
	}

	require.Equal(t, int32(1), upstream.connections.Load())
	require.Equal(t, int32(2), resolver.calls.Load(), "an empty primary group should trigger one candidate-group resolution")
	require.Contains(t, accountRepo.calls(), primaryGroupID)
	require.Contains(t, accountRepo.calls(), candidateGroupID)
}
