//go:build unit

package handler

import (
	"context"
	"errors"
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

type routingForwardUsers struct {
	service.UserRepository
}

func (routingForwardUsers) GetByID(_ context.Context, id int64) (*service.User, error) {
	return &service.User{ID: id, Status: service.StatusActive}, nil
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
	attempts             []service.SmartRoutingAttempt
	health               *smartrouting.Module
	metrics              map[int64]service.SmartRoutingAccountMetric
	flipMetricsOnFailure bool
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
	if r.flipMetricsOnFailure && !a.Success && !a.HealthOnly {
		r.metrics = map[int64]service.SmartRoutingAccountMetric{
			1: {SpeedMs: 10_000, ErrorRate: 1},
			2: {SpeedMs: 100, ErrorRate: 0},
		}
	}
	if r.health != nil {
		r.health.ObserveRoutingAttempt(a)
	}
}

func (r *routingForwardResolver) RoutingAccountMetrics(context.Context, int64, string, string, bool) map[int64]service.SmartRoutingAccountMetric {
	return r.metrics
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

func (r *routingForwardResolver) SuppressSameAccountRetry(err *service.UpstreamFailoverError) bool {
	return r.health != nil && r.health.SuppressSameAccountRetry(err)
}

type routingForwardUpstream struct {
	service.HTTPUpstream
	calls                     []int64
	partial                   bool
	disconnectBeforeCompleted bool
	disconnectAfterOutput     bool
	failureStatus             int
	succeedAfter              int
	failedSiteCalls           int
}

type routingForwardReadErrorBody struct {
	*strings.Reader
	err  error
	done bool
}

func (r *routingForwardReadErrorBody) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF && !r.done {
		r.done = true
		if r.err != nil {
			return 0, r.err
		}
	}
	return n, err
}

func (r *routingForwardReadErrorBody) Close() error {
	return nil
}

func (u *routingForwardUpstream) Do(_ *http.Request, _ string, id int64, _ int) (*http.Response, error) {
	u.calls = append(u.calls, id)
	status, body := 200, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_routed\",\"object\":\"response\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"
	if id == 1 {
		u.failedSiteCalls++
		if u.disconnectBeforeCompleted || u.disconnectAfterOutput {
			if u.disconnectAfterOutput {
				body = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"
			} else {
				body = "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_disconnected\",\"status\":\"in_progress\"}}\n\n"
			}
			return &http.Response{
				StatusCode: status,
				Header:     http.Header{"Content-Type": {"text/event-stream"}},
				Body:       &routingForwardReadErrorBody{Reader: strings.NewReader(body), err: errors.New("stream closed before response.completed")},
			}, nil
		}
		if u.succeedAfter > 0 && u.failedSiteCalls > u.succeedAfter {
			return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		}
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

func TestSmartRoutingPoolRecoveryStaysOnWebsiteWithoutCooldown(t *testing.T) {
	// A routed pool website gets one fast same-site retry. The first request
	// recovers on that retry; a later request can use the recovered website
	// immediately without a cooldown or a forced cross-group switch.
	upstream := &routingForwardUpstream{failureStatus: 503, succeedAfter: 1}
	h := newOpenAIResponsesFailoverTestHandler(t, upstream)
	cfg := &config.Config{RunMode: config.RunModeSimple}
	groups := routingForwardGroups{groups: map[int64]*service.Group{
		3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
	}}
	accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
		{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"api_key": "test-one", "pool_mode": true, "pool_mode_retry_status_codes": []any{float64(503)}}},
	}}}
	billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
	h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, routingForwardUsers{}, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
	resolver := &routingForwardResolver{health: &smartrouting.Module{}}
	h.gatewayService.SetSmartRoutingResolver(resolver)

	for range 2 {
		c, rec := newOpenAIResponsesFailoverTestContext(t, context.Background())
		key, _ := c.Get(string(middleware.ContextKeyAPIKey))
		apiKey := key.(*service.APIKey)
		apiKey.Group = groups.groups[3131]
		ctx := service.WithAPIKeyContext(c.Request.Context(), apiKey)
		state := &service.SmartRoutingRequestState{MainGroupID: 3131, CurrentGroupID: 3131, EndpointType: "responses", MaxGroupSwitches: 2, PrimaryGroupSnapshot: apiKey.Group}
		c.Request = c.Request.WithContext(service.WithSmartRoutingState(ctx, state))
		h.Responses(c)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "resp_routed")
	}

	require.Equal(t, []int64{1, 1, 1}, upstream.calls, "a routed pool website may recover on one same-site retry and remain available")
	require.Len(t, resolver.attempts, 3)
	require.False(t, resolver.attempts[0].HealthFinal)
	require.True(t, resolver.attempts[1].Success)
	require.True(t, resolver.attempts[2].Success)
	for _, attempt := range resolver.attempts {
		require.False(t, attempt.HealthOnly, "a recovered website must not receive a terminal failure notification")
	}
}

