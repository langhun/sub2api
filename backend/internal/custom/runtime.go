// Package custom contains the application-specific Overlay extension points.
package custom

import "github.com/google/wire"

// ProviderSet constructs the Overlay runtime at the composition root.
var ProviderSet = wire.NewSet(NewRuntime)

// Runtime owns dependencies shared by custom modules as they are introduced.
type Runtime struct{}

// NewRuntime constructs the empty Overlay runtime.
func NewRuntime() *Runtime {
	return &Runtime{}
}
