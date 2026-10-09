package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildOpenAIResponseFailedSSE_DefaultSequenceAndError(t *testing.T) {
	frame := buildOpenAIResponseFailedSSE("resp_test", "gpt-test", nil, "upstream unavailable")
	require.True(t, strings.HasPrefix(frame, "event: response.failed\ndata: "))
	payload := strings.TrimSpace(strings.TrimPrefix(frame, "event: response.failed\ndata: "))
	require.True(t, gjson.Valid(payload))
	require.Equal(t, "response.failed", gjson.Get(payload, "type").String())
	require.Zero(t, gjson.Get(payload, "sequence_number").Int())
	require.Equal(t, "resp_test", gjson.Get(payload, "response.id").String())
	require.Equal(t, "gpt-test", gjson.Get(payload, "response.model").String())
	require.Equal(t, "failed", gjson.Get(payload, "response.status").String())
	require.Equal(t, "upstream_error", gjson.Get(payload, "response.error.code").String())
	require.Equal(t, "upstream unavailable", gjson.Get(payload, "response.error.message").String())
}
