package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestProxyLatencyInfoHashRoundTripPreservesRuntimeFields(t *testing.T) {
	score := 88
	now := time.Now().UTC().Truncate(time.Second)
	info := &service.ProxyLatencyInfo{
		Success:             true,
		Message:             "Proxy is accessible",
		QualityStatus:       "good",
		QualityScore:        &score,
		HealthStatus:        "healthy",
		FailoverSwitchCount: ptrInt64RepositoryTest(3),
		UpdatedAt:           now,
	}

	fields, err := proxyLatencyInfoToHash(info)
	require.NoError(t, err)
	require.Contains(t, fields, "success")
	require.Contains(t, fields, "quality_score")
	require.Contains(t, fields, "failover_switch_count")

	stringFields := make(map[string]string, len(fields))
	for key, value := range fields {
		stringFields[key] = value.(string)
	}
	roundTrip, err := proxyLatencyInfoFromHash(stringFields)
	require.NoError(t, err)
	require.True(t, roundTrip.Success)
	require.Equal(t, "good", roundTrip.QualityStatus)
	require.NotNil(t, roundTrip.QualityScore)
	require.Equal(t, 88, *roundTrip.QualityScore)
	require.NotNil(t, roundTrip.FailoverSwitchCount)
	require.Equal(t, int64(3), *roundTrip.FailoverSwitchCount)
	require.True(t, roundTrip.UpdatedAt.Equal(now))
}

func TestProxyLatencyInfoFromHashIgnoresInvalidFieldValues(t *testing.T) {
	info, err := proxyLatencyInfoFromHash(map[string]string{
		"success":        "false",
		"message":        `"failed"`,
		"quality_score":  "not-json",
		"health_status":  `"cooldown"`,
		"last_fail_at":   "123",
		"last_fail_time": "2026-05-12",
		"updated_at":     `"2026-05-12T00:00:00Z"`,
	})
	require.NoError(t, err)
	require.False(t, info.Success)
	require.Equal(t, "failed", info.Message)
	require.Equal(t, "cooldown", info.HealthStatus)
	require.Nil(t, info.QualityScore)
}

func ptrInt64RepositoryTest(value int64) *int64 {
	return &value
}
