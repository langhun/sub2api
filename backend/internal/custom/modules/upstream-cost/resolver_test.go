package upstreamcost

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestResolverUsesFreshPerKeyMultiplier(t *testing.T) {
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	account := probeAccount(map[string]any{
		"status":      "ok",
		"fresh_until": now.Add(time.Hour).Format(time.RFC3339Nano),
		"data": map[string]any{
			"billing_scope":             "token",
			"resolved_rate_multiplier":  0.08,
			"peak_rate_enabled":         false,
			"effective_rate_multiplier": 0.08,
		},
	})

	multiplier, ok := NewResolver().ResolveAccountUsageMultiplier(account, now)
	require.True(t, ok)
	require.Equal(t, 0.08, multiplier)
}

func TestResolverFallsBackWhenSnapshotIsStale(t *testing.T) {
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	account := probeAccount(map[string]any{
		"status":      "ok",
		"fresh_until": now.Add(-time.Second).Format(time.RFC3339Nano),
		"data": map[string]any{
			"billing_scope":            "token",
			"resolved_rate_multiplier": 0.08,
			"peak_rate_enabled":        false,
		},
	})

	_, ok := NewResolver().ResolveAccountUsageMultiplier(account, now)
	require.False(t, ok)
}

func TestResolverAppliesPeakMultiplierAtUsageTime(t *testing.T) {
	now := time.Date(2026, time.July, 31, 10, 0, 0, 0, time.UTC)
	account := probeAccount(map[string]any{
		"status":      "ok",
		"fresh_until": now.Add(time.Hour).Format(time.RFC3339Nano),
		"data": map[string]any{
			"billing_scope":            "token",
			"resolved_rate_multiplier": 0.08,
			"peak_rate_enabled":        true,
			"peak_start":               "09:00",
			"peak_end":                 "18:00",
			"peak_rate_multiplier":     1.5,
			"timezone":                 "UTC",
		},
	})

	multiplier, ok := NewResolver().ResolveAccountUsageMultiplier(account, now)
	require.True(t, ok)
	require.Equal(t, 0.12, multiplier)
}

func probeAccount(snapshot map[string]any) *service.Account {
	return &service.Account{
		Platform: "openai",
		Type:     "apikey",
		Extra: map[string]any{
			probeEnabledExtraKey: true,
			probeExtraKey:        snapshot,
		},
	}
}
