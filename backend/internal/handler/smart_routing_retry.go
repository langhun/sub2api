package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func retrySmartRoutingCapacity(c *gin.Context, err error, accountID int64, maxSwitches int) bool {
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
	if maxSwitches < 0 || state.Attempt+state.CapacityRetries >= maxSwitches+1 {
		return false
	}
	if state.FailedAccountIDs == nil {
		state.FailedAccountIDs = map[int64]struct{}{}
	}
	if _, failed := state.FailedAccountIDs[accountID]; failed {
		return false
	}
	state.FailedAccountIDs[accountID] = struct{}{}
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
		// Reserve the request budget only after final account admission. A
		// rejected health/probe lease never starts an upstream request and must
		// not consume one of the hard attempt slots.
		if !service.SmartRoutingAttemptBudgetAvailable(c.Request.Context()) {
			release()
			state.FinalErrorStatus = http.StatusServiceUnavailable
			return zero, &service.UpstreamFailoverError{
				StatusCode:        http.StatusServiceUnavailable,
				Reason:            service.SmartRoutingAttemptBudgetExceededReason,
				Scope:             service.GatewayFailureScopeRequest,
				NextAccountAction: service.NextAccountStop,
				ClientMessage:     "smart routing attempt budget exhausted",
			}
		}
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
