package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAdminRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()

	userHandler := NewUserHandler(adminSvc, nil, nil, nil)
	groupHandler := NewGroupHandler(adminSvc, nil, nil)
	proxyHandler := NewProxyHandler(adminSvc)
	redeemHandler := NewRedeemHandler(adminSvc, nil)

	router.GET("/api/v1/admin/users", userHandler.List)
	router.GET("/api/v1/admin/users/:id", userHandler.GetByID)
	router.POST("/api/v1/admin/users/:id/auth-identities", userHandler.BindAuthIdentity)
	router.POST("/api/v1/admin/users", userHandler.Create)
	router.PUT("/api/v1/admin/users/:id", userHandler.Update)
	router.DELETE("/api/v1/admin/users/:id", userHandler.Delete)
	router.POST("/api/v1/admin/users/:id/balance", userHandler.UpdateBalance)
	router.GET("/api/v1/admin/users/:id/api-keys", userHandler.GetUserAPIKeys)
	router.GET("/api/v1/admin/users/:id/usage", userHandler.GetUserUsage)

	router.GET("/api/v1/admin/groups", groupHandler.List)
	router.GET("/api/v1/admin/groups/all", groupHandler.GetAll)
	router.GET("/api/v1/admin/groups/:id", groupHandler.GetByID)
	router.POST("/api/v1/admin/groups", groupHandler.Create)
	router.PUT("/api/v1/admin/groups/:id", groupHandler.Update)
	router.DELETE("/api/v1/admin/groups/:id", groupHandler.Delete)
	router.GET("/api/v1/admin/groups/:id/stats", groupHandler.GetStats)
	router.GET("/api/v1/admin/groups/:id/api-keys", groupHandler.GetGroupAPIKeys)

	router.GET("/api/v1/admin/proxies", proxyHandler.List)
	router.GET("/api/v1/admin/proxies/all", proxyHandler.GetAll)
	router.GET("/api/v1/admin/proxies/:id", proxyHandler.GetByID)
	router.POST("/api/v1/admin/proxies", proxyHandler.Create)
	router.PUT("/api/v1/admin/proxies/:id", proxyHandler.Update)
	router.DELETE("/api/v1/admin/proxies/:id", proxyHandler.Delete)
	proxySubscriptionHandler := NewProxySubscriptionHandler(adminSvc)
	router.GET("/api/v1/admin/proxies/subscriptions", proxySubscriptionHandler.List)
	router.GET("/api/v1/admin/proxies/subscriptions/:id", proxySubscriptionHandler.GetByID)
	router.POST("/api/v1/admin/proxies/subscriptions", proxySubscriptionHandler.Create)
	router.PUT("/api/v1/admin/proxies/subscriptions/:id", proxySubscriptionHandler.Update)
	router.DELETE("/api/v1/admin/proxies/subscriptions/:id", proxySubscriptionHandler.Delete)
	router.POST("/api/v1/admin/proxies/subscriptions/:id/refresh", proxySubscriptionHandler.Refresh)
	router.GET("/api/v1/admin/proxies/subscriptions/:id/nodes", proxySubscriptionHandler.ListNodes)
	router.GET("/api/v1/admin/proxies/subscriptions/:id/proxies", proxySubscriptionHandler.ListProxies)
	router.POST("/api/v1/admin/proxies/batch-delete", proxyHandler.BatchDelete)
	router.POST("/api/v1/admin/proxies/unassign-accounts", proxyHandler.UnassignAccounts)
	router.POST("/api/v1/admin/proxies/:id/test", proxyHandler.Test)
	router.POST("/api/v1/admin/proxies/:id/quality-check", proxyHandler.CheckQuality)
	router.GET("/api/v1/admin/proxies/:id/stats", proxyHandler.GetStats)
	router.GET("/api/v1/admin/proxies/:id/accounts", proxyHandler.GetProxyAccounts)

	router.GET("/api/v1/admin/redeem-codes", redeemHandler.List)
	router.GET("/api/v1/admin/redeem-codes/:id", redeemHandler.GetByID)
	router.POST("/api/v1/admin/redeem-codes", redeemHandler.Generate)
	router.DELETE("/api/v1/admin/redeem-codes/:id", redeemHandler.Delete)
	router.POST("/api/v1/admin/redeem-codes/batch-delete", redeemHandler.BatchDelete)
	router.POST("/api/v1/admin/redeem-codes/:id/expire", redeemHandler.Expire)
	router.GET("/api/v1/admin/redeem-codes/:id/stats", redeemHandler.GetStats)

	return router, adminSvc
}

