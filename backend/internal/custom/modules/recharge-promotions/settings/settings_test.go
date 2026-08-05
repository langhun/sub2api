package settings

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValuesRoundTripNormalizesCampaignTimesToUTC(t *testing.T) {
	start := time.Date(2026, time.August, 5, 12, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	end := start.Add(24 * time.Hour)
	config := Config{Enabled: true, BonusRate: 20, MinAmount: 10.5, MaxBonus: 30, StartAt: start, EndAt: end}

	values, err := Values(config)
	require.NoError(t, err)
	require.Equal(t, "2026-08-05T04:30:00Z", values[KeyStartAt])
	require.Equal(t, "2026-08-06T04:30:00Z", values[KeyEndAt])
	require.Equal(t, Config{Enabled: true, BonusRate: 20, MinAmount: 10.5, MaxBonus: 30, StartAt: start.UTC(), EndAt: end.UTC()}, FromValues(values))
}

func TestFromValuesUsesSafeDefaultsForInvalidValues(t *testing.T) {
	config := FromValues(map[string]string{
		KeyEnabled:   "not-a-bool",
		KeyBonusRate: "NaN",
		KeyMinAmount: "-1",
		KeyMaxBonus:  "not-a-number",
		KeyStartAt:   "not-a-time",
	})
	require.Equal(t, Default(), config)
}

func TestValidateCampaignRules(t *testing.T) {
	valid := Config{Enabled: true, BonusRate: 100, MinAmount: 0, MaxBonus: 0}
	require.NoError(t, Validate(valid))

	invalidCases := []struct {
		name   string
		config Config
		match  string
	}{
		{name: "enabled zero rate", config: Config{Enabled: true}, match: KeyBonusRate},
		{name: "rate too high", config: Config{BonusRate: 100.01}, match: KeyBonusRate},
		{name: "rate precision", config: Config{BonusRate: 1.001}, match: KeyBonusRate},
		{name: "negative minimum", config: Config{MinAmount: -0.01}, match: KeyMinAmount},
		{name: "minimum precision", config: Config{MinAmount: 0.001}, match: KeyMinAmount},
		{name: "invalid maximum", config: Config{MaxBonus: math.Inf(1)}, match: KeyMaxBonus},
		{name: "invalid schedule", config: Config{StartAt: time.Unix(2, 0), EndAt: time.Unix(1, 0)}, match: KeyStartAt},
	}
	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.ErrorContains(t, Validate(testCase.config), testCase.match)
		})
	}
}

func TestReaderWritesValidatedValues(t *testing.T) {
	store := &storeStub{values: map[string]string{}}
	reader := New(store)
	config := Config{Enabled: true, BonusRate: 15, MinAmount: 20, MaxBonus: 5}
	require.NoError(t, reader.Write(context.Background(), config))
	require.Equal(t, "true", store.values[KeyEnabled])
	require.Equal(t, "15.00", store.values[KeyBonusRate])

	config.BonusRate = 0
	require.ErrorContains(t, reader.Write(context.Background(), config), KeyBonusRate)
	require.Equal(t, 1, store.writes)
}

func TestPublicIncludesCampaignEstimateSettings(t *testing.T) {
	public := Public(Config{Enabled: true, BonusRate: 10, MinAmount: 20, MaxBonus: 5})
	require.Equal(t, true, public[KeyEnabled])
	require.Equal(t, 10.0, public[KeyBonusRate])
	require.Equal(t, 20.0, public[KeyMinAmount])
	require.Equal(t, 5.0, public[KeyMaxBonus])
	require.Equal(t, "", public[KeyStartAt])
	require.Equal(t, "", public[KeyEndAt])
}

type storeStub struct {
	values map[string]string
	writes int
}

func (s *storeStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *storeStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		s.values[key] = value
	}
	s.writes++
	return nil
}
