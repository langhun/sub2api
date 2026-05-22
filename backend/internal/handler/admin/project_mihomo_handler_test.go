package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type projectMihomoSettingRepoStub struct {
	values map[string]string
}

func (s *projectMihomoSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (s *projectMihomoSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s.values == nil {
		return "", service.ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *projectMihomoSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *projectMihomoSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *projectMihomoSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *projectMihomoSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *projectMihomoSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type projectMihomoAdminServiceStub struct {
	proxies []service.Proxy
	created []service.CreateProxyInput
	updated []struct {
		id    int64
		input service.UpdateProxyInput
	}
}

func (s *projectMihomoAdminServiceStub) ListProxies(context.Context, int, int, string, string, string, string, string) ([]service.Proxy, int64, error) {
	return append([]service.Proxy(nil), s.proxies...), int64(len(s.proxies)), nil
}

func (s *projectMihomoAdminServiceStub) CreateProxy(_ context.Context, input *service.CreateProxyInput) (*service.Proxy, error) {
	s.created = append(s.created, *input)
	return &service.Proxy{ID: int64(len(s.created)), Name: input.Name, Protocol: input.Protocol, Host: input.Host, Port: input.Port, Status: service.StatusActive}, nil
}

func (s *projectMihomoAdminServiceStub) UpdateProxy(_ context.Context, id int64, input *service.UpdateProxyInput) (*service.Proxy, error) {
	s.updated = append(s.updated, struct {
		id    int64
		input service.UpdateProxyInput
	}{id: id, input: *input})
	return &service.Proxy{ID: id, Name: input.Name, Protocol: input.Protocol, Host: input.Host, Port: input.Port, Status: input.Status}, nil
}

type projectMihomoSourceRepoStub struct {
	enabled []service.ProxySubscriptionSource
}

func (s *projectMihomoSourceRepoStub) Create(context.Context, *service.ProxySubscriptionSource) error {
	return nil
}

func (s *projectMihomoSourceRepoStub) GetByID(context.Context, int64) (*service.ProxySubscriptionSource, error) {
	return nil, service.ErrProxyNotFound
}

func (s *projectMihomoSourceRepoStub) List(context.Context, int, int, string, *bool) ([]service.ProxySubscriptionSource, int64, error) {
	out := append([]service.ProxySubscriptionSource(nil), s.enabled...)
	return out, int64(len(out)), nil
}

func (s *projectMihomoSourceRepoStub) ListEnabled(context.Context) ([]service.ProxySubscriptionSource, error) {
	return append([]service.ProxySubscriptionSource(nil), s.enabled...), nil
}

func (s *projectMihomoSourceRepoStub) ListDueForRefresh(context.Context, time.Time, int) ([]service.ProxySubscriptionSource, error) {
	return append([]service.ProxySubscriptionSource(nil), s.enabled...), nil
}

func (s *projectMihomoSourceRepoStub) Update(context.Context, *service.ProxySubscriptionSource) error {
	return nil
}

func (s *projectMihomoSourceRepoStub) Delete(context.Context, int64) error {
	return nil
}

func setupMihomoRouter(projectSvc *service.MihomoService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewProxyHandler(newStubAdminService(), projectSvc)
	router.GET("/api/v1/admin/proxies/mihomo", handler.GetMihomo)
	router.PUT("/api/v1/admin/proxies/mihomo", handler.UpdateMihomo)
	router.POST("/api/v1/admin/proxies/mihomo/sync", handler.SyncMihomo)
	return router
}

func TestMihomoHandlerReturnsNotFoundWhenServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewProxyHandler(newStubAdminService())
	router.GET("/api/v1/admin/proxies/mihomo", handler.GetMihomo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/mihomo", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMihomoHandlerRejectsInvalidJSON(t *testing.T) {
	router := setupMihomoRouter(service.NewMihomoService(&projectMihomoSettingRepoStub{}, &projectMihomoSourceRepoStub{}, &projectMihomoAdminServiceStub{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/mihomo", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMihomoHandlerSyncRejectsInvalidSubscriptionURL(t *testing.T) {
	router := setupMihomoRouter(service.NewMihomoService(
		&projectMihomoSettingRepoStub{},
		&projectMihomoSourceRepoStub{
			enabled: []service.ProxySubscriptionSource{{ID: 1, Name: "bad-source", URL: "file:///tmp/sub.yaml", SourceFormat: service.ProxySubscriptionSourceFormatAuto, Enabled: true}},
		},
		&projectMihomoAdminServiceStub{},
	))

	body, err := json.Marshal(map[string]any{
		"controller_url": "http://127.0.0.1:9097",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/mihomo/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "PROJECT_MIHOMO_SUBSCRIPTION_INVALID", errors.Reason(parseHandlerResponseReason(t, rec.Body.Bytes())))
}

func TestMihomoHandlerUpdateAndSyncSuccess(t *testing.T) {
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/configs", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer controller.Close()
	subscription := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n"))
	}))
	defer subscription.Close()

	adminStub := &projectMihomoAdminServiceStub{}
	repo := &projectMihomoSettingRepoStub{}
	router := setupMihomoRouter(service.NewMihomoService(
		repo,
		&projectMihomoSourceRepoStub{
			enabled: []service.ProxySubscriptionSource{{ID: 1, Name: "source-a", URL: subscription.URL, SourceFormat: service.ProxySubscriptionSourceFormatClashYAML, Enabled: true}},
		},
		adminStub,
	))

	body, err := json.Marshal(map[string]any{
		"controller_url": controller.URL,
		"auto_optimize":  true,
		"country_filter": "日本",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/mihomo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/mihomo", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"auto_optimize":true`)
	require.Contains(t, rec.Body.String(), `"country_filter":"日本"`)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/mihomo/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminStub.created, 4)
}

func parseHandlerResponseReason(t *testing.T, body []byte) error {
	t.Helper()
	var payload struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	return errors.New(http.StatusBadRequest, payload.Reason, "")
}
