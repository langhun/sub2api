package gamehall

import (
	"context"
	"testing"

	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/stretchr/testify/require"
)

type gameHallSettingsStore struct{ values map[string]string }

func (s gameHallSettingsStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (gameHallSettingsStore) SetMultiple(context.Context, map[string]string) error { return nil }

func TestRegistrySettingsAdapterReadsGameHallConfiguration(t *testing.T) {
	registry := customsettings.NewRegistry(gameHallSettingsStore{values: map[string]string{
		"game_hall_enabled":                 "true",
		"game_slots_enabled":                "true",
		"game_slots_min_bet":                "2",
		"game_slots_max_bet":                "50",
		"game_exchange_min_amount":          "5",
		"game_exchange_max_amount":          "100",
		"game_exchange_daily_limit":         "200",
		"game_exchange_allow_dg_to_balance": "false",
	}})

	config, err := NewRegistrySettingsAdapter(registry).Read(context.Background())
	require.NoError(t, err)
	require.True(t, config.Enabled)
	require.True(t, config.SlotsEnabled)
	require.Equal(t, 2.0, config.SlotsMinBet)
	require.Equal(t, 50.0, config.SlotsMaxBet)
	require.Equal(t, 5.0, config.ExchangeMinAmount)
	require.Equal(t, 100.0, config.ExchangeMaxAmount)
	require.Equal(t, 200.0, config.ExchangeDailyLimit)
	require.False(t, config.ExchangeAllowDGToBalance)
}

func TestRegistrySettingsAdapterRequiresRegistry(t *testing.T) {
	_, err := NewRegistrySettingsAdapter(nil).Read(context.Background())
	require.ErrorContains(t, err, "custom settings registry is required")
}
