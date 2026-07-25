package walletextension

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes attaches the authenticated direct-transfer routes.
func (m *Module) RegisterRoutes(
	r *gin.Engine,
	jwtAuth middleware.JWTAuthMiddleware,
	adminAuth middleware.AdminAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
) {
	if m == nil || m.User == nil || r == nil {
		return
	}
	v1 := r.Group("/api/v1")
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	authenticated.Use(gin.HandlerFunc(auditLog))
	transfer := authenticated.Group("/transfer")
	transfer.POST("", m.User.Transfer)
	transfer.POST("/validate", m.User.Preview)
	transfer.GET("/receiver", m.User.ResolveRecipient)
	transfer.GET("/receivers", m.User.SearchRecipients)
	transfer.GET("/history", m.User.History)
	transfer.GET("/stats", m.User.Stats)
	transfer.GET("/leaderboard", m.User.Leaderboard)

	if m.Admin == nil {
		return
	}
	admin := v1.Group("/admin")
	admin.Use(gin.HandlerFunc(adminAuth))
	admin.Use(gin.HandlerFunc(auditLog))
	admin.Use(middleware.AdminComplianceGuard(settingService))
	transfers := admin.Group("/transfers")
	transfers.GET("", m.Admin.ListTransfers)
	transfers.GET("/stats", m.Admin.FeeStats)
	transfers.PUT("/:id/freeze", m.Admin.FreezeTransfer)
	transfers.PUT("/:id/revoke", m.Admin.RevokeTransfer)
	transfers.POST("/batch", m.Admin.BatchDistribute)
}
