package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type smartRoutingBudgetHTTPUpstream struct {
	sends       int
	lastRequest *http.Request
}

func (u *smartRoutingBudgetHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.sends++
	u.lastRequest = req
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}, nil
}

func (u *smartRoutingBudgetHTTPUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.sends++
	u.lastRequest = req
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}, nil
}

func TestSmartRoutingOpenAIGatewayRegistersRedirectBudgetForHTTPAndWSTurn(t *testing.T) {
	testCases := []struct {
		name              string
		forward           func(*OpenAIGatewayService, *http.Request, string, *Account) (*http.Response, error)
		reserve           func(context.Context, int64) error
		wantTotalAttempts int
	}{
		{
			name:              "http request",
			forward:           (*OpenAIGatewayService).doOpenAIUpstream,
			reserve:           ReserveSmartRoutingOpenAIRequestAttempt,
			wantTotalAttempts: 2,
		},
		{
			name:              "websocket turn",
			forward:           (*OpenAIGatewayService).doOpenAIWSTurnUpstream,
			reserve:           ReserveSmartRoutingOpenAIWSTurnAttempt,
			wantTotalAttempts: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := &smartRoutingBudgetHTTPUpstream{}
			gateway := &OpenAIGatewayService{httpUpstream: upstream}
			state := &SmartRoutingRequestState{
				SmartRoutingActive:      true,
				MainGroupID:             10,
				CurrentGroupID:          20,
				SmartRoutingMaxAttempts: 8,
				SmartRoutingMaxDuration: time.Minute,
			}
			ctx := WithSmartRoutingState(context.Background(), state)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://upstream.test/v1/responses", http.NoBody)
			require.NoError(t, err)

			resp, err := tc.forward(gateway, req, "", &Account{ID: 31, Platform: PlatformOpenAI})
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, upstream.lastRequest)
			reserver := HTTPUpstreamRedirectAttemptReserverFromContext(upstream.lastRequest.Context())
			require.NotNil(t, reserver, "the native HTTP transport must receive a request-scoped redirect reserver")
			require.NoError(t, reserver(upstream.lastRequest.Context()), "one 307/308 replay remains within the two-send site cap")
			err = reserver(upstream.lastRequest.Context())
			var budgetErr *UpstreamFailoverError
			require.ErrorAs(t, err, &budgetErr)
			require.Equal(t, SmartRoutingSiteAttemptBudgetExceededReason, budgetErr.Reason)
			require.Equal(t, tc.wantTotalAttempts, state.TotalAttempts)
		})
	}
}

func TestOrdinaryOpenAIGatewayRequestDoesNotRegisterRedirectBudget(t *testing.T) {
	upstream := &smartRoutingBudgetHTTPUpstream{}
	gateway := &OpenAIGatewayService{httpUpstream: upstream}
	req, err := http.NewRequest(http.MethodPost, "https://upstream.test/v1/responses", http.NoBody)
	require.NoError(t, err)

	_, err = gateway.doOpenAIUpstream(req, "", &Account{ID: 31, Platform: PlatformOpenAI})
	require.NoError(t, err)
	require.NotNil(t, upstream.lastRequest)
	require.Nil(t, HTTPUpstreamRedirectAttemptReserverFromContext(upstream.lastRequest.Context()), "ordinary single-group requests retain the prior redirect path")
}

