package custom_test

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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
		custom.NewRuntime(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil),
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
	require.True(t, exists)
	_, exists = routes["GET /api/v1/transfer/stats"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/transfer/leaderboard"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/admin/transfers/batch"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/checkin"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/public/leaderboard/balance"]
	require.True(t, exists)
	_, exists = routes["POST /api/v1/redpacket"]
	require.True(t, exists)
	_, exists = routes["GET /api/v1/admin/blindbox/prize-items"]
	require.True(t, exists)
}
