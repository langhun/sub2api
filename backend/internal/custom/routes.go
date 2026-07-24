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
	if runtime == nil || runtime.GameHall == nil {
		return
	}
	runtime.GameHall.RegisterRoutes(r, jwtAuth, adminAuth, auditLog, settingService)
}
