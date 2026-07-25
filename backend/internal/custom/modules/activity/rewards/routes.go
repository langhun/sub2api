package rewards

import (
	legacyhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	legacy "github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Module owns the activity admin HTTP surface for blind-box prize management
// and durable reward delivery operations.
type Module struct {
	Admin *legacyhandler.BlindboxHandler
}

// NewModule adapts the established admin handler without changing endpoint
// payloads, status codes, or validation behavior.
func NewModule(adminHandler *legacyhandler.BlindboxHandler) *Module {
	return &Module{Admin: adminHandler}
}

// NewLegacyModule is the temporary composition bridge while the admin handler
// continues to use the established BlindBoxService implementation.
func NewLegacyModule(blindboxService *legacy.BlindBoxService) *Module {
	return NewModule(legacyhandler.NewBlindboxHandler(blindboxService))
}

// RegisterRoutes attaches the established admin endpoints to the caller's
// already-protected /api/v1/admin group. It deliberately installs no
// middleware: the existing admin authentication, audit, and compliance guards
// remain owned by the server route composition.
//
// The old registerBlindboxRoutes and registerRewardDeliveryRoutes calls must be
// removed before this method is invoked, otherwise Gin rejects duplicate paths.
func (m *Module) RegisterRoutes(adminRoutes *gin.RouterGroup) {
	if m == nil || m.Admin == nil || adminRoutes == nil {
		return
	}

	blindbox := adminRoutes.Group("/blindbox")
	blindbox.GET("/prize-items", m.Admin.ListPrizeItems)
	blindbox.POST("/prize-items", m.Admin.CreatePrizeItem)
	blindbox.PUT("/prize-items/:id", m.Admin.UpdatePrizeItem)
	blindbox.DELETE("/prize-items/:id", m.Admin.DeletePrizeItem)
	blindbox.GET("/stats", m.Admin.GetStats)

	deliveries := adminRoutes.Group("/reward-deliveries")
	deliveries.GET("", m.Admin.ListRewardDeliveries)
	deliveries.POST("/:id/retry", m.Admin.RetryRewardDelivery)
	deliveries.POST("/:id/compensate", m.Admin.CompensateRewardDelivery)
}
