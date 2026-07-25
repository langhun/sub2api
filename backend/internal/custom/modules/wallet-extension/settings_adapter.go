package walletextension

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
)

// NewRegistrySettingsAdapter projects only wallet-owned transfer policy and
// the activity-owned transfer leaderboard gate. The module never reads the
// broad core setting view directly.
func NewRegistrySettingsAdapter(registry *customsettings.Registry) contract.SettingsReader {
	return registrySettingsAdapter{registry: registry}
}

type registrySettingsAdapter struct{ registry *customsettings.Registry }

func (a registrySettingsAdapter) GetWalletExtensionSettings(ctx context.Context) (contract.Settings, error) {
	snapshot, err := a.read(ctx)
	if err != nil {
		return contract.Settings{}, err
	}
	settings := snapshot.WalletExtension
	return contract.Settings{DirectTransfer: contract.DirectTransferSettings{
		Enabled:         settings.DirectTransferEnabled,
		FeeRate:         settings.DirectTransferFeeRate,
		MinimumAmount:   settings.DirectTransferMinAmount,
		MaximumAmount:   settings.DirectTransferMaxAmount,
		DailyLimit:      settings.DirectTransferDailyLimit,
		DailyCountLimit: settings.DirectTransferDailyCountLimit,
		VIPFeeExempt:    settings.DirectTransferVIPFeeExempt,
	}}, nil
}

func (a registrySettingsAdapter) GetWalletTransferLeaderboardSettings(ctx context.Context) (TransferLeaderboardSettings, error) {
	snapshot, err := a.read(ctx)
	if err != nil {
		return TransferLeaderboardSettings{}, err
	}
	return TransferLeaderboardSettings{
		Enabled: snapshot.Activity.LeaderboardEnabled && snapshot.Activity.LeaderboardTransferEnabled,
	}, nil
}

func (a registrySettingsAdapter) read(ctx context.Context) (customsettings.Snapshot, error) {
	if a.registry == nil {
		return customsettings.Snapshot{}, fmt.Errorf("custom settings registry is required")
	}
	snapshot, err := a.registry.Read(ctx)
	if err != nil {
		return customsettings.Snapshot{}, fmt.Errorf("read custom wallet settings: %w", err)
	}
	return snapshot, nil
}

var (
	_ contract.SettingsReader           = registrySettingsAdapter{}
	_ transferLeaderboardSettingsReader = registrySettingsAdapter{}
)
