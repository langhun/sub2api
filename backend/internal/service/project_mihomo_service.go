package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	mihomoConfigFilename = "config.yaml"
	mihomoHTTPTimeout    = 10 * time.Second
)

var (
	ErrMihomoSubscriptionRequired = infraerrors.BadRequest("PROJECT_MIHOMO_SUBSCRIPTION_REQUIRED", "at least one enabled proxy subscription source is required")
	ErrMihomoSubscriptionInvalid  = infraerrors.BadRequest("PROJECT_MIHOMO_SUBSCRIPTION_INVALID", "proxy subscription source url must be a valid http or https URL")
	ErrMihomoControllerRequired   = infraerrors.BadRequest("PROJECT_MIHOMO_CONTROLLER_REQUIRED", "controller_url is required")
	ErrMihomoProtocolInvalid      = infraerrors.BadRequest("PROJECT_MIHOMO_PROTOCOL_INVALID", "protocol must be one of http, https, socks5, socks5h")
	ErrMihomoControllerInvalid    = infraerrors.BadRequest("PROJECT_MIHOMO_CONTROLLER_INVALID", "controller_url must be a valid http or https URL")
	ErrMihomoSettingsCorrupted    = infraerrors.InternalServer("PROJECT_MIHOMO_SETTINGS_CORRUPTED", "mihomo settings are corrupted")
	ErrMihomoListenerCountInvalid = infraerrors.BadRequest("PROJECT_MIHOMO_LISTENER_COUNT_INVALID", "listener_count must be between 1 and 32")
	ErrMihomoStartPortInvalid     = infraerrors.BadRequest("PROJECT_MIHOMO_START_PORT_INVALID", "start_port must be between 1 and 65535")
	ErrMihomoPortRangeInvalid     = infraerrors.BadRequest("PROJECT_MIHOMO_PORT_RANGE_INVALID", "listener ports exceed valid range")
)

type MihomoSettings struct {
	Protocol         string   `json:"protocol"`
	TargetHost       string   `json:"target_host"`
	StartPort        int      `json:"start_port"`
	ListenerCount    int      `json:"listener_count"`
	ControllerURL    string   `json:"controller_url"`
	ControllerSecret string   `json:"controller_secret"`
	ProxyNamePrefix  string   `json:"proxy_name_prefix"`
	ListenerRegions  []string `json:"listener_regions"`
	AutoOptimize     bool     `json:"auto_optimize"`
	CountryFilter    string   `json:"country_filter"`
}

type MihomoProxy struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

type MihomoStatus struct {
	Settings         MihomoSettings `json:"settings"`
	ConfigPath       string         `json:"config_path"`
	Proxies          []MihomoProxy  `json:"proxies"`
	AvailableRegions []string       `json:"available_regions"`
}

type MihomoSyncResult struct {
	ConfigPath string        `json:"config_path"`
	Proxies    []MihomoProxy `json:"proxies"`
	Created    int           `json:"created"`
	Reused     int           `json:"reused"`
	Reloaded   bool          `json:"reloaded"`
}

type MihomoService struct {
	settingRepo  SettingRepository
	sourceRepo   ProxySubscriptionSourceRepository
	adminService mihomoAdminService
}

type mihomoAdminService interface {
	ListProxies(ctx context.Context, page, pageSize int, protocol, status, search string, sortBy, sortOrder string) ([]Proxy, int64, error)
	CreateProxy(ctx context.Context, input *CreateProxyInput) (*Proxy, error)
	UpdateProxy(ctx context.Context, id int64, input *UpdateProxyInput) (*Proxy, error)
}

func NewMihomoService(settingRepo SettingRepository, sourceRepo ProxySubscriptionSourceRepository, adminService mihomoAdminService) *MihomoService {
	return &MihomoService{settingRepo: settingRepo, sourceRepo: sourceRepo, adminService: adminService}
}

