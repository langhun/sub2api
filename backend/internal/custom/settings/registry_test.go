package settings

import (
	"context"
	"fmt"
	"testing"

	rechargepromotionsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
	"github.com/stretchr/testify/require"
)

type memoryStore struct {
	values map[string]string
	writes int
}

func (s *memoryStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *memoryStore) SetMultiple(_ context.Context, values map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for key, value := range values {
		s.values[key] = value
	}
	s.writes++
	return nil
}

func TestRegistryReadUsesLegacyDefaultsAndProjectsPublicValues(t *testing.T) {
	registry := NewRegistry(&memoryStore{values: map[string]string{}})

	snapshot, err := registry.Read(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0.1, snapshot.Activity.CheckinMinBalance)
	require.Equal(t, "default", snapshot.BrandHome.DefaultHomepage)
	require.True(t, snapshot.AppleTheme.Enabled)
	require.Equal(t, "auto", snapshot.AppleTheme.DefaultThemeMode)
	require.True(t, snapshot.Activity.LeaderboardEnabled)
	require.True(t, snapshot.GameHall.ExchangeAllowDGToBalance)
	require.True(t, snapshot.RegistrationProtection.AliasEnabled)
	require.True(t, snapshot.RegistrationProtection.FingerprintEnabled)

	public := Public(snapshot)
	require.Equal(t, false, public["checkin_enabled"])
	require.Equal(t, true, public["leaderboard_enabled"])
	require.NotContains(t, public, "transfer_enabled")
	require.Equal(t, false, public["game_hall_enabled"])
	require.Equal(t, "default", public["default_homepage"])
	require.Equal(t, "glass", public["theme_style"])
	require.Equal(t, true, public["enable_apple_theme"])
	require.Equal(t, "auto", public["default_theme_mode"])
	require.Equal(t, 0.01, public["game_exchange_min_amount"])
	require.Equal(t, 500.0, public["payment_quick_amount_max"])
}

func TestRegistryWritePersistsRegistrationProtectionSwitches(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	registry := NewRegistry(store)
	snapshot, err := registry.Read(context.Background())
	require.NoError(t, err)
	snapshot.RegistrationProtection.AliasEnabled = false
	snapshot.RegistrationProtection.FingerprintEnabled = false

	require.NoError(t, registry.Write(context.Background(), snapshot))
	require.Equal(t, "false", store.values["registration_alias_protection_enabled"])
	require.Equal(t, "false", store.values["registration_reward_fingerprint_protection_enabled"])

	reloaded, err := registry.Read(context.Background())
	require.NoError(t, err)
	require.False(t, reloaded.RegistrationProtection.AliasEnabled)
	require.False(t, reloaded.RegistrationProtection.FingerprintEnabled)
}

func TestRegistryWriteValidatesBeforeOneAggregatePersistence(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	registry := NewRegistry(store)
	snapshot, err := registry.Read(context.Background())
	require.NoError(t, err)

	snapshot.Activity.CheckinEnabled = true
	snapshot.BrandHome.DefaultHomepage = "dino"
	snapshot.AppleTheme.Enabled = false
	snapshot.AppleTheme.DefaultThemeMode = "dark"
	snapshot.GameHall.Enabled = true
	require.NoError(t, registry.Write(context.Background(), snapshot))
	require.Equal(t, 1, store.writes)
	require.Equal(t, "true", store.values["checkin_enabled"])
	require.Equal(t, "dino", store.values["default_homepage"])
	require.Equal(t, "classic", store.values["theme_style"])
	require.Equal(t, "false", store.values["enable_apple_theme"])
	require.Equal(t, "dark", store.values["default_theme_mode"])
	require.NotContains(t, store.values, "transfer_enabled")
	require.Equal(t, "true", store.values["game_hall_enabled"])
}

func TestRegistryWriteRejectsInvalidModuleValuesWithoutWriting(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	registry := NewRegistry(store)
	snapshot, err := registry.Read(context.Background())
	require.NoError(t, err)
	snapshot.GameHall.SlotsMaxBet = snapshot.GameHall.SlotsMinBet - 0.01

	err = registry.Write(context.Background(), snapshot)
	require.ErrorContains(t, err, "game_slots_max_bet")
	require.Zero(t, store.writes)
}

func TestRegistryWritesPaymentQuickAmountLimit(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	registry := NewRegistry(store)
	snapshot, err := registry.Read(context.Background())
	require.NoError(t, err)
	snapshot.PaymentDisplay.QuickAmountMax = 1000
	require.NoError(t, registry.Write(context.Background(), snapshot))
	require.Equal(t, "1000.00000000", store.values["payment_quick_amount_max"])
}

func TestMergePublicValuesRejectsOverlappingOwnership(t *testing.T) {
	_, err := mergePublicValues(map[string]any{"shared": true}, map[string]any{"shared": false})
	require.EqualError(t, err, fmt.Sprintf("duplicate custom public setting key %q", "shared"))
}

func TestRechargePromotionTieredPercentageCompatibility(t *testing.T) {
	rules := []rechargepromotionsettings.TieredPercentageRule{
		{MinimumAmount: 10, MaximumAmount: 50, PercentageRate: 5},
		{MinimumAmount: 50, MaximumAmount: 0, PercentageRate: 8},
	}
	snapshot := Snapshot{RechargePromotions: rechargepromotionsettings.Default()}
	mode := rechargepromotionsettings.ModeTieredPercentage
	patch := Patch{RechargePromotionMode: &mode, RechargePromotionRateTiers: &rules}
	require.True(t, patch.HasChanges())

	updated := patch.Apply(snapshot)
	require.Equal(t, rules, updated.RechargePromotions.TieredPercentageRules)
	compatibility := CompatibilityFromSnapshot(updated)
	require.Equal(t, rules, compatibility.RechargePromotionRateTiers)
	require.Equal(t, rules, compatibility.Public().RechargePromotionRateTiers)
}
