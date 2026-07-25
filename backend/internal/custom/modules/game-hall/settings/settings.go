// Package settings owns the persisted configuration for the Overlay game hall.
package settings

import (
	"context"
	"fmt"
	"math"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	KeyGameHallEnabled              = "game_hall_enabled"
	KeyGameSlotsEnabled             = "game_slots_enabled"
	KeyGameSlotsMinBet              = "game_slots_min_bet"
	KeyGameSlotsMaxBet              = "game_slots_max_bet"
	KeyGameExchangeMinAmount        = "game_exchange_min_amount"
	KeyGameExchangeMaxAmount        = "game_exchange_max_amount"
	KeyGameExchangeDailyLimit       = "game_exchange_daily_limit"
	KeyGameExchangeAllowDGToBalance = "game_exchange_allow_dg_to_balance"
)

type Config struct {
	Enabled                  bool
	SlotsEnabled             bool
	SlotsMinBet              float64
	SlotsMaxBet              float64
	ExchangeMinAmount        float64
	ExchangeMaxAmount        float64
	ExchangeDailyLimit       float64
	ExchangeAllowDGToBalance bool
}

func Default() Config {
	return Config{
		SlotsMinBet:              0.01,
		SlotsMaxBet:              1000,
		ExchangeMinAmount:        0.01,
		ExchangeMaxAmount:        1000,
		ExchangeDailyLimit:       1000,
		ExchangeAllowDGToBalance: true,
	}
}

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("game hall settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read game hall settings: %w", err)
	}
	return FromValues(values), nil
}

func (r *Reader) Write(ctx context.Context, config Config) error {
	values, err := Values(config)
	if err != nil {
		return err
	}
	if r == nil || r.store == nil {
		return fmt.Errorf("game hall settings store is required")
	}
	return r.store.SetMultiple(ctx, values)
}

func Keys() []string {
	return []string{
		KeyGameHallEnabled, KeyGameSlotsEnabled, KeyGameSlotsMinBet, KeyGameSlotsMaxBet,
		KeyGameExchangeMinAmount, KeyGameExchangeMaxAmount, KeyGameExchangeDailyLimit,
		KeyGameExchangeAllowDGToBalance,
	}
}

func FromValues(values map[string]string) Config {
	defaults := Default()
	return Config{
		Enabled:                  contract.Bool(values, KeyGameHallEnabled, false),
		SlotsEnabled:             contract.Bool(values, KeyGameSlotsEnabled, false),
		SlotsMinBet:              contract.Float(values, KeyGameSlotsMinBet, defaults.SlotsMinBet),
		SlotsMaxBet:              contract.Float(values, KeyGameSlotsMaxBet, defaults.SlotsMaxBet),
		ExchangeMinAmount:        contract.Float(values, KeyGameExchangeMinAmount, defaults.ExchangeMinAmount),
		ExchangeMaxAmount:        contract.Float(values, KeyGameExchangeMaxAmount, defaults.ExchangeMaxAmount),
		ExchangeDailyLimit:       contract.Float(values, KeyGameExchangeDailyLimit, defaults.ExchangeDailyLimit),
		ExchangeAllowDGToBalance: contract.Bool(values, KeyGameExchangeAllowDGToBalance, defaults.ExchangeAllowDGToBalance),
	}
}

func Values(config Config) (map[string]string, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	return map[string]string{
		KeyGameHallEnabled:              fmt.Sprintf("%t", config.Enabled),
		KeyGameSlotsEnabled:             fmt.Sprintf("%t", config.SlotsEnabled),
		KeyGameSlotsMinBet:              contract.FormatFloat(config.SlotsMinBet, 8),
		KeyGameSlotsMaxBet:              contract.FormatFloat(config.SlotsMaxBet, 8),
		KeyGameExchangeMinAmount:        contract.FormatFloat(config.ExchangeMinAmount, 8),
		KeyGameExchangeMaxAmount:        contract.FormatFloat(config.ExchangeMaxAmount, 8),
		KeyGameExchangeDailyLimit:       contract.FormatFloat(config.ExchangeDailyLimit, 8),
		KeyGameExchangeAllowDGToBalance: fmt.Sprintf("%t", config.ExchangeAllowDGToBalance),
	}, nil
}

func Validate(config Config) error {
	if !finitePositive(config.SlotsMinBet) {
		return fmt.Errorf("game_slots_min_bet must be finite and greater than 0")
	}
	if !finiteNonnegative(config.SlotsMaxBet) || config.SlotsMaxBet < config.SlotsMinBet {
		return fmt.Errorf("game_slots_max_bet must be finite and greater than or equal to game_slots_min_bet")
	}
	if !finitePositive(config.ExchangeMinAmount) {
		return fmt.Errorf("game_exchange_min_amount must be finite and greater than 0")
	}
	if !finiteNonnegative(config.ExchangeMaxAmount) || (config.ExchangeMaxAmount > 0 && config.ExchangeMaxAmount < config.ExchangeMinAmount) {
		return fmt.Errorf("game_exchange_max_amount must be 0 or greater than or equal to game_exchange_min_amount")
	}
	if !finiteNonnegative(config.ExchangeDailyLimit) {
		return fmt.Errorf("game_exchange_daily_limit must be finite and nonnegative")
	}
	return nil
}

func Public(config Config) map[string]any {
	return map[string]any{
		KeyGameHallEnabled:              config.Enabled,
		KeyGameSlotsEnabled:             config.SlotsEnabled,
		KeyGameSlotsMinBet:              config.SlotsMinBet,
		KeyGameSlotsMaxBet:              config.SlotsMaxBet,
		KeyGameExchangeMinAmount:        config.ExchangeMinAmount,
		KeyGameExchangeMaxAmount:        config.ExchangeMaxAmount,
		KeyGameExchangeDailyLimit:       config.ExchangeDailyLimit,
		KeyGameExchangeAllowDGToBalance: config.ExchangeAllowDGToBalance,
	}
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
