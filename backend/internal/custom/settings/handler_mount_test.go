package settings

import (
	"context"
	"encoding/json"
	"testing"

	codeformatsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format/settings"
	"github.com/stretchr/testify/require"
)

func TestHandlerMountProjectsAndAppliesCustomSettingsWithoutCoreFieldKnowledge(t *testing.T) {
	store := &memoryStore{values: map[string]string{
		"checkin_enabled": "true",
	}}
	mount := NewHandlerMount(NewRegistry(store))

	admin, err := mount.Admin(context.Background())
	require.NoError(t, err)
	require.Equal(t, true, admin["checkin_enabled"])

	public, err := mount.Public(context.Background())
	require.NoError(t, err)
	require.Equal(t, true, public["checkin_enabled"])
	require.Equal(t, "default", public["default_homepage"])
	require.NotContains(t, public, "checkin_min_balance")

	formats := codeformatsettings.Default()
	formats.RedPacket.Prefix = "RP"
	payload, err := json.Marshal(map[string]any{
		"default_homepage":     "dino",
		"code_format_settings": formats,
	})
	require.NoError(t, err)
	changed, err := mount.ValidateUpdate(context.Background(), payload)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mount.ApplyUpdate(context.Background(), payload))
	require.Equal(t, "dino", store.values["default_homepage"])
	require.NotContains(t, store.values, "transfer_enabled")
	require.Equal(t, formats, codeformatsettings.FromValues(store.values))
	admin, err = mount.Admin(context.Background())
	require.NoError(t, err)
	require.Equal(t, formats, decodeCodeFormatSettings(t, admin["code_format_settings"]))
}

func TestHandlerMountRejectsInvalidCodeFormatSettings(t *testing.T) {
	mount := NewHandlerMount(NewRegistry(&memoryStore{values: map[string]string{}}))
	changed, err := mount.ValidateUpdate(context.Background(), json.RawMessage(`{"code_format_settings":{}}`))
	require.True(t, changed)
	require.Error(t, err)
}

func decodeCodeFormatSettings(t *testing.T, value any) codeformatsettings.Config {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	var settings codeformatsettings.Config
	require.NoError(t, json.Unmarshal(encoded, &settings))
	return settings
}
