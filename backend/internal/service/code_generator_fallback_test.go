package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedeemServiceWithoutInjectedGeneratorPreservesLegacyDefaults(t *testing.T) {
	service := NewRedeemService(nil, nil, nil, nil, nil, nil, nil, nil)

	randomCode, err := service.GenerateRandomCode()
	require.NoError(t, err)
	require.Regexp(t, `^[0-9A-F]{8}(?:-[0-9A-F]{8}){3}$`, randomCode)

	repo := &redeemGenerateRepo{}
	service = NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil)
	_, err = service.GenerateCodes(context.Background(), GenerateCodesRequest{Count: 1, Type: RedeemTypeBalance, Value: 1})
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	require.Regexp(t, `^[0-9A-F]{8}(?:-[0-9A-F]{8}){3}$`, repo.created[0].Code)
}

func TestConfigureCodeGeneratorOverridesCoreDefaults(t *testing.T) {
	generator := redeemGenerateCodeGenerator{code: "CUSTOM-00"}
	redeem := NewRedeemService(nil, nil, nil, nil, nil, nil, nil, nil)
	admin := &adminServiceImpl{}

	ConfigureCodeGenerator(redeem, admin, generator)

	randomCode, err := redeem.GenerateRandomCode()
	require.NoError(t, err)
	require.Equal(t, "CUSTOM-00", randomCode)
	require.Equal(t, generator, admin.codeGenerator)
}
