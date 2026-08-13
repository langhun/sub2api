package payment

import "testing"

func TestExternalProviderDefinitionRegistration(t *testing.T) {
	const providerKey = "external-definition-test"
	RegisterExternalProviderDefinition(ExternalProviderDefinition{
		ProviderKey:       providerKey,
		VisibleMethods:    []PaymentType{providerKey},
		CurrencyConfigKey: "fiat",
		DefaultCurrency:   "CNY",
		WebhookPath:       "/external-test/",
	})

	definition, ok := ExternalProviderDefinitionFor(providerKey)
	if !ok {
		t.Fatal("registered external provider definition was not found")
	}
	if definition.WebhookPath != "external-test" {
		t.Fatalf("webhook path = %q, want external-test", definition.WebhookPath)
	}
	if !IsExternalProvider(providerKey) {
		t.Fatal("registered external provider key was not recognized")
	}
}
