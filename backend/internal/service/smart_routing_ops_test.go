package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSmartRoutingOpsEventFreezesAttemptGroup(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetOpsAccountGroup(c, 11, 2)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{AccountID: 11, UpstreamStatusCode: 503})
	SetOpsAccountGroup(c, 11, 22)
	value, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := value.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.EqualValues(t, 2, events[0].GroupID)
	require.EqualValues(t, 22, OpsAccountGroup(c, 11))
}
