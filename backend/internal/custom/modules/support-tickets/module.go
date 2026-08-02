package supporttickets

// Module groups the user and administrator handlers exposed by this Overlay.
type Module struct {
	User  *UserHandler
	Admin *AdminHandler
}

func NewModule(service *Service) *Module {
	return &Module{User: NewUserHandler(service), Admin: NewAdminHandler(service)}
}
