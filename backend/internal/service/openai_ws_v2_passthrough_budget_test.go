package service

import (
	"context"
	"errors"
	"testing"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

type passthroughBudgetFrameConnRecorder struct {
	writes int
}

func (c *passthroughBudgetFrameConnRecorder) ReadFrame(context.Context) (coderws.MessageType, []byte, error) {
	return coderws.MessageText, nil, errOpenAIWSConnClosed
}

func (c *passthroughBudgetFrameConnRecorder) WriteFrame(context.Context, coderws.MessageType, []byte) error {
	c.writes++
	return nil
}

func (c *passthroughBudgetFrameConnRecorder) Close() error {
	return nil
}

func TestOpenAIWSPassthroughBudgetFrameConnReservesBeforeRealResponseCreateWrite(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          20,
		SmartRoutingMaxAttempts: 8,
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	inner := &passthroughBudgetFrameConnRecorder{}
	conn := &openAIWSPassthroughBudgetFrameConn{
		inner:     inner,
		ctx:       ctx,
		accountID: 31,
	}

	require.NoError(t, conn.WriteFrame(ctx, coderws.MessageText, []byte(`{"type":"session.update"}`)))
	require.Equal(t, 1, inner.writes, "non-turn control frames must not consume the turn budget")

	responseCreate := []byte(`{"type":"response.create","model":"gpt-5.5"}`)
	require.NoError(t, conn.WriteFrame(ctx, coderws.MessageText, responseCreate))
	require.NoError(t, conn.WriteFrame(ctx, coderws.MessageText, responseCreate))
	require.Equal(t, 3, inner.writes)

	err := conn.WriteFrame(ctx, coderws.MessageText, responseCreate)
	var budgetErr *UpstreamFailoverError
	require.ErrorAs(t, err, &budgetErr)
	require.Equal(t, SmartRoutingSiteAttemptBudgetExceededReason, budgetErr.Reason)
	require.True(t, budgetErr.ShouldRetryNextAccount())
	retryPayload, retryCurrentTurn := OpenAIWSCurrentTurnRetryPayload(err)
	require.True(t, retryCurrentTurn, "a stateless later turn can be retried on a replacement candidate")
	require.JSONEq(t, string(responseCreate), string(retryPayload))
	require.Equal(t, 3, inner.writes, "a rejected response.create must not reach the upstream")
}

func TestOpenAIWSPassthroughBudgetFrameConnKeepsOrdinaryRequestsUnchanged(t *testing.T) {
	inner := &passthroughBudgetFrameConnRecorder{}
	conn := &openAIWSPassthroughBudgetFrameConn{
		inner:     inner,
		ctx:       context.Background(),
		accountID: 31,
	}
	responseCreate := []byte(`{"type":"response.create","model":"gpt-5.5"}`)

	for i := 0; i < SmartRoutingMaxSiteAttempts+1; i++ {
		require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, responseCreate))
	}
	require.Equal(t, SmartRoutingMaxSiteAttempts+1, inner.writes)
}

func TestOpenAIWSPassthroughBudgetErrorRemainsFailoverEligible(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          20,
		SmartRoutingMaxAttempts: 8,
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	conn := &openAIWSPassthroughBudgetFrameConn{
		inner:     &passthroughBudgetFrameConnRecorder{},
		ctx:       ctx,
		accountID: 31,
	}
	responseCreate := []byte(`{"type":"response.create","model":"gpt-5.5"}`)
	require.NoError(t, conn.WriteFrame(ctx, coderws.MessageText, responseCreate))
	require.NoError(t, conn.WriteFrame(ctx, coderws.MessageText, responseCreate))

	err := conn.WriteFrame(ctx, coderws.MessageText, responseCreate)
	var budgetErr *UpstreamFailoverError
	require.ErrorAs(t, err, &budgetErr)
	wrapped := wrapOpenAIWSIngressTurnError("write_upstream", err, false)
	require.True(t, isOpenAIWSIngressTurnRetryable(wrapped))
	require.True(t, errors.As(wrapped, &budgetErr))
	require.True(t, budgetErr.ShouldRetryNextAccount(), "the outer handler must be able to select another candidate")
}

func TestOpenAIWSPassthroughUnsafeLaterTurnDoesNotReplayFirstTurn(t *testing.T) {
	state := &SmartRoutingRequestState{
		SmartRoutingActive:      true,
		MainGroupID:             10,
		CurrentGroupID:          20,
		SmartRoutingMaxAttempts: 8,
	}
	ctx := WithSmartRoutingState(context.Background(), state)
	inner := &passthroughBudgetFrameConnRecorder{}
	conn := &openAIWSPassthroughBudgetFrameConn{
		inner:     inner,
		ctx:       ctx,
		accountID: 31,
	}
	require.NoError(t, conn.WriteFrame(ctx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.5"}`)))
	continuation := []byte(`{"type":"response.create","model":"gpt-5.5","previous_response_id":"resp_first"}`)
	require.NoError(t, conn.WriteFrame(ctx, coderws.MessageText, continuation))

	err := conn.WriteFrame(ctx, coderws.MessageText, continuation)
	var closeErr *OpenAIWSClientCloseError
	require.ErrorAs(t, err, &closeErr)
	var budgetErr *UpstreamFailoverError
	require.False(t, errors.As(err, &budgetErr), "an unsafe continuation must not be unwrapped as a failover error")
	_, retryCurrentTurn := OpenAIWSCurrentTurnRetryPayload(err)
	require.False(t, retryCurrentTurn)
	require.Equal(t, 2, inner.writes, "the rejected continuation must not reach the upstream")
}

func TestOpenAIWSPassthroughSuccessfulTerminalClassification(t *testing.T) {
	require.True(t, openAIWSPassthroughIsSuccessfulTerminalOutput([]byte(`{"type":"response.completed"}`)))
	require.True(t, openAIWSPassthroughIsSuccessfulTerminalOutput([]byte(`{"type":"response.done"}`)))
	require.False(t, openAIWSPassthroughIsSuccessfulTerminalOutput([]byte(`{"type":"response.failed"}`)))
	require.False(t, openAIWSPassthroughIsSuccessfulTerminalOutput([]byte(`{"type":"response.incomplete"}`)))
}
