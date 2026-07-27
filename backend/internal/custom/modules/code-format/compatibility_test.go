package codeformat

import (
	"context"
	"regexp"
	"testing"

	codeformatsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format/settings"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/stretchr/testify/require"
)

func TestGeneratorPreservesLegacyDefaultRandomRedeemShape(t *testing.T) {
	code, err := (*Generator)(nil).GenerateDefaultRedeemCode(context.Background())

	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^[A-HJ-NP-Z2-9]{8}(?:-[A-HJ-NP-Z2-9]{8}){3}$`), code)
}

func TestGeneratorPreservesFormatMappingAndLegacyHex(t *testing.T) {
	store := &generatorStore{values: map[string]string{
		codeformatsettings.KeyBalance:     `{"prefix":"BAL","character_set":"numeric","separator":"","group_length":2,"group_count":1}`,
		codeformatsettings.KeyConcurrency: `{"prefix":"CON","character_set":"numeric","separator":"","group_length":2,"group_count":1}`,
		codeformatsettings.KeyInvitation:  `{"prefix":"INV","character_set":"hex","separator":"","group_length":2,"group_count":1}`,
	}}
	generator := NewGenerator(customsettings.NewRegistry(store))

	for _, codeType := range []string{"balance", "admin_balance", "checkin", "checkin_luck"} {
		code, err := generator.GenerateCode(context.Background(), codeType)
		require.NoError(t, err, codeType)
		require.Regexp(t, `^BAL[0-9]{2}$`, code, codeType)
	}
	for _, codeType := range []string{"concurrency", "admin_concurrency"} {
		code, err := generator.GenerateCode(context.Background(), codeType)
		require.NoError(t, err, codeType)
		require.Regexp(t, `^CON[0-9]{2}$`, code, codeType)
	}

	invitation, err := generator.GenerateCode(context.Background(), "invitation")
	require.NoError(t, err)
	require.Regexp(t, `^INV[A-HJ-NP-Z2-9]{2}$`, invitation)
}

func TestGeneratorUsesDefaultsForEmptyPersistedConfiguration(t *testing.T) {
	generator := NewGenerator(customsettings.NewRegistry(&generatorStore{values: map[string]string{}}))

	code, err := generator.GenerateRedPacketCode(context.Background())

	require.NoError(t, err)
	require.Regexp(t, `^[A-HJ-NP-Z2-9]{24}$`, code)
}
