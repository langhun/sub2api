// Package settings owns the persisted configuration for Overlay wallet extensions.
package settings

import (
	"context"
	"fmt"
	"math"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	keyTransferEnabled         = "transfer_enabled"
	keyTransferFeeRate         = "transfer_fee_rate"
	keyTransferMinAmount       = "transfer_min_amount"
	keyTransferMaxAmount       = "transfer_max_amount"
	keyTransferDailyLimit      = "transfer_daily_limit"
	keyTransferDailyCountLimit = "transfer_daily_count_limit"
	keyTransferVIPFeeExempt    = "transfer_vip_fee_exempt"
)

type Config struct {
	DirectTransferEnabled         bool
	DirectTransferFeeRate         float64
	DirectTransferMinAmount       float64
	DirectTransferMaxAmount       float64
	DirectTransferDailyLimit      float64
	DirectTransferDailyCountLimit int
	DirectTransferVIPFeeExempt    bool
}

func Default() Config {
	return Config{
		DirectTransferFeeRate:         0.01,
		DirectTransferMinAmount:       0.01,
		DirectTransferMaxAmount:       1000,
		DirectTransferDailyLimit:      1000,
		DirectTransferDailyCountLimit: 50,
	}
}

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("wallet extension settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read wallet extension settings: %w", err)
	}
	return FromValues(values), nil
}

func (r *Reader) Write(ctx context.Context, config Config) error {
	values, err := Values(config)
	if err != nil {
		return err
	}
	if r == nil || r.store == nil {
		return fmt.Errorf("wallet extension settings store is required")
	}
	return r.store.SetMultiple(ctx, values)
}

func Keys() []string {
	return []string{
		keyTransferEnabled, keyTransferFeeRate, keyTransferMinAmount, keyTransferMaxAmount,
		keyTransferDailyLimit, keyTransferDailyCountLimit, keyTransferVIPFeeExempt,
	}
}

func FromValues(values map[string]string) Config {
	defaults := Default()
	return Config{
		DirectTransferEnabled:         contract.Bool(values, keyTransferEnabled, false),
		DirectTransferFeeRate:         contract.Float(values, keyTransferFeeRate, defaults.DirectTransferFeeRate),
		DirectTransferMinAmount:       contract.Float(values, keyTransferMinAmount, defaults.DirectTransferMinAmount),
		DirectTransferMaxAmount:       contract.Float(values, keyTransferMaxAmount, defaults.DirectTransferMaxAmount),
		DirectTransferDailyLimit:      contract.Float(values, keyTransferDailyLimit, defaults.DirectTransferDailyLimit),
		DirectTransferDailyCountLimit: contract.PositiveInt(values, keyTransferDailyCountLimit, defaults.DirectTransferDailyCountLimit),
		DirectTransferVIPFeeExempt:    contract.Bool(values, keyTransferVIPFeeExempt, false),
	}
}

func Values(config Config) (map[string]string, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	return map[string]string{
		keyTransferEnabled:         fmt.Sprintf("%t", config.DirectTransferEnabled),
		keyTransferFeeRate:         contract.FormatFloat(config.DirectTransferFeeRate, 6),
		keyTransferMinAmount:       contract.FormatFloat(config.DirectTransferMinAmount, 8),
		keyTransferMaxAmount:       contract.FormatFloat(config.DirectTransferMaxAmount, 8),
		keyTransferDailyLimit:      contract.FormatFloat(config.DirectTransferDailyLimit, 8),
		keyTransferDailyCountLimit: fmt.Sprintf("%d", config.DirectTransferDailyCountLimit),
		keyTransferVIPFeeExempt:    fmt.Sprintf("%t", config.DirectTransferVIPFeeExempt),
	}, nil
}

func Validate(config Config) error {
	if !finiteNonnegative(config.DirectTransferFeeRate) || config.DirectTransferFeeRate > 1 {
		return fmt.Errorf("transfer_fee_rate must be finite and between 0 and 1")
	}
	for name, value := range map[string]float64{
		"transfer_min_amount":  config.DirectTransferMinAmount,
		"transfer_max_amount":  config.DirectTransferMaxAmount,
		"transfer_daily_limit": config.DirectTransferDailyLimit,
	} {
		if !finiteNonnegative(value) {
			return fmt.Errorf("%s must be finite and nonnegative", name)
		}
	}
	if config.DirectTransferMaxAmount > 0 && config.DirectTransferMaxAmount < config.DirectTransferMinAmount {
		return fmt.Errorf("transfer_max_amount must be 0 or greater than or equal to transfer_min_amount")
	}
	if config.DirectTransferDailyCountLimit < 0 {
		return fmt.Errorf("transfer_daily_count_limit must be nonnegative")
	}
	return nil
}

func Public(config Config) map[string]any {
	return map[string]any{keyTransferEnabled: config.DirectTransferEnabled}
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
