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
	KeyClusterEnabled     = "registration_cluster_protection_enabled"
	KeyClusterWindow6h    = "registration_cluster_window_6h"
	KeyClusterWindow24h   = "registration_cluster_window_24h"
	KeyClusterMedium      = "registration_cluster_medium_threshold"
	KeyClusterHigh        = "registration_cluster_high_threshold"
	KeyAutoHighLock       = "registration_auto_high_risk_lock_enabled"
	KeyTrialEnabled       = "registration_trial_enabled"
	KeyTrialHours         = "registration_trial_hours"
	KeyTrialCostLimit     = "registration_trial_cost_limit"
	KeyTrialRPM           = "registration_trial_rpm_limit"
	KeyTrialConcurrency   = "registration_trial_concurrency"
	KeyPaymentUnlock      = "registration_payment_unlock_enabled"
	KeyPaymentAmount      = "registration_payment_unlock_amount"
)

type Config struct {
	AliasEnabled       bool
	FingerprintEnabled bool
	ClusterEnabled     bool
	ClusterWindow6h    int
	ClusterWindow24h   int
	ClusterMedium      int
	ClusterHigh        int
	AutoHighLock       bool
	TrialEnabled       bool
	TrialHours         int
	TrialCostLimit     float64
	TrialRPM           int
	TrialConcurrency   int
	PaymentUnlock      bool
	PaymentAmount      float64
}

func Default() Config {
	return Config{
		AliasEnabled: true, FingerprintEnabled: true, ClusterEnabled: true,
		ClusterWindow6h: 3, ClusterWindow24h: 5, ClusterMedium: 3, ClusterHigh: 5,
		AutoHighLock: true, TrialEnabled: true, TrialHours: 24, TrialCostLimit: 2,
		TrialRPM: 10, TrialConcurrency: 2, PaymentUnlock: true, PaymentAmount: 10,
	}
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

// Keys is the complete set of settings consumed by the registration
// protection module. Keeping this list complete is important: the admin
// settings endpoint persists the flat keys, while every runtime policy read
// uses this list to load them back. Omitting a key silently restores its
// default value on the next read.
func Keys() []string {
	return []string{
		KeyAliasEnabled,
		KeyFingerprintEnabled,
		KeyClusterEnabled,
		KeyClusterWindow6h,
		KeyClusterWindow24h,
		KeyClusterMedium,
		KeyClusterHigh,
		KeyAutoHighLock,
		KeyTrialEnabled,
		KeyTrialHours,
		KeyTrialCostLimit,
		KeyTrialRPM,
		KeyTrialConcurrency,
		KeyPaymentUnlock,
		KeyPaymentAmount,
	}
}

func FromValues(values map[string]string) Config {
	defaults := Default()
	return Config{
		AliasEnabled:       contract.Bool(values, KeyAliasEnabled, defaults.AliasEnabled),
		FingerprintEnabled: contract.Bool(values, KeyFingerprintEnabled, defaults.FingerprintEnabled),
		ClusterEnabled:     contract.Bool(values, KeyClusterEnabled, defaults.ClusterEnabled),
		ClusterWindow6h:    contract.PositiveInt(values, KeyClusterWindow6h, defaults.ClusterWindow6h),
		ClusterWindow24h:   contract.PositiveInt(values, KeyClusterWindow24h, defaults.ClusterWindow24h),
		ClusterMedium:      contract.PositiveInt(values, KeyClusterMedium, defaults.ClusterMedium),
		ClusterHigh:        contract.PositiveInt(values, KeyClusterHigh, defaults.ClusterHigh),
		AutoHighLock:       contract.Bool(values, KeyAutoHighLock, defaults.AutoHighLock),
		TrialEnabled:       contract.Bool(values, KeyTrialEnabled, defaults.TrialEnabled),
		TrialHours:         contract.PositiveInt(values, KeyTrialHours, defaults.TrialHours),
		TrialCostLimit:     contract.Float(values, KeyTrialCostLimit, defaults.TrialCostLimit),
		TrialRPM:           contract.PositiveInt(values, KeyTrialRPM, defaults.TrialRPM),
		TrialConcurrency:   contract.PositiveInt(values, KeyTrialConcurrency, defaults.TrialConcurrency),
		PaymentUnlock:      contract.Bool(values, KeyPaymentUnlock, defaults.PaymentUnlock),
		PaymentAmount:      contract.Float(values, KeyPaymentAmount, defaults.PaymentAmount),
	}
}

func Values(config Config) (map[string]string, error) {
	return map[string]string{
		KeyAliasEnabled:       fmt.Sprintf("%t", config.AliasEnabled),
		KeyFingerprintEnabled: fmt.Sprintf("%t", config.FingerprintEnabled),
		KeyClusterEnabled:     fmt.Sprintf("%t", config.ClusterEnabled), KeyClusterWindow6h: fmt.Sprintf("%d", config.ClusterWindow6h),
		KeyClusterWindow24h: fmt.Sprintf("%d", config.ClusterWindow24h), KeyClusterMedium: fmt.Sprintf("%d", config.ClusterMedium),
		KeyClusterHigh: fmt.Sprintf("%d", config.ClusterHigh), KeyAutoHighLock: fmt.Sprintf("%t", config.AutoHighLock),
		KeyTrialEnabled: fmt.Sprintf("%t", config.TrialEnabled), KeyTrialHours: fmt.Sprintf("%d", config.TrialHours),
		KeyTrialCostLimit: fmt.Sprintf("%g", config.TrialCostLimit), KeyTrialRPM: fmt.Sprintf("%d", config.TrialRPM),
		KeyTrialConcurrency: fmt.Sprintf("%d", config.TrialConcurrency), KeyPaymentUnlock: fmt.Sprintf("%t", config.PaymentUnlock),
		KeyPaymentAmount: fmt.Sprintf("%g", config.PaymentAmount),
	}, nil
}

func Validate(config Config) error {
	if config.ClusterWindow6h <= 0 || config.ClusterWindow24h <= 0 || config.ClusterWindow24h < config.ClusterWindow6h {
		return fmt.Errorf("registration cluster windows must be positive and ordered")
	}
	if config.ClusterMedium <= 0 || config.ClusterHigh < config.ClusterMedium || config.TrialHours <= 0 || config.TrialRPM <= 0 || config.TrialConcurrency <= 0 || config.TrialCostLimit <= 0 || config.PaymentAmount <= 0 {
		return fmt.Errorf("registration protection limits must be positive")
	}
	return nil
}

func Public(config Config) map[string]any {
	return map[string]any{
		KeyAliasEnabled:       config.AliasEnabled,
		KeyFingerprintEnabled: config.FingerprintEnabled,
		KeyClusterEnabled:     config.ClusterEnabled, KeyClusterWindow6h: config.ClusterWindow6h, KeyClusterWindow24h: config.ClusterWindow24h,
		KeyClusterMedium: config.ClusterMedium, KeyClusterHigh: config.ClusterHigh, KeyAutoHighLock: config.AutoHighLock,
		KeyTrialEnabled: config.TrialEnabled, KeyTrialHours: config.TrialHours, KeyTrialCostLimit: config.TrialCostLimit,
		KeyTrialRPM: config.TrialRPM, KeyTrialConcurrency: config.TrialConcurrency, KeyPaymentUnlock: config.PaymentUnlock,
		KeyPaymentAmount: config.PaymentAmount,
	}
}
