package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type routeFullAccountQueue struct{ helperConcurrencyCacheStub }

func (*routeFullAccountQueue) IncrementAccountWaitCount(context.Context, int64, int) (bool, error) {
	return false, nil
}

func TestSmartRoutingCapacityFailureDefersErrorUntilCandidatesExhausted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, smart := range []bool{true, false} {
		t.Run(map[bool]string{true: "smart", false: "single"}[smart], func(t *testing.T) {
			s := &service.SmartRoutingRequestState{MainGroupID: 2, CurrentGroupID: 2, SmartRoutingActive: smart, MaxGroupSwitches: 2, TriedGroups: map[int64]struct{}{}}
			ctx := service.WithSmartRoutingState(context.Background(), s)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)
			h := &OpenAIGatewayHandler{gatewayService: &service.OpenAIGatewayService{}, concurrencyHelper: NewConcurrencyHelper(service.NewConcurrencyService(&routeFullAccountQueue{}), SSEPingFormatComment, 0)}
			groupID := int64(2)
			selection := &service.AccountSelectionResult{Account: &service.Account{ID: 11}, WaitPlan: &service.AccountWaitPlan{AccountID: 11, MaxConcurrency: 1, MaxWaiting: 1, Timeout: time.Second}}
			started := false
			release, result := h.acquireResponsesAccountSlot(c, &groupID, "session", selection, false, &started, zap.NewNop())
			require.Nil(t, release)
			if smart {
				require.Equal(t, openAISlotAcquireRouteRetry, result)
				require.Empty(t, w.Body.String())
				require.False(t, c.Writer.Written())
				require.Contains(t, s.FailedAccountIDs, int64(11))
				require.NotContains(t, s.ExhaustedGroups, int64(2), "other accounts in this group may still have capacity")
			} else {
				require.Equal(t, openAISlotAcquireFailed, result)
				require.Equal(t, 429, w.Code)
			}
		})
	}
}

func TestSmartRoutingCapacityRetryRespectsRequestBoundary(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		err                        error
		cancel, output, continuity bool
		want                       bool
	}{
		{"account timeout", &ConcurrencyError{SlotType: "account", IsTimeout: true}, false, false, false, true},
		{"user timeout", &ConcurrencyError{SlotType: "user", IsTimeout: true}, false, false, false, false},
		{"client cancelled", nil, true, false, false, false},
		{"visible output", nil, false, true, false, false},
		{"continuation", nil, false, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			state := &service.SmartRoutingRequestState{SmartRoutingActive: true, MainGroupID: 2, CurrentGroupID: 2, ContinuityLocked: tc.continuity}
			ctx = service.WithSmartRoutingState(ctx, state)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)
			if tc.cancel {
				cancel()
			}
			if tc.output {
				_, _ = c.Writer.Write([]byte("visible"))
			}
			require.Equal(t, tc.want, retrySmartRoutingCapacity(c, tc.err, 11, 2))
		})
	}
}

func TestSmartRoutingCapacityRetriesAreBoundedAcrossReselection(t *testing.T) {
	s := &service.SmartRoutingRequestState{SmartRoutingActive: true, MainGroupID: 2, CurrentGroupID: 2}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(service.WithSmartRoutingState(context.Background(), s))
	require.True(t, retrySmartRoutingCapacity(c, nil, 11, 1))
	require.False(t, retrySmartRoutingCapacity(c, nil, 11, 1), "same account cannot spin")
	require.True(t, retrySmartRoutingCapacity(c, nil, 12, 1))
	require.False(t, retrySmartRoutingCapacity(c, nil, 13, 1), "group changes must not reset the request budget")
	require.Empty(t, s.ExhaustedGroups)
}

type routingAdmissionFixture struct {
	allowed            bool
	suppressPoolRetry  bool
	acquires, releases int
	attempts           []service.SmartRoutingAttempt
}

func (f *routingAdmissionFixture) AcquireRoutingAccount(context.Context, int64, string, string, bool) (string, bool) {
	f.acquires++
	return "probe-token", f.allowed
}

func (f *routingAdmissionFixture) ReleaseRoutingAccount(int64, string, string, bool, string) {
	f.releases++
}

