package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCodeFormatSettingsUsesValidOverrides(t *testing.T) {
	settings := parseCodeFormatSettings(map[string]string{
		SettingKeyCodeFormatBalance: `{"prefix":"BAL","character_set":"numeric","separator":"-","group_length":3,"group_count":2}`,
	})

	require.Equal(t, "BAL", settings.Balance.Prefix)
	require.Equal(t, CodeCharacterSetNumeric, settings.Balance.CharacterSet)
	require.Equal(t, DefaultCompactRedeemCodeFormat(), settings.Invitation)
}

func TestParseCodeFormatSettingsFallsBackForInvalidValues(t *testing.T) {
	settings := parseCodeFormatSettings(map[string]string{
		SettingKeyCodeFormatBalance:    `{not-json}`,
		SettingKeyCodeFormatInvitation: `{"character_set":"hex","group_length":0,"group_count":1}`,
	})

	require.Equal(t, DefaultCompactRedeemCodeFormat(), settings.Balance)
	require.Equal(t, DefaultCompactRedeemCodeFormat(), settings.Invitation)
}

func TestParseCodeFormatSettingsMigratesLegacyHex(t *testing.T) {
	settings := parseCodeFormatSettings(map[string]string{
		SettingKeyCodeFormatBalance: `{"prefix":"BAL","character_set":"hex","separator":"","group_length":32,"group_count":1}`,
	})

	require.Equal(t, CodeCharacterSetAlphanumeric, settings.Balance.CharacterSet)
	require.Equal(t, "BAL", settings.Balance.Prefix)
}

func TestAppendCodeFormatUpdatesRoundTrips(t *testing.T) {
	want := DefaultCodeFormatSettings()
	want.RedPacket.Prefix = "RP"
	updates := map[string]string{}

	require.NoError(t, appendCodeFormatUpdates(updates, want))
	require.Equal(t, want, parseCodeFormatSettings(updates))
}

func TestAppendCodeFormatUpdatesRejectsInvalidFormat(t *testing.T) {
	settings := DefaultCodeFormatSettings()
	settings.Subscription.GroupCount = 0

	require.Error(t, appendCodeFormatUpdates(map[string]string{}, settings))
}

func TestAppendCodeFormatUpdatesTreatsOmittedSettingsAsDefaults(t *testing.T) {
	updates := map[string]string{}
	require.NoError(t, appendCodeFormatUpdates(updates, CodeFormatSettings{}))
	require.Equal(t, DefaultCodeFormatSettings(), parseCodeFormatSettings(updates))
}

func TestCodeFormatSettingsMapsInternalAdjustmentTypes(t *testing.T) {
	settings := DefaultCodeFormatSettings()
	settings.Balance.Prefix = "BAL"
	settings.Concurrency.Prefix = "CON"
	settings.Subscription.Prefix = "SUB"
	settings.Invitation.Prefix = "INV"

	require.Equal(t, "BAL", settings.RedeemFormat(AdjustmentTypeAdminBalance).Prefix)
	require.Equal(t, "CON", settings.RedeemFormat(AdjustmentTypeAdminConcurrency).Prefix)
	require.Equal(t, "BAL", settings.RedeemFormat(AdjustmentTypeCheckinLuck).Prefix)
	require.Equal(t, "INV", settings.RedeemFormat(RedeemTypeInvitation).Prefix)
}
