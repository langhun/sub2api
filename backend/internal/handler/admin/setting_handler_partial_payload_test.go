//go:build unit

package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

// Saving settings is a whole-document PUT. A client that sends only the field it
// cares about must not reset everything else: a payload as small as
// `{"risk_control_enabled":true}` used to clear site_name, after which
// getStringOrDefault rendered the empty value as the built-in default and the
// login page silently changed name.

func TestUpdateSettingsPartialPayloadKeepsUnsentKeys(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySiteName:         "Example Gateway",
		service.SettingKeySiteSubtitle:     "Example Gateway Platform",
		service.SettingKeySMTPHost:         "smtp.example.com",
		service.SettingKeySMTPFrom:         "noreply@example.com",
		service.SettingKeyTurnstileEnabled: "true",
	})

	rec := doUpdateSettings(t, h, map[string]any{"risk_control_enabled": true}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "true", repo.values[service.SettingKeyRiskControlEnabled],
		"the field the caller actually sent must be written")

	require.Equal(t, "Example Gateway", repo.values[service.SettingKeySiteName])
	require.Equal(t, "Example Gateway Platform", repo.values[service.SettingKeySiteSubtitle])
	require.Equal(t, "smtp.example.com", repo.values[service.SettingKeySMTPHost])
	require.Equal(t, "noreply@example.com", repo.values[service.SettingKeySMTPFrom])
	require.Equal(t, "true", repo.values[service.SettingKeyTurnstileEnabled])
}

// A full payload keeps whole-document semantics: fields explicitly set to their
// zero value are still cleared.
func TestUpdateSettingsFullPayloadStillClearsSentEmptyFields(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySiteName: "Example Gateway",
	})

	rec := doUpdateSettings(t, h, map[string]any{"site_name": ""}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "", repo.values[service.SettingKeySiteName],
		"an explicitly sent empty value is a deliberate clear, not an omission")
}

// smtp_from_email is the one request field whose JSON name differs from its
// setting key; the alias keeps it from being treated as always-omitted.
func TestUpdateSettingsSMTPFromAliasIsWritable(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeySMTPFrom: "old@example.com",
	})

	rec := doUpdateSettings(t, h, map[string]any{"smtp_from_email": "new@example.com"}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "new@example.com", repo.values[service.SettingKeySMTPFrom])
}

func TestUpdateSettingsModelPlazaSwitchesPersist(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyModelPlazaEnabled:     "false",
		service.SettingKeyModelPlazaRequireAuth: "false",
		service.SettingKeyModelPlazaDescription: "old description",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"model_plaza_enabled":      true,
		"model_plaza_require_auth": true,
		"model_plaza_description":  "new description",
	}, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, "true", repo.values[service.SettingKeyModelPlazaEnabled])
	require.Equal(t, "true", repo.values[service.SettingKeyModelPlazaRequireAuth])
	require.Equal(t, "new description", repo.values[service.SettingKeyModelPlazaDescription])
	require.Contains(t, rec.Body.String(), `"model_plaza_enabled":true`)
	require.Contains(t, rec.Body.String(), `"model_plaza_require_auth":true`)
	require.Contains(t, rec.Body.String(), `"model_plaza_description":"new description"`)

	// A later partial save must retain the previously enabled Plaza settings.
	rec = doUpdateSettings(t, h, map[string]any{"registration_enabled": true}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyModelPlazaEnabled])
	require.Equal(t, "true", repo.values[service.SettingKeyModelPlazaRequireAuth])
	require.Equal(t, "new description", repo.values[service.SettingKeyModelPlazaDescription])
}

func TestUpdateSettingsOmittedPasskeyPreservesStoredValueAndEchoesIt(t *testing.T) {
	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyPasskeyEnabled: "true",
	}}
	cfg := &config.Config{
		Default: config.DefaultConfig{UserConcurrency: 5},
		WebAuthn: config.WebAuthnConfig{
			Enabled:   true,
			RPID:      "example.com",
			RPOrigins: []string{"https://example.com"},
		},
	}
	svc := service.NewSettingService(repo, cfg)
	h := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	rec := doUpdateSettings(t, h, map[string]any{"registration_enabled": true}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyPasskeyEnabled])
	require.Contains(t, rec.Body.String(), `"passkey_enabled":true`)
}

func TestUpdateSettingsPasskeyEnableRequiresWebAuthnConfiguration(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyPasskeyEnabled: "false",
	})

	rec := doUpdateSettings(t, h, map[string]any{"passkey_enabled": true}, nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "valid WebAuthn RP ID")
	require.Equal(t, "false", repo.values[service.SettingKeyPasskeyEnabled])
}
