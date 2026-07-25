package gamehall

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes attaches the module's authenticated user and admin APIs.
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

		gameHall := authenticated.Group("/game-hall")
		gameHall.GET("/status", m.User.Status)
		gameHall.POST("/exchange", m.User.Exchange)
		gameHall.POST("/play", m.User.Play)
		gameHall.GET("/transactions", m.User.Transactions)
		gameHall.GET("/rounds", m.User.Rounds)
	}

	if m.Admin != nil {
		admin := v1.Group("/admin")
		admin.Use(gin.HandlerFunc(adminAuth))
		admin.Use(gin.HandlerFunc(auditLog))
		admin.Use(middleware.AdminComplianceGuard(settingService))

		gameHall := admin.Group("/game-hall")
		gameHall.GET("/transactions", m.Admin.Transactions)
		gameHall.GET("/rounds", m.Admin.Rounds)
		gameHall.GET("/users/:user_id/access", m.Admin.UserAccess)
		gameHall.PUT("/users/:user_id/access", m.Admin.UpdateUserAccess)
	}
}
