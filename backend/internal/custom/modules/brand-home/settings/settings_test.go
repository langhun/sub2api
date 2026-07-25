package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromValuesKeepsLegacyDinoSetting(t *testing.T) {
	config := FromValues(map[string]string{KeyDefaultHomepage: "DINO"})
	require.Equal(t, HomepageDino, config.DefaultHomepage)
}

func TestFromValuesDefaultsInvalidLegacySetting(t *testing.T) {
	config := FromValues(map[string]string{KeyDefaultHomepage: "unknown"})
	require.Equal(t, HomepageDefault, config.DefaultHomepage)
}

func TestValuesRejectsUnsupportedHomepage(t *testing.T) {
	_, err := Values(Config{DefaultHomepage: "other"})
	require.Error(t, err)
}
