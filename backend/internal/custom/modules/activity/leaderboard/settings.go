package leaderboard

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

const (
	settingKeyLeaderboardEnabled            = "leaderboard_enabled"
	settingKeyLeaderboardBalanceEnabled     = "leaderboard_balance_enabled"
	settingKeyLeaderboardConsumptionEnabled = "leaderboard_consumption_enabled"
	settingKeyLeaderboardCheckinEnabled     = "leaderboard_checkin_enabled"
	settingKeyLeaderboardIncludeAdmin       = "leaderboard_include_admin"
)

var leaderboardSettingKeys = []string{
	settingKeyLeaderboardEnabled,
	settingKeyLeaderboardBalanceEnabled,
	settingKeyLeaderboardConsumptionEnabled,
	settingKeyLeaderboardCheckinEnabled,
	settingKeyLeaderboardIncludeAdmin,
}

// SettingsStore is the narrow persistence boundary for activity leaderboard
// switches. It deliberately cannot read unrelated application settings.
type SettingsStore interface {
	ReadActivityLeaderboardSettings(ctx context.Context) (map[string]string, error)
}

// SettingsReader maps the module-owned setting slice to its public contract.
// A storage failure preserves the legacy fail-open defaults for these switches.
type SettingsReader struct {
	store SettingsStore
}

func NewSettingsReader(store SettingsStore) *SettingsReader {
	return &SettingsReader{store: store}
}

func (r *SettingsReader) GetActivityLeaderboardSettings(ctx context.Context) (contract.LeaderboardFeatureSettings, error) {
	if r == nil || r.store == nil {
		return defaultLeaderboardFeatureSettings(nil), nil
	}
	values, err := r.store.ReadActivityLeaderboardSettings(ctx)
	if err != nil {
		return defaultLeaderboardFeatureSettings(nil), nil
	}
	return defaultLeaderboardFeatureSettings(values), nil
}

// NewSettingsStore returns the module-owned Ent implementation of the narrow
// activity leaderboard settings contract.
func NewSettingsStore(client *dbent.Client) SettingsStore {
	return &entSettingsStore{client: client}
}

type entSettingsStore struct {
	client *dbent.Client
}

func (s *entSettingsStore) ReadActivityLeaderboardSettings(ctx context.Context) (map[string]string, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("activity leaderboard settings store is unavailable")
	}
	items, err := s.client.Setting.Query().Where(setting.KeyIn(leaderboardSettingKeys...)).All(ctx)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(items))
	for _, item := range items {
		values[item.Key] = item.Value
	}
	return values, nil
}

func defaultLeaderboardFeatureSettings(values map[string]string) contract.LeaderboardFeatureSettings {
	return contract.LeaderboardFeatureSettings{
		Enabled:            values[settingKeyLeaderboardEnabled] != "false",
		BalanceEnabled:     values[settingKeyLeaderboardBalanceEnabled] != "false",
		ConsumptionEnabled: values[settingKeyLeaderboardConsumptionEnabled] != "false",
		CheckinEnabled:     values[settingKeyLeaderboardCheckinEnabled] != "false",
		IncludeAdmin:       values[settingKeyLeaderboardIncludeAdmin] == "true",
	}
}
