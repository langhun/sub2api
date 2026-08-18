// Package settings owns registration-protection configuration for the Overlay.
package settings

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	KeyAliasEnabled       = "registration_alias_protection_enabled"
	KeyFingerprintEnabled = "registration_reward_fingerprint_protection_enabled"
)

type Config struct {
	AliasEnabled       bool
	FingerprintEnabled bool
}

func Default() Config {
	return Config{AliasEnabled: true, FingerprintEnabled: true}
}

type Reader struct {
	store contract.Store
}

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("registration protection settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read registration protection settings: %w", err)
	}
	return FromValues(values), nil
}

func (r *Reader) Write(ctx context.Context, config Config) error {
	if r == nil || r.store == nil {
		return fmt.Errorf("registration protection settings store is required")
	}
	values, err := Values(config)
	if err != nil {
		return err
	}
	return r.store.SetMultiple(ctx, values)
}

func Keys() []string { return []string{KeyAliasEnabled, KeyFingerprintEnabled} }

func FromValues(values map[string]string) Config {
	defaults := Default()
	return Config{
		AliasEnabled:       contract.Bool(values, KeyAliasEnabled, defaults.AliasEnabled),
		FingerprintEnabled: contract.Bool(values, KeyFingerprintEnabled, defaults.FingerprintEnabled),
	}
}

func Values(config Config) (map[string]string, error) {
	return map[string]string{
		KeyAliasEnabled:       fmt.Sprintf("%t", config.AliasEnabled),
		KeyFingerprintEnabled: fmt.Sprintf("%t", config.FingerprintEnabled),
	}, nil
}

func Validate(Config) error { return nil }

func Public(config Config) map[string]any {
	return map[string]any{
		KeyAliasEnabled:       config.AliasEnabled,
		KeyFingerprintEnabled: config.FingerprintEnabled,
	}
}
