package redpacket

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes attaches the extracted endpoints at their existing public
// paths. The legacy route registrations must be removed before this method is
// called, otherwise Gin will reject duplicate handlers.
func (m *Module) RegisterRoutes(
	r *gin.Engine,
	jwtAuth middleware.JWTAuthMiddleware,
	adminAuth middleware.AdminAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
) {
	if m == nil || r == nil {
		return
	}
	v1 := r.Group("/api/v1")
	if m.User != nil {
		authenticated := v1.Group("")
		authenticated.Use(gin.HandlerFunc(jwtAuth))
		authenticated.Use(middleware.BackendModeUserGuard(settingService))
		authenticated.Use(gin.HandlerFunc(auditLog))
		redPackets := authenticated.Group("/redpacket")
		redPackets.POST("", m.User.Create)
		redPackets.POST("/claim", m.User.Claim)
		redPackets.GET("/my", m.User.Mine)
		redPackets.GET("/:id", m.User.Detail)
	}
	if m.Admin != nil {
		admin := v1.Group("/admin")
		admin.Use(gin.HandlerFunc(adminAuth))
		admin.Use(gin.HandlerFunc(auditLog))
		admin.Use(middleware.AdminComplianceGuard(settingService))
		admin.Group("/redpackets").GET("", m.Admin.ListAll)
	}
}
