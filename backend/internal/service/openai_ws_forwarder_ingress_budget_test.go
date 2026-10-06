package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSmartRoutingTurnAttemptBudgetIsIsolatedAndResetAfterSuccess(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		SmartRoutingMaxAttempts: 2,
		SmartRoutingMaxDuration: time.Minute,
	}
	ctx := WithSmartRoutingState(context.Background(), state)

	require.True(t, SmartRoutingTurnAttemptBudgetAvailable(ctx))
	require.True(t, SmartRoutingTurnAttemptBudgetAvailable(ctx))
	require.False(t, SmartRoutingTurnAttemptBudgetAvailable(ctx))
	require.Equal(t, 2, state.turnAttemptCount)
	require.Equal(t, 0, state.TotalAttempts, "WebSocket turn budget must not mutate HTTP/request routing telemetry")
	require.True(t, state.SmartRoutingStartedAt.IsZero(), "WebSocket turn budget must not mutate request-level timing")

	CompleteSmartRoutingTurnAttemptBudget(ctx)
	require.True(t, SmartRoutingTurnAttemptBudgetAvailable(ctx), "the next logical response.create starts a fresh budget")
	require.Equal(t, 1, state.turnAttemptCount)
}

func TestSmartRoutingTurnAttemptBudgetPersistsAcrossFailoverReentryWithoutWallClockExpiry(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		SmartRoutingMaxAttempts: 2,
		SmartRoutingMaxDuration: time.Minute,
	}
	firstCtx := WithSmartRoutingState(context.Background(), state)
	require.True(t, SmartRoutingTurnAttemptBudgetAvailable(firstCtx))

	reenteredCtx := WithSmartRoutingState(context.Background(), state)
	require.True(t, SmartRoutingTurnAttemptBudgetAvailable(reenteredCtx), "account/group retry re-entry shares the unfinished turn budget")
	require.False(t, SmartRoutingTurnAttemptBudgetAvailable(reenteredCtx))

	state.attemptMu.Lock()
	state.turnAttemptCount = 1
	state.turnAttemptStart = time.Now().Add(-2 * time.Minute)
	state.attemptMu.Unlock()
	require.True(t, SmartRoutingTurnAttemptBudgetAvailable(reenteredCtx), "a long-running turn must not be cancelled by a wall-clock routing deadline")
	require.False(t, SmartRoutingTurnAttemptBudgetAvailable(reenteredCtx), "the send-count budget still stops the next attempt")
}

func TestReserveSmartRoutingOpenAIWSTurnAttemptReturnsTypedStopOnlyWhenActive(t *testing.T) {
	require.NoError(t, reserveSmartRoutingOpenAIWSTurnAttempt(context.Background()), "ordinary single-group requests are unaffected")

	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		SmartRoutingMaxAttempts: 1,
		SmartRoutingMaxDuration: time.Minute,
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	require.NoError(t, reserveSmartRoutingOpenAIWSTurnAttempt(ctx))

	err := reserveSmartRoutingOpenAIWSTurnAttempt(ctx)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, SmartRoutingTurnAttemptBudgetExceededReason, failoverErr.Reason)
	require.Equal(t, GatewayFailureScopeRequest, failoverErr.Scope)
	require.Equal(t, NextAccountStop, failoverErr.NextAccountAction)
	require.True(t, failoverErr.RequestScopedTransient)
	require.False(t, failoverErr.ShouldReportAccountScheduleFailure(), "budget exhaustion is not an upstream-account health failure")
}
