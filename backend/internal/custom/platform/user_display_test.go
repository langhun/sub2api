package platform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserDisplayName(t *testing.T) {
	tests := []struct {
		username string
		email    string
		userID   int64
		want     string
	}{
		{username: " Alice ", email: "alice@example.com", userID: 7, want: "Alice"},
		{email: " alice@example.com ", userID: 7, want: "alice@example.com"},
		{userID: 7, want: "用户 #7"},
		{want: "用户"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, UserDisplayName(tt.username, tt.email, tt.userID))
	}
}

func TestMaskedUserDisplayName(t *testing.T) {
	require.Equal(t, "Alice", MaskedUserDisplayName(" Alice ", "alice@example.com", 7))
	require.Equal(t, "p***n@example.com", MaskedUserDisplayName("", " person@example.com ", 7))
	require.Equal(t, "用户 #7", MaskedUserDisplayName("", "", 7))
}
