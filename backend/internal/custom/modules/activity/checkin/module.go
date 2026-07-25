package checkin

import (
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	legacy "github.com/Wei-Shaw/sub2api/internal/service"
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

// NewLegacyModule is the temporary composition bridge for the current core
// services. The legacy dependency is contained here and in adapter.go.
func NewLegacyModule(checkinService *legacy.CheckinService, blindboxService *legacy.BlindBoxService) *Module {
	return NewModule(
		NewLegacyAdapter(checkinService),
		NewLegacyBlindboxRecordsAdapter(blindboxService),
	)
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
