package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingHandler_UpdateSettingsWritesOverlaySettingsThroughRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{}}
	settingsService := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(settingsService, nil, nil, nil, nil, nil, nil)
	handler.SetSettingsMount(customsettings.NewHandlerMount(customsettings.ProvideRegistry(settingsService)))

	body, err := json.Marshal(map[string]any{
		"checkin_enabled":   true,
		"transfer_enabled":  true,
		"game_hall_enabled": true,
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

	handler.UpdateSettings(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "true", repo.values["checkin_enabled"])
	require.Equal(t, "true", repo.values["transfer_enabled"])
	require.Equal(t, "true", repo.values["game_hall_enabled"])
}
