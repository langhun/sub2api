package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// A clean upstream EOF after semantic Responses output is not a complete
// response. The service returns ErrOpenAIUpstreamStreamTruncated and leaves the
// already-written output intact; the handler must add exactly one protocol
// terminal event so strict clients do not observe a silent EOF.
func TestOpenAIEnsureForwardErrorResponse_CleanEOFAfterOutputEmitsSingleResponseFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, EndpointResponses, nil)
	_, err := c.Writer.WriteString(
		"event: response.output_text.delta\n" +
			`data: {"type":"response.output_text.delta","delta":"partial","sequence_number":4}` +
			"\n\n",
	)
	require.NoError(t, err)
	service.SetOpenAIResponsesLastVisibleSequenceNumber(c, 4)

	truncated := service.ErrOpenAIUpstreamStreamTruncated
	require.False(t, openAIForwardErrorAlreadyCommunicated(c, 0, truncated),
		"clean EOF without a terminal event is not already communicated")

	h := &OpenAIGatewayHandler{}
	require.True(t, h.ensureOpenAIStreamReadErrorResponse(c, truncated, true))

	assertSingleResponsesFailedAfterOutput(t, w.Body.String())
	require.Contains(t, w.Body.String(), `"code":"upstream_stream_truncated"`)
	require.Contains(t, w.Body.String(), "Upstream response stream ended before completion")
}

func TestGatewayEnsureForwardErrorResponse_CleanEOFAfterOutputEmitsSingleResponseFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, EndpointResponses, nil)
	_, err := c.Writer.WriteString(
		"event: response.output_text.delta\n" +
			`data: {"type":"response.output_text.delta","delta":"partial","sequence_number":4}` +
			"\n\n",
	)
	require.NoError(t, err)
	service.SetOpenAIResponsesLastVisibleSequenceNumber(c, 4)

	truncated := service.ErrOpenAIUpstreamStreamTruncated
	require.False(t, gatewayForwardErrorAlreadyCommunicated(c, 0, truncated),
		"clean EOF without a terminal event is not already communicated")

	h := &GatewayHandler{}
	require.True(t, h.ensureForwardErrorResponse(c, true))

	assertSingleResponsesFailedAfterOutput(t, w.Body.String())
}

func assertSingleResponsesFailedAfterOutput(t *testing.T, body string) {
	t.Helper()
	require.Contains(t, body, `"delta":"partial"`)
	require.Equal(t, 1, strings.Count(body, "event: response.failed\n"),
		"clean EOF must receive exactly one Responses terminal event")
	require.NotContains(t, body, "event: error\n")

	frames := strings.Split(strings.TrimSuffix(body, "\n\n"), "\n\n")
	require.Len(t, frames, 2)
	require.Equal(t, "event: response.output_text.delta", strings.Split(frames[0], "\n")[0])
	require.Equal(t, "event: response.failed", strings.Split(frames[1], "\n")[0])

	dataLine := strings.TrimPrefix(strings.Split(frames[1], "\n")[1], "data: ")
	var event struct {
		Type           string `json:"type"`
		SequenceNumber int    `json:"sequence_number"`
		Response       struct {
			Object string `json:"object"`
			Status string `json:"status"`
			Error  struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"response"`
	}
	require.NoError(t, json.Unmarshal([]byte(dataLine), &event))
	require.Equal(t, "response.failed", event.Type)
	require.Equal(t, "response", event.Response.Object)
	require.Equal(t, "failed", event.Response.Status)
	require.NotEmpty(t, event.Response.Error.Code)
	require.NotEmpty(t, event.Response.Error.Message)
	require.Equal(t, int64(5), int64(event.SequenceNumber))
}
