package supporttickets

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSettingsDefaultsAndPublicValues(t *testing.T) {
	config := SettingsFromValues(map[string]string{})
	require.True(t, config.Enabled)

	values, err := SettingsValues(Settings{Enabled: false})
	require.NoError(t, err)
	require.Equal(t, "false", values[KeyEnabled])
}
