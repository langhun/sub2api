// Package settings owns persisted configuration for the Overlay Apple theme.
package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	KeyEnabled          = "enable_apple_theme"
	KeyDefaultThemeMode = "default_theme_mode"
	KeyLegacyThemeStyle = "theme_style"
)

const (
	ModeAuto  = "auto"
	ModeLight = "light"
	ModeDark  = "dark"
	ModeTime  = "time"
)

const (
	LegacyThemeGlass   = "glass"
	LegacyThemeClassic = "classic"
)

type Config struct {
	Enabled          bool
	DefaultThemeMode string
}

func Default() Config {
	return Config{Enabled: true, DefaultThemeMode: ModeAuto}
}

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("apple theme settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read apple theme settings: %w", err)
	}
	return FromValues(values), nil
}

func (r *Reader) Write(ctx context.Context, config Config) error {
	values, err := Values(config)
	if err != nil {
		return err
	}
	if r == nil || r.store == nil {
		return fmt.Errorf("apple theme settings store is required")
	}
	return r.store.SetMultiple(ctx, values)
}

func Keys() []string {
	return []string{KeyEnabled, KeyDefaultThemeMode, KeyLegacyThemeStyle}
}

func FromValues(values map[string]string) Config {
	config := Default()
	if raw, ok := values[KeyEnabled]; ok {
		if enabled, err := strconv.ParseBool(strings.TrimSpace(raw)); err == nil {
			config.Enabled = enabled
		}
	} else if strings.EqualFold(strings.TrimSpace(values[KeyLegacyThemeStyle]), LegacyThemeClassic) {
		config.Enabled = false
	}
	config.DefaultThemeMode = normalizeMode(values[KeyDefaultThemeMode])
	return config
}

func Values(config Config) (map[string]string, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	legacyStyle := LegacyThemeClassic
	if config.Enabled {
		legacyStyle = LegacyThemeGlass
	}
	return map[string]string{
		KeyEnabled:          strconv.FormatBool(config.Enabled),
		KeyDefaultThemeMode: config.DefaultThemeMode,
		KeyLegacyThemeStyle: legacyStyle,
	}, nil
}

func Validate(config Config) error {
	if config.DefaultThemeMode != ModeAuto && config.DefaultThemeMode != ModeLight && config.DefaultThemeMode != ModeDark && config.DefaultThemeMode != ModeTime {
		return fmt.Errorf("default_theme_mode must be %q, %q, %q, or %q", ModeAuto, ModeLight, ModeDark, ModeTime)
	}
	return nil
}

func Public(config Config) map[string]any {
	legacyStyle := LegacyThemeClassic
	if config.Enabled {
		legacyStyle = LegacyThemeGlass
	}
	return map[string]any{
		KeyEnabled:          config.Enabled,
		KeyDefaultThemeMode: config.DefaultThemeMode,
		KeyLegacyThemeStyle: legacyStyle,
	}
}

func normalizeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ModeLight:
		return ModeLight
	case ModeDark:
		return ModeDark
	case ModeTime:
		return ModeTime
	default:
		return ModeAuto
	}
}
