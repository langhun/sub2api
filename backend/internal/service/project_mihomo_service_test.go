package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestMihomoRenderConfigUsesOptimizeSettings(t *testing.T) {
	svc := NewMihomoService(nil, nil, nil)
	settings := DefaultMihomoSettings()
	settings.ListenerCount = 2
	settings.StartPort = 41001
	settings.ProxyNamePrefix = "proj"
	settings.AutoOptimize = true
	settings.CountryFilter = "美国"
	settings.ListenerRegions = []string{"香港", ""}

	data, err := svc.renderConfig(&settings, []mihomoProvider{{Name: "provider-1", Path: "/tmp/provider-1.yaml"}})
	if err != nil {
		t.Fatalf("renderConfig() error = %v", err)
	}

	var cfg struct {
		ProxyGroups []map[string]any `yaml:"proxy-groups"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal rendered config: %v\n%s", err, string(data))
	}
	if len(cfg.ProxyGroups) != 2 {
		t.Fatalf("expected 2 proxy groups, got %d", len(cfg.ProxyGroups))
	}
	if got := cfg.ProxyGroups[0]["type"]; got != "url-test" {
		t.Fatalf("first group type = %v, want url-test", got)
	}
	if got := cfg.ProxyGroups[0]["filter"]; got != "香港" {
		t.Fatalf("first group filter = %v, want 香港", got)
	}
	if _, ok := cfg.ProxyGroups[1]["filter"]; ok {
		t.Fatalf("second group should not have filter when auto optimize is enabled: %#v", cfg.ProxyGroups[1])
	}
	if got := cfg.ProxyGroups[1]["url"]; got == "" || got == nil {
		t.Fatalf("second group should include health check url: %#v", cfg.ProxyGroups[1])
	}
}

func TestMihomoRenderConfigUsesCountryFilterWithoutAutoOptimize(t *testing.T) {
	svc := NewMihomoService(nil, nil, nil)
	settings := DefaultMihomoSettings()
	settings.ListenerCount = 1
	settings.AutoOptimize = false
	settings.CountryFilter = "日本"
	settings.ListenerRegions = []string{""}

	data, err := svc.renderConfig(&settings, []mihomoProvider{{Name: "provider-1", Path: "/tmp/provider-1.yaml"}})
	if err != nil {
		t.Fatalf("renderConfig() error = %v", err)
	}

	text := string(data)
	if !strings.Contains(text, "type: url-test") {
		t.Fatalf("expected country filter group to use url-test, got:\n%s", text)
	}
	if !strings.Contains(text, "filter: 日本") {
		t.Fatalf("expected rendered config to include country filter, got:\n%s", text)
	}
}

func TestMihomoSetSettingsRejectsInvalidControllerURL(t *testing.T) {
	svc := NewMihomoService(&projectMihomoSettingRepoStub{}, nil, nil)
	settings := DefaultMihomoSettings()
	settings.ControllerURL = "http:///missing-host"

	_, err := svc.SetSettings(context.Background(), &settings)
	if !errors.Is(err, ErrMihomoControllerInvalid) {
		t.Fatalf("SetSettings() error = %v, want %v", err, ErrMihomoControllerInvalid)
	}
}

func TestMihomoResolveProvidersRejectsInvalidSubscriptionURL(t *testing.T) {
	svc := NewMihomoService(
		&projectMihomoSettingRepoStub{},
		&projectMihomoSourceRepoStub{
			enabled: []ProxySubscriptionSource{{ID: 1, Name: "bad-source", URL: "file:///tmp/sub.yaml", SourceFormat: ProxySubscriptionSourceFormatAuto, Enabled: true}},
		},
		nil,
	)

	_, err := svc.resolveProviders(context.Background())
	if !errors.Is(err, ErrMihomoSubscriptionInvalid) {
		t.Fatalf("resolveProviders() error = %v, want %v", err, ErrMihomoSubscriptionInvalid)
	}
}

func TestMihomoGetSettingsReturnsCorruptionError(t *testing.T) {
	svc := NewMihomoService(&projectMihomoSettingRepoStub{
		values: map[string]string{
			SettingKeyMihomoSettings: "{bad json",
		},
	}, nil, nil)

	_, err := svc.GetSettings(context.Background())
	if !errors.Is(err, ErrMihomoSettingsCorrupted) {
		t.Fatalf("GetSettings() error = %v, want %v", err, ErrMihomoSettingsCorrupted)
	}
}

func TestMihomoSyncReturnsReloadFailure(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	subscription := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n"))
	}))
	defer subscription.Close()
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/configs" {
			t.Fatalf("unexpected reload request %s %s", r.Method, r.URL.String())
		}
		http.Error(w, "reload failed", http.StatusBadGateway)
	}))
	defer controller.Close()

	admin := &projectMihomoAdminServiceStub{}
	svc := NewMihomoService(
		&projectMihomoSettingRepoStub{},
		&projectMihomoSourceRepoStub{
			enabled: []ProxySubscriptionSource{{ID: 1, Name: "source-a", URL: subscription.URL, SourceFormat: ProxySubscriptionSourceFormatClashYAML, Enabled: true}},
		},
		admin,
	)
	settings := DefaultMihomoSettings()
	settings.ControllerURL = controller.URL
	settings.ListenerCount = 1
	settings.ListenerRegions = []string{""}

	result, err := svc.Sync(context.Background(), &settings)
	if err == nil {
		t.Fatalf("Sync() error = nil, want reload failure")
	}
	if result != nil {
		t.Fatalf("Sync() result = %#v, want nil on reload failure", result)
	}
	if len(admin.created) != 1 {
		t.Fatalf("created proxies = %d, want 1", len(admin.created))
	}
}

func TestMihomoSyncReactivatesDisabledProxy(t *testing.T) {
	admin := &projectMihomoAdminServiceStub{
		proxies: []Proxy{
			{ID: 7, Name: "mihomo-01", Protocol: "socks5h", Host: "127.0.0.1", Port: 41001, Status: StatusDisabled},
		},
	}
	svc := NewMihomoService(nil, nil, admin)
	settings := DefaultMihomoSettings()
	settings.ListenerCount = 1
	settings.ListenerRegions = []string{""}

	created, reused, err := svc.syncProxyRows(context.Background(), &settings, &settings, svc.buildProxies(&settings))
	if err != nil {
		t.Fatalf("syncProxyRows() error = %v", err)
	}
	if created != 0 || reused != 1 {
		t.Fatalf("syncProxyRows() created=%d reused=%d, want created=0 reused=1", created, reused)
	}
	if len(admin.updated) != 1 {
		t.Fatalf("updated proxies = %d, want 1", len(admin.updated))
	}
	update := admin.updated[0]
	if update.id != 7 || update.input.Status != StatusActive {
		t.Fatalf("update = %#v, want id 7 active status", update)
	}
}

func TestMihomoSyncReusesExistingProxyAcrossPrefixChangeAndDisablesStaleProxy(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	subscription := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n"))
	}))
	defer subscription.Close()
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/configs" {
			t.Fatalf("unexpected reload request %s %s", r.Method, r.URL.String())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer controller.Close()

	previous := DefaultMihomoSettings()
	previous.ProxyNamePrefix = "old-prefix"
	previous.ListenerCount = 2
	previous.ListenerRegions = []string{"", ""}
	previous.ControllerURL = controller.URL
	previousJSON, err := json.Marshal(&previous)
	if err != nil {
		t.Fatalf("marshal previous settings: %v", err)
	}

	admin := &projectMihomoAdminServiceStub{
		proxies: []Proxy{
			{ID: 7, Name: "old-prefix-01", Protocol: "socks5h", Host: previous.TargetHost, Port: previous.StartPort, Status: StatusActive},
			{ID: 8, Name: "old-prefix-02", Protocol: "socks5h", Host: previous.TargetHost, Port: previous.StartPort + 1, Status: StatusActive},
		},
	}
	repo := &projectMihomoSettingRepoStub{
		values: map[string]string{
			SettingKeyMihomoSettings: string(previousJSON),
		},
	}
	svc := NewMihomoService(
		repo,
		&projectMihomoSourceRepoStub{
			enabled: []ProxySubscriptionSource{{ID: 1, Name: "source-a", URL: subscription.URL, SourceFormat: ProxySubscriptionSourceFormatClashYAML, Enabled: true}},
		},
		admin,
	)

	next := previous
	next.ProxyNamePrefix = "new-prefix"
	next.ListenerCount = 1
	next.ListenerRegions = []string{""}

	result, err := svc.Sync(context.Background(), &next)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.Created != 0 || result.Reused != 1 {
		t.Fatalf("Sync() created=%d reused=%d, want created=0 reused=1", result.Created, result.Reused)
	}
	if len(admin.created) != 0 {
		t.Fatalf("created proxies = %d, want 0", len(admin.created))
	}
	if len(admin.updated) != 2 {
		t.Fatalf("updated proxies = %d, want 2", len(admin.updated))
	}

	updatesByID := map[int64]UpdateProxyInput{}
	for _, update := range admin.updated {
		updatesByID[update.id] = update.input
	}
	reused := updatesByID[7]
	if reused.Name != "new-prefix-01" || reused.Port != previous.StartPort || reused.Status != StatusActive {
		t.Fatalf("reused update = %#v, want renamed active proxy on original port", reused)
	}
	disabled := updatesByID[8]
	if disabled.Name != "old-prefix-02" || disabled.Status != StatusDisabled {
		t.Fatalf("disabled update = %#v, want stale proxy disabled", disabled)
	}
}

func TestDefaultMihomoSettingsRespectsDataDir(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	settings := DefaultMihomoSettings()
	if settings.TargetHost != "127.0.0.1" || settings.ControllerURL != "http://127.0.0.1:9097" {
		t.Fatalf("DefaultMihomoSettings() with DATA_DIR outside /app/data = host %q controller %q", settings.TargetHost, settings.ControllerURL)
	}

	t.Setenv("DATA_DIR", "/app/data")
	settings = DefaultMihomoSettings()
	if settings.TargetHost != "mihomo-sub2api" || settings.ControllerURL != "http://mihomo-sub2api:9097" {
		t.Fatalf("DefaultMihomoSettings() with DATA_DIR=/app/data = host %q controller %q", settings.TargetHost, settings.ControllerURL)
	}
}

type projectMihomoSettingRepoStub struct {
	values map[string]string
}

func (s *projectMihomoSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *projectMihomoSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
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
	proxies []Proxy
	created []CreateProxyInput
	updated []struct {
		id    int64
		input UpdateProxyInput
	}
}

func (s *projectMihomoAdminServiceStub) ListProxies(context.Context, int, int, string, string, string, string, string) ([]Proxy, int64, error) {
	return append([]Proxy(nil), s.proxies...), int64(len(s.proxies)), nil
}

func (s *projectMihomoAdminServiceStub) CreateProxy(_ context.Context, input *CreateProxyInput) (*Proxy, error) {
	s.created = append(s.created, *input)
	proxy := &Proxy{ID: int64(len(s.proxies) + len(s.created)), Name: input.Name, Protocol: input.Protocol, Host: input.Host, Port: input.Port, Status: StatusActive}
	return proxy, nil
}

type projectMihomoSourceRepoStub struct {
	enabled []ProxySubscriptionSource
}

func (s *projectMihomoSourceRepoStub) Create(context.Context, *ProxySubscriptionSource) error {
	return nil
}

func (s *projectMihomoSourceRepoStub) GetByID(context.Context, int64) (*ProxySubscriptionSource, error) {
	return nil, ErrProxyNotFound
}

func (s *projectMihomoSourceRepoStub) List(context.Context, int, int, string, *bool) ([]ProxySubscriptionSource, int64, error) {
	out := append([]ProxySubscriptionSource(nil), s.enabled...)
	return out, int64(len(out)), nil
}

func (s *projectMihomoSourceRepoStub) ListEnabled(context.Context) ([]ProxySubscriptionSource, error) {
	return append([]ProxySubscriptionSource(nil), s.enabled...), nil
}

func (s *projectMihomoSourceRepoStub) ListDueForRefresh(context.Context, time.Time, int) ([]ProxySubscriptionSource, error) {
	return append([]ProxySubscriptionSource(nil), s.enabled...), nil
}

func (s *projectMihomoSourceRepoStub) Update(context.Context, *ProxySubscriptionSource) error {
	return nil
}

func (s *projectMihomoSourceRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (s *projectMihomoAdminServiceStub) UpdateProxy(_ context.Context, id int64, input *UpdateProxyInput) (*Proxy, error) {
	s.updated = append(s.updated, struct {
		id    int64
		input UpdateProxyInput
	}{id: id, input: *input})
	proxy := &Proxy{ID: id, Name: input.Name, Protocol: input.Protocol, Host: input.Host, Port: input.Port, Status: input.Status}
	return proxy, nil
}
