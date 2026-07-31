// Package settings owns the persisted configuration for Overlay activity features.
package settings

import (
	"context"
	"fmt"
	"math"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	keyCheckinEnabled            = "checkin_enabled"
	keyCheckinMinBalance         = "checkin_min_balance"
	keyCheckinMaxBalance         = "checkin_max_balance"
	keyCheckinLuckEnabled        = "checkin_luck_enabled"
	keyCheckinLuckMinMultiplier  = "checkin_luck_min_multiplier"
	keyCheckinLuckMaxMultiplier  = "checkin_luck_max_multiplier"
	keyCheckinBlindboxEnabled    = "checkin_blindbox_enabled"
	keyCheckinBlindboxTrigger    = "checkin_blindbox_trigger_type"
	keyCheckinBlindboxInterval   = "checkin_blindbox_interval"
	keyRedPacketEnabled          = "redpacket_enabled"
	keyRedPacketMaxCount         = "redpacket_max_count"
	keyRedPacketExpireHours      = "redpacket_expire_hours"
	keyUsageQueryEnabled         = "usage_query_enabled"
	keyLeaderboardEnabled        = "leaderboard_enabled"
	keyLeaderboardBalanceEnabled = "leaderboard_balance_enabled"
	keyLeaderboardConsumeEnabled = "leaderboard_consumption_enabled"
	keyLeaderboardCheckinEnabled = "leaderboard_checkin_enabled"
	keyLeaderboardIncludeAdmin   = "leaderboard_include_admin"
)

// Config contains all activity-owned configuration. Values intentionally keep
// the established persisted keys so existing deployments need no data rewrite.
type Config struct {
	CheckinEnabled                bool
	CheckinMinBalance             float64
	CheckinMaxBalance             float64
	CheckinLuckEnabled            bool
	CheckinLuckMinMultiplier      float64
	CheckinLuckMaxMultiplier      float64
	CheckinBlindboxEnabled        bool
	CheckinBlindboxTriggerType    string
	CheckinBlindboxInterval       int
	RedPacketEnabled              bool
	RedPacketMaxCount             int
	RedPacketExpireHours          int
	UsageQueryEnabled             bool
	LeaderboardEnabled            bool
	LeaderboardBalanceEnabled     bool
	LeaderboardConsumptionEnabled bool
	LeaderboardCheckinEnabled     bool
	LeaderboardIncludeAdmin       bool
}

func Default() Config {
	return Config{
		CheckinMinBalance:             0.1,
		CheckinMaxBalance:             1,
		CheckinLuckMinMultiplier:      0.1,
		CheckinLuckMaxMultiplier:      3,
		CheckinBlindboxTriggerType:    "streak",
		CheckinBlindboxInterval:       7,
		RedPacketMaxCount:             100,
		RedPacketExpireHours:          24,
		UsageQueryEnabled:             true,
		LeaderboardEnabled:            true,
		LeaderboardBalanceEnabled:     true,
		LeaderboardConsumptionEnabled: true,
		LeaderboardCheckinEnabled:     true,
	}
}

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("activity settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read activity settings: %w", err)
	}
	return FromValues(values), nil
}

func (r *Reader) Write(ctx context.Context, config Config) error {
	values, err := Values(config)
	if err != nil {
		return err
	}
	if r == nil || r.store == nil {
		return fmt.Errorf("activity settings store is required")
	}
	return r.store.SetMultiple(ctx, values)
}

func Keys() []string {
	return []string{
		keyCheckinEnabled, keyCheckinMinBalance, keyCheckinMaxBalance,
		keyCheckinLuckEnabled, keyCheckinLuckMinMultiplier, keyCheckinLuckMaxMultiplier,
		keyCheckinBlindboxEnabled, keyCheckinBlindboxTrigger, keyCheckinBlindboxInterval,
		keyRedPacketEnabled, keyRedPacketMaxCount, keyRedPacketExpireHours,
		keyUsageQueryEnabled, keyLeaderboardEnabled, keyLeaderboardBalanceEnabled,
		keyLeaderboardConsumeEnabled, keyLeaderboardCheckinEnabled,
		keyLeaderboardIncludeAdmin,
	}
}

func FromValues(values map[string]string) Config {
	defaults := Default()
	return Config{
		CheckinEnabled:                contract.Bool(values, keyCheckinEnabled, false),
		CheckinMinBalance:             contract.Float(values, keyCheckinMinBalance, defaults.CheckinMinBalance),
		CheckinMaxBalance:             contract.Float(values, keyCheckinMaxBalance, defaults.CheckinMaxBalance),
		CheckinLuckEnabled:            contract.Bool(values, keyCheckinLuckEnabled, false),
		CheckinLuckMinMultiplier:      contract.Float(values, keyCheckinLuckMinMultiplier, defaults.CheckinLuckMinMultiplier),
		CheckinLuckMaxMultiplier:      contract.Float(values, keyCheckinLuckMaxMultiplier, defaults.CheckinLuckMaxMultiplier),
		CheckinBlindboxEnabled:        contract.Bool(values, keyCheckinBlindboxEnabled, false),
		CheckinBlindboxTriggerType:    parseTrigger(values[keyCheckinBlindboxTrigger], defaults.CheckinBlindboxTriggerType),
		CheckinBlindboxInterval:       contract.PositiveInt(values, keyCheckinBlindboxInterval, defaults.CheckinBlindboxInterval),
		RedPacketEnabled:              contract.Bool(values, keyRedPacketEnabled, false),
		RedPacketMaxCount:             contract.PositiveInt(values, keyRedPacketMaxCount, defaults.RedPacketMaxCount),
		RedPacketExpireHours:          contract.PositiveInt(values, keyRedPacketExpireHours, defaults.RedPacketExpireHours),
		UsageQueryEnabled:             contract.Bool(values, keyUsageQueryEnabled, defaults.UsageQueryEnabled),
		LeaderboardEnabled:            contract.Bool(values, keyLeaderboardEnabled, defaults.LeaderboardEnabled),
		LeaderboardBalanceEnabled:     contract.Bool(values, keyLeaderboardBalanceEnabled, defaults.LeaderboardBalanceEnabled),
		LeaderboardConsumptionEnabled: contract.Bool(values, keyLeaderboardConsumeEnabled, defaults.LeaderboardConsumptionEnabled),
		LeaderboardCheckinEnabled:     contract.Bool(values, keyLeaderboardCheckinEnabled, defaults.LeaderboardCheckinEnabled),
		LeaderboardIncludeAdmin:       contract.Bool(values, keyLeaderboardIncludeAdmin, false),
	}
}

