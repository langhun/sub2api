package settings

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromValuesUsesDefaultAndParsesLimit(t *testing.T) {
	require.Equal(t, 500.0, FromValues(map[string]string{}).QuickAmountMax)
	require.Equal(t, 250.5, FromValues(map[string]string{KeyQuickAmountMax: "250.5"}).QuickAmountMax)
}

func TestValuesAllowsZeroAsUnlimited(t *testing.T) {
	values, err := Values(Config{QuickAmountMax: 0})
	require.NoError(t, err)
	require.Equal(t, "0.00000000", values[KeyQuickAmountMax])
}

func TestValidateRejectsNegativeAndNonFiniteValues(t *testing.T) {
	for _, value := range []float64{-1, math.Inf(1), math.NaN()} {
		require.Error(t, Validate(Config{QuickAmountMax: value}))
	}
}
