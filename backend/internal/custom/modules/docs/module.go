package docs

type Module struct{ Handler *Handler }

func NewModule(service *Service) *Module { return &Module{Handler: NewHandler(service)} }
