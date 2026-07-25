// Package settings composes all Overlay-owned configuration at one fixed
// composition-root entry point.
package settings

import (
	"context"
	"fmt"

	activitysettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/settings"
	brandhomesettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/brand-home/settings"
	gamehallsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall/settings"
	walletsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/settings"
	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
	"github.com/Wei-Shaw/sub2api/internal/handler/settingsext"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

// ProviderSet lets the composition root wire this registry as one Overlay
// dependency without exporting its module internals to core settings packages.
var ProviderSet = wire.NewSet(ProvideRegistry, ProvideHandlerSettingsMount)

// Snapshot is the complete typed configuration owned by Overlay modules.
type Snapshot struct {
	Activity        activitysettings.Config
	BrandHome       brandhomesettings.Config
	WalletExtension walletsettings.Config
	GameHall        gamehallsettings.Config
}

// Registry is the sole aggregate for Overlay settings. Core settings code can
// consume this as a compatibility projection without owning module fields.
type Registry struct {
	store contract.Store
}

// ProvideRegistry is the Wire-friendly fixed entry point for custom.Runtime.
func ProvideRegistry(settingService *service.SettingService) *Registry {
	return NewRegistry(serviceStore{settings: settingService})
}

func ProvideHandlerSettingsMount(registry *Registry) settingsext.Mount {
	return NewHandlerMount(registry)
}

func NewRegistry(store contract.Store) *Registry {
	return &Registry{store: store}
}

func (r *Registry) Read(ctx context.Context) (Snapshot, error) {
	if r == nil || r.store == nil {
		return Snapshot{}, fmt.Errorf("custom settings store is required")
	}
	activityConfig, err := activitysettings.New(r.store).Read(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	brandHomeConfig, err := brandhomesettings.New(r.store).Read(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	walletConfig, err := walletsettings.New(r.store).Read(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	gameHallConfig, err := gamehallsettings.New(r.store).Read(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Activity: activityConfig, BrandHome: brandHomeConfig, WalletExtension: walletConfig, GameHall: gameHallConfig}, nil
}

func (r *Registry) Validate(snapshot Snapshot) error {
	if err := activitysettings.Validate(snapshot.Activity); err != nil {
		return err
	}
	if err := brandhomesettings.Validate(snapshot.BrandHome); err != nil {
		return err
	}
	if err := walletsettings.Validate(snapshot.WalletExtension); err != nil {
		return err
	}
	if err := gamehallsettings.Validate(snapshot.GameHall); err != nil {
		return err
	}
	return nil
}

// Write validates all module values before performing one batch persistence
// operation. Individual modules own their encoding; this registry owns the
// aggregate write boundary.
func (r *Registry) Write(ctx context.Context, snapshot Snapshot) error {
	if r == nil || r.store == nil {
		return fmt.Errorf("custom settings store is required")
	}
	if err := r.Validate(snapshot); err != nil {
		return err
	}
	values, err := Values(snapshot)
	if err != nil {
		return err
	}
	return r.store.SetMultiple(ctx, values)
}

func (r *Registry) Public(ctx context.Context) (map[string]any, error) {
	snapshot, err := r.Read(ctx)
	if err != nil {
		return nil, err
	}
	return Public(snapshot), nil
}

// UsageQueryEnabled exposes the narrow activity setting needed by the generic
// API gateway without reintroducing a dependency on legacy balance settings.
func (r *Registry) UsageQueryEnabled(ctx context.Context) (bool, error) {
	snapshot, err := r.Read(ctx)
	if err != nil {
		return false, err
	}
	return snapshot.Activity.UsageQueryEnabled, nil
}

func Values(snapshot Snapshot) (map[string]string, error) {
	activityValues, err := activitysettings.Values(snapshot.Activity)
	if err != nil {
		return nil, err
	}
	brandHomeValues, err := brandhomesettings.Values(snapshot.BrandHome)
	if err != nil {
		return nil, err
	}
	walletValues, err := walletsettings.Values(snapshot.WalletExtension)
	if err != nil {
		return nil, err
	}
	gameHallValues, err := gamehallsettings.Values(snapshot.GameHall)
	if err != nil {
		return nil, err
	}
	return mergeStringValues(activityValues, brandHomeValues, walletValues, gameHallValues)
}

func Public(snapshot Snapshot) map[string]any {
	values, err := mergePublicValues(
		activitysettings.Public(snapshot.Activity),
		brandhomesettings.Public(snapshot.BrandHome),
		walletsettings.Public(snapshot.WalletExtension),
		gamehallsettings.Public(snapshot.GameHall),
	)
	if err != nil {
		panic(err)
	}
	return values
}

func mergeStringValues(groups ...map[string]string) (map[string]string, error) {
	merged := make(map[string]string)
	for _, group := range groups {
		for key, value := range group {
			if _, exists := merged[key]; exists {
				return nil, fmt.Errorf("duplicate custom setting key %q", key)
			}
			merged[key] = value
		}
	}
	return merged, nil
}

func mergePublicValues(groups ...map[string]any) (map[string]any, error) {
	merged := make(map[string]any)
	for _, group := range groups {
		for key, value := range group {
			if _, exists := merged[key]; exists {
				return nil, fmt.Errorf("duplicate custom public setting key %q", key)
			}
			merged[key] = value
		}
	}
	return merged, nil
}

type serviceStore struct{ settings *service.SettingService }

func (s serviceStore) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if s.settings == nil {
		return nil, fmt.Errorf("setting service is required")
	}
	return s.settings.GetMultiple(ctx, keys)
}

func (s serviceStore) SetMultiple(ctx context.Context, values map[string]string) error {
	if s.settings == nil {
		return fmt.Errorf("setting service is required")
	}
	return s.settings.SetMultiple(ctx, values)
}
