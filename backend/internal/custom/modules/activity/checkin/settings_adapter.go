package checkin

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
)

// NewRegistrySettingsAdapter projects the activity-owned part of the Overlay
// settings registry into the check-in contract. It keeps the service unaware
// of the registry's other modules and of the core SettingService.
func NewRegistrySettingsAdapter(registry *customsettings.Registry) contract.SettingsReader {
	return registrySettingsAdapter{registry: registry}
}

type registrySettingsAdapter struct{ registry *customsettings.Registry }

func (a registrySettingsAdapter) GetActivitySettings(ctx context.Context) (contract.Settings, error) {
	if a.registry == nil {
		return contract.Settings{}, fmt.Errorf("custom settings registry is required")
	}
	snapshot, err := a.registry.Read(ctx)
	if err != nil {
		return contract.Settings{}, fmt.Errorf("read custom activity settings: %w", err)
	}
	settings := snapshot.Activity
	return contract.Settings{
		Checkin: contract.CheckinSettings{
			Enabled:           settings.CheckinEnabled,
			MinimumReward:     settings.CheckinMinBalance,
			MaximumReward:     settings.CheckinMaxBalance,
			LuckEnabled:       settings.CheckinLuckEnabled,
			MinimumMultiplier: settings.CheckinLuckMinMultiplier,
			MaximumMultiplier: settings.CheckinLuckMaxMultiplier,
		},
		Blindbox: contract.BlindboxSettings{
			Enabled:     settings.CheckinBlindboxEnabled,
			TriggerType: settings.CheckinBlindboxTriggerType,
			Interval:    settings.CheckinBlindboxInterval,
		},
	}, nil
}

// CodeFormatGenerator is the small capability required to create a stable
// ledger code for a check-in adjustment. *service.SettingService satisfies it
// at the composition root without becoming a module dependency.
type CodeFormatGenerator interface {
	GenerateCode(ctx context.Context, codeType string) (string, error)
}

// NewCodeFormatGenerator adapts code-format generation to the check-in ledger
// port. Callers may supply the existing core implementation or a test double.
func NewCodeFormatGenerator(source CodeFormatGenerator) CheckinCodeGenerator {
	return checkinCodeFormatGenerator{source: source}
}

type checkinCodeFormatGenerator struct{ source CodeFormatGenerator }

func (g checkinCodeFormatGenerator) GenerateCheckinCode(ctx context.Context, adjustmentType string) (string, error) {
	if g.source == nil {
		return "", fmt.Errorf("check-in code generator is required")
	}
	return g.source.GenerateCode(ctx, adjustmentType)
}

// BalanceCacheSource is the only existing cache operation check-in needs
// after committing a balance change.
type BalanceCacheSource interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

// NewBalanceCacheInvalidator translates the platform cache operation into the
// Activity contract without exposing a concrete cache service to the module.
func NewBalanceCacheInvalidator(source BalanceCacheSource) contract.BalanceCacheInvalidator {
	return balanceCacheInvalidator{source: source}
}

type balanceCacheInvalidator struct{ source BalanceCacheSource }

func (i balanceCacheInvalidator) InvalidateBalance(ctx context.Context, userID int64) error {
	if i.source == nil {
		return nil
	}
	return i.source.InvalidateUserBalance(ctx, userID)
}

var (
	_ contract.SettingsReader          = registrySettingsAdapter{}
	_ CheckinCodeGenerator             = checkinCodeFormatGenerator{}
	_ contract.BalanceCacheInvalidator = balanceCacheInvalidator{}
)
