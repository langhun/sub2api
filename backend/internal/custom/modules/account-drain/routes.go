package accountdrain

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (m *Module) RegisterRoutes(r *gin.Engine, adminAuth middleware.AdminAuthMiddleware, auditLog middleware.AuditLogMiddleware, settingService *service.SettingService) {
	if m == nil || m.Handler == nil || r == nil {
		return
	}
	admin := r.Group("/api/v1/admin")
	admin.Use(gin.HandlerFunc(adminAuth))
	admin.Use(gin.HandlerFunc(auditLog))
	admin.Use(middleware.AdminComplianceGuard(settingService))
	plans := admin.Group("/account-drain/plans")
	plans.GET("", m.Handler.List)
	plans.POST("", m.Handler.Create)
	plans.POST("/:id/stop", m.Handler.Stop)
}
