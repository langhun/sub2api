package redpacket

import "github.com/Wei-Shaw/sub2api/internal/custom/platform"

// Module groups the Activity red-packet HTTP adapter and expiry worker.
type Module struct {
	User   *Handler
	Admin  *Handler
	Expiry ExpiryWorker
}

func NewModule(service Service, expiry ExpiryWorker) *Module {
	return NewModuleWithIdempotency(service, expiry, nil)
}

// NewModuleWithIdempotency composes red-packet HTTP routes with a generic
// platform idempotency port while keeping the worker and service unchanged.
func NewModuleWithIdempotency(service Service, expiry ExpiryWorker, coordinator platform.IdempotencyCoordinator) *Module {
	handler := NewHandlerWithIdempotency(service, coordinator)
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
