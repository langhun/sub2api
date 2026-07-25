package checkin

import (
	"context"
	"testing"

	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/stretchr/testify/require"
)

type adapterSettingsStore struct{ values map[string]string }

func (s adapterSettingsStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (adapterSettingsStore) SetMultiple(context.Context, map[string]string) error { return nil }

type adapterCodeGenerator struct {
	codeType string
}

func (g *adapterCodeGenerator) GenerateCode(_ context.Context, codeType string) (string, error) {
	g.codeType = codeType
	return "checkin-ledger-code", nil
}

type adapterBalanceCache struct {
	userID int64
}

func (c *adapterBalanceCache) InvalidateUserBalance(_ context.Context, userID int64) error {
	c.userID = userID
	return nil
}

func TestRegistrySettingsAdapterMapsCompleteCheckinSettings(t *testing.T) {
	registry := customsettings.NewRegistry(adapterSettingsStore{values: map[string]string{
		"checkin_enabled":               "true",
		"checkin_min_balance":           "2.5",
		"checkin_max_balance":           "9.5",
		"checkin_luck_enabled":          "true",
		"checkin_luck_min_multiplier":   "0.5",
		"checkin_luck_max_multiplier":   "4",
		"checkin_blindbox_enabled":      "true",
		"checkin_blindbox_trigger_type": "total",
		"checkin_blindbox_interval":     "12",
	}})

	settings, err := NewRegistrySettingsAdapter(registry).GetActivitySettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, 2.5, settings.Checkin.MinimumReward)
	require.Equal(t, 9.5, settings.Checkin.MaximumReward)
	require.True(t, settings.Checkin.Enabled)
	require.True(t, settings.Checkin.LuckEnabled)
	require.Equal(t, 0.5, settings.Checkin.MinimumMultiplier)
	require.Equal(t, 4.0, settings.Checkin.MaximumMultiplier)
	require.True(t, settings.Blindbox.Enabled)
	require.Equal(t, "total", settings.Blindbox.TriggerType)
	require.Equal(t, 12, settings.Blindbox.Interval)
}

func TestCodeFormatGeneratorDelegatesCheckinAdjustmentType(t *testing.T) {
	source := &adapterCodeGenerator{}

	code, err := NewCodeFormatGenerator(source).GenerateCheckinCode(context.Background(), adjustmentTypeCheckinLuck)

	require.NoError(t, err)
	require.Equal(t, "checkin-ledger-code", code)
	require.Equal(t, adjustmentTypeCheckinLuck, source.codeType)
}

func TestBalanceCacheInvalidatorDelegatesToPlatformCache(t *testing.T) {
	cache := &adapterBalanceCache{}

	err := NewBalanceCacheInvalidator(cache).InvalidateBalance(context.Background(), 9)

	require.NoError(t, err)
	require.Equal(t, int64(9), cache.userID)
}
