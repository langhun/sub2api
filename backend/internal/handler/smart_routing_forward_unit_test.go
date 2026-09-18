//go:build unit

package handler

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	smartrouting "github.com/Wei-Shaw/sub2api/internal/custom/modules/smart-routing"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type routingForwardGroups struct {
	service.GroupRepository
	groups map[int64]*service.Group
}

func (r routingForwardGroups) GetByIDLite(_ context.Context, id int64) (*service.Group, error) {
	return r.groups[id], nil
}
func (r routingForwardGroups) GetByID(ctx context.Context, id int64) (*service.Group, error) {
	return r.GetByIDLite(ctx, id)
}

type routingForwardAccounts struct {
	openAIImagesFailoverAccountRepo
}

func (r routingForwardAccounts) ListSchedulableByGroupIDAndPlatform(_ context.Context, id int64, platform string) ([]service.Account, error) {
	var accounts []service.Account
	for _, a := range r.accounts {
		if a.Platform == platform && len(a.GroupIDs) == 1 && a.GroupIDs[0] == id {
			accounts = append(accounts, a)
		}
	}
	return accounts, nil
}

type routingForwardResolver struct {
	attempts []service.SmartRoutingAttempt
	health   *smartrouting.Module
}

func (r *routingForwardResolver) Resolve(ctx context.Context, key *service.APIKey, model, _ string, excluded map[int64]struct{}) (*service.SmartRouteDecision, error) {
	s := service.SmartRoutingStateFromContext(ctx)
	s.SmartRoutingActive, s.Observer, s.RequestedModel = true, r, model
	s.Strategy = "speed"
	if _, exhausted := s.ExhaustedGroups[3131]; exhausted {
		if _, exhausted = s.ExhaustedGroups[3132]; exhausted {
			s.NoRoute = true
			return nil, nil
		}
		return &service.SmartRouteDecision{GroupID: 3132}, nil
	}
	return &service.SmartRouteDecision{GroupID: 3131}, nil
}
func (r *routingForwardResolver) ObserveRoutingAttempt(a service.SmartRoutingAttempt) {
	r.attempts = append(r.attempts, a)
	if r.health != nil {
		r.health.ObserveRoutingAttempt(a)
	}
}

func (r *routingForwardResolver) AcquireRoutingAccount(ctx context.Context, id int64, model, endpoint string, stream bool) (string, bool) {
	if r.health == nil {
		return "", true
	}
	return r.health.AcquireRoutingAccount(ctx, id, model, endpoint, stream)
}

func (r *routingForwardResolver) ReleaseRoutingAccount(id int64, model, endpoint string, stream bool, token string) {
	if r.health != nil {
		r.health.ReleaseRoutingAccount(id, model, endpoint, stream, token)
	}
}

type routingForwardUpstream struct {
	service.HTTPUpstream
	calls         []int64
	partial       bool
	failureStatus int
}

