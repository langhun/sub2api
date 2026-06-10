//go:build unit

package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSendHTTPErrorAndEnd_EmitsStructuredCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	svc := &AccountTestService{}
	err := svc.sendHTTPErrorAndEnd(ctx, http.StatusTooManyRequests, "upstream limited")

	require.Error(t, err)
	require.Contains(t, recorder.Body.String(), `"type":"error"`)
	require.Contains(t, recorder.Body.String(), `"code":"429"`)
	require.Contains(t, recorder.Body.String(), `"error":"upstream limited"`)
}
