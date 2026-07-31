package settings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type publicSettingsSourceStub struct {
	value any
	err   error
}

func (s publicSettingsSourceStub) GetPublicSettingsForInjection(context.Context) (any, error) {
	return s.value, s.err
}

func TestInjectionProviderProjectsOverlaySettingsFromRegistry(t *testing.T) {
	registry := NewRegistry(&memoryStore{values: map[string]string{
		"checkin_enabled":   "true",
		"game_hall_enabled": "true",
	}})
	provider := NewInjectionProvider(publicSettingsSourceStub{value: map[string]any{
		"site_name":         "Sub2API",
		"checkin_enabled":   false,
		"game_hall_enabled": false,
	}}, registry)

	result, err := provider.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	settings, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Sub2API", settings["site_name"])
	require.Equal(t, true, settings["checkin_enabled"])
	require.NotContains(t, settings, "transfer_enabled")
	require.Equal(t, true, settings["game_hall_enabled"])
}