func DefaultMihomoSettings() MihomoSettings {
	targetHost, controllerURL := defaultMihomoEndpoint()
	return MihomoSettings{
		Protocol:        "socks5h",
		TargetHost:      targetHost,
		StartPort:       41001,
		ListenerCount:   4,
		ControllerURL:   controllerURL,
		ProxyNamePrefix: "mihomo",
		ListenerRegions: make([]string, 4),
	}
}

type mihomoProvider struct {
	Name string
	Path string
}

func (s *MihomoService) GetSettings(ctx context.Context) (*MihomoSettings, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyMihomoSettings)
	if err != nil {
		if err == ErrSettingNotFound || strings.Contains(err.Error(), "not found") {
			defaults := DefaultMihomoSettings()
			return &defaults, nil
		}
		return nil, fmt.Errorf("get project mihomo settings: %w", err)
	}
	settings := DefaultMihomoSettings()
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return nil, ErrMihomoSettingsCorrupted.WithCause(err)
		}
	}
	s.normalize(&settings)
	return &settings, nil
}

func (s *MihomoService) SetSettings(ctx context.Context, settings *MihomoSettings) (*MihomoSettings, error) {
	normalized := *settings
	s.normalize(&normalized)
	if err := s.validate(&normalized); err != nil {
		return nil, err
	}
	data, err := json.Marshal(&normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal project mihomo settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyMihomoSettings, string(data)); err != nil {
		return nil, fmt.Errorf("save project mihomo settings: %w", err)
	}
	return &normalized, nil
}

func (s *MihomoService) GetStatus(ctx context.Context) (*MihomoStatus, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &MihomoStatus{
		Settings:         *settings,
		ConfigPath:       s.configPath(),
		Proxies:          s.buildProxies(settings),
		AvailableRegions: s.availableRegions(settings),
	}, nil
}

func (s *MihomoService) Sync(ctx context.Context, settings *MihomoSettings) (*MihomoSyncResult, error) {
	previous, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	saved, err := s.SetSettings(ctx, settings)
	if err != nil {
		return nil, err
	}
	providers, err := s.resolveProviders(ctx)
	if err != nil {
		return nil, err
	}
	configPath, err := s.writeConfig(saved, providers)
	if err != nil {
		return nil, err
	}
	proxies := s.buildProxies(saved)
	created, reused, err := s.syncProxyRows(ctx, previous, saved, proxies)
	if err != nil {
		return nil, err
	}
	if err := s.reloadConfig(ctx, saved, configPath); err != nil {
		return nil, fmt.Errorf("reload mihomo config: %w", err)
	}
	return &MihomoSyncResult{ConfigPath: configPath, Proxies: proxies, Created: created, Reused: reused, Reloaded: true}, nil
}

func (s *MihomoService) normalize(settings *MihomoSettings) {
	defaults := DefaultMihomoSettings()
	settings.Protocol = strings.ToLower(defaultString(settings.Protocol, defaults.Protocol))
	settings.TargetHost = defaultString(settings.TargetHost, defaults.TargetHost)
	settings.ControllerURL = defaultString(settings.ControllerURL, defaults.ControllerURL)
	if !strings.Contains(settings.ControllerURL, "://") {
		settings.ControllerURL = "http://" + settings.ControllerURL
	}
	settings.ControllerSecret = strings.TrimSpace(settings.ControllerSecret)
	settings.ProxyNamePrefix = defaultString(settings.ProxyNamePrefix, defaults.ProxyNamePrefix)
	settings.CountryFilter = strings.TrimSpace(settings.CountryFilter)
	if settings.StartPort == 0 {
		settings.StartPort = defaults.StartPort
	}
	if settings.ListenerCount == 0 {
		settings.ListenerCount = defaults.ListenerCount
	}
	settings.ListenerRegions = normalizeMihomoListenerRegions(settings.ListenerRegions, settings.ListenerCount)
}

