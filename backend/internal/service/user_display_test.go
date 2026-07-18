package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		userID   int64
		want     string
	}{
		{name: "username first", username: " Alice ", email: "alice@example.com", userID: 7, want: "Alice"},
		{name: "email fallback", email: " alice@example.com ", userID: 7, want: "alice@example.com"},
		{name: "id fallback", userID: 7, want: "用户 #7"},
		{name: "generic fallback", want: "用户"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, UserDisplayName(tt.username, tt.email, tt.userID))
		})
	}
}

func TestMaskedUserDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		userID   int64
		want     string
	}{
		{name: "username remains complete", username: " Alice ", email: "alice@example.com", userID: 7, want: "Alice"},
		{name: "email fallback is masked", email: " person@example.com ", userID: 7, want: "p***n@example.com"},
		{name: "id fallback", userID: 7, want: "用户 #7"},
		{name: "generic fallback", want: "用户"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, MaskedUserDisplayName(tt.username, tt.email, tt.userID))
		})
	}
}
