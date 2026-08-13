package provider

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type externalFactoryTestProvider struct{}

func (externalFactoryTestProvider) Name() string        { return "test" }
func (externalFactoryTestProvider) ProviderKey() string { return "external-test" }
func (externalFactoryTestProvider) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{"external-test"}
}
func (externalFactoryTestProvider) CreatePayment(context.Context, payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	return nil, nil
}
func (externalFactoryTestProvider) QueryOrder(context.Context, string) (*payment.QueryOrderResponse, error) {
	return nil, nil
}
func (externalFactoryTestProvider) VerifyNotification(context.Context, string, map[string]string) (*payment.PaymentNotification, error) {
	return nil, nil
}
func (externalFactoryTestProvider) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, nil
}

func TestCreateProviderUsesExternalFactory(t *testing.T) {
	RegisterExternalProvider("external-test", func(instanceID string, config map[string]string) (payment.Provider, error) {
		require.Equal(t, "42", instanceID)
		require.Equal(t, "value", config["key"])
		return externalFactoryTestProvider{}, nil
	})

	provider, err := CreateProvider("external-test", "42", map[string]string{"key": "value"})
	require.NoError(t, err)
	require.Equal(t, "external-test", provider.ProviderKey())
}
