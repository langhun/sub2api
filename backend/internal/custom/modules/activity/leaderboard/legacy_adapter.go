package leaderboard

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// NewLegacyModule creates a runnable activity leaderboard module while the
// core leaderboard read model is still hosted by the legacy service package.
// The adapter is read-only and contains no wallet implementation dependency.
func NewLegacyModule(settingService *service.SettingService, leaderboardService *service.LeaderboardService) *Module {
	return NewModule(NewLegacySettingsReader(settingService), NewLegacyReaders(leaderboardService))
}

// NewLegacySettingsReader adapts the current public setting projection to the
// activity contract. It is intentionally a one-way mapping.
func NewLegacySettingsReader(settingService *service.SettingService) contract.LeaderboardSettingsReader {
	if settingService == nil {
		return nil
	}
	return legacySettingsReader{settings: settingService}
}

type legacySettingsReader struct {
	settings *service.SettingService
}

func (r legacySettingsReader) GetActivityLeaderboardSettings(ctx context.Context) (contract.LeaderboardFeatureSettings, error) {
	settings := r.settings.GetLeaderboardSettings(ctx)
	return toActivityLeaderboardSettings(settings), nil
}

func toActivityLeaderboardSettings(settings service.BalanceFeatureSettings) contract.LeaderboardFeatureSettings {
	return contract.LeaderboardFeatureSettings{
		Enabled:              settings.LeaderboardEnabled,
		BalanceEnabled:       settings.LeaderboardBalanceEnabled,
		ConsumptionEnabled:   settings.LeaderboardConsumptionEnabled,
		CheckinEnabled:       settings.LeaderboardCheckinEnabled,
		TransferEnabled:      settings.TransferEnabled,
		TransferBoardEnabled: settings.LeaderboardTransferEnabled,
		IncludeAdmin:         settings.LeaderboardIncludeAdmin,
	}
}

// NewLegacyReaders exposes the four legacy public leaderboard queries through
// activity's read-side contracts. Callers receive no reader when the legacy
// service is absent, so the HTTP adapter fails closed with ErrUnavailable.
func NewLegacyReaders(leaderboardService *service.LeaderboardService) Readers {
	if leaderboardService == nil {
		return Readers{}
	}
	reader := &legacyReadModel{leaderboards: leaderboardService}
	return Readers{
		Balance:     reader,
		Consumption: reader,
		Checkin:     reader,
		Transfer:    reader,
	}
}

type legacyReadModel struct {
	leaderboards *service.LeaderboardService
}

func (r *legacyReadModel) ListBalanceLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	result, err := r.leaderboards.GetBalanceLeaderboard(ctx, query.Page, query.PageSize)
	return toActivityLeaderboardPage(result), err
}

func (r *legacyReadModel) ListConsumptionLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	result, err := r.leaderboards.GetConsumptionLeaderboard(ctx, string(query.Period), query.Page, query.PageSize)
	return toActivityLeaderboardPage(result), err
}

func (r *legacyReadModel) ListCheckinLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	result, err := r.leaderboards.GetCheckinLeaderboard(ctx, query.Page, query.PageSize)
	return toActivityLeaderboardPage(result), err
}

func (r *legacyReadModel) ListTransferLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	result, err := r.leaderboards.GetTransferLeaderboard(ctx, string(query.Period), query.Page, query.PageSize)
	return toActivityLeaderboardPage(result), err
}

func toActivityLeaderboardPage(result *service.LeaderboardResult) contract.LeaderboardPage {
	if result == nil {
		return contract.LeaderboardPage{}
	}
	entries := make([]contract.LeaderboardEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		entries = append(entries, contract.LeaderboardEntry{
			Rank:       entry.Rank,
			Username:   entry.Username,
			Value:      entry.Value,
			ExtraInt:   entry.ExtraInt,
			ExtraInt2:  entry.ExtraInt2,
			ExtraFloat: entry.ExtraFloat,
			ExtraDate:  entry.ExtraDate,
		})
	}
	return contract.LeaderboardPage{Entries: entries, Total: result.Total}
}