func TestUserHandlerEndpoints(t *testing.T) {
	router, _ := setupAdminRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/1", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	bindBody := map[string]any{
		"provider_type":    "wechat",
		"provider_key":     "wechat-main",
		"provider_subject": "union-123",
		"metadata":         map[string]any{"source": "admin-repair"},
		"channel": map[string]any{
			"channel":         "open",
			"channel_app_id":  "wx-open",
			"channel_subject": "openid-123",
		},
	}
	body, _ := json.Marshal(bindBody)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/1/auth-identities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	createBody := map[string]any{"email": "new@example.com", "password": "pass123", "balance": 1, "concurrency": 2}
	body, _ = json.Marshal(createBody)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	updateBody := map[string]any{"email": "updated@example.com"}
	body, _ = json.Marshal(updateBody)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/1", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/1/balance", bytes.NewBufferString(`{"balance":1,"operation":"add"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/1/api-keys", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/1/usage?period=today", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandlerBindAuthIdentityMapsRequest(t *testing.T) {
	router, adminSvc := setupAdminRouter()

	body, err := json.Marshal(map[string]any{
		"provider_type":    "oidc",
		"provider_key":     "https://issuer.example",
		"provider_subject": "subject-123",
		"issuer":           "https://issuer.example",
		"metadata":         map[string]any{"report_id": 12},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/9/auth-identities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(9), adminSvc.boundAuthIdentityFor)
	require.NotNil(t, adminSvc.boundAuthIdentity)
	require.Equal(t, "oidc", adminSvc.boundAuthIdentity.ProviderType)
	require.Equal(t, "https://issuer.example", adminSvc.boundAuthIdentity.ProviderKey)
	require.Equal(t, "subject-123", adminSvc.boundAuthIdentity.ProviderSubject)
	require.Nil(t, adminSvc.boundAuthIdentity.Channel)
	require.Equal(t, float64(12), adminSvc.boundAuthIdentity.Metadata["report_id"])
}

func TestGroupHandlerEndpoints(t *testing.T) {
	router, _ := setupAdminRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/all", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/2", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body, _ := json.Marshal(map[string]any{"name": "new", "platform": "anthropic", "subscription_type": "standard"})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body, _ = json.Marshal(map[string]any{"name": "update"})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/2", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/groups/2", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/2/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/2/api-keys", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestGroupHandlerGetStatsReturnsAggregatedValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	adminSvc.apiKeys = make([]service.APIKey, 0, 205)
	for i := 1; i <= 205; i++ {
		status := service.StatusDisabled
		if i <= 203 {
			status = service.StatusActive
		}
		adminSvc.apiKeys = append(adminSvc.apiKeys, service.APIKey{
			ID:     int64(i),
			Status: status,
		})
	}

	usageRepo := &groupStatsUsageRepoStub{
		stats: []usagestats.GroupStat{
			{GroupID: 2, Requests: 11, ActualCost: 1.2},
			{GroupID: 2, Requests: 9, ActualCost: 2.3},
		},
	}
	dashboardSvc := service.NewDashboardService(usageRepo, nil, nil, nil)
	groupHandler := NewGroupHandler(adminSvc, dashboardSvc, nil)

	router.GET("/api/v1/admin/groups/:id/stats", groupHandler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/2/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(205), data["total_api_keys"])
	require.Equal(t, float64(203), data["active_api_keys"])
	require.Equal(t, float64(20), data["total_requests"])
	require.InDelta(t, 3.5, data["total_cost"], 1e-9)
	require.Len(t, adminSvc.lastGetGroupAPIKeyCalls, 2)
	require.Equal(t, []groupAPIKeyCall{
		{groupID: 2, page: 1, pageSize: 200},
		{groupID: 2, page: 2, pageSize: 200},
	}, adminSvc.lastGetGroupAPIKeyCalls)
}

type groupStatsUsageRepoStub struct {
	service.UsageLogRepository
	stats []usagestats.GroupStat
}

func (s *groupStatsUsageRepoStub) GetGroupStatsWithFilters(
	_ context.Context,
	_, _ time.Time,
	_, _, _, groupID int64,
	_ *int16,
	_ *bool,
	_ *int8,
) ([]usagestats.GroupStat, error) {
	if groupID <= 0 {
		return nil, nil
	}
	return s.stats, nil
}

func TestProxyHandlerEndpoints(t *testing.T) {
	router, _ := setupAdminRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/all", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/4", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body, _ := json.Marshal(map[string]any{"name": "proxy", "protocol": "http", "host": "localhost", "port": 8080})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body, _ = json.Marshal(map[string]any{"name": "proxy2"})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/4", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/proxies/4", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/batch-delete", bytes.NewBufferString(`{"ids":[1,2]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/unassign-accounts", bytes.NewBufferString(`{"proxy_ids":[1,2]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/4/test", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/4/quality-check", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/4/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/4/accounts", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/subscriptions", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/subscriptions/4/refresh", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestProxyHandlerUpdateAllowsClearingCredentials(t *testing.T) {
	router, adminSvc := setupAdminRouter()

	body, err := json.Marshal(map[string]any{
		"username": nil,
		"password": nil,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/4", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, adminSvc.updatedProxies)
	last := adminSvc.updatedProxies[len(adminSvc.updatedProxies)-1]
	require.NotNil(t, last.Username)
	require.NotNil(t, last.Password)
	require.Equal(t, "", *last.Username)
	require.Equal(t, "", *last.Password)
}

func TestProxyHandlerListSupportsRuntimeStatusFilter(t *testing.T) {
	router, _ := setupAdminRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies?runtime_status=failed", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestProxySubscriptionHandlerRequestMapping(t *testing.T) {
	router, adminSvc := setupAdminRouter()

	body, err := json.Marshal(map[string]any{
		"name":                   "sub-a",
		"url":                    "https://example.com/sub",
		"source_format":          "clash_yaml",
		"enabled":                true,
		"refresh_interval_hours": 12,
		"auto_add_to_pool":       true,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, adminSvc.lastCreateProxySubscriptionInput)
	require.Equal(t, "sub-a", adminSvc.lastCreateProxySubscriptionInput.Name)
	require.Equal(t, "https://example.com/sub", adminSvc.lastCreateProxySubscriptionInput.URL)
	require.Equal(t, "clash_yaml", adminSvc.lastCreateProxySubscriptionInput.SourceFormat)
	require.Equal(t, 12, adminSvc.lastCreateProxySubscriptionInput.RefreshIntervalHours)
	require.True(t, adminSvc.lastCreateProxySubscriptionInput.AutoAddToPool)

	updateName := "sub-b"
	body, err = json.Marshal(map[string]any{
		"name":                   updateName,
		"refresh_interval_hours": 24,
	})
	require.NoError(t, err)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/subscriptions/9", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(9), adminSvc.lastUpdateProxySubscriptionID)
	require.NotNil(t, adminSvc.lastUpdateProxySubscriptionInput)
	require.NotNil(t, adminSvc.lastUpdateProxySubscriptionInput.Name)
	require.Equal(t, updateName, *adminSvc.lastUpdateProxySubscriptionInput.Name)
	require.NotNil(t, adminSvc.lastUpdateProxySubscriptionInput.RefreshIntervalHours)
	require.Equal(t, 24, *adminSvc.lastUpdateProxySubscriptionInput.RefreshIntervalHours)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/proxies/subscriptions/7", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), adminSvc.lastDeleteProxySubscriptionID)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/subscriptions/5/nodes", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(5), adminSvc.lastListProxySubscriptionNodesID)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/subscriptions/6/proxies", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(6), adminSvc.lastListSubscriptionProxiesID)
}

func TestRedeemHandlerEndpoints(t *testing.T) {
	router, _ := setupAdminRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/5", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body, _ := json.Marshal(map[string]any{"count": 1, "type": "balance", "value": 10})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/redeem-codes/5", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes/batch-delete", bytes.NewBufferString(`{"ids":[1,2]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes/5/expire", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/5/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestStubAdminServiceGetGroupAPIKeysPaginatesLikeContract(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.apiKeys = []service.APIKey{
		{ID: 1, Status: service.StatusActive},
		{ID: 2, Status: service.StatusDisabled},
		{ID: 3, Status: service.StatusActive},
	}

	page1, total, err := adminSvc.GetGroupAPIKeys(context.Background(), 2, 1, 2)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, page1, 2)
	require.EqualValues(t, []int64{1, 2}, []int64{page1[0].ID, page1[1].ID})

	page2, total, err := adminSvc.GetGroupAPIKeys(context.Background(), 2, 2, 2)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, page2, 1)
	require.EqualValues(t, int64(3), page2[0].ID)

	pageOutOfRange, total, err := adminSvc.GetGroupAPIKeys(context.Background(), 2, 3, 2)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Empty(t, pageOutOfRange)

	pageDefaultSize, total, err := adminSvc.GetGroupAPIKeys(context.Background(), 2, 1, 0)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, pageDefaultSize, 3)

	require.Equal(t, []groupAPIKeyCall{
		{groupID: 2, page: 1, pageSize: 2},
		{groupID: 2, page: 2, pageSize: 2},
		{groupID: 2, page: 3, pageSize: 2},
		{groupID: 2, page: 1, pageSize: 0},
	}, adminSvc.lastGetGroupAPIKeyCalls)
}
