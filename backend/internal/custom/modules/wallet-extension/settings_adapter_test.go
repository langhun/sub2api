package walletextension

import (
	"context"
	"testing"

	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/stretchr/testify/require"
)

type walletSettingsStore struct{ values map[string]string }

func (s walletSettingsStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (walletSettingsStore) SetMultiple(context.Context, map[string]string) error { return nil }

func TestRegistrySettingsAdapterMapsWalletAndLeaderboardPolicy(t *testing.T) {
	registry := customsettings.NewRegistry(walletSettingsStore{values: map[string]string{
		"transfer_enabled":             "true",
		"transfer_fee_rate":            "0.025",
		"transfer_min_amount":          "2.5",
		"transfer_max_amount":          "250",
		"transfer_daily_limit":         "500",
		"transfer_daily_count_limit":   "8",
		"transfer_vip_fee_exempt":      "true",
		"leaderboard_enabled":          "true",
		"leaderboard_transfer_enabled": "true",
	}})

	adapter := NewRegistrySettingsAdapter(registry)
	settings, err := adapter.GetWalletExtensionSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.DirectTransfer.Enabled)
	require.Equal(t, 0.025, settings.DirectTransfer.FeeRate)
	require.Equal(t, 2.5, settings.DirectTransfer.MinimumAmount)
	require.Equal(t, 250.0, settings.DirectTransfer.MaximumAmount)
	require.Equal(t, 500.0, settings.DirectTransfer.DailyLimit)
	require.Equal(t, 8, settings.DirectTransfer.DailyCountLimit)
	require.True(t, settings.DirectTransfer.VIPFeeExempt)

	leaderboardReader, ok := adapter.(transferLeaderboardSettingsReader)
	require.True(t, ok)
	leaderboard, err := leaderboardReader.GetWalletTransferLeaderboardSettings(context.Background())
	require.NoError(t, err)
	require.True(t, leaderboard.Enabled)
}

func TestRegistrySettingsAdapterRequiresRegistry(t *testing.T) {
	_, err := NewRegistrySettingsAdapter(nil).GetWalletExtensionSettings(context.Background())
	require.ErrorContains(t, err, "custom settings registry is required")
}
