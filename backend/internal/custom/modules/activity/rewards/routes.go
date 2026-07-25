package rewards

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Module owns the activity admin HTTP surface for blind-box prize management
// and durable reward delivery operations.
type Module struct {
	Admin   AdminRouteHandler
	Rewards *Service
	Outbox  Outbox
	Runner  DeliveryRunner
}

// DeliveryRunner is the check-in-facing immediate delivery capability. It
// deliberately exposes no worker lifecycle or shared service implementation.
type DeliveryRunner interface {
	RunByID(ctx context.Context, id int64) error
}

// AdminRouteHandler is the narrow HTTP surface bound under the pre-existing
// admin middleware chain. Both the module-owned handler and the temporary
// compatibility handler satisfy it.
type AdminRouteHandler interface {
	ListPrizeItems(*gin.Context)
	CreatePrizeItem(*gin.Context)
	UpdatePrizeItem(*gin.Context)
	DeletePrizeItem(*gin.Context)
	GetStats(*gin.Context)
	ListRewardDeliveries(*gin.Context)
	RetryRewardDelivery(*gin.Context)
	CompensateRewardDelivery(*gin.Context)
}

// NewModule constructs the module-owned admin HTTP surface.
func NewModule(adminHandler AdminRouteHandler) *Module {
	return &Module{Admin: adminHandler}
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
