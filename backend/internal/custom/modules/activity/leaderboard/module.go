package leaderboard

import (
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
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

// NewDatabaseModule constructs the operational activity leaderboard without a
// dependency on the legacy leaderboard or settings services.
func NewDatabaseModule(client *dbent.Client, db *sql.DB) *Module {
	repository := NewRepository(client, db)
	return NewModule(NewSettingsReader(NewSettingsStore(client)), Readers{
		Balance:     repository,
		Consumption: repository,
		Checkin:     repository,
	})
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
}
