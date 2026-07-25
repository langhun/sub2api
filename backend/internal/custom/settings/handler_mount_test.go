package settings

import (
	"context"
	"encoding/json"
	"testing"

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

	payload := json.RawMessage(`{"default_homepage":"dino","transfer_enabled":true}`)
	changed, err := mount.ValidateUpdate(context.Background(), payload)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mount.ApplyUpdate(context.Background(), payload))
	require.Equal(t, "dino", store.values["default_homepage"])
	require.Equal(t, "true", store.values["transfer_enabled"])
}
