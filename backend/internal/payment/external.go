package payment

import (
	"strings"
	"sync"
)

// ExternalProviderDefinition describes an Overlay-owned payment provider using
// generic payment configuration and routing metadata.
type ExternalProviderDefinition struct {
	ProviderKey                       string
	SensitiveConfigFields             []string
	PendingOrderProtectedConfigFields []string
	VisibleMethods                    []PaymentType
	DisplayNameConfigKey              string
	DefaultDisplayName                string
	CurrencyConfigKey                 string
	DefaultCurrency                   string
	WebhookPath                       string
	WebhookOrderID                    func(rawBody string) string
}

var externalProviderDefinitions = struct {
	sync.RWMutex
	items map[string]ExternalProviderDefinition
}{items: make(map[string]ExternalProviderDefinition)}

// RegisterExternalProviderDefinition installs Overlay provider metadata.
func RegisterExternalProviderDefinition(definition ExternalProviderDefinition) {
	key := strings.TrimSpace(definition.ProviderKey)
	if key == "" {
		return
	}
	definition.ProviderKey = key
	definition.WebhookPath = strings.Trim(strings.TrimSpace(definition.WebhookPath), "/")
	externalProviderDefinitions.Lock()
	defer externalProviderDefinitions.Unlock()
	externalProviderDefinitions.items[key] = definition
}

// ExternalProviderDefinitionFor returns a registered Overlay provider definition.
func ExternalProviderDefinitionFor(providerKey string) (ExternalProviderDefinition, bool) {
	externalProviderDefinitions.RLock()
	defer externalProviderDefinitions.RUnlock()
	definition, ok := externalProviderDefinitions.items[strings.TrimSpace(providerKey)]
	return definition, ok
}

// ExternalProviderDefinitions returns a copy suitable for route composition.
func ExternalProviderDefinitions() []ExternalProviderDefinition {
	externalProviderDefinitions.RLock()
	defer externalProviderDefinitions.RUnlock()
	definitions := make([]ExternalProviderDefinition, 0, len(externalProviderDefinitions.items))
	for _, definition := range externalProviderDefinitions.items {
		definitions = append(definitions, definition)
	}
	return definitions
}

// IsExternalProvider reports whether the provider key belongs to an Overlay module.
func IsExternalProvider(providerKey string) bool {
	_, ok := ExternalProviderDefinitionFor(providerKey)
	return ok
}
