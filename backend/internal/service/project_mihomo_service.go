package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"gopkg.in/yaml.v3"
)

const (
	projectMihomoConfigFilename = "config.yaml"
	projectMihomoProviderName   = "project-subscription"
	projectMihomoProviderPath   = "./providers/project-subscription.yaml"
	projectMihomoHTTPTimeout    = 10 * time.Second
	projectMihomoReloadPath     = "/root/.config/mihomo/config.yaml"
)

var (
	ErrProjectMihomoSubscriptionRequired = infraerrors.BadRequest("PROJECT_MIHOMO_SUBSCRIPTION_REQUIRED", "subscription_url is required")
	ErrProjectMihomoControllerRequired   = infraerrors.BadRequest("PROJECT_MIHOMO_CONTROLLER_REQUIRED", "controller_url is required")
	ErrProjectMihomoProtocolInvalid      = infraerrors.BadRequest("PROJECT_MIHOMO_PROTOCOL_INVALID", "protocol must be one of http, https, socks5, socks5h")
	ErrProjectMihomoListenerCountInvalid = infraerrors.BadRequest("PROJECT_MIHOMO_LISTENER_COUNT_INVALID", "listener_count must be between 1 and 32")
	ErrProjectMihomoStartPortInvalid     = infraerrors.BadRequest("PROJECT_MIHOMO_START_PORT_INVALID", "start_port must be between 1 and 65535")
	ErrProjectMihomoPortRangeInvalid     = infraerrors.BadRequest("PROJECT_MIHOMO_PORT_RANGE_INVALID", "listener ports exceed valid range")
)

type ProjectMihomoSettings struct {
	SubscriptionURL  string   `json:"subscription_url"`
	SubscriptionURLs []string `json:"subscription_urls"`
	SubscriptionUA   string   `json:"subscription_user_agent"`
	UpdateInterval   int      `json:"update_interval"`
	Protocol         string   `json:"protocol"`
	TargetHost       string   `json:"target_host"`
	StartPort        int      `json:"start_port"`
	ListenerCount    int      `json:"listener_count"`
	ControllerURL    string   `json:"controller_url"`
	ControllerSecret string   `json:"controller_secret"`
	ProxyNamePrefix  string   `json:"proxy_name_prefix"`
	ListenerRegions  []string `json:"listener_regions"`
}

type ProjectMihomoProxy struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

type ProjectMihomoStatus struct {
	Settings         ProjectMihomoSettings `json:"settings"`
	ConfigPath       string                `json:"config_path"`
	Proxies          []ProjectMihomoProxy  `json:"proxies"`
	AvailableRegions []string              `json:"available_regions"`
}

type ProjectMihomoSyncResult struct {
	ConfigPath string               `json:"config_path"`
	Proxies    []ProjectMihomoProxy `json:"proxies"`
	Created    int                  `json:"created"`
	Reused     int                  `json:"reused"`
	Reloaded   bool                 `json:"reloaded"`
}

type ProjectMihomoService struct {
	settingRepo  SettingRepository
	adminService AdminService
}

func NewProjectMihomoService(settingRepo SettingRepository, adminService AdminService) *ProjectMihomoService {
	return &ProjectMihomoService{settingRepo: settingRepo, adminService: adminService}
}

func DefaultProjectMihomoSettings() ProjectMihomoSettings {
	targetHost := "mihomo-sub2api"
	controllerURL := "http://mihomo-sub2api:9097"
	if _, err := os.Stat("/app/data"); err != nil {
		targetHost = "127.0.0.1"
		controllerURL = "http://127.0.0.1:9097"
	}
	return ProjectMihomoSettings{
		SubscriptionUA:  "sub2api/mihomo",
		UpdateInterval:  3600,
		Protocol:        "socks5h",
		TargetHost:      targetHost,
		StartPort:       41001,
		ListenerCount:   4,
		ControllerURL:   controllerURL,
		ProxyNamePrefix: "project-mihomo",
		ListenerRegions: make([]string, 4),
	}
}

func (s *ProjectMihomoService) GetSettings(ctx context.Context) (*ProjectMihomoSettings, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyProjectMihomoSettings)
	if err != nil {
		if err == ErrSettingNotFound || strings.Contains(err.Error(), "not found") {
			defaults := DefaultProjectMihomoSettings()
			return &defaults, nil
		}
		return nil, fmt.Errorf("get project mihomo settings: %w", err)
	}
	settings := DefaultProjectMihomoSettings()
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			defaults := DefaultProjectMihomoSettings()
			return &defaults, nil
		}
	}
	s.normalize(&settings)
	return &settings, nil
}

func (s *ProjectMihomoService) SetSettings(ctx context.Context, settings *ProjectMihomoSettings) (*ProjectMihomoSettings, error) {
	normalized := *settings
	s.normalize(&normalized)
	if err := s.validate(&normalized); err != nil {
		return nil, err
	}
	data, err := json.Marshal(&normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal project mihomo settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyProjectMihomoSettings, string(data)); err != nil {
		return nil, fmt.Errorf("save project mihomo settings: %w", err)
	}
	return &normalized, nil
}

