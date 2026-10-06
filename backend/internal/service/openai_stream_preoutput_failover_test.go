package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesPartialFirstOutputEventCanFailOver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	readErr := errors.New("upstream disconnected before SSE event boundary")
	service := &OpenAIGatewayService{cfg: &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize:                     defaultMaxLineSize,
			OpenAIFirstOutputTimeoutSeconds: 30,
		},
	}}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: &openAIStreamReadThenErrorCloser{
			reader: strings.NewReader(
				"event: response.output_text.delta\n" +
					`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n",
			),
			err: readErr,
		},
	}

	_, err := service.handleStreamingResponse(
		c.Request.Context(),
		resp,
		c,
		&Account{ID: 1, Platform: PlatformOpenAI, Name: "partial-first-output"},
		time.Now(),
		"gpt-5.6",
		"gpt-5.6",
	)

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "an attempt-local partial SSE frame has not been sent to the client")
	require.Empty(t, recorder.Body.String(), "the incomplete frame must not leak before trying another upstream")
	require.False(t, IsResponseCommitted(c))
}

func TestOpenAIResponsesDiscardedPreambleDoesNotAdvanceRetrySequence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &OpenAIGatewayService{cfg: &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	firstAttempt := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body: &openAIStreamReadThenErrorCloser{
			reader: strings.NewReader(
				"event: response.created\n" +
					`data: {"type":"response.created","sequence_number":90,"response":{"id":"resp_failed_attempt"}}` + "\n\n",
			),
			err: errors.New("first upstream disconnected before output"),
		},
	}
	_, err := service.handleStreamingResponse(
		c.Request.Context(),
		firstAttempt,
		c,
		&Account{ID: 1, Platform: PlatformOpenAI, Name: "failed-first-attempt"},
		time.Now(),
		"gpt-5.6",
		"gpt-5.6",
	)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Empty(t, recorder.Body.String(), "the failed attempt's preamble must stay private")
	_, hasSequence := OpenAIResponsesLastVisibleSequenceNumber(c)
	require.False(t, hasSequence, "a discarded attempt must not update request-visible sequence state")

	retryAttempt := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body: &openAIStreamReadThenErrorCloser{
			reader: strings.NewReader(
				"event: response.created\n" +
					`data: {"type":"response.created","sequence_number":1,"response":{"id":"resp_retry"}}` + "\n\n" +
					"event: response.output_text.delta\n" +
					`data: {"type":"response.output_text.delta","sequence_number":2,"delta":"partial"}` + "\n\n",
			),
			err: errors.New("retry upstream disconnected after output"),
		},
	}
	_, err = service.handleStreamingResponse(
		c.Request.Context(),
		retryAttempt,
		c,
		&Account{ID: 2, Platform: PlatformOpenAI, Name: "retry-attempt"},
		time.Now(),
		"gpt-5.6",
		"gpt-5.6",
	)
	require.Error(t, err)
	require.Contains(t, recorder.Body.String(), `"sequence_number":3`)
	require.NotContains(t, recorder.Body.String(), `"sequence_number":91`)
	require.Equal(t, int64(2), OpenAIResponsesNextVisibleSequenceNumber(c)-1)
}