func TestSmartRoutingSpeedPolicyCannotStealPoolRetry(t *testing.T) {
	upstream := &routingForwardUpstream{failureStatus: 503, succeedAfter: 2}
	h := newOpenAIResponsesFailoverTestHandler(t, upstream)
	cfg := &config.Config{RunMode: config.RunModeSimple}
	groups := routingForwardGroups{groups: map[int64]*service.Group{
		3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
	}}
	accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
		{ID: 1, Priority: 0, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"api_key": "test-one", "pool_mode": true, "pool_mode_retry_status_codes": []any{float64(503)}}},
		{ID: 2, Priority: 0, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"api_key": "test-two", "pool_mode": true}},
	}}}
	billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
	h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, routingForwardUsers{}, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
	resolver := &routingForwardResolver{
		health:               &smartrouting.Module{},
		flipMetricsOnFailure: true,
		metrics: map[int64]service.SmartRoutingAccountMetric{
			1: {SpeedMs: 100, ErrorRate: 0},
			2: {SpeedMs: 10_000, ErrorRate: 0},
		},
	}
	h.gatewayService.SetSmartRoutingResolver(resolver)

	c, rec := newOpenAIResponsesFailoverTestContext(t, context.Background())
	key, _ := c.Get(string(middleware.ContextKeyAPIKey))
	apiKey := key.(*service.APIKey)
	apiKey.Group = groups.groups[3131]
	ctx := service.WithAPIKeyContext(c.Request.Context(), apiKey)
	state := &service.SmartRoutingRequestState{MainGroupID: 3131, CurrentGroupID: 3131, EndpointType: "responses", MaxGroupSwitches: 2, PrimaryGroupSnapshot: apiKey.Group}
	c.Request = c.Request.WithContext(service.WithSmartRoutingState(ctx, state))
	h.Responses(c)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, []int64{1, 1, 2}, upstream.calls, "speed re-scoring must not steal the one same-site pool retry, but the next failure must switch websites")
	// The exhausted website also emits one health-only terminal observation
	// before the scheduler selects the next website.
	require.Len(t, resolver.attempts, 4)
	require.False(t, resolver.attempts[0].Success)
	require.False(t, resolver.attempts[1].Success)
	require.True(t, resolver.attempts[2].HealthFinal)
	require.True(t, resolver.attempts[2].HealthOnly)
	require.True(t, resolver.attempts[3].Success)
	require.Equal(t, int64(2), resolver.attempts[3].AccountID)
	require.Zero(t, state.RetryAccountID)
}

