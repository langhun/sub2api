package service

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIPassthroughReadErrorClosesPartialSSEFrameBeforeFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	readErr := io.ErrUnexpectedEOF
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"rid-partial-frame"}},
		Body: &openAIStreamReadThenErrorCloser{
			reader: strings.NewReader(`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n"),
			err:    readErr,
		},
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}}

	_, err := svc.handleStreamingResponsePassthrough(
		c.Request.Context(), resp, c,
		&Account{ID: 1, Platform: PlatformOpenAI},
		time.Now(), "gpt-5", "gpt-5",
	)

	require.ErrorIs(t, err, readErr)
	require.False(t, isUpstreamFailoverError(err), "a semantic delta written downstream must not be replayed")
	var eventTypes []string
	var eventPayloads [][]byte
	forEachOpenAISSEFrame(rec.Body.String(), func(eventType string, payload []byte) {
		eventTypes = append(eventTypes, eventType)
		eventPayloads = append(eventPayloads, append([]byte(nil), payload...))
	})
	require.Equal(t, []string{"response.output_text.delta", "response.failed"}, eventTypes)
	require.True(t, gjson.ValidBytes(eventPayloads[0]))
	require.Equal(t, "partial", gjson.GetBytes(eventPayloads[0], "delta").String())
	require.True(t, gjson.ValidBytes(eventPayloads[1]))
	require.Equal(t, OpenAIUpstreamStreamReadErrorCode, gjson.GetBytes(eventPayloads[1], "response.error.code").String())
	require.True(t, IsResponseCommitted(c))
}

func TestOpenAIPassthroughCleanEOFAfterDeltaWritesFailureTerminal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"rid-clean-eof"}},
		Body: io.NopCloser(strings.NewReader(
			`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n\n",
		)),
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}}

	_, err := svc.handleStreamingResponsePassthrough(
		c.Request.Context(), resp, c,
		&Account{ID: 1, Platform: PlatformOpenAI},
		time.Now(), "gpt-5", "gpt-5",
	)

	require.ErrorIs(t, err, ErrOpenAIUpstreamStreamTruncated)
	require.False(t, isUpstreamFailoverError(err), "a semantic delta written downstream must not be replayed")
	var eventTypes []string
	var eventPayloads [][]byte
	forEachOpenAISSEFrame(rec.Body.String(), func(eventType string, payload []byte) {
		eventTypes = append(eventTypes, eventType)
		eventPayloads = append(eventPayloads, append([]byte(nil), payload...))
	})
	require.Equal(t, []string{"response.output_text.delta", "response.failed"}, eventTypes)
	require.Equal(t, "partial", gjson.GetBytes(eventPayloads[0], "delta").String())
	require.Equal(t, OpenAIUpstreamStreamTruncatedCode, gjson.GetBytes(eventPayloads[1], "response.error.code").String())
	require.True(t, IsResponseCommitted(c))
}

func TestOpenAIPassthroughCleanEOFAfterTerminalClosesSSEFrame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"rid-terminal-eof"}},
		Body: io.NopCloser(strings.NewReader(
			`data: {"type":"response.completed","response":{"id":"resp_terminal","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}}` + "\n",
		)),
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}}

	_, err := svc.handleStreamingResponsePassthrough(
		c.Request.Context(), resp, c,
		&Account{ID: 1, Platform: PlatformOpenAI},
		time.Now(), "gpt-5", "gpt-5",
	)

	require.NoError(t, err)
	require.True(t, strings.HasSuffix(rec.Body.String(), "\n\n"), "terminal SSE event must have its blank-line delimiter")
	var eventTypes []string
	var eventPayloads [][]byte
	forEachOpenAISSEFrame(rec.Body.String(), func(eventType string, payload []byte) {
		eventTypes = append(eventTypes, eventType)
		eventPayloads = append(eventPayloads, append([]byte(nil), payload...))
	})
	require.Equal(t, []string{"response.completed"}, eventTypes)
	require.True(t, gjson.ValidBytes(eventPayloads[0]))
	require.Equal(t, "resp_terminal", gjson.GetBytes(eventPayloads[0], "response.id").String())
}

func isUpstreamFailoverError(err error) bool {
	var failoverErr *UpstreamFailoverError
	return errors.As(err, &failoverErr)
}
