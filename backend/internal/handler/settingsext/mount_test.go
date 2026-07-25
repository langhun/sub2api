package settingsext

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergePreservesCoreFieldsAndLetsMountedExtensionsOwnMigratedKeys(t *testing.T) {
	type corePayload struct {
		Core     bool   `json:"core"`
		Migrated string `json:"migrated"`
	}

	merged, err := Merge(corePayload{Core: true, Migrated: "legacy"}, map[string]any{
		"extension": true,
		"migrated":  "overlay",
	})
	require.NoError(t, err)
	require.Equal(t, true, merged["core"])
	require.Equal(t, true, merged["extension"])
	require.Equal(t, "overlay", merged["migrated"])
}
