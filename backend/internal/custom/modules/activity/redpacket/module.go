package redpacket

// Module groups the Activity red-packet HTTP adapter and expiry worker.
type Module struct {
	User   *Handler
	Admin  *Handler
	Expiry ExpiryWorker
}

func NewModule(service Service, expiry ExpiryWorker) *Module {
	handler := NewHandler(service)
	return &Module{User: handler, Admin: handler, Expiry: expiry}
}

// Start launches periodic expiry only after Runtime has assembled its core
// adapters. It is safe to call more than once.
func (m *Module) Start() {
	if m != nil && m.Expiry != nil {
		m.Expiry.Start()
	}
}

// Stop waits for an in-flight expiry cycle before application shutdown.
func (m *Module) Stop() {
	if m != nil && m.Expiry != nil {
		m.Expiry.Stop()
	}
}
