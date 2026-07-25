// Package settings owns the persisted code-format configuration for custom
// features while preserving the established setting keys consumed by core
// redemption services.
package settings

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type Config = service.CodeFormatSettings

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("code format settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read code format settings: %w", err)
	}
	return service.ParseCodeFormatSettings(values), nil
}

func Keys() []string {
	return []string{
		service.SettingKeyCodeFormatBalance,
		service.SettingKeyCodeFormatConcurrency,
		service.SettingKeyCodeFormatSubscription,
		service.SettingKeyCodeFormatInvitation,
		service.SettingKeyCodeFormatRedPacket,
	}
}

func Values(config Config) (map[string]string, error) {
	return service.CodeFormatSettingsValues(config)
}

func Validate(config Config) error {
	return config.Validate()
}
