package handler

import (
	"context"
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
