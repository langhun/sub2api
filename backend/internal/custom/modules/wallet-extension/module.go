// Package walletextension owns the wallet-extension Overlay boundary.
package walletextension

const (
	// ModuleID is the stable registry identifier for the wallet-extension Overlay.
	ModuleID = "wallet_extension"
	// ModuleVersion tracks the public contract version, not an implementation release.
	ModuleVersion = "v1"
)

// Status describes the implementation maturity of an Overlay module.
type Status string

const (
	// StatusContractOnly means the module exposes dependency contracts but no runtime behavior yet.
	StatusContractOnly Status = "contract_only"
	// StatusOperational means the module owns active HTTP routes or workers.
	StatusOperational Status = "operational"
)

// Dependency identifies a core capability required by an Overlay module.
type Dependency string

const (
	DependencyAccount     Dependency = "core.account"
	DependencyBalance     Dependency = "core.balance"
	DependencyAudit       Dependency = "core.audit"
	DependencySettings    Dependency = "core.settings"
	DependencyTransaction Dependency = "core.transaction"
)

// Manifest is static module metadata for future registry and runtime composition.
type Manifest struct {
	ID           string
	Version      string
	Status       Status
	Dependencies []Dependency
}

// Metadata declares wallet-extension's core dependencies without wiring an implementation.
var Metadata = Manifest{
	ID:      ModuleID,
	Version: ModuleVersion,
	Status:  StatusOperational,
	Dependencies: []Dependency{
		DependencyAccount,
		DependencyBalance,
		DependencyAudit,
		DependencySettings,
		DependencyTransaction,
	},
}

// Module groups the handlers exposed by wallet-extension.
type Module struct {
	User  *UserHandler
	Admin *AdminHandler
}

// NewModule constructs wallet-extension's HTTP module from its module-owned
// service. The service obtains only the narrow platform adapters it needs.
func NewModule(directTransferService *Service) *Module {
	return &Module{
		User:  NewUserHandler(directTransferService),
		Admin: NewAdminHandler(directTransferService),
	}
}
