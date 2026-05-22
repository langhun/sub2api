//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGetStreamTimeoutSettings_InvalidJSONReturnsError(t *testing.T) {
	repo := newMockSettingRepo()
	repo.data[SettingKeyStreamTimeoutSettings] = "not-json"
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetStreamTimeoutSettings(context.Background())
	require.ErrorIs(t, err, ErrStreamTimeoutSettingsCorrupt)
	require.Nil(t, settings)
}

func TestGetRectifierSettings_InvalidJSONReturnsError(t *testing.T) {
	repo := newMockSettingRepo()
	repo.data[SettingKeyRectifierSettings] = "not-json"
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetRectifierSettings(context.Background())
	require.ErrorIs(t, err, ErrRectifierSettingsCorrupt)
	require.Nil(t, settings)
}

func TestGetBetaPolicySettings_InvalidJSONReturnsError(t *testing.T) {
	repo := newMockSettingRepo()
	repo.data[SettingKeyBetaPolicySettings] = "not-json"
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetBetaPolicySettings(context.Background())
	require.ErrorIs(t, err, ErrBetaPolicySettingsCorrupt)
	require.Nil(t, settings)
}