func (s *ProjectMihomoService) GetStatus(ctx context.Context) (*ProjectMihomoStatus, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &ProjectMihomoStatus{
		Settings:         *settings,
		ConfigPath:       s.configPath(),
		Proxies:          s.buildProxies(settings),
		AvailableRegions: s.availableRegions(settings),
	}, nil
}

func (s *ProjectMihomoService) Sync(ctx context.Context, settings *ProjectMihomoSettings) (*ProjectMihomoSyncResult, error) {
	saved, err := s.SetSettings(ctx, settings)
	if err != nil {
		return nil, err
	}
	configPath, err := s.writeConfig(saved)
	if err != nil {
		return nil, err
	}
	proxies := s.buildProxies(saved)
	created, reused, err := s.syncProxyRows(ctx, saved, proxies)
	if err != nil {
		return nil, err
	}
	reloaded := s.reloadConfig(ctx, saved, projectMihomoReloadPath) == nil
	return &ProjectMihomoSyncResult{ConfigPath: configPath, Proxies: proxies, Created: created, Reused: reused, Reloaded: reloaded}, nil
}

func (s *ProjectMihomoService) normalize(settings *ProjectMihomoSettings) {
	defaults := DefaultProjectMihomoSettings()
	settings.SubscriptionURLs = normalizeProjectMihomoSubscriptionURLs(settings.SubscriptionURLs, settings.SubscriptionURL)
	if len(settings.SubscriptionURLs) > 0 {
		settings.SubscriptionURL = settings.SubscriptionURLs[0]
	} else {
		settings.SubscriptionURL = ""
	}
	settings.SubscriptionUA = defaultString(settings.SubscriptionUA, defaults.SubscriptionUA)
	settings.Protocol = strings.ToLower(defaultString(settings.Protocol, defaults.Protocol))
	settings.TargetHost = defaultString(settings.TargetHost, defaults.TargetHost)
	settings.ControllerURL = defaultString(settings.ControllerURL, defaults.ControllerURL)
	if !strings.Contains(settings.ControllerURL, "://") {
		settings.ControllerURL = "http://" + settings.ControllerURL
	}
	settings.ControllerSecret = strings.TrimSpace(settings.ControllerSecret)
	settings.ProxyNamePrefix = defaultString(settings.ProxyNamePrefix, defaults.ProxyNamePrefix)
	if settings.UpdateInterval <= 0 {
		settings.UpdateInterval = defaults.UpdateInterval
	}
	if settings.StartPort == 0 {
		settings.StartPort = defaults.StartPort
	}
	if settings.ListenerCount == 0 {
		settings.ListenerCount = defaults.ListenerCount
	}
	settings.ListenerRegions = normalizeProjectMihomoListenerRegions(settings.ListenerRegions, settings.ListenerCount)
}

func (s *ProjectMihomoService) validate(settings *ProjectMihomoSettings) error {
	if len(settings.SubscriptionURLs) == 0 {
		return ErrProjectMihomoSubscriptionRequired
	}
	if strings.TrimSpace(settings.ControllerURL) == "" {
		return ErrProjectMihomoControllerRequired
	}
	switch settings.Protocol {
	case "http", "https", "socks5", "socks5h":
	default:
		return ErrProjectMihomoProtocolInvalid
	}
	if settings.ListenerCount < 1 || settings.ListenerCount > 32 {
		return ErrProjectMihomoListenerCountInvalid
	}
	if settings.StartPort < 1 || settings.StartPort > 65535 {
		return ErrProjectMihomoStartPortInvalid
	}
	if settings.StartPort+settings.ListenerCount-1 > 65535 {
		return ErrProjectMihomoPortRangeInvalid
	}
	return nil
}

func (s *ProjectMihomoService) buildProxies(settings *ProjectMihomoSettings) []ProjectMihomoProxy {
	out := make([]ProjectMihomoProxy, 0, settings.ListenerCount)
	for i := 0; i < settings.ListenerCount; i++ {
		out = append(out, ProjectMihomoProxy{Name: fmt.Sprintf("%s-%02d", settings.ProxyNamePrefix, i+1), Protocol: settings.Protocol, Host: settings.TargetHost, Port: settings.StartPort + i})
	}
	return out
}

