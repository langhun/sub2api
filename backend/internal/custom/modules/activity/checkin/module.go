package checkin

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/gin-gonic/gin"
)

// Module groups the activity-owned check-in HTTP surface.
type Module struct {
	Handler *Handler
}

// NewModule builds a check-in module from activity contracts. Route composition
// remains with the caller so it can reuse the existing authenticated group.
func NewModule(service contract.CheckinService, blindboxReader contract.BlindboxRecordsReader) *Module {
	return &Module{Handler: NewHandler(service, blindboxReader)}
}

// NewOperationalModule constructs the runnable check-in module from narrow
// Activity/platform ports. It is the target composition API for root runtime
// wiring and has no dependency on the legacy CheckinService or BlindBoxService.
func NewOperationalModule(deps Dependencies, blindboxReader contract.BlindboxRecordsReader) (*Module, error) {
	service, err := NewService(deps)
	if err != nil {
		return nil, fmt.Errorf("construct activity check-in service: %w", err)
	}
	return NewModule(service, blindboxReader), nil
}

// RegisterRoutes attaches the established five check-in endpoints to an
// already-authenticated /api/v1 group. It deliberately does not install any
// middleware, preserving the route owner's existing authentication, backend
// mode guard, and audit policy.
func (m *Module) RegisterRoutes(authenticated *gin.RouterGroup) {
	if m == nil || m.Handler == nil || authenticated == nil {
		return
	}

	checkin := authenticated.Group("/checkin")
	checkin.POST("", m.Handler.Checkin)
	checkin.POST("/luck", m.Handler.LuckCheckin)
	checkin.GET("/status", m.Handler.GetStatus)
	checkin.GET("/calendar", m.Handler.GetCalendar)
	checkin.GET("/blindbox/records", m.Handler.GetBlindboxRecords)
}
