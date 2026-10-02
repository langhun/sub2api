// Package settings owns presentation-only payment settings for the Overlay.
package settings

import (
	"context"
	"fmt"
	"math"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const KeyQuickAmountMax = "payment_quick_amount_max"

// Config controls which preset recharge amount is the largest one shown to
// users. Zero means that all built-in presets remain available.
type Config struct {
	QuickAmountMax float64
}

func Default() Config { return Config{QuickAmountMax: 500} }

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("payment display settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read payment display settings: %w", err)
	}
	return FromValues(values), nil
}

func Keys() []string { return []string{KeyQuickAmountMax} }

func FromValues(values map[string]string) Config {
	defaults := Default()
	return Config{QuickAmountMax: contract.Float(values, KeyQuickAmountMax, defaults.QuickAmountMax)}
}

func Values(config Config) (map[string]string, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	return map[string]string{
		KeyQuickAmountMax: contract.FormatFloat(config.QuickAmountMax, 8),
	}, nil
}

func Validate(config Config) error {
	if math.IsNaN(config.QuickAmountMax) || math.IsInf(config.QuickAmountMax, 0) || config.QuickAmountMax < 0 {
		return fmt.Errorf("%s must be finite and nonnegative", KeyQuickAmountMax)
	}
	return nil
}

func Public(config Config) map[string]any {
	return map[string]any{KeyQuickAmountMax: config.QuickAmountMax}
}
