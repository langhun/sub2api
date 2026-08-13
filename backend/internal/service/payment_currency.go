package service

import (
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func paymentProviderConfigCurrency(providerKey string, cfg map[string]string) string {
	switch strings.TrimSpace(providerKey) {
	case payment.TypeStripe, payment.TypeAirwallex:
		currency, err := payment.NormalizePaymentCurrency(cfg["currency"])
		if err == nil {
			return currency
		}
	}
	if definition, ok := payment.ExternalProviderDefinitionFor(providerKey); ok {
		currency, err := payment.NormalizePaymentCurrency(cfg[definition.CurrencyConfigKey])
		if err == nil {
			return currency
		}
		if currency, err := payment.NormalizePaymentCurrency(definition.DefaultCurrency); err == nil {
			return currency
		}
	}
	return payment.DefaultPaymentCurrency
}

func PaymentOrderCurrency(order *dbent.PaymentOrder) string {
	if snapshot := psOrderProviderSnapshot(order); snapshot != nil {
		if currency, err := payment.NormalizePaymentCurrency(snapshot.Currency); err == nil {
			return currency
		}
	}
	return payment.DefaultPaymentCurrency
}
