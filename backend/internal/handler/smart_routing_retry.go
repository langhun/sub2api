package handler

import (
	"errors"
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

func observeSmartRoutingForward(c *gin.Context, accountID int64, stream bool, started time.Time, result any, err error) {
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
	service.ObserveSmartRoutingAttempt(c.Request.Context(), accountID, stream, time.Since(started), firstToken, err)
}