func TestSmartRoutingPoolRetriesWebsiteBeforeQuarantine(t *testing.T) {
	for _, tc := range []struct {
		name                string
		status              int
		customRetryCodes    []any
		wantFailedSiteCalls int
	}{
		{name: "402 is website failure by default", status: 402, wantFailedSiteCalls: 1},
		{name: "401 releases website after one same-site retry", status: 401, wantFailedSiteCalls: 2},
		{name: "403 balance exhaustion switches website immediately", status: 403, wantFailedSiteCalls: 1},
		{name: "429 releases website after one same-site retry", status: 429, wantFailedSiteCalls: 2},
		{name: "configured 503 releases website after one same-site retry", status: 503, customRetryCodes: []any{float64(503)}, wantFailedSiteCalls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := &routingForwardUpstream{failureStatus: tc.status}
			h := newOpenAIResponsesFailoverTestHandler(t, upstream)
			cfg := &config.Config{RunMode: config.RunModeSimple}
			groups := routingForwardGroups{groups: map[int64]*service.Group{
				3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
				3132: {ID: 3132, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
			}}
			failedSiteCredentials := map[string]any{"api_key": "test-one", "pool_mode": true}
			if tc.customRetryCodes != nil {
				failedSiteCredentials["pool_mode_retry_status_codes"] = tc.customRetryCodes
			}
			accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
				{ID: 1, Priority: 0, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: failedSiteCredentials},
				{ID: 2, Priority: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"api_key": "test-two", "pool_mode": true}},
			}}}
			billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billing.Stop)
			channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
			h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, routingForwardUsers{}, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
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
			wantCalls := make([]int64, 0, tc.wantFailedSiteCalls+2)
			for range tc.wantFailedSiteCalls {
				wantCalls = append(wantCalls, 1)
			}
			wantCalls = append(wantCalls, 2, 2)
			require.Equal(t, wantCalls, upstream.calls, "retry a website once, then move to the next website and keep the next request on the healthy website")
			wantAttempts := tc.wantFailedSiteCalls + 2
			if tc.wantFailedSiteCalls > 1 {
				wantAttempts++ // terminal website-health notification; not SQL telemetry
			}
			if tc.status == 403 {
				wantAttempts++ // deterministic balance failure still records a health-only terminal event
			}
			require.Len(t, resolver.attempts, wantAttempts, "local admission rejection is not an upstream failure")
			require.Equal(t, tc.status, resolver.attempts[0].StatusCode)
			if tc.wantFailedSiteCalls > 1 {
				for i := 0; i < tc.wantFailedSiteCalls; i++ {
					require.False(t, resolver.attempts[i].HealthFinal, "pool member failures remain attempt telemetry")
				}
				require.True(t, resolver.attempts[tc.wantFailedSiteCalls].HealthFinal)
				require.True(t, resolver.attempts[tc.wantFailedSiteCalls].HealthOnly)
			} else if tc.status == 403 {
				require.False(t, resolver.attempts[0].HealthFinal, "the first 403 remains attempt telemetry")
				require.True(t, resolver.attempts[1].HealthFinal)
				require.True(t, resolver.attempts[1].HealthOnly)
			} else {
				require.True(t, resolver.attempts[0].HealthFinal, "non-retryable website failures quarantine immediately")
			}
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
				3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
				3132: {ID: 3132, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
			}}
			accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
				{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"access_token": "test-one"}},
				{ID: 2, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3132}, Credentials: map[string]any{"access_token": "test-two"}},
			}}}
			billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billing.Stop)
			channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
			h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, routingForwardUsers{}, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
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

