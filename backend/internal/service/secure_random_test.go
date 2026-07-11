//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecureRandomHelpersStayWithinBounds(t *testing.T) {
	for range 100 {
		value, err := secureRandomIntN(7)
		require.NoError(t, err)
		require.GreaterOrEqual(t, value, 0)
		require.Less(t, value, 7)

		fraction, err := secureRandomFloat64()
		require.NoError(t, err)
		require.GreaterOrEqual(t, fraction, 0.0)
		require.Less(t, fraction, 1.0)
	}
}

func TestSecureRandomIntNRejectsInvalidBound(t *testing.T) {
	_, err := secureRandomIntN(0)
	require.Error(t, err)
}
