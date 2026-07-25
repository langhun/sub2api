// Package gamehall owns the custom game-hall HTTP surface.
package gamehall

import "github.com/Wei-Shaw/sub2api/internal/custom/platform"

const ModuleID = "game_hall"

// Module groups the handlers exposed by the game-hall Overlay.
type Module struct {
	User  *UserHandler
	Admin *AdminHandler
}

// NewModule constructs the game-hall HTTP module. Its service implementation
// is supplied by the Overlay composition root.
func NewModule(gameHallService *GameHallService) *Module {
	return NewModuleWithIdempotency(gameHallService, nil)
}

// NewModuleWithIdempotency injects the generic platform coordinator used by
// mutating user routes. A nil coordinator keeps the historical direct path.
func NewModuleWithIdempotency(gameHallService *GameHallService, coordinator platform.IdempotencyCoordinator) *Module {
	return &Module{
		User:  NewUserHandlerWithIdempotency(gameHallService, coordinator),
		Admin: NewAdminHandler(gameHallService),
	}
}
