package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeysIncludesEveryRuntimeSetting(t *testing.T) {
	keys := Keys()
	require.ElementsMatch(t, []string{
		KeyAliasEnabled,
		KeyFingerprintEnabled,
		KeyClusterEnabled,
		KeyClusterWindow6h,
		KeyClusterWindow24h,
		KeyClusterMedium,
		KeyClusterHigh,
		KeyAutoHighLock,
		KeyTrialEnabled,
		KeyTrialHours,
		KeyTrialCostLimit,
		KeyTrialRPM,
		KeyTrialConcurrency,
		KeyPaymentUnlock,
		KeyPaymentAmount,
	}, keys)
}

func TestFromValuesPreservesTrialAndRiskSwitches(t *testing.T) {
	config := FromValues(map[string]string{
		KeyClusterEnabled: "false",
		KeyAutoHighLock:   "false",
		KeyTrialEnabled:   "false",
		KeyPaymentUnlock:  "false",
	})

	require.False(t, config.ClusterEnabled)
	require.False(t, config.AutoHighLock)
	require.False(t, config.TrialEnabled)
	require.False(t, config.PaymentUnlock)
}
