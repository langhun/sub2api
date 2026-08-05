package supporttickets

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const supportTicketRequestBodyLimit int64 = 16 * 1024

func (m *Module) RegisterRoutes(r *gin.Engine, jwtAuth middleware.JWTAuthMiddleware, adminAuth middleware.AdminAuthMiddleware, auditLog middleware.AuditLogMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if m == nil || r == nil {
		return
	}
	v1 := r.Group("/api/v1")
	if m.User != nil {
		user := v1.Group("")
		user.Use(gin.HandlerFunc(jwtAuth), middleware.BackendModeUserGuard(settingService))
		tickets := user.Group("/support/tickets")
		tickets.Use(middleware.RequestBodyLimit(supportTicketRequestBodyLimit), panelRateLimiter.Global(), gin.HandlerFunc(auditLog))
		tickets.GET("", m.User.List)
		tickets.POST("", panelRateLimiter.Heavy(), m.User.Create)
		tickets.GET("/:public_id", m.User.Get)
		tickets.POST("/:public_id/messages", panelRateLimiter.Heavy(), m.User.Reply)
		tickets.POST("/:public_id/reopen", m.User.Reopen)
	}
	if m.Admin != nil {
		admin := v1.Group("/admin")
		admin.Use(gin.HandlerFunc(adminAuth), middleware.AdminComplianceGuard(settingService))
		tickets := admin.Group("/support/tickets")
		tickets.Use(middleware.RequestBodyLimit(supportTicketRequestBodyLimit), panelRateLimiter.Global(), gin.HandlerFunc(auditLog))
		tickets.GET("", m.Admin.List)
		tickets.POST("", panelRateLimiter.Heavy(), m.Admin.Create)
		tickets.GET("/:public_id", m.Admin.Get)
		tickets.POST("/:public_id/messages", m.Admin.Reply)
		tickets.PATCH("/:public_id", m.Admin.Update)
	}
}