func (f *routingAdmissionFixture) ObserveRoutingAttempt(a service.SmartRoutingAttempt) {
	f.attempts = append(f.attempts, a)
}

func (f *routingAdmissionFixture) SuppressSameAccountRetry(*service.UpstreamFailoverError) bool {
	return f.suppressPoolRetry
}

func TestSmartRoutingForwardAdmissionAndCleanup(t *testing.T) {
	for _, scenario := range []string{"success", "failure", "cancel", "request_error", "panic", "denied", "single"} {
		t.Run(scenario, func(t *testing.T) {
			controller := &routingAdmissionFixture{allowed: scenario != "denied"}
			state := &service.SmartRoutingRequestState{SmartRoutingActive: scenario != "single", Observer: controller, RequestedModel: "gpt-5", EndpointType: "responses", CurrentGroupID: 2}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx = service.WithSmartRoutingState(service.WithAPIKeyContext(ctx, &service.APIKey{ID: 9}), state)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)
			calls := 0
			account := &service.Account{ID: 11}
			run := func() {
				_, err := forwardWithSmartRouting(c, account, false, func() (*service.OpenAIForwardResult, error) {
					calls++
					switch scenario {
					case "cancel":
						cancel()
						return nil, context.Canceled
					case "request_error":
						return nil, &service.UpstreamFailoverError{StatusCode: 400, Scope: service.GatewayFailureScopeRequest}
					case "failure":
						return nil, &service.UpstreamFailoverError{StatusCode: 503}
					case "panic":
						panic("forward panic")
					}
					return &service.OpenAIForwardResult{}, nil
				})
				if scenario == "denied" {
					var failure *service.UpstreamFailoverError
					require.ErrorAs(t, err, &failure)
					require.False(t, failure.ShouldReportAccountScheduleFailure())
				}
			}
			if scenario == "panic" {
				require.Panics(t, run)
			} else {
				run()
			}
			if scenario == "single" {
				require.Zero(t, controller.acquires)
				require.Zero(t, controller.releases)
			} else {
				require.Equal(t, 1, controller.acquires)
				if scenario == "denied" {
					require.Zero(t, calls)
					require.Zero(t, controller.releases)
				} else {
					require.Equal(t, 1, controller.releases)
				}
			}
			if scenario == "success" || scenario == "failure" {
				require.Len(t, controller.attempts, 1)
				require.Equal(t, "probe-token", controller.attempts[0].ProbeToken)
			} else {
				require.Empty(t, controller.attempts, "orchestration failures must not penalize the upstream")
			}
			require.Empty(t, state.ProbeToken)
		})
	}
}

func TestSmartRoutingBudgetCountsOnlyAdmittedUpstreamForwards(t *testing.T) {
	controller := &routingAdmissionFixture{allowed: false}
	state := &service.SmartRoutingRequestState{
		SmartRoutingActive:      true,
		Observer:                controller,
		RequestedModel:          "gpt-5",
		EndpointType:            "responses",
		CurrentGroupID:          2,
		SmartRoutingMaxAttempts: 1,
		SmartRoutingMaxDuration: time.Minute,
	}
	ctx := service.WithSmartRoutingState(service.WithAPIKeyContext(context.Background(), &service.APIKey{ID: 9}), state)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)

	_, err := forwardWithSmartRouting(c, &service.Account{ID: 11}, false, func() (*service.OpenAIForwardResult, error) {
		t.Fatal("health-denied account must not start an upstream forward")
		return nil, nil
	})
	var failure *service.UpstreamFailoverError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, service.SmartRoutingAccountUnavailableReason, failure.Reason)
	require.Zero(t, state.TotalAttempts, "a rejected admission is not an upstream attempt")
	require.Equal(t, 1, controller.acquires)
	require.Zero(t, controller.releases, "no lease was acquired")
}

