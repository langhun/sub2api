package gamehall

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModuleRegistersUserAndAdminRoutes(t *testing.T) {
	router := gin.New()
	module := NewModule(nil)
	passthrough := func(c *gin.Context) { c.Next() }

	module.RegisterRoutes(
		router,
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for _, route := range []string{
		"GET /api/v1/game-hall/status",
		"POST /api/v1/game-hall/exchange",
		"POST /api/v1/game-hall/play",
		"GET /api/v1/game-hall/transactions",
		"GET /api/v1/game-hall/rounds",
		"GET /api/v1/admin/game-hall/transactions",
		"GET /api/v1/admin/game-hall/rounds",
		"GET /api/v1/admin/game-hall/users/:user_id/access",
		"PUT /api/v1/admin/game-hall/users/:user_id/access",
	} {
		_, ok := routes[route]
		require.Truef(t, ok, "missing module route %s", route)
	}
}
