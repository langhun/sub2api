package custom

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes is the single route registration entry point for custom modules.
func RegisterRoutes(
	r *gin.Engine,
	runtime *Runtime,
	jwtAuth middleware.JWTAuthMiddleware,
	adminAuth middleware.AdminAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
) {
	if runtime == nil {
		return
	}
	if runtime.GameHall != nil {
		runtime.GameHall.RegisterRoutes(r, jwtAuth, adminAuth, auditLog, settingService)
	}
	if runtime.ActivityCheckin != nil {
		v1 := r.Group("/api/v1")
		authenticated := v1.Group("")
		authenticated.Use(gin.HandlerFunc(jwtAuth))
		authenticated.Use(middleware.BackendModeUserGuard(settingService))
		authenticated.Use(gin.HandlerFunc(auditLog))
		runtime.ActivityCheckin.RegisterRoutes(authenticated)
	}
	if runtime.ActivityLeaderboard != nil {
		runtime.ActivityLeaderboard.RegisterRoutes(r)
	}
	if runtime.ActivityRedPacket != nil {
		runtime.ActivityRedPacket.RegisterRoutes(r, jwtAuth, adminAuth, auditLog, settingService)
	}
	if runtime.ActivityRewardsHTTP != nil {
		v1 := r.Group("/api/v1")
		admin := v1.Group("/admin")
		admin.Use(gin.HandlerFunc(adminAuth))
		admin.Use(gin.HandlerFunc(auditLog))
		admin.Use(middleware.AdminComplianceGuard(settingService))
		runtime.ActivityRewardsHTTP.RegisterRoutes(admin)
	}
	if runtime.WalletExtension != nil {
		runtime.WalletExtension.RegisterRoutes(r, jwtAuth, adminAuth, auditLog, settingService)
	}
}
