package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromValuesKeepsLegacyDinoSetting(t *testing.T) {
	config := FromValues(map[string]string{KeyDefaultHomepage: "DINO"})
	require.Equal(t, HomepageDino, config.DefaultHomepage)
}

func TestFromValuesKeepsPixelSetting(t *testing.T) {
	config := FromValues(map[string]string{KeyDefaultHomepage: "PIXEL"})
	require.Equal(t, HomepagePixel, config.DefaultHomepage)
}

func TestFromValuesDefaultsInvalidLegacySetting(t *testing.T) {
	config := FromValues(map[string]string{KeyDefaultHomepage: "unknown"})
	require.Equal(t, HomepageDefault, config.DefaultHomepage)
	require.Equal(t, ThemeGlass, config.ThemeStyle)
}

func TestFromValuesNormalizesThemeStyle(t *testing.T) {
	config := FromValues(map[string]string{KeyThemeStyle: " CLASSIC "})
	require.Equal(t, ThemeClassic, config.ThemeStyle)

	config = FromValues(map[string]string{KeyThemeStyle: "unsupported"})
	require.Equal(t, ThemeGlass, config.ThemeStyle)
}

func TestValuesRejectsUnsupportedHomepage(t *testing.T) {
	_, err := Values(Config{DefaultHomepage: "other"})
	require.Error(t, err)
}

func TestValuesRejectsUnsupportedTheme(t *testing.T) {
	_, err := Values(Config{DefaultHomepage: HomepageDefault, ThemeStyle: "other"})
	require.ErrorContains(t, err, "theme_style")
}
