//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestExternalProviderDefinitionIntegratesPaymentConfiguration(t *testing.T) {
	const providerKey = "external-payment-service-test"
	payment.RegisterExternalProviderDefinition(payment.ExternalProviderDefinition{
		ProviderKey:                       providerKey,
		SensitiveConfigFields:             []string{"apiToken"},
		PendingOrderProtectedConfigFields: []string{"apiToken", "fiat"},
		VisibleMethods:                    []payment.PaymentType{providerKey},
		CurrencyConfigKey:                 "fiat",
		DefaultCurrency:                   "CNY",
		WebhookPath:                       "external-payment-service-test",
	})

	require.NoError(t, validateProviderRequest(providerKey, "External", providerKey))
	require.True(t, isSensitiveProviderConfigField(providerKey, "apiToken"))
	require.False(t, isSensitiveProviderConfigField(providerKey, "fiat"))
	require.True(t, hasPendingOrderProtectedConfigChange(providerKey,
		map[string]string{"apiToken": "old", "fiat": "CNY"},
		map[string]string{"apiToken": "new", "fiat": "CNY"},
	))
	require.False(t, hasPendingOrderProtectedConfigChange(providerKey,
		map[string]string{"apiToken": "same", "fiat": "CNY"},
		map[string]string{"apiToken": "same", "fiat": "CNY"},
	))
	require.Equal(t, []string{providerKey}, enabledVisibleMethodsForProvider(providerKey, ""))
	require.Equal(t, "CNY", paymentProviderConfigCurrency(providerKey, map[string]string{"fiat": "CNY"}))
	require.Equal(t, "CNY", paymentProviderConfigCurrency(providerKey, map[string]string{}))
}
