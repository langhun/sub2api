package provider

import (
	"fmt"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

type ExternalFactory func(instanceID string, config map[string]string) (payment.Provider, error)

var externalFactories = struct {
	sync.RWMutex
	items map[string]ExternalFactory
}{items: make(map[string]ExternalFactory)}

// RegisterExternalProvider adds an Overlay-owned provider constructor.
func RegisterExternalProvider(key string, factory ExternalFactory) {
	if key == "" || factory == nil {
		return
	}
	externalFactories.Lock()
	defer externalFactories.Unlock()
	externalFactories.items[key] = factory
}

func externalProvider(key, instanceID string, config map[string]string) (payment.Provider, error) {
	externalFactories.RLock()
	factory := externalFactories.items[key]
	externalFactories.RUnlock()
	if factory == nil {
		return nil, nil
	}
	return factory(instanceID, config)
}

// CreateProvider creates a Provider from a provider key, instance ID and decrypted config.
func CreateProvider(providerKey string, instanceID string, config map[string]string) (payment.Provider, error) {
	switch providerKey {
	case payment.TypeEasyPay:
		return NewEasyPay(instanceID, config)
	case payment.TypeAlipay:
		return NewAlipay(instanceID, config)
	case payment.TypeWxpay:
		return NewWxpay(instanceID, config)
	case payment.TypeStripe:
		return NewStripe(instanceID, config)
	case payment.TypeAirwallex:
		return NewAirwallex(instanceID, config)
	default:
		if p, err := externalProvider(providerKey, instanceID, config); p != nil || err != nil {
			return p, err
		}
		return nil, fmt.Errorf("unknown provider key: %s", providerKey)
	}
}
