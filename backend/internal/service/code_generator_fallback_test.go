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
	require.Regexp(t, `^[A-HJ-NP-Z2-9]{8}(?:-[A-HJ-NP-Z2-9]{8}){3}$`, randomCode)

	repo := &redeemGenerateRepo{}
	service = NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil)
	_, err = service.GenerateCodes(context.Background(), GenerateCodesRequest{Count: 1, Type: RedeemTypeBalance, Value: 1})
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	require.Regexp(t, `^[A-HJ-NP-Z2-9]{32}$`, repo.created[0].Code)
}
