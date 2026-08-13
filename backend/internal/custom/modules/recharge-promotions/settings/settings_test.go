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

func TestTieredPercentageSettingsRoundTrip(t *testing.T) {
	rules := []TieredPercentageRule{
		{MinimumAmount: 50, MaximumAmount: 100, PercentageRate: 8.5},
		{MinimumAmount: 100, MaximumAmount: 0, PercentageRate: 10},
		{MinimumAmount: 10, MaximumAmount: 50, PercentageRate: 5},
	}
	values, err := Values(Config{Enabled: true, Mode: ModeTieredPercentage, MaximumBonus: 20, TieredPercentageRules: rules})
	require.NoError(t, err)
	require.Contains(t, Keys(), KeyTieredPercentageRules)

	decoded := FromValues(values)
	require.Equal(t, ModeTieredPercentage, decoded.Mode)
	require.Equal(t, []TieredPercentageRule{
		{MinimumAmount: 10, MaximumAmount: 50, PercentageRate: 5},
		{MinimumAmount: 50, MaximumAmount: 100, PercentageRate: 8.5},
		{MinimumAmount: 100, MaximumAmount: 0, PercentageRate: 10},
	}, decoded.TieredPercentageRules)
	require.Equal(t, decoded.TieredPercentageRules, Public(decoded)[KeyTieredPercentageRules])
}

func TestValidateTieredPercentageRules(t *testing.T) {
	valid := Config{Enabled: true, Mode: ModeTieredPercentage, TieredPercentageRules: []TieredPercentageRule{
		{MinimumAmount: 10, MaximumAmount: 50, PercentageRate: 5},
		{MinimumAmount: 50, MaximumAmount: 100, PercentageRate: 8},
		{MinimumAmount: 100, MaximumAmount: 0, PercentageRate: 10},
	}}
	require.NoError(t, Validate(valid))

	tests := []struct {
		name  string
		rules []TieredPercentageRule
		want  string
	}{
		{name: "empty", rules: nil, want: "requires at least one rule"},
		{name: "gap", rules: []TieredPercentageRule{{MinimumAmount: 10, MaximumAmount: 49, PercentageRate: 5}, {MinimumAmount: 50, MaximumAmount: 0, PercentageRate: 8}}, want: "continuous"},
		{name: "overlap", rules: []TieredPercentageRule{{MinimumAmount: 10, MaximumAmount: 51, PercentageRate: 5}, {MinimumAmount: 50, MaximumAmount: 0, PercentageRate: 8}}, want: "continuous"},
		{name: "early unbounded", rules: []TieredPercentageRule{{MinimumAmount: 10, MaximumAmount: 0, PercentageRate: 5}, {MinimumAmount: 50, MaximumAmount: 0, PercentageRate: 8}}, want: "greater than minimum"},
		{name: "bounded final", rules: []TieredPercentageRule{{MinimumAmount: 10, MaximumAmount: 50, PercentageRate: 5}}, want: "final maximum_amount must be 0"},
		{name: "decreasing percentage", rules: []TieredPercentageRule{{MinimumAmount: 10, MaximumAmount: 50, PercentageRate: 8}, {MinimumAmount: 50, MaximumAmount: 0, PercentageRate: 5}}, want: "nondecreasing"},
		{name: "invalid precision", rules: []TieredPercentageRule{{MinimumAmount: 10.001, MaximumAmount: 0, PercentageRate: 5}}, want: "two-decimal minimum_amount"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Config{Enabled: true, Mode: ModeTieredPercentage, TieredPercentageRules: test.rules}
			require.ErrorContains(t, Validate(config), test.want)
		})
	}
}
