package leaderboard

import (
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"

	"github.com/gin-gonic/gin"
)

// Module owns the activity leaderboard's public HTTP surface.
type Module struct {
	Handler *Handler
}

func NewModule(settings contract.LeaderboardSettingsReader, readers Readers) *Module {
	return &Module{Handler: NewHandler(NewService(settings, readers))}
}

// RegisterRoutes preserves the existing public endpoint paths. It does not add
// middleware because these four leaderboard endpoints are intentionally public.
func (m *Module) RegisterRoutes(r *gin.Engine) {
	if r == nil || m == nil || m.Handler == nil {
		return
	}
	leaderboard := r.Group("/api/v1/public/leaderboard")
	leaderboard.GET("/balance", m.Handler.Balance)
	leaderboard.GET("/consumption", m.Handler.Consumption)
	leaderboard.GET("/checkin", m.Handler.Checkin)
	leaderboard.GET("/transfer", m.Handler.Transfer)
}
