//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistrationInvitationCodeFormat(t *testing.T) {
	t.Parallel()

	validCases := []string{
		"DG-ABC123",
		"DG-000000",
		"dg-zz9999",
		" DG-A1B2C3 ",
	}
	for _, code := range validCases {
		require.True(t, IsRegistrationInvitationCodeFormat(code), code)
	}

	invalidCases := []string{
		"",
		"DG-ABCDE",
		"DG-ABCDEFG",
		"DG_ABC123",
		"AB-ABC123",
		"DG-ABC12-",
		"DG-ABC 23",
		"DG-ABC!23",
	}
	for _, code := range invalidCases {
		require.False(t, IsRegistrationInvitationCodeFormat(code), code)
	}
}

func TestGenerateRegistrationInvitationCode(t *testing.T) {
	t.Parallel()

	code, err := GenerateRegistrationInvitationCode()
	require.NoError(t, err)
	require.True(t, IsRegistrationInvitationCodeFormat(code))
	require.Equal(t, NormalizeRegistrationInvitationCode(code), code)
}
