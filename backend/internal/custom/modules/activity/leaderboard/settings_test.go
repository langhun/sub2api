package leaderboard

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/stretchr/testify/require"
)

type settingsStoreStub struct {
	values map[string]string
	err    error
	calls  int
}

func (s *settingsStoreStub) ReadActivityLeaderboardSettings(context.Context) (map[string]string, error) {
	s.calls++
	return s.values, s.err
}

func TestSettingsReaderMapsOnlyActivityLeaderboardSwitches(t *testing.T) {
	store := &settingsStoreStub{values: map[string]string{
		settingKeyLeaderboardEnabled:            "true",
		settingKeyLeaderboardBalanceEnabled:     "false",
		settingKeyLeaderboardConsumptionEnabled: "true",
		settingKeyLeaderboardCheckinEnabled:     "false",
		settingKeyLeaderboardTransferEnabled:    "true",
		settingKeyLeaderboardIncludeAdmin:       "true",
		settingKeyTransferEnabled:               "true",
	}}

	settings, err := NewSettingsReader(store).GetActivityLeaderboardSettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, contract.LeaderboardFeatureSettings{
		Enabled:              true,
		BalanceEnabled:       false,
		ConsumptionEnabled:   true,
		CheckinEnabled:       false,
		TransferEnabled:      true,
		TransferBoardEnabled: true,
		IncludeAdmin:         true,
	}, settings)
	require.Equal(t, 1, store.calls)
}

func TestSettingsReaderPreservesLegacyDefaultsWhenStorageFails(t *testing.T) {
	settings, err := NewSettingsReader(&settingsStoreStub{err: errors.New("database unavailable")}).GetActivityLeaderboardSettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, contract.LeaderboardFeatureSettings{
		Enabled:            true,
		BalanceEnabled:     true,
		ConsumptionEnabled: true,
		CheckinEnabled:     true,
	}, settings)
}

func TestSettingsReaderKeepsTransferLeaderboardDisabledByDefault(t *testing.T) {
	settings, err := NewSettingsReader(&settingsStoreStub{}).GetActivityLeaderboardSettings(context.Background())

	require.NoError(t, err)
	require.False(t, settings.TransferEnabled)
	require.False(t, settings.TransferBoardEnabled)
}
