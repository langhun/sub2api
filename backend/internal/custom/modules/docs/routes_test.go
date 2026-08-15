package docs

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
		"GET /api/v1/docs/:slug",
		"GET /api/v1/docs/:slug/history",
		"POST /api/v1/docs/:slug/revisions",
		"POST /api/v1/admin/docs/:slug/revisions/:revision_id/revert",
		"POST /api/v1/admin/docs/:slug/revisions/:revision_id/review",
	} {
		_, ok := routes[route]
		require.Truef(t, ok, "missing route %s", route)
	}
}
