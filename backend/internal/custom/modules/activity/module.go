// Package activity owns the custom activity Overlay boundary.
package activity

const (
	// ModuleID is the stable registry identifier for the activity Overlay.
	ModuleID = "activity"
	// ModuleVersion tracks the public contract version, not an implementation release.
	ModuleVersion = "v1"
)

// Status describes the implementation maturity of an Overlay module.
type Status string

const (
	// StatusContractOnly means the module exposes dependency contracts but no runtime behavior yet.
	StatusContractOnly Status = "contract_only"
	// StatusOperational means the module owns active routes and runtime workers.
	StatusOperational Status = "operational"
)

// Dependency identifies a core capability required by an Overlay module.
type Dependency string

const (
	DependencyAccount      Dependency = "core.account"
	DependencyBalance      Dependency = "core.balance"
	DependencySettings     Dependency = "core.settings"
	DependencyAudit        Dependency = "core.audit"
	DependencyCodeIssuer   Dependency = "core.code_issuer"
	DependencySubscription Dependency = "core.subscription"
	DependencyNotification Dependency = "core.notification"
	DependencyCache        Dependency = "core.cache"
)

// Manifest is static module metadata for future registry and runtime composition.
type Manifest struct {
	ID           string
	Version      string
	Status       Status
	Dependencies []Dependency
}

// Metadata declares activity's core dependencies without wiring an implementation.
var Metadata = Manifest{
	ID:      ModuleID,
	Version: ModuleVersion,
	Status:  StatusOperational,
	Dependencies: []Dependency{
		DependencyAccount,
		DependencyBalance,
		DependencySettings,
		DependencyAudit,
		DependencyCodeIssuer,
		DependencySubscription,
		DependencyNotification,
		DependencyCache,
	},
}
