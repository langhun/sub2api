package payment

import "testing"

func TestExternalProviderDefinitionRegistration(t *testing.T) {
	const providerKey = "external-definition-test"
	RegisterExternalProviderDefinition(ExternalProviderDefinition{
		ProviderKey:          providerKey,
		VisibleMethods:       []PaymentType{providerKey},
		DisplayNameConfigKey: "displayName",
		DefaultDisplayName:   "Default label",
		CurrencyConfigKey:    "fiat",
		DefaultCurrency:      "CNY",
		WebhookPath:          "/external-test/",
	})

	definition, ok := ExternalProviderDefinitionFor(providerKey)
	if !ok {
		t.Fatal("registered external provider definition was not found")
	}
	if definition.WebhookPath != "external-test" {
		t.Fatalf("webhook path = %q, want external-test", definition.WebhookPath)
	}
	if definition.DisplayNameConfigKey != "displayName" || definition.DefaultDisplayName != "Default label" {
		t.Fatalf("display name metadata = (%q, %q), want (displayName, Default label)", definition.DisplayNameConfigKey, definition.DefaultDisplayName)
	}
	if !IsExternalProvider(providerKey) {
		t.Fatal("registered external provider key was not recognized")
	}
}
