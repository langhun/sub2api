package service

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesVisibleSequenceIsMonotonic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	_, ok := OpenAIResponsesLastVisibleSequenceNumber(c)
	require.False(t, ok)
	require.Equal(t, int64(0), OpenAIResponsesNextVisibleSequenceNumber(c))

	SetOpenAIResponsesLastVisibleSequenceNumber(c, 4)
	last, ok := OpenAIResponsesLastVisibleSequenceNumber(c)
	require.True(t, ok)
	require.Equal(t, int64(4), last)
	require.Equal(t, int64(5), OpenAIResponsesNextVisibleSequenceNumber(c))

	SetOpenAIResponsesLastVisibleSequenceNumber(c, 2)
	last, ok = OpenAIResponsesLastVisibleSequenceNumber(c)
	require.True(t, ok)
	require.Equal(t, int64(4), last)

	SetOpenAIResponsesLastVisibleSequenceNumber(c, -1)
	last, ok = OpenAIResponsesLastVisibleSequenceNumber(c)
	require.True(t, ok)
	require.Equal(t, int64(4), last)
}
