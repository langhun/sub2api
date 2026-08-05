// Package settings owns the persisted configuration for the Overlay recharge
// promotion campaign.
package settings

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	KeyEnabled   = "recharge_promotion_enabled"
	KeyBonusRate = "recharge_promotion_bonus_rate"
	KeyMinAmount = "recharge_promotion_min_amount"
	KeyMaxBonus  = "recharge_promotion_max_bonus"
	KeyStartAt   = "recharge_promotion_start_at"
	KeyEndAt     = "recharge_promotion_end_at"
)

// Config contains one global recharge-promotion campaign. Zero StartAt or
// EndAt means that side of the campaign window is unrestricted.
type Config struct {
	Enabled   bool
	BonusRate float64
	MinAmount float64
	MaxBonus  float64
	StartAt   time.Time
	EndAt     time.Time
}

func Default() Config { return Config{} }

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("recharge promotion settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read recharge promotion settings: %w", err)
	}
	return FromValues(values), nil
}

func (r *Reader) Write(ctx context.Context, config Config) error {
	values, err := Values(config)
	if err != nil {
		return err
	}
	if r == nil || r.store == nil {
		return fmt.Errorf("recharge promotion settings store is required")
	}
	return r.store.SetMultiple(ctx, values)
}

func Keys() []string {
	return []string{KeyEnabled, KeyBonusRate, KeyMinAmount, KeyMaxBonus, KeyStartAt, KeyEndAt}
}

func FromValues(values map[string]string) Config {
	defaults := Default()
	return Config{
		Enabled:   contract.Bool(values, KeyEnabled, defaults.Enabled),
		BonusRate: contract.Float(values, KeyBonusRate, defaults.BonusRate),
		MinAmount: contract.Float(values, KeyMinAmount, defaults.MinAmount),
		MaxBonus:  contract.Float(values, KeyMaxBonus, defaults.MaxBonus),
		StartAt:   parseStoredTime(values[KeyStartAt]),
		EndAt:     parseStoredTime(values[KeyEndAt]),
	}
}

func Values(config Config) (map[string]string, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	return map[string]string{
		KeyEnabled:   fmt.Sprintf("%t", config.Enabled),
		KeyBonusRate: contract.FormatFloat(config.BonusRate, 2),
		KeyMinAmount: contract.FormatFloat(config.MinAmount, 2),
		KeyMaxBonus:  contract.FormatFloat(config.MaxBonus, 2),
		KeyStartAt:   FormatTime(config.StartAt),
		KeyEndAt:     FormatTime(config.EndAt),
	}, nil
}

func Validate(config Config) error {
	if !finiteNonnegative(config.BonusRate) || config.BonusRate > 100 || !hasTwoDecimalPlaces(config.BonusRate) {
		return fmt.Errorf("recharge_promotion_bonus_rate must be between 0 and 100 with at most two decimal places")
	}
	if config.Enabled && config.BonusRate <= 0 {
		return fmt.Errorf("recharge_promotion_bonus_rate must be greater than 0 when recharge promotion is enabled")
	}
	if !finiteNonnegative(config.MinAmount) || !hasTwoDecimalPlaces(config.MinAmount) {
		return fmt.Errorf("recharge_promotion_min_amount must be nonnegative with at most two decimal places")
	}
	if !finiteNonnegative(config.MaxBonus) || !hasTwoDecimalPlaces(config.MaxBonus) {
		return fmt.Errorf("recharge_promotion_max_bonus must be nonnegative with at most two decimal places")
	}
	if !config.StartAt.IsZero() && !config.EndAt.IsZero() && !config.StartAt.Before(config.EndAt) {
		return fmt.Errorf("recharge_promotion_start_at must be before recharge_promotion_end_at")
	}
	return nil
}

// Public returns only the values needed to render the client-side estimate.
// The server remains the source of truth when an order is created.
func Public(config Config) map[string]any {
	return map[string]any{
		KeyEnabled:   config.Enabled,
		KeyBonusRate: config.BonusRate,
		KeyMinAmount: config.MinAmount,
		KeyMaxBonus:  config.MaxBonus,
		KeyStartAt:   FormatTime(config.StartAt),
		KeyEndAt:     FormatTime(config.EndAt),
	}
}

func parseStoredTime(value string) time.Time {
	parsed, err := ParseTime(value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// ParseTime parses a campaign boundary. An empty value means that boundary is
// unrestricted; non-empty values must be RFC3339.
func ParseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("must be RFC3339: %w", err)
	}
	return parsed.UTC(), nil
}

// FormatTime returns the canonical UTC representation used by persistence and
// the public settings projection.
func FormatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func hasTwoDecimalPlaces(value float64) bool {
	return math.Abs(value*100-math.Round(value*100)) < 1e-9
}
