//go:build unit

package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func validBalanceFeatureSettings() BalanceFeatureSettings {
	return BalanceFeatureSettings{
		GameSlotsMinBet:            0.01,
		GameSlotsMaxBet:            100,
		CheckinMinBalance:          0.1,
		CheckinMaxBalance:          1,
		CheckinLuckMinMultiplier:   0.1,
		CheckinLuckMaxMultiplier:   3,
		TransferFeeRate:            0.01,
		TransferMinAmount:          0.01,
		TransferMaxAmount:          1000,
		TransferDailyLimit:         1000,
		CheckinBlindboxInterval:    7,
		CheckinBlindboxTriggerType: "streak",
		TransferDailyCountLimit:    50,
		RedPacketMaxCount:          100,
		RedPacketExpireHours:       24,
	}
}

func TestBalanceFeatureSettingsValidate(t *testing.T) {
	require.NoError(t, validBalanceFeatureSettings().Validate())

	cases := map[string]func(*BalanceFeatureSettings){
		"reversed checkin range":  func(s *BalanceFeatureSettings) { s.CheckinMaxBalance = s.CheckinMinBalance - 0.01 },
		"invalid luck number":     func(s *BalanceFeatureSettings) { s.CheckinLuckMinMultiplier = math.NaN() },
		"fee above one":           func(s *BalanceFeatureSettings) { s.TransferFeeRate = 1.01 },
		"reversed transfer range": func(s *BalanceFeatureSettings) { s.TransferMaxAmount = s.TransferMinAmount / 2 },
		"negative expiry":         func(s *BalanceFeatureSettings) { s.RedPacketExpireHours = -1 },
		"invalid trigger":         func(s *BalanceFeatureSettings) { s.CheckinBlindboxTriggerType = "random" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			settings := validBalanceFeatureSettings()
			mutate(&settings)
			require.Error(t, settings.Validate())
		})
	}
}
