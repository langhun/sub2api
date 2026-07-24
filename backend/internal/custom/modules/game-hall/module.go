// Package gamehall owns the custom game-hall HTTP surface.
package gamehall

const ModuleID = "game_hall"

// Module groups the handlers exposed by the game-hall Overlay.
type Module struct {
	User  *UserHandler
	Admin *AdminHandler
}

// NewModule constructs the game-hall HTTP module. Its service implementation
// is supplied by the Overlay composition root.
func NewModule(gameHallService *GameHallService) *Module {
	return &Module{
		User:  NewUserHandler(gameHallService),
		Admin: NewAdminHandler(gameHallService),
	}
}