func (s *MihomoService) validate(settings *MihomoSettings) error {
	if strings.TrimSpace(settings.ControllerURL) == "" {
		return ErrMihomoControllerRequired
	}
	controllerURL, err := url.Parse(settings.ControllerURL)
	if err != nil || controllerURL.Host == "" || (controllerURL.Scheme != "http" && controllerURL.Scheme != "https") {
		return ErrMihomoControllerInvalid
	}
	switch settings.Protocol {
	case "http", "https", "socks5", "socks5h":
	default:
		return ErrMihomoProtocolInvalid
	}
	if settings.ListenerCount < 1 || settings.ListenerCount > 32 {
		return ErrMihomoListenerCountInvalid
	}
	if settings.StartPort < 1 || settings.StartPort > 65535 {
		return ErrMihomoStartPortInvalid
	}
	if settings.StartPort+settings.ListenerCount-1 > 65535 {
		return ErrMihomoPortRangeInvalid
	}
	return nil
}

func (s *MihomoService) resolveProviders(ctx context.Context) ([]mihomoProvider, error) {
	if s.sourceRepo == nil {
		return nil, fmt.Errorf("project mihomo source repository is not configured")
	}
	sources, err := s.sourceRepo.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled proxy subscription sources: %w", err)
	}
	if len(sources) == 0 {
		return nil, ErrMihomoSubscriptionRequired
	}

	providersDir := filepath.Join(s.configDir(), "providers")
	if err := os.MkdirAll(providersDir, 0o700); err != nil {
		return nil, fmt.Errorf("create mihomo providers dir: %w", err)
	}

	providers := make([]mihomoProvider, 0, len(sources))
	for _, source := range sources {
		rawURL := strings.TrimSpace(source.URL)
		parsedURL, err := url.Parse(rawURL)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return nil, ErrMihomoSubscriptionInvalid
		}
		payload, err := s.fetchSubscriptionContent(ctx, rawURL)
		if err != nil {
			return nil, fmt.Errorf("fetch proxy subscription source %d: %w", source.ID, err)
		}
		nodes, _, parseErrs := parseProxySubscriptionPayload(payload, source.SourceFormat)
		filteredNodes, filterErrs := filterRuntimeCandidateSubscriptionNodes(nodes)
		parseErrs = append(parseErrs, filterErrs...)
		if len(filteredNodes) == 0 {
			if len(parseErrs) > 0 {
				return nil, fmt.Errorf("proxy subscription source %d produced no usable nodes: %s", source.ID, parseErrs[len(parseErrs)-1].Message)
			}
			return nil, fmt.Errorf("proxy subscription source %d produced no usable nodes", source.ID)
		}

		filename := fmt.Sprintf("mihomo-subscription-%d.yaml", source.ID)
		path := filepath.Join(providersDir, filename)
		if err := os.WriteFile(path, []byte(buildRuntimeCandidateProviderContent(filteredNodes)), 0o600); err != nil {
			return nil, fmt.Errorf("write provider file for source %d: %w", source.ID, err)
		}
		name := strings.TrimSpace(source.Name)
		if name == "" {
			name = fmt.Sprintf("source-%d", source.ID)
		}
		providers = append(providers, mihomoProvider{
			Name: fmt.Sprintf("%s-%d", sanitizeMihomoProviderName(name), source.ID),
			Path: path,
		})
	}
	return providers, nil
}

func (s *MihomoService) buildProxies(settings *MihomoSettings) []MihomoProxy {
	out := make([]MihomoProxy, 0, settings.ListenerCount)
	for i := 0; i < settings.ListenerCount; i++ {
		out = append(out, MihomoProxy{Name: fmt.Sprintf("%s-%02d", settings.ProxyNamePrefix, i+1), Protocol: settings.Protocol, Host: settings.TargetHost, Port: settings.StartPort + i})
	}
	return out
}

