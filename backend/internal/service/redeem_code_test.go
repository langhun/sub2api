//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistrationInvitationCodeFormat(t *testing.T) {
	t.Parallel()

	validCases := []string{
		"ABCD-EFGH-IJKL-MNOP",
		"0000-1111-2222-3333",
		"abcd-zz99-yy88-xx77",
		" ABCD-EFGH-IJKL-MNOP ",
	}
	for _, code := range validCases {
		require.True(t, IsRegistrationInvitationCodeFormat(code), code)
	}

	invalidCases := []string{
		"",
		"ABCD-EFGH-IJKL",
		"ABCD-EFGH-IJKL-MNOP-QRST",
		"ABCD_EFGH_IJKL_MNOP",
		"AB-EFGH-IJKL-MNOP",
		"ABCD-EFGH-IJKL-MNO-",
		"ABCD-EFGH-IJKL MNO",
		"ABCD-EFGH-IJKL-MNO!",
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
