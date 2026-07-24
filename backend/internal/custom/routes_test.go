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
		custom.NewRuntime(nil, nil, nil, nil),
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
}