func (s *MihomoService) syncProxyRows(ctx context.Context, previousSettings, settings *MihomoSettings, proxies []MihomoProxy) (int, int, error) {
	existing, err := s.listMihomoProxyRows(ctx, previousSettings, settings)
	if err != nil {
		return 0, 0, err
	}
	created, reused := 0, 0
	matchedIDs := make(map[int64]struct{}, len(proxies))
	for _, expected := range proxies {
		var match *Proxy
		for i := range existing {
			if _, used := matchedIDs[existing[i].ID]; used {
				continue
			}
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
		matchedIDs[match.ID] = struct{}{}
		reused++
		if match.Name != expected.Name || match.Protocol != expected.Protocol || match.Host != expected.Host || match.Port != expected.Port || match.Status != StatusActive {
			if _, err := s.adminService.UpdateProxy(ctx, match.ID, &UpdateProxyInput{Name: expected.Name, Protocol: expected.Protocol, Host: expected.Host, Port: expected.Port, Status: StatusActive}); err != nil {
				return created, reused, err
			}
		}
	}
	for i := range existing {
		if _, ok := matchedIDs[existing[i].ID]; ok || existing[i].Status == StatusDisabled {
			continue
		}
		if _, err := s.adminService.UpdateProxy(ctx, existing[i].ID, &UpdateProxyInput{
			Name:     existing[i].Name,
			Protocol: existing[i].Protocol,
			Host:     existing[i].Host,
			Port:     existing[i].Port,
			Status:   StatusDisabled,
		}); err != nil {
			return created, reused, err
		}
	}
	return created, reused, nil
}

func (s *MihomoService) listMihomoProxyRows(ctx context.Context, settings ...*MihomoSettings) ([]Proxy, error) {
	if s.adminService == nil {
		return nil, fmt.Errorf("project mihomo admin service is not configured")
	}
	const pageSize = 500
	searches := make([]string, 0, len(settings))
	seenSearches := map[string]struct{}{}
	for _, cfg := range settings {
		if cfg == nil {
			continue
		}
		search := strings.TrimSpace(cfg.ProxyNamePrefix)
		if search == "" {
			continue
		}
		if _, ok := seenSearches[search]; ok {
			continue
		}
		seenSearches[search] = struct{}{}
		searches = append(searches, search)
	}
	out := []Proxy{}
	seenIDs := map[int64]struct{}{}
	for _, search := range searches {
		for page := 1; ; page++ {
			items, total, err := s.adminService.ListProxies(ctx, page, pageSize, "", "", search, "id", "asc")
			if err != nil {
				return nil, err
			}
			for _, item := range items {
				if _, ok := seenIDs[item.ID]; ok {
					continue
				}
				seenIDs[item.ID] = struct{}{}
				out = append(out, item)
			}
			if len(items) < pageSize || page >= int((total+int64(pageSize)-1)/int64(pageSize)) {
				break
			}
		}
	}
	return out, nil
}

func (s *MihomoService) writeConfig(settings *MihomoSettings, providers []mihomoProvider) (string, error) {
	if err := os.MkdirAll(filepath.Join(s.configDir(), "providers"), 0o700); err != nil {
		return "", fmt.Errorf("create mihomo dir: %w", err)
	}
	content, err := s.renderConfig(settings, providers)
	if err != nil {
		return "", err
	}
	path := s.configPath()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", fmt.Errorf("write mihomo config: %w", err)
	}
	return path, nil
}

func (s *MihomoService) renderConfig(settings *MihomoSettings, providers []mihomoProvider) ([]byte, error) {
	providerConfigs := map[string]any{}
	providerNames := make([]string, 0, len(providers))
	for _, provider := range providers {
		providerNames = append(providerNames, provider.Name)
		providerConfigs[provider.Name] = map[string]any{
			"type": "file",
			"path": provider.Path,
			"health-check": map[string]any{
				"enable":   true,
				"url":      "https://www.gstatic.com/generate_204",
				"interval": 300,
				"timeout":  5000,
				"lazy":     true,
			},
		}
	}
	listeners := make([]map[string]any, 0, settings.ListenerCount)
	groups := make([]map[string]any, 0, settings.ListenerCount)
	for i, proxy := range s.buildProxies(settings) {
		listeners = append(listeners, map[string]any{"name": proxy.Name, "type": "mixed", "port": proxy.Port, "listen": "0.0.0.0", "udp": true, "proxy": proxy.Name})
		groups = append(groups, mihomoProxyGroup(proxy.Name, providerNames, mihomoGroupFilter(settings, i), settings.AutoOptimize))
	}
	root := map[string]any{"mode": "rule", "allow-lan": true, "bind-address": "*", "external-controller": controllerListenAddress(settings.ControllerURL), "secret": settings.ControllerSecret, "log-level": "info", "proxy-providers": providerConfigs, "proxy-groups": groups, "listeners": listeners}
	data, err := yaml.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("marshal mihomo config: %w", err)
	}
	return data, nil
}