func TestSmartRoutingResponsesDisconnectBeforeCompletionFailsOver(t *testing.T) {
	upstream := &routingForwardUpstream{disconnectBeforeCompleted: true}
	h := newOpenAIResponsesFailoverTestHandler(t, upstream)
	cfg := &config.Config{RunMode: config.RunModeSimple}
	groups := routingForwardGroups{groups: map[int64]*service.Group{
		3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
		3132: {ID: 3132, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
	}}
	accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
		{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"access_token": "test-one"}},
		{ID: 2, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3132}, Credentials: map[string]any{"access_token": "test-two"}},
	}}}
	billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
	h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, routingForwardUsers{}, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
	resolver := &routingForwardResolver{}
	h.gatewayService.SetSmartRoutingResolver(resolver)

	c, rec := newOpenAIResponsesFailoverTestContext(t, context.Background())
	key, _ := c.Get(string(middleware.ContextKeyAPIKey))
	apiKey := key.(*service.APIKey)
	apiKey.Group = groups.groups[3131]
	c.Request.Body = io.NopCloser(strings.NewReader(`{"model":"gpt-5.1","stream":true,"input":"hello"}`))
	ctx := service.WithAPIKeyContext(c.Request.Context(), apiKey)
	state := &service.SmartRoutingRequestState{
		MainGroupID:          3131,
		CurrentGroupID:       3131,
		EndpointType:         "responses",
		MaxGroupSwitches:     2,
		PrimaryGroupSnapshot: apiKey.Group,
	}
	c.Request = c.Request.WithContext(service.WithSmartRoutingState(ctx, state))

	h.Responses(c)

	require.Equal(t, []int64{1, 2}, upstream.calls, "a pre-completion disconnect before semantic output must fail over to the next candidate")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "resp_routed")
	require.NotContains(t, rec.Body.String(), "stream closed before response.completed")
	require.NotContains(t, rec.Body.String(), "response.failed")
	require.Len(t, resolver.attempts, 2)
	require.False(t, resolver.attempts[0].Success)
	require.True(t, resolver.attempts[1].Success)
	require.Equal(t, int64(3132), resolver.attempts[1].GroupID)
}

func TestSmartRoutingResponsesDisconnectAfterTokenDoesNotReplay(t *testing.T) {
	upstream := &routingForwardUpstream{disconnectAfterOutput: true}
	h := newOpenAIResponsesFailoverTestHandler(t, upstream)
	cfg := &config.Config{RunMode: config.RunModeSimple}
	groups := routingForwardGroups{groups: map[int64]*service.Group{
		3131: {ID: 3131, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
		3132: {ID: 3132, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard},
	}}
	accounts := routingForwardAccounts{openAIImagesFailoverAccountRepo{accounts: []service.Account{
		{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3131}, Credentials: map[string]any{"access_token": "test-one"}},
		{ID: 2, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, GroupIDs: []int64{3132}, Credentials: map[string]any{"access_token": "test-two"}},
	}}}
	billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{}, groups, nil, nil, nil)
	h.gatewayService = service.NewOpenAIGatewayService(accounts, nil, nil, routingForwardUsers{}, nil, nil, nil, cfg, nil, nil, nil, nil, billing, upstream, nil, nil, nil, nil, channels, nil, nil, nil)
	resolver := &routingForwardResolver{}
	h.gatewayService.SetSmartRoutingResolver(resolver)

	c, rec := newOpenAIResponsesFailoverTestContext(t, context.Background())
	key, _ := c.Get(string(middleware.ContextKeyAPIKey))
	apiKey := key.(*service.APIKey)
	apiKey.Group = groups.groups[3131]
	c.Request.Body = io.NopCloser(strings.NewReader(`{"model":"gpt-5.1","stream":true,"input":"hello"}`))
	ctx := service.WithAPIKeyContext(c.Request.Context(), apiKey)
	state := &service.SmartRoutingRequestState{
		MainGroupID:          3131,
		CurrentGroupID:       3131,
		EndpointType:         "responses",
		MaxGroupSwitches:     2,
		PrimaryGroupSnapshot: apiKey.Group,
	}
	c.Request = c.Request.WithContext(service.WithSmartRoutingState(ctx, state))

	h.Responses(c)

	require.Equal(t, []int64{1}, upstream.calls, "once semantic output is visible, replaying the request would duplicate downstream output")
	require.NotContains(t, rec.Body.String(), "resp_routed")
	require.Contains(t, rec.Body.String(), "partial")
	require.Contains(t, rec.Body.String(), "response.failed", "a Responses stream must terminate with a protocol terminal event")
	require.NotContains(t, rec.Body.String(), "event: error\n", "legacy error events make Codex report stream closed before response.completed")
	require.GreaterOrEqual(t, len(resolver.attempts), 1)
	require.False(t, resolver.attempts[0].Success)
}
