package docs

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const docsRequestBodyLimit int64 = 256 * 1024

func (m *Module) RegisterRoutes(r *gin.Engine, jwtAuth middleware.JWTAuthMiddleware, adminAuth middleware.AdminAuthMiddleware, auditLog middleware.AuditLogMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if m == nil || m.Handler == nil || r == nil {
		return
	}
	v1 := r.Group("/api/v1")
	user := v1.Group("")
	user.Use(gin.HandlerFunc(jwtAuth), middleware.BackendModeUserGuard(settingService), gin.HandlerFunc(auditLog))
	user.Use(middleware.RequestBodyLimit(docsRequestBodyLimit), panelRateLimiter.Global())
	userDocs := user.Group("/docs/:slug")
	userDocs.GET("", m.Handler.List)
	userDocs.GET("/history", m.Handler.List)
	userDocs.POST("/revisions", panelRateLimiter.Heavy(), m.Handler.Create)

	admin := v1.Group("/admin")
	admin.Use(gin.HandlerFunc(adminAuth), middleware.AdminComplianceGuard(settingService), gin.HandlerFunc(auditLog))
	admin.Use(middleware.RequestBodyLimit(docsRequestBodyLimit), panelRateLimiter.Global())
	adminDocs := admin.Group("/docs/:slug")
	adminDocs.POST("/revisions/:revision_id/revert", panelRateLimiter.Heavy(), m.Handler.Revert)
	adminDocs.POST("/revisions/:revision_id/review", panelRateLimiter.Heavy(), m.Handler.Review)
}