func (s *MihomoService) fetchSubscriptionContent(ctx context.Context, rawURL string) ([]byte, error) {
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               45 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	})
	if err != nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription fetch failed: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
}

func mihomoProxyGroup(name string, providerNames []string, filter string, autoOptimize bool) map[string]any {
	groupType := "select"
	if autoOptimize || filter != "" {
		groupType = "url-test"
	}
	group := map[string]any{"name": name, "type": groupType, "use": providerNames}
	if filter != "" {
		group["filter"] = filter
	}
	if groupType == "url-test" {
		group["url"] = "https://www.gstatic.com/generate_204"
		group["interval"] = 300
		group["tolerance"] = 50
	}
	return group
}

func mihomoGroupFilter(settings *MihomoSettings, index int) string {
	if settings == nil {
		return ""
	}
	if index >= 0 && index < len(settings.ListenerRegions) {
		if region := strings.TrimSpace(settings.ListenerRegions[index]); region != "" {
			return region
		}
	}
	if !settings.AutoOptimize {
		return strings.TrimSpace(settings.CountryFilter)
	}
	return ""
}

func (s *MihomoService) reloadConfig(ctx context.Context, settings *MihomoSettings, configPath string) error {
	client, err := httpclient.GetClient(httpclient.Options{Timeout: mihomoHTTPTimeout})
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
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("reload mihomo config: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (s *MihomoService) availableRegions(settings *MihomoSettings) []string {
	regions := []string{"香港", "台湾", "日本", "韩国", "新加坡", "美国", "英国", "德国", "法国", "加拿大", "澳大利亚"}
	extra := 1
	if settings != nil {
		extra += len(settings.ListenerRegions)
	}
	seen := make(map[string]struct{}, len(regions)+extra)
	out := make([]string, 0, len(regions)+extra)
	appendRegion := func(region string) {
		region = strings.TrimSpace(region)
		if region == "" {
			return
		}
		key := strings.ToLower(region)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, region)
	}
	for _, region := range regions {
		appendRegion(region)
	}
	if settings != nil {
		appendRegion(settings.CountryFilter)
		for _, region := range settings.ListenerRegions {
			appendRegion(region)
		}
	}
	return out
}

func (s *MihomoService) configDir() string {
	if dataDir := strings.TrimSpace(os.Getenv("DATA_DIR")); dataDir != "" {
		return filepath.Join(dataDir, "mihomo")
	}
	if _, err := os.Stat("/app/data"); err == nil {
		return "/app/data/mihomo"
	}
	return filepath.Join(".", "data", "mihomo")
}

func defaultMihomoEndpoint() (string, string) {
	if dataDir := strings.TrimSpace(os.Getenv("DATA_DIR")); dataDir != "" {
		if filepath.ToSlash(filepath.Clean(dataDir)) == "/app/data" {
			return "mihomo-sub2api", "http://mihomo-sub2api:9097"
		}
		return "127.0.0.1", "http://127.0.0.1:9097"
	}
	if _, err := os.Stat("/app/data"); err == nil {
		return "mihomo-sub2api", "http://mihomo-sub2api:9097"
	}
	return "127.0.0.1", "http://127.0.0.1:9097"
}

func (s *MihomoService) configPath() string {
	return filepath.Join(s.configDir(), mihomoConfigFilename)
}

func normalizeMihomoListenerRegions(regions []string, count int) []string {
	out := make([]string, count)
	for i := 0; i < count; i++ {
		if i < len(regions) {
			out[i] = strings.TrimSpace(regions[i])
		}
	}
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

func sanitizeMihomoProviderName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "source"
	}
	var b strings.Builder
	b.Grow(len(value))
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		return "source"
	}
	return name
}