func (u *routingForwardUpstream) Do(_ *http.Request, _ string, id int64, _ int) (*http.Response, error) {
	u.calls = append(u.calls, id)
	status, body := 200, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_routed\",\"object\":\"response\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"
	if id == 1 {
		status, body = 503, `{"error":{"type":"server_error","message":"unavailable"}}`
		if u.failureStatus > 0 {
			status = u.failureStatus
			body = `{"error":{"type":"insufficient_quota","message":"upstream account balance exhausted"}}`
		}
		if u.partial {
			status = 200
			body = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"visible\"}\n\ndata: {\"type\":\"error\",\"error\":{\"type\":\"server_error\",\"message\":\"broken\"}}\n\n"
		}
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestSmartRoutingPoolFailureNotRetriedOnNextRequest(t *testing.T) {
	for _, status := range []int{402, 403, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			upstream := &routingForwardUpstream{failureStatus: status}
			h := newOpenAIResponsesFailoverTestHandler(t, upstream)
			cfg := &config.Config{RunMode: config.RunModeSimple}
			groups := routingForwardGroups{groups: map[int64]*service.Group{
				3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive},
				3132: {ID: 3132, Platform: service.PlatformOpenAI, Status: service.StatusActive},
			}}
			accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
				{ID: 1, Priority: 0, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"api_key": "test-one", "pool_mode": true}},
				{ID: 2, Priority: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"api_key": "test-two", "pool_mode": true}},
			}}}
			billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billing.Stop)
			channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
			h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
			resolver := &routingForwardResolver{health: &smartrouting.Module{}}
			h.gatewayService.SetSmartRoutingResolver(resolver)
			for range 2 {
				c, rec := newOpenAIResponsesFailoverTestContext(t, context.Background())
				key, _ := c.Get(string(middleware.ContextKeyAPIKey))
				apiKey, ok := key.(*service.APIKey)
				require.True(t, ok)
				apiKey.Group = groups.groups[3131]
				ctx := service.WithAPIKeyContext(c.Request.Context(), apiKey)
				state := &service.SmartRoutingRequestState{MainGroupID: 3131, CurrentGroupID: 3131, EndpointType: "responses", MaxGroupSwitches: 2, PrimaryGroupSnapshot: apiKey.Group}
				c.Request = c.Request.WithContext(service.WithSmartRoutingState(ctx, state))
				h.Responses(c)
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				require.Contains(t, rec.Body.String(), "resp_routed")
			}
			require.Equal(t, []int64{1, 2, 2}, upstream.calls, "next request must skip the failed pool account and use its healthy peer")
			require.Len(t, resolver.attempts, 3, "local admission rejection is not an upstream failure")
			require.Equal(t, status, resolver.attempts[0].StatusCode)
		})
	}
}

func TestSmartRoutingResponsesRealForwardFailover(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "before_output", true: "after_output"}[partial], func(t *testing.T) {
			upstream := &routingForwardUpstream{partial: partial}
			h := newOpenAIResponsesFailoverTestHandler(t, upstream)
			cfg := &config.Config{RunMode: config.RunModeSimple}
			groups := routingForwardGroups{groups: map[int64]*service.Group{
				3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive},
				3132: {ID: 3132, Platform: service.PlatformOpenAI, Status: service.StatusActive},
			}}
			accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
				{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"access_token": "test-one"}},
				{ID: 2, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3132}, Credentials: map[string]any{"access_token": "test-two"}},
			}}}
			billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billing.Stop)
			channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
			h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
			resolver := &routingForwardResolver{}
			h.gatewayService.SetSmartRoutingResolver(resolver)
			c, rec := newOpenAIResponsesFailoverTestContext(t, context.Background())
			key, _ := c.Get(string(middleware.ContextKeyAPIKey))
			apiKey := key.(*service.APIKey)
			apiKey.Group = groups.groups[3131]
			ctx := service.WithAPIKeyContext(c.Request.Context(), apiKey)
			state := &service.SmartRoutingRequestState{MainGroupID: 3131, CurrentGroupID: 3131, EndpointType: "responses", MaxGroupSwitches: 2, PrimaryGroupSnapshot: apiKey.Group}
			c.Request = c.Request.WithContext(service.WithSmartRoutingState(ctx, state))
			if partial {
				c.Request.Body = io.NopCloser(strings.NewReader(`{"model":"gpt-5.1","stream":true,"input":"hello"}`))
			}
			h.Responses(c)
			if partial {
				require.Equal(t, []int64{1}, upstream.calls, "visible output must not be replayed on a candidate")
				require.Contains(t, rec.Body.String(), "visible")
			} else {
				require.Equal(t, []int64{1, 2}, upstream.calls, rec.Body.String())
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				require.Len(t, resolver.attempts, 2)
				require.False(t, resolver.attempts[0].Success)
				require.True(t, resolver.attempts[1].Success)
				require.Equal(t, int64(3132), resolver.attempts[1].GroupID)
			}
		})
	}
}
