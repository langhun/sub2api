package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type poolActionResponse struct {
	Code int            `json:"code"`
	Data map[string]any `json:"data"`
}

func setupProxyPoolRouter(adminSvc *stubAdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewProxyHandler(adminSvc)
	router.POST("/api/v1/admin/proxies/pool-membership", h.UpdatePoolMembership)
	router.POST("/api/v1/admin/proxies/clear-cooldown", h.ClearCooldown)
	return router
}

func TestProxyHandlerUpdatePoolMembership(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupProxyPoolRouter(adminSvc)

	body, err := json.Marshal(map[string]any{
		"ids":     []int64{1, 2, 2, 3},
		"enabled": true,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/pool-membership", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{1, 2, 2, 3}, adminSvc.lastPoolMembershipIDs)
	require.True(t, adminSvc.lastPoolMembershipEnabled)

	var resp poolActionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, float64(4), resp.Data["updated"])
	require.Equal(t, true, resp.Data["enabled"])
}

func TestProxyHandlerClearCooldown(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupProxyPoolRouter(adminSvc)

	body, err := json.Marshal(map[string]any{
		"ids": []int64{9, 10},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/clear-cooldown", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{9, 10}, adminSvc.lastClearCooldownIDs)

	var resp poolActionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, float64(2), resp.Data["cleared"])
}