func (s *ProjectMihomoService) syncProxyRows(ctx context.Context, settings *ProjectMihomoSettings, proxies []ProjectMihomoProxy) (int, int, error) {
	existing, err := s.adminService.GetAllProxies(ctx)
	if err != nil {
		return 0, 0, err
	}
	created, reused := 0, 0
	for _, expected := range proxies {
		var match *Proxy
		for i := range existing {
			if strings.EqualFold(strings.TrimSpace(existing[i].Name), expected.Name) || (existing[i].Host == expected.Host && existing[i].Port == expected.Port) {
				match = &existing[i]
				break
			}
		}
		if match == nil {
			if _, err := s.adminService.CreateProxy(ctx, &CreateProxyInput{Name: expected.Name, Protocol: expected.Protocol, Host: expected.Host, Port: expected.Port}); err != nil {
				return created, reused, err
			}
			created++
			continue
		}
		reused++
		if match.Name != expected.Name || match.Protocol != expected.Protocol || match.Host != expected.Host || match.Port != expected.Port || match.Status != StatusActive {
			if _, err := s.adminService.UpdateProxy(ctx, match.ID, &UpdateProxyInput{Name: expected.Name, Protocol: expected.Protocol, Host: expected.Host, Port: expected.Port, Status: StatusActive}); err != nil {
				return created, reused, err
			}
		}
	}
	return created, reused, nil
}

func (s *ProjectMihomoService) writeConfig(settings *ProjectMihomoSettings) (string, error) {
	if err := os.MkdirAll(filepath.Join(s.configDir(), "providers"), 0o755); err != nil {
		return "", fmt.Errorf("create mihomo dir: %w", err)
	}
	content, err := s.renderConfig(settings)
	if err != nil {
		return "", err
	}
	path := s.configPath()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", fmt.Errorf("write mihomo config: %w", err)
	}
	return path, nil
}

func (s *ProjectMihomoService) renderConfig(settings *ProjectMihomoSettings) ([]byte, error) {
	providerConfigs := map[string]any{}
	providerNames := make([]string, 0, len(settings.SubscriptionURLs))
	for i, rawURL := range settings.SubscriptionURLs {
		name := projectMihomoProviderName
		path := projectMihomoProviderPath
		if len(settings.SubscriptionURLs) > 1 {
			name = fmt.Sprintf("%s-%02d", projectMihomoProviderName, i+1)
			path = fmt.Sprintf("./providers/%s.yaml", name)
		}
		providerNames = append(providerNames, name)
		providerConfigs[name] = map[string]any{"type": "http", "url": rawURL, "path": path, "interval": settings.UpdateInterval, "header": map[string]any{"User-Agent": []string{settings.SubscriptionUA}}, "health-check": map[string]any{"enable": true, "url": "https://www.gstatic.com/generate_204", "interval": 300, "timeout": 5000, "lazy": true}}
	}
	listeners := make([]map[string]any, 0, settings.ListenerCount)
	groups := make([]map[string]any, 0, settings.ListenerCount)
	for _, proxy := range s.buildProxies(settings) {
		listeners = append(listeners, map[string]any{"name": proxy.Name, "type": "mixed", "port": proxy.Port, "listen": "0.0.0.0", "udp": true, "proxy": proxy.Name})
		groups = append(groups, map[string]any{"name": proxy.Name, "type": "select", "use": providerNames})
	}
	root := map[string]any{"mode": "rule", "allow-lan": true, "bind-address": "*", "external-controller": controllerListenAddress(settings.ControllerURL), "secret": settings.ControllerSecret, "log-level": "info", "proxy-providers": providerConfigs, "proxy-groups": groups, "listeners": listeners}
	data, err := yaml.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("marshal mihomo config: %w", err)
	}
	return data, nil
}

func (s *ProjectMihomoService) reloadConfig(ctx context.Context, settings *ProjectMihomoSettings, configPath string) error {
	client, err := httpclient.GetClient(httpclient.Options{Timeout: projectMihomoHTTPTimeout})
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{"path": configPath})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(settings.ControllerURL, "/")+"/configs?force=true", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if settings.ControllerSecret != "" {
		req.Header.Set("Authorization", "Bearer "+settings.ControllerSecret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("reload mihomo config: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (s *ProjectMihomoService) availableRegions(settings *ProjectMihomoSettings) []string { return nil }

func (s *ProjectMihomoService) configDir() string {
	if dataDir := strings.TrimSpace(os.Getenv("DATA_DIR")); dataDir != "" {
		return filepath.Join(dataDir, "mihomo")
	}
	if _, err := os.Stat("/app/data"); err == nil {
		return "/app/data/mihomo"
	}
	return filepath.Join(".", "data", "mihomo")
}

func (s *ProjectMihomoService) configPath() string {
	return filepath.Join(s.configDir(), projectMihomoConfigFilename)
}

func normalizeProjectMihomoListenerRegions(regions []string, count int) []string {
	out := make([]string, count)
	for i := 0; i < count; i++ {
		if i < len(regions) {
			out[i] = strings.TrimSpace(regions[i])
		}
	}
	return out
}

func normalizeProjectMihomoSubscriptionURLs(urls []string, fallback string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	appendOne := func(value string) {
		for _, item := range strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == '\r' }) {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	for _, item := range urls {
		appendOne(item)
	}
	appendOne(fallback)
	return out
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func controllerListenAddress(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "0.0.0.0:9097"
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Port() != "" {
		return net.JoinHostPort("0.0.0.0", parsed.Port())
	}
	return "0.0.0.0:9097"
}