func TestSmartRoutingOpenAIRequestBudgetCapsRealSendsPerGroupAndAccount(t *testing.T) {
	ctx := WithSmartRoutingState(context.Background(), &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          20,
		SmartRoutingMaxAttempts: 8,
		SmartRoutingMaxDuration: time.Minute,
	})
	upstream := &smartRoutingBudgetHTTPUpstream{}
	service := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{ID: 31, Platform: PlatformOpenAI}

	for i := 0; i < SmartRoutingMaxSiteAttempts; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://upstream.test/v1/responses", http.NoBody)
		require.NoError(t, err)
		resp, err := service.doOpenAIUpstream(req, "", account)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://upstream.test/v1/responses", http.NoBody)
	require.NoError(t, err)
	_, err = service.doOpenAIUpstream(req, "", account)
	var budgetErr *UpstreamFailoverError
	require.ErrorAs(t, err, &budgetErr)
	require.Equal(t, SmartRoutingSiteAttemptBudgetExceededReason, budgetErr.Reason)
	require.Equal(t, GatewayFailureScopeAccount, budgetErr.Scope)
	require.True(t, budgetErr.ShouldRetryNextAccount())
	require.True(t, budgetErr.RequestScopedTransient, "a local routing cap must not lower upstream health")
	require.False(t, budgetErr.ShouldReportAccountScheduleFailure(), "a local routing cap is not an upstream health failure")
	require.Equal(t, 2, upstream.sends, "the third request must be rejected before reaching the same site")

	state := SmartRoutingStateFromContext(ctx)
	state.CurrentGroupID = 21
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, "https://upstream.test/v1/responses", http.NoBody)
	require.NoError(t, err)
	_, err = service.doOpenAIUpstream(req, "", account)
	require.NoError(t, err, "the same account ID in another group has an independent route budget")
	require.Equal(t, 3, upstream.sends)

	state.CurrentGroupID = 20
	otherAccount := &Account{ID: 32, Platform: PlatformOpenAI}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, "https://upstream.test/v1/responses", http.NoBody)
	require.NoError(t, err)
	_, err = service.doOpenAIUpstream(req, "", otherAccount)
	require.NoError(t, err, "another website in the same group has an independent route budget")
	require.Equal(t, 4, upstream.sends)
}

func TestSmartRoutingOpenAIWebSocketTurnBudgetSharesSiteCountAndResets(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          20,
		SmartRoutingMaxAttempts: 8,
		SmartRoutingMaxDuration: time.Minute,
	}
	ctx := WithSmartRoutingState(context.Background(), state)

	require.NoError(t, ReserveSmartRoutingOpenAIWSTurnAttempt(ctx, 31))
	require.NoError(t, ReserveSmartRoutingOpenAIWSTurnAttempt(ctx, 31))
	err := ReserveSmartRoutingOpenAIWSTurnAttempt(ctx, 31)
	var budgetErr *UpstreamFailoverError
	require.ErrorAs(t, err, &budgetErr)
	require.Equal(t, SmartRoutingSiteAttemptBudgetExceededReason, budgetErr.Reason)
	require.Equal(t, 2, state.turnAttemptCount)

	require.NoError(t, ReserveSmartRoutingOpenAIWSTurnAttempt(ctx, 32), "a second website may receive its own bounded attempts")
	CompleteSmartRoutingTurnAttemptBudget(ctx)
	require.NoError(t, ReserveSmartRoutingOpenAIWSTurnAttempt(ctx, 31), "a successful terminal turn starts a fresh per-site budget")
	require.Equal(t, 1, state.siteAttempts[smartRoutingSiteAttemptKey{groupID: 20, accountID: 31}])
}

func TestSmartRoutingOpenAIBudgetAllowsFailoverAfterSlowUpstreamFailure(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          10,
		SmartRoutingMaxAttempts: 4,
		SmartRoutingMaxDuration: 15 * time.Millisecond,
	}
	ctx := WithSmartRoutingState(context.Background(), state)

	require.NoError(t, ReserveSmartRoutingOpenAIRequestAttempt(ctx, 31))
	time.Sleep(30 * time.Millisecond)
	require.True(t, SmartRoutingRetryBudgetAllows(ctx), "elapsed generation time must not suppress a safe failover")

	state.CurrentGroupID = 20
	require.NoError(t, ReserveSmartRoutingOpenAIRequestAttempt(ctx, 32), "a slow failed attempt must still be able to reach the next candidate")
	require.Equal(t, 2, state.TotalAttempts)
}

func TestOrdinaryOpenAIRequestDoesNotUseSmartRoutingSendBudget(t *testing.T) {
	upstream := &smartRoutingBudgetHTTPUpstream{}
	service := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{ID: 31, Platform: PlatformOpenAI}

	for i := 0; i < SmartRoutingMaxSiteAttempts+1; i++ {
		req, err := http.NewRequest(http.MethodPost, "https://upstream.test/v1/responses", http.NoBody)
		require.NoError(t, err)
		_, err = service.doOpenAIUpstream(req, "", account)
		require.NoError(t, err)
	}
	require.Equal(t, SmartRoutingMaxSiteAttempts+1, upstream.sends)
}