func Values(config Config) (map[string]string, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	return map[string]string{
		keyCheckinEnabled:            fmt.Sprintf("%t", config.CheckinEnabled),
		keyCheckinMinBalance:         contract.FormatFloat(config.CheckinMinBalance, 8),
		keyCheckinMaxBalance:         contract.FormatFloat(config.CheckinMaxBalance, 8),
		keyCheckinLuckEnabled:        fmt.Sprintf("%t", config.CheckinLuckEnabled),
		keyCheckinLuckMinMultiplier:  contract.FormatFloat(config.CheckinLuckMinMultiplier, 8),
		keyCheckinLuckMaxMultiplier:  contract.FormatFloat(config.CheckinLuckMaxMultiplier, 8),
		keyCheckinBlindboxEnabled:    fmt.Sprintf("%t", config.CheckinBlindboxEnabled),
		keyCheckinBlindboxTrigger:    config.CheckinBlindboxTriggerType,
		keyCheckinBlindboxInterval:   fmt.Sprintf("%d", config.CheckinBlindboxInterval),
		keyRedPacketEnabled:          fmt.Sprintf("%t", config.RedPacketEnabled),
		keyRedPacketMaxCount:         fmt.Sprintf("%d", config.RedPacketMaxCount),
		keyRedPacketExpireHours:      fmt.Sprintf("%d", config.RedPacketExpireHours),
		keyUsageQueryEnabled:         fmt.Sprintf("%t", config.UsageQueryEnabled),
		keyLeaderboardEnabled:        fmt.Sprintf("%t", config.LeaderboardEnabled),
		keyLeaderboardBalanceEnabled: fmt.Sprintf("%t", config.LeaderboardBalanceEnabled),
		keyLeaderboardConsumeEnabled: fmt.Sprintf("%t", config.LeaderboardConsumptionEnabled),
		keyLeaderboardCheckinEnabled: fmt.Sprintf("%t", config.LeaderboardCheckinEnabled),
		keyLeaderboardIncludeAdmin:   fmt.Sprintf("%t", config.LeaderboardIncludeAdmin),
	}, nil
}

func Validate(config Config) error {
	if err := validateRange("checkin balance", config.CheckinMinBalance, config.CheckinMaxBalance); err != nil {
		return err
	}
	if err := validateRange("checkin luck multiplier", config.CheckinLuckMinMultiplier, config.CheckinLuckMaxMultiplier); err != nil {
		return err
	}
	if config.CheckinBlindboxInterval < 0 || config.RedPacketMaxCount < 0 || config.RedPacketExpireHours < 0 {
		return fmt.Errorf("activity count, interval, and expiry settings must be nonnegative")
	}
	if config.CheckinBlindboxTriggerType != "streak" && config.CheckinBlindboxTriggerType != "total" {
		return fmt.Errorf("checkin_blindbox_trigger_type must be streak or total")
	}
	return nil
}

func Public(config Config) map[string]any {
	return map[string]any{
		keyCheckinEnabled:            config.CheckinEnabled,
		keyCheckinLuckEnabled:        config.CheckinLuckEnabled,
		keyCheckinBlindboxEnabled:    config.CheckinBlindboxEnabled,
		keyRedPacketEnabled:          config.RedPacketEnabled,
		keyUsageQueryEnabled:         config.UsageQueryEnabled,
		keyLeaderboardEnabled:        config.LeaderboardEnabled,
		keyLeaderboardBalanceEnabled: config.LeaderboardBalanceEnabled,
		keyLeaderboardConsumeEnabled: config.LeaderboardConsumptionEnabled,
		keyLeaderboardCheckinEnabled: config.LeaderboardCheckinEnabled,
		keyLeaderboardIncludeAdmin:   config.LeaderboardIncludeAdmin,
	}
}

func parseTrigger(value, fallback string) string {
	if value == "streak" || value == "total" {
		return value
	}
	return fallback
}

func validateRange(name string, minValue, maxValue float64) error {
	if !finiteNonnegative(minValue) || !finiteNonnegative(maxValue) {
		return fmt.Errorf("%s values must be finite and nonnegative", name)
	}
	if maxValue < minValue {
		return fmt.Errorf("%s maximum must be greater than or equal to minimum", name)
	}
	return nil
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
