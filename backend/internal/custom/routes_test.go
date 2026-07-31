package custom_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom"
	activitycheckin "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/checkin"
	activityleaderboard "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/leaderboard"
	activityredpacket "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/redpacket"
	activityrewards "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/rewards"
	gamehall "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageQuerySettingsStub struct {
	enabled bool
	err     error
}

func (s usageQuerySettingsStub) UsageQueryEnabled(context.Context) (bool, error) {
	return s.enabled, s.err
}

func TestRegisterRoutesDoesNotAddRoutesWithoutRuntime(t *testing.T) {
	router := gin.New()

	custom.RegisterRoutes(router, nil, nil, nil, nil, nil)

	require.Empty(t, router.Routes())
}

func TestRegisterRoutesMountsGameHallModule(t *testing.T) {
	router := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }

	custom.RegisterRoutes(
		router,
		&custom.Runtime{
			ActivityCheckin:     activitycheckin.NewModule(nil, nil),
			ActivityLeaderboard: activityleaderboard.NewModule(nil, activityleaderboard.Readers{}),
			ActivityRedPacket:   activityredpacket.NewModule(nil, nil),
			ActivityRewardsHTTP: activityrewards.NewModule(routeHandlerStub{}),
			GameHall:            gamehall.NewModule(nil),
		},
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	_, exists := routes["POST /api/v1/game-hall/play"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/admin/game-hall/rounds"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/transfer"]
	require.False(t, exists)
	_, exists = routes["GET /api/v1/transfer/stats"]
	require.False(t, exists)
	_, exists = routes["POST /api/v1/admin/transfers/batch"]
	require.False(t, exists)
	_, exists = routes["POST /api/v1/checkin"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/public/leaderboard/balance"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/redpacket"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/admin/blindbox/prize-items"]
	require.True(t, exists)
}

func TestRegisterRoutesUsageQueryGateOnlyControlsUsageEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		settings   usageQuerySettingsStub
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "disabled v1 usage",
			settings:   usageQuerySettingsStub{enabled: false},
			method:     http.MethodGet,
			path:       "/v1/usage",
			wantStatus: http.StatusForbidden,
			wantBody:   `{"type":"error","error":{"type":"permission_error","message":"Usage query is disabled"}}`,
		},
		{
			name:       "disabled antigravity usage",
			settings:   usageQuerySettingsStub{enabled: false},
			method:     http.MethodGet,
			path:       "/antigravity/v1/usage",
			wantStatus: http.StatusForbidden,
			wantBody:   `{"type":"error","error":{"type":"permission_error","message":"Usage query is disabled"}}`,
		},
		{
			name:       "settings unavailable",
			settings:   usageQuerySettingsStub{err: errors.New("store unavailable")},
			method:     http.MethodGet,
			path:       "/v1/usage",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"type":"error","error":{"type":"api_error","message":"Usage query settings are unavailable"}}`,
		},
		{
			name:       "enabled reaches upstream usage handler",
			settings:   usageQuerySettingsStub{enabled: true},
			method:     http.MethodGet,
			path:       "/v1/usage",
			wantStatus: http.StatusOK,
			wantBody:   `{"route":"v1 usage"}`,
		},
		{
			name:       "other route bypasses disabled gate",
			settings:   usageQuerySettingsStub{enabled: false},
			method:     http.MethodGet,
			path:       "/v1/models",
			wantStatus: http.StatusOK,
			wantBody:   `{"route":"models"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			custom.RegisterRoutes(router, &custom.Runtime{UsageQuerySettings: tt.settings}, nil, nil, nil, nil)
			router.GET("/v1/usage", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"route": "v1 usage"}) })
			router.GET("/antigravity/v1/usage", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"route": "antigravity usage"}) })
			router.GET("/v1/models", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"route": "models"}) })

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, nil))

			require.Equal(t, tt.wantStatus, recorder.Code)
			require.JSONEq(t, tt.wantBody, recorder.Body.String())
		})
	}
}

type routeHandlerStub struct{}

func (routeHandlerStub) ListPrizeItems(*gin.Context)           {}
func (routeHandlerStub) CreatePrizeItem(*gin.Context)          {}
func (routeHandlerStub) UpdatePrizeItem(*gin.Context)          {}
func (routeHandlerStub) DeletePrizeItem(*gin.Context)          {}
func (routeHandlerStub) GetStats(*gin.Context)                 {}
func (routeHandlerStub) ListRewardDeliveries(*gin.Context)     {}
func (routeHandlerStub) RetryRewardDelivery(*gin.Context)      {}
func (routeHandlerStub) CompensateRewardDelivery(*gin.Context) {}
