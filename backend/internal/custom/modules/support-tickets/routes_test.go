package supporttickets

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModuleRegistersUserAndAdminRoutes(t *testing.T) {
	router := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }
	NewModule(nil).RegisterRoutes(router,
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
		nil,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"GET /api/v1/support/tickets",
		"POST /api/v1/support/tickets",
		"GET /api/v1/support/tickets/:public_id",
		"POST /api/v1/support/tickets/:public_id/messages",
		"POST /api/v1/support/tickets/:public_id/reopen",
		"GET /api/v1/admin/support/tickets",
		"GET /api/v1/admin/support/tickets/:public_id",
		"POST /api/v1/admin/support/tickets/:public_id/messages",
		"PATCH /api/v1/admin/support/tickets/:public_id",
	} {
		_, ok := routes[route]
		require.Truef(t, ok, "missing route %s", route)
	}
}