func TestSmartRoutingBudgetFailureRecordsTerminalStatusWithoutForward(t *testing.T) {
	controller := &routingAdmissionFixture{allowed: true}
	state := &service.SmartRoutingRequestState{
		SmartRoutingActive:      true,
		Observer:                controller,
		RequestedModel:          "gpt-5",
		EndpointType:            "responses",
		CurrentGroupID:          2,
		SmartRoutingMaxAttempts: 1,
		SmartRoutingMaxDuration: time.Minute,
		TotalAttempts:           1,
		SmartRoutingStartedAt:   time.Now(),
	}
	ctx := service.WithSmartRoutingState(service.WithAPIKeyContext(context.Background(), &service.APIKey{ID: 9}), state)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)

	_, err := forwardWithSmartRouting(c, &service.Account{ID: 11}, false, func() (*service.OpenAIForwardResult, error) {
		t.Fatal("budget-exhausted request must not start another upstream forward")
		return nil, nil
	})
	var failure *service.UpstreamFailoverError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, service.SmartRoutingAttemptBudgetExceededReason, failure.Reason)
	require.Equal(t, 503, state.FinalErrorStatus)
	require.Equal(t, 1, state.TotalAttempts)
	require.Equal(t, 1, controller.releases)
}

func TestSmartRoutingForwardEnforcesWallClockDeadline(t *testing.T) {
	controller := &routingAdmissionFixture{allowed: true}
	state := &service.SmartRoutingRequestState{
		SmartRoutingActive:      true,
		Observer:                controller,
		RequestedModel:          "gpt-5",
		EndpointType:            "responses",
		CurrentGroupID:          2,
		SmartRoutingMaxAttempts: 2,
		SmartRoutingMaxDuration: 15 * time.Millisecond,
	}
	ctx := service.WithSmartRoutingState(service.WithAPIKeyContext(context.Background(), &service.APIKey{ID: 9}), state)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)

	started := time.Now()
	_, err := forwardWithSmartRouting(c, &service.Account{ID: 11}, false, func() (*service.OpenAIForwardResult, error) {
		<-c.Request.Context().Done()
		return nil, c.Request.Context().Err()
	})

	var failure *service.UpstreamFailoverError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, service.SmartRoutingAttemptBudgetExceededReason, failure.Reason)
	require.Equal(t, 503, state.FinalErrorStatus)
	require.Less(t, time.Since(started), 500*time.Millisecond)
	require.Equal(t, 1, state.TotalAttempts)
	require.Equal(t, 1, controller.acquires)
	require.Equal(t, 1, controller.releases)
}

func TestSmartRoutingForwardSuppressesOnlyUnsafePoolRetries(t *testing.T) {
	tests := []struct {
		name        string
		maxAttempts int
		response    string
		suppress    bool
		wantRetry   bool
	}{
		{name: "deterministic balance failure", maxAttempts: 12, response: "Insufficient account balance", suppress: true, wantRetry: false},
		{name: "request budget exhausted", maxAttempts: 1, response: "service busy", wantRetry: false},
		{name: "transient failure with budget", maxAttempts: 12, response: "service busy", wantRetry: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &routingAdmissionFixture{allowed: true, suppressPoolRetry: tt.suppress}
			state := &service.SmartRoutingRequestState{
				SmartRoutingActive: true, Observer: controller, RequestedModel: "gpt-5",
				EndpointType: "responses", CurrentGroupID: 2,
				SmartRoutingMaxAttempts: tt.maxAttempts, SmartRoutingMaxDuration: time.Minute,
			}
			ctx := service.WithSmartRoutingState(service.WithAPIKeyContext(context.Background(), &service.APIKey{ID: 9}), state)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(ctx)
			upstreamFailure := &service.UpstreamFailoverError{
				StatusCode: http.StatusServiceUnavailable, RetryableOnSameAccount: true,
				ResponseBody: []byte(tt.response),
			}
			if tt.response == "Insufficient account balance" {
				upstreamFailure.StatusCode = http.StatusForbidden
			}

			_, err := forwardWithSmartRouting(c, &service.Account{ID: 11}, false, func() (*service.OpenAIForwardResult, error) {
				return nil, upstreamFailure
			})

			require.ErrorIs(t, err, upstreamFailure)
			require.Equal(t, tt.wantRetry, upstreamFailure.RetryableOnSameAccount)
		})
	}
}
