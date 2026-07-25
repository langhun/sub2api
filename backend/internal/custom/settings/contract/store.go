// Package contract contains the narrow persistence port shared by Overlay
// settings modules. It deliberately exposes no core setting-domain types.
package contract

import (
	"context"
	"math"
	"strconv"
	"strings"
)

// Store persists named setting values. The core SettingService supplies the
// implementation at the composition root, while modules only depend on this
// small port.
type Store interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, values map[string]string) error
}

func Bool(values map[string]string, key string, fallback bool) bool {
	raw, ok := values[key]
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return parsed
}

func Float(values map[string]string, key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(values[key]), 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}

// PositiveInt preserves the legacy configuration behavior: an absent, zero,
// or invalid value resolves to the module default.
func PositiveInt(values map[string]string, key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(values[key]))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func FormatFloat(value float64, precision int) string {
	return strconv.FormatFloat(value, 'f', precision, 64)
}
