package custom

import (
	"context"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	usageQueryPath            = "/v1/usage"
	antigravityUsageQueryPath = "/antigravity/v1/usage"
)

// UsageQuerySettings is the only activity configuration consumed by the
// custom usage-query route gate.
type UsageQuerySettings interface {
	UsageQueryEnabled(context.Context) (bool, error)
}

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
	if runtime.UsageQuerySettings != nil {
		r.Use(activityUsageQueryGate(runtime.UsageQuerySettings))
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
}

func activityUsageQueryGate(settings UsageQuerySettings) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet || !isActivityUsageQueryPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		enabled, err := settings.UsageQueryEnabled(c.Request.Context())
		if err != nil {
			activityUsageQueryError(c, http.StatusServiceUnavailable, "api_error", "Usage query settings are unavailable")
			return
		}
		if !enabled {
			activityUsageQueryError(c, http.StatusForbidden, "permission_error", "Usage query is disabled")
			return
		}
		c.Next()
	}
}

func isActivityUsageQueryPath(path string) bool {
	return path == usageQueryPath || path == antigravityUsageQueryPath
}

func activityUsageQueryError(c *gin.Context, status int, errorType, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errorType,
			"message": message,
		},
	})
}
