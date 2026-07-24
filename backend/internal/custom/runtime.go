// Package custom contains the application-specific Overlay extension points.
package custom

import (
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	gamehall "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

// ProviderSet constructs the Overlay runtime at the composition root.
var ProviderSet = wire.NewSet(NewRuntime)

// Runtime owns dependencies shared by custom modules as they are introduced.
type Runtime struct {
	GameHall *gamehall.Module
}

// NewRuntime constructs the enabled custom modules at the composition root.
func NewRuntime(client *dbent.Client, db *sql.DB, settingService *service.SettingService, billingCache *service.BillingCacheService) *Runtime {
	gameHallService := gamehall.NewGameHallService(gamehall.NewGameHallRepository(client, db), settingService, billingCache)
	return &Runtime{GameHall: gamehall.NewModule(gameHallService)}
}
