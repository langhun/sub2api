package redpacket

import (
	"context"
	"testing"

	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/stretchr/testify/require"
)

func TestRegistrySettingsAdapterProjectsActivityRedPacketSettings(t *testing.T) {
	registry := customsettings.NewRegistry(redPacketSettingsStore{values: map[string]string{
		"redpacket_enabled":      "true",
		"redpacket_max_count":    "25",
		"redpacket_expire_hours": "48",
	}})

	settings, err := NewRegistrySettingsAdapter(registry).GetActivityRedPacketSettings(context.Background())

	require.NoError(t, err)
	require.Equal(t, true, settings.Enabled)
	require.Equal(t, 25, settings.MaximumCount)
	require.Equal(t, 48, settings.ExpireHours)
}

func TestZeroFeeAdapterKeepsRedPacketsFree(t *testing.T) {
	quote, err := NewZeroFeeAdapter().QuoteRedPacketFee(context.Background(), 7, 10)

	require.NoError(t, err)
	require.Zero(t, quote.Rate)
	require.Zero(t, quote.Amount)
}

func TestSettingsCodeGeneratorUsesOnlyTheCodeGenerationPort(t *testing.T) {
	code, err := NewSettingsCodeGenerator(redPacketCodeSource{code: "RP-123"}).GenerateRedPacketCode(context.Background())

	require.NoError(t, err)
	require.Equal(t, "RP-123", code)
}

func TestSettingsCodeGeneratorRejectsMissingSource(t *testing.T) {
	_, err := NewSettingsCodeGenerator(nil).GenerateRedPacketCode(context.Background())

	require.Error(t, err)
}

type redPacketCodeSource struct{ code string }

func (s redPacketCodeSource) GenerateRedPacketCode(context.Context) (string, error) {
	return s.code, nil
}

type redPacketSettingsStore struct{ values map[string]string }

func (s redPacketSettingsStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (redPacketSettingsStore) SetMultiple(context.Context, map[string]string) error { return nil }
