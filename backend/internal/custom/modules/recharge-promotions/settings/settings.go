// Package settings owns the persisted configuration for the recharge-promotion Overlay.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	KeyEnabled        = "recharge_promotion_enabled"
	KeyMode           = "recharge_promotion_mode"
	KeyPercentageRate = "recharge_promotion_percentage_rate"
	KeyMinimumAmount  = "recharge_promotion_minimum_amount"
	KeyMaximumBonus   = "recharge_promotion_maximum_bonus"
	KeyTierRules      = "recharge_promotion_tier_rules"
	KeyStartAt        = "recharge_promotion_start_at"
	KeyEndAt          = "recharge_promotion_end_at"

	ModePercentage = "percentage"
	ModeTiered     = "tiered"
)

type TierRule struct {
	MinimumAmount float64 `json:"minimum_amount"`
	BonusCredit   float64 `json:"bonus_credit"`
}

// Config contains one global campaign. Zero campaign boundaries are open-ended.
type Config struct {
	Enabled        bool
	Mode           string
	PercentageRate float64
	MinimumAmount  float64
	MaximumBonus   float64
	TierRules      []TierRule
	StartAt        time.Time
	EndAt          time.Time
}

func Default() Config { return Config{Mode: ModePercentage, TierRules: []TierRule{}} }

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
	return []string{KeyEnabled, KeyMode, KeyPercentageRate, KeyMinimumAmount, KeyMaximumBonus, KeyTierRules, KeyStartAt, KeyEndAt}
}

func FromValues(values map[string]string) Config {
	defaults := Default()
	config := Config{
		Enabled:        contract.Bool(values, KeyEnabled, defaults.Enabled),
		Mode:           normalizeMode(values[KeyMode]),
		PercentageRate: contract.Float(values, KeyPercentageRate, defaults.PercentageRate),
		MinimumAmount:  contract.Float(values, KeyMinimumAmount, defaults.MinimumAmount),
		MaximumBonus:   contract.Float(values, KeyMaximumBonus, defaults.MaximumBonus),
		TierRules:      parseTierRules(values[KeyTierRules]),
		StartAt:        parseStoredTime(values[KeyStartAt]),
		EndAt:          parseStoredTime(values[KeyEndAt]),
	}
	if config.TierRules == nil {
		config.TierRules = []TierRule{}
	}
	return config
}

func Values(config Config) (map[string]string, error) {
	config.Mode = normalizeMode(config.Mode)
	config.TierRules = normalizeTierRules(config.TierRules)
	if err := Validate(config); err != nil {
		return nil, err
	}
	tiers, err := json.Marshal(normalizeTierRules(config.TierRules))
	if err != nil {
		return nil, fmt.Errorf("encode recharge promotion tier rules: %w", err)
	}
	return map[string]string{
		KeyEnabled:        fmt.Sprintf("%t", config.Enabled),
		KeyMode:           normalizeMode(config.Mode),
		KeyPercentageRate: contract.FormatFloat(config.PercentageRate, 2),
		KeyMinimumAmount:  contract.FormatFloat(config.MinimumAmount, 2),
		KeyMaximumBonus:   contract.FormatFloat(config.MaximumBonus, 2),
		KeyTierRules:      string(tiers),
		KeyStartAt:        FormatTime(config.StartAt),
		KeyEndAt:          FormatTime(config.EndAt),
	}, nil
}

func Validate(config Config) error {
	if normalizeMode(config.Mode) != config.Mode && config.Mode != "" {
		return fmt.Errorf("%s must be %q or %q", KeyMode, ModePercentage, ModeTiered)
	}
	if !finiteNonnegative(config.PercentageRate) || config.PercentageRate > 100 || !twoDecimal(config.PercentageRate) {
		return fmt.Errorf("%s must be between 0 and 100 with at most two decimal places", KeyPercentageRate)
	}
	if !finiteNonnegative(config.MinimumAmount) || !twoDecimal(config.MinimumAmount) {
		return fmt.Errorf("%s must be nonnegative with at most two decimal places", KeyMinimumAmount)
	}
	if !finiteNonnegative(config.MaximumBonus) || !twoDecimal(config.MaximumBonus) {
		return fmt.Errorf("%s must be nonnegative with at most two decimal places", KeyMaximumBonus)
	}
	if !config.StartAt.IsZero() && !config.EndAt.IsZero() && !config.StartAt.Before(config.EndAt) {
		return fmt.Errorf("%s must be before %s", KeyStartAt, KeyEndAt)
	}
	mode := normalizeMode(config.Mode)
	if config.Enabled && mode == ModePercentage && config.PercentageRate <= 0 {
		return fmt.Errorf("%s must be greater than 0 when percentage promotion is enabled", KeyPercentageRate)
	}
	if mode == ModeTiered {
		if len(config.TierRules) == 0 && config.Enabled {
			return fmt.Errorf("%s requires at least one rule when tiered promotion is enabled", KeyTierRules)
		}
		for _, rule := range config.TierRules {
			if !finitePositive(rule.MinimumAmount) || !finitePositive(rule.BonusCredit) || !twoDecimal(rule.MinimumAmount) || !twoDecimal(rule.BonusCredit) {
				return fmt.Errorf("%s entries require positive two-decimal minimum_amount and bonus_credit", KeyTierRules)
			}
		}
		for index := 1; index < len(config.TierRules); index++ {
			if config.TierRules[index-1].MinimumAmount >= config.TierRules[index].MinimumAmount {
				return fmt.Errorf("%s minimum amounts must be strictly increasing", KeyTierRules)
			}
		}
	}
	return nil
}

func Public(config Config) map[string]any {
	return map[string]any{
		KeyEnabled:        config.Enabled,
		KeyMode:           normalizeMode(config.Mode),
		KeyPercentageRate: config.PercentageRate,
		KeyMinimumAmount:  config.MinimumAmount,
		KeyMaximumBonus:   config.MaximumBonus,
		KeyTierRules:      normalizeTierRules(config.TierRules),
		KeyStartAt:        FormatTime(config.StartAt),
		KeyEndAt:          FormatTime(config.EndAt),
	}
}

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

func FormatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func normalizeMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), ModeTiered) {
		return ModeTiered
	}
	return ModePercentage
}

func parseStoredTime(value string) time.Time {
	parsed, err := ParseTime(value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func parseTierRules(value string) []TierRule {
	var rules []TierRule
	if strings.TrimSpace(value) == "" || json.Unmarshal([]byte(value), &rules) != nil {
		return []TierRule{}
	}
	return normalizeTierRules(rules)
}

func normalizeTierRules(rules []TierRule) []TierRule {
	result := append([]TierRule(nil), rules...)
	sort.Slice(result, func(i, j int) bool { return result[i].MinimumAmount < result[j].MinimumAmount })
	if result == nil {
		return []TierRule{}
	}
	return result
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
func finitePositive(value float64) bool { return value > 0 && finiteNonnegative(value) }
func twoDecimal(value float64) bool     { return math.Abs(value*100-math.Round(value*100)) < 1e-9 }
