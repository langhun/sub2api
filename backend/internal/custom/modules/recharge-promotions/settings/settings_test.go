package settings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTieredSettingsRoundTripAndSort(t *testing.T) {
	config := Config{Enabled: true, Mode: ModeTiered, TierRules: []TierRule{{MinimumAmount: 50, BonusCredit: 8}, {MinimumAmount: 10, BonusCredit: 1}}}
	values, err := Values(config)
	require.NoError(t, err)
	decoded := FromValues(values)
	require.Equal(t, []TierRule{{MinimumAmount: 10, BonusCredit: 1}, {MinimumAmount: 50, BonusCredit: 8}}, decoded.TierRules)
}

func TestValidateRejectsInvalidEnabledCampaign(t *testing.T) {
	require.ErrorContains(t, Validate(Config{Enabled: true, Mode: ModePercentage}), KeyPercentageRate)
	require.ErrorContains(t, Validate(Config{Enabled: true, Mode: ModeTiered}), KeyTierRules)
}

func TestValidateRejectsDuplicateTierThresholds(t *testing.T) {
	require.ErrorContains(t, Validate(Config{Mode: ModeTiered, TierRules: []TierRule{{MinimumAmount: 10, BonusCredit: 1}, {MinimumAmount: 10, BonusCredit: 2}}}), "strictly increasing")
}
