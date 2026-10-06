package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// retrySmartRoutingCapacity permits a bounded re-selection after an account
// capacity admission failure. Capacity re-selection is request/group-level
// orchestration: it must not use the gateway's account-switch budget or the
// Attempt telemetry counter as a shared budget. The latter counts completed
// upstream attempts and can otherwise make a request's capacity budget depend
// on unrelated provider failures.
//
// MaxGroupSwitches is the routing contract's single source of truth. The
// extra +1 is the documented final capacity re-entry after the configured
// number of candidate-group transitions (for the default of two switches,
// three capacity retries are allowed). A zero value is tolerated for legacy
// internal callers and uses the same default as the group failover guard.
func retrySmartRoutingCapacity(c *gin.Context, err error, accountID int64) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if err != nil {
		var capacity *ConcurrencyError
		var full *WaitQueueFullError
		accountCapacity := errors.As(err, &capacity) && capacity.SlotType == "account"
		accountQueueFull := errors.As(err, &full) && full.SlotType == "account"
		if !accountCapacity && !accountQueueFull {
			return false
		}
	}
	ctx := c.Request.Context()
	state := service.SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || !service.SmartRoutingContractValid(ctx) || state.ContinuityLocked || ctx.Err() != nil || gatewayStreamSemanticOutputStarted(c) || accountID <= 0 {
		return false
	}
	maxGroupSwitches := state.MaxGroupSwitches
	if maxGroupSwitches < 0 {
		return false
	}
	if maxGroupSwitches == 0 {
		maxGroupSwitches = 2
	}
	if state.CapacityRetries >= maxGroupSwitches+1 {
		return false
	}
	if service.SmartRoutingAccountFailed(ctx, accountID, state.CurrentGroupID) {
		return false
	}
	service.MarkSmartRoutingAccountFailed(ctx, accountID, state.CurrentGroupID)
	state.CapacityRetries++
	return true
}

func observeSmartRoutingForward(c *gin.Context, account *service.Account, stream bool, started time.Time, result any, err error) {
	var firstToken *int
	switch r := result.(type) {
	case *service.OpenAIForwardResult:
		if r == nil && err == nil {
			return
		}
		if r != nil {
			firstToken = r.FirstTokenMs
		}
	case *service.ForwardResult:
		if r == nil && err == nil {
			return
		}
		if r != nil {
			firstToken = r.FirstTokenMs
		}
	}
	service.ObserveSmartRoutingAttempt(c.Request.Context(), account, stream, time.Since(started), firstToken, err)
}

func forwardWithSmartRouting[T any](c *gin.Context, account *service.Account, stream bool, forward func() (T, error)) (T, error) {
	var zero T
	if account == nil {
		return zero, errors.New("smart routing account is nil")
	}
	release, err := service.BeginSmartRoutingAttempt(c.Request.Context(), account.ID)
	if err != nil {
		return zero, err
	}
	if state := service.SmartRoutingStateFromContext(c.Request.Context()); state != nil && state.SmartRoutingActive {
		// Count only at the transport boundary. GatewayService and OpenAI
		// transports reserve immediately before each real upstream send,
		// including protocol-repair and redirect attempts. Admission itself
		// must not consume a slot because a later capacity/profit veto may
		// prevent any network request.
		service.PrepareSmartRoutingOpenAIRequestBudget(c.Request.Context())
	}
	defer release()
	originalRequest := c.Request
	attemptCtx, cancelAttempt := service.SmartRoutingAttemptContext(originalRequest.Context())
	defer cancelAttempt()
	defer func() { c.Request = originalRequest }()
	c.Request = originalRequest.WithContext(attemptCtx)
	started := time.Now()
	result, err := forward()
	// Observe against the uncancelled request context so a budget deadline is
	// still attributable as the terminal attempt before the request returns.
	c.Request = originalRequest
	if errors.Is(err, context.DeadlineExceeded) && errors.Is(attemptCtx.Err(), context.DeadlineExceeded) && originalRequest.Context().Err() == nil {
		if state := service.SmartRoutingStateFromContext(originalRequest.Context()); state != nil {
			state.FinalErrorStatus = http.StatusServiceUnavailable
		}
		err = &service.UpstreamFailoverError{
			StatusCode:        http.StatusServiceUnavailable,
			Reason:            service.SmartRoutingAttemptBudgetExceededReason,
			Scope:             service.GatewayFailureScopeRequest,
			NextAccountAction: service.NextAccountStop,
			ClientMessage:     "smart routing time budget exhausted",
		}
	}
	observeSmartRoutingForward(c, account, stream, started, result, err)
	return result, err
}
