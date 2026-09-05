package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromValuesUsesLegacyThemeStyleWhenNewSwitchIsMissing(t *testing.T) {
	require.False(t, FromValues(map[string]string{KeyLegacyThemeStyle: " CLASSIC "}).Enabled)
	require.True(t, FromValues(map[string]string{KeyLegacyThemeStyle: "glass"}).Enabled)
}

func TestFromValuesPrefersExplicitSwitchAndNormalizesMode(t *testing.T) {
	config := FromValues(map[string]string{
		KeyEnabled:          "false",
		KeyDefaultThemeMode: " DARK ",
		KeyLegacyThemeStyle: LegacyThemeGlass,
	})
	require.False(t, config.Enabled)
	require.Equal(t, ModeDark, config.DefaultThemeMode)
}

func TestFromValuesNormalizesTimeMode(t *testing.T) {
	config := FromValues(map[string]string{KeyDefaultThemeMode: " TIME "})
	require.Equal(t, ModeTime, config.DefaultThemeMode)
}

func TestValuesWritesNewAndLegacyCompatibilityKeys(t *testing.T) {
	values, err := Values(Config{Enabled: true, DefaultThemeMode: ModeLight})
	require.NoError(t, err)
	require.Equal(t, "true", values[KeyEnabled])
	require.Equal(t, ModeLight, values[KeyDefaultThemeMode])
	require.Equal(t, LegacyThemeGlass, values[KeyLegacyThemeStyle])
}

func TestValidateRejectsUnsupportedMode(t *testing.T) {
	require.ErrorContains(t, Validate(Config{Enabled: true, DefaultThemeMode: "sepia"}), KeyDefaultThemeMode)
}

func TestValidateAcceptsTimeMode(t *testing.T) {
	require.NoError(t, Validate(Config{Enabled: true, DefaultThemeMode: ModeTime}))
}
