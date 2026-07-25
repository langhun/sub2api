package gamehall

import (
	"context"
	"fmt"

	gamehallsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall/settings"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
)

// RegistrySettingsAdapter keeps game-hall configuration independent from the
// upstream SettingService while using the Overlay registry's typed snapshot.
type RegistrySettingsAdapter struct {
	registry *customsettings.Registry
}

func NewRegistrySettingsAdapter(registry *customsettings.Registry) *RegistrySettingsAdapter {
	return &RegistrySettingsAdapter{registry: registry}
}

func (a *RegistrySettingsAdapter) Read(ctx context.Context) (gamehallsettings.Config, error) {
	if a == nil || a.registry == nil {
		return gamehallsettings.Config{}, fmt.Errorf("custom settings registry is required")
	}
	snapshot, err := a.registry.Read(ctx)
	if err != nil {
		return gamehallsettings.Config{}, err
	}
	return snapshot.GameHall, nil
}
