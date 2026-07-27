package settings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	codeformatsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format/settings"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOverlaySettingsMountWritesOnlyOverlayOwnedValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &overlaySettingRepository{values: map[string]string{}}
	settingService := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := adminhandler.NewSettingHandler(settingService, nil, nil, nil, nil, nil, nil)
	handler.SetSettingsMount(customsettings.NewHandlerMount(customsettings.ProvideRegistry(settingService)))

	formats := codeformatsettings.Default()
	formats.RedPacket.Prefix = "RP"
	body, err := json.Marshal(map[string]any{
		"checkin_enabled":      true,
		"transfer_enabled":     true,
		"game_hall_enabled":    true,
		"code_format_settings": formats,
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	requestContext, _ := gin.CreateTestContext(recorder)
	requestContext.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", nil)
	requestContext.Request.Header.Set("Content-Type", "application/json")
	requestContext.Request.Body = io.NopCloser(bytes.NewReader(body))

	handler.UpdateSettings(requestContext)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "true", repo.values["checkin_enabled"])
	require.Equal(t, "true", repo.values["transfer_enabled"])
	require.Equal(t, "true", repo.values["game_hall_enabled"])
	require.Equal(t, formats, codeformatsettings.FromValues(repo.values))
}

type overlaySettingRepository struct {
	values map[string]string
}

func (r *overlaySettingRepository) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *overlaySettingRepository) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}

func (r *overlaySettingRepository) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (r *overlaySettingRepository) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (r *overlaySettingRepository) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		r.values[key] = value
	}
	return nil
}

func (r *overlaySettingRepository) GetAll(context.Context) (map[string]string, error) {
	values := make(map[string]string, len(r.values))
	for key, value := range r.values {
		values[key] = value
	}
	return values, nil
}

func (r *overlaySettingRepository) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

var _ service.SettingRepository = (*overlaySettingRepository)(nil)
