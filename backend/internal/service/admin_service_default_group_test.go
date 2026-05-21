package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCreateAccountDefaultGroupID_PrefersExactMatch(t *testing.T) {
	groupID, ok := resolveCreateAccountDefaultGroupID(PlatformAntigravity, []Group{
		{ID: 11, Name: PlatformAntigravity + "-default-2"},
		{ID: 22, Name: PlatformAntigravity + "-default"},
	})

	require.True(t, ok)
	require.Equal(t, int64(22), groupID)
}

func TestResolveCreateAccountDefaultGroupID_FallsBackToLegacyAntigravityDefault(t *testing.T) {
	groupID, ok := resolveCreateAccountDefaultGroupID(PlatformAntigravity, []Group{
		{ID: 22, Name: PlatformAntigravity + "-default-2"},
		{ID: 11, Name: PlatformAntigravity + "-default-1"},
	})

	require.True(t, ok)
	require.Equal(t, int64(11), groupID)
}
