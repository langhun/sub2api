package settings

import (
	"context"
	"fmt"
	"testing"

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
	require.True(t, snapshot.Activity.LeaderboardEnabled)
	require.Equal(t, 0.01, snapshot.WalletExtension.DirectTransferMinAmount)
	require.True(t, snapshot.GameHall.ExchangeAllowDGToBalance)

	public := Public(snapshot)
	require.Equal(t, false, public["checkin_enabled"])
	require.Equal(t, true, public["leaderboard_enabled"])
	require.Equal(t, false, public["transfer_enabled"])
	require.Equal(t, false, public["game_hall_enabled"])
	require.Equal(t, "default", public["default_homepage"])
	require.Equal(t, 0.01, public["game_exchange_min_amount"])
}

func TestRegistryWriteValidatesBeforeOneAggregatePersistence(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	registry := NewRegistry(store)
	snapshot, err := registry.Read(context.Background())
	require.NoError(t, err)

	snapshot.Activity.CheckinEnabled = true
	snapshot.BrandHome.DefaultHomepage = "dino"
	snapshot.WalletExtension.DirectTransferEnabled = true
	snapshot.GameHall.Enabled = true
	require.NoError(t, registry.Write(context.Background(), snapshot))
	require.Equal(t, 1, store.writes)
	require.Equal(t, "true", store.values["checkin_enabled"])
	require.Equal(t, "dino", store.values["default_homepage"])
	require.Equal(t, "true", store.values["transfer_enabled"])
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

func TestMergePublicValuesRejectsOverlappingOwnership(t *testing.T) {
	_, err := mergePublicValues(map[string]any{"shared": true}, map[string]any{"shared": false})
	require.EqualError(t, err, fmt.Sprintf("duplicate custom public setting key %q", "shared"))
}
