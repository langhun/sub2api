package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type updateTrackingAdminService struct {
	*stubAdminService
	updateCalled bool
}

func (s *updateTrackingAdminService) UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.updateCalled = true
	return s.stubAdminService.UpdateAccount(ctx, id, input)
}

func newAllowlistAccountTestService() *service.AccountTestService {
	return service.NewAccountTestService(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{
					Enabled:       true,
					UpstreamHosts: []string{"api.openai.com"},
				},
			},
		},
		nil,
		nil,
	)
}

func TestAccountHandlerCreate_InvalidBaseURLRejectedBeforePersist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	handler := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		newAllowlistAccountTestService(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router := gin.New()
	router.POST("/api/v1/admin/accounts", handler.Create)

	body := map[string]any{
		"name":     "relay-openai",
		"platform": "openai",
		"type":     "apikey",
		"credentials": map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://relay.example.com/v1",
		},
		"concurrency": 1,
		"priority":    1,
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, adminSvc.createdAccounts)
	require.Contains(t, rec.Body.String(), "security.url_allowlist.upstream_hosts")
	require.Contains(t, rec.Body.String(), "relay.example.com")
}

func TestAccountHandlerUpdate_InvalidBaseURLRejectedBeforePersist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := &updateTrackingAdminService{stubAdminService: newStubAdminService()}
	handler := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		newAllowlistAccountTestService(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router := gin.New()
	router.PUT("/api/v1/admin/accounts/:id", handler.Update)

	body := map[string]any{
		"credentials": map[string]any{
			"base_url": "https://relay.example.com/v1",
		},
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/3", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.False(t, adminSvc.updateCalled)
	require.Contains(t, rec.Body.String(), "security.url_allowlist.upstream_hosts")
	require.Contains(t, rec.Body.String(), "relay.example.com")
}
