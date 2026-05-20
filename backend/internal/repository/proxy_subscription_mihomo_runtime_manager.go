package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	appconfig "github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	mihomoConfig "github.com/metacubex/mihomo/config"
	mihomoConstant "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/hub/route"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel/statistic"
	"gopkg.in/yaml.v3"
)

type proxySubscriptionMihomoRuntimeManager struct {
	cfg         appconfig.ProxySubscriptionMihomoConfig
	dataDir     string
	listenerDir string

	mu          sync.Mutex
	runtimes    map[string]*mihomoRuntimeState
	initialized bool
}

type mihomoRuntimeState struct {
	RuntimeID    string
	ConfigPath   string
	ProviderPath string
	ListenerHost string
	ListenerPort int
	Stopped      bool
}

type mihomoConfigFile struct {
	AllowLAN       bool                      `yaml:"allow-lan"`
	BindAddress    string                    `yaml:"bind-address,omitempty"`
	SocksPort      int                       `yaml:"socks-port,omitempty"`
	Mode           string                    `yaml:"mode"`
	LogLevel       string                    `yaml:"log-level"`
	IPv6           bool                      `yaml:"ipv6"`
	Listeners      []mihomoListener          `yaml:"listeners,omitempty"`
	ProxyProviders map[string]mihomoProvider `yaml:"proxy-providers,omitempty"`
	ProxyGroups    []mihomoProxyGroup        `yaml:"proxy-groups"`
	Rules          []string                  `yaml:"rules"`
}

type mihomoListener struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Listen string `yaml:"listen"`
	Port   int    `yaml:"port"`
	UDP    bool   `yaml:"udp"`
	Proxy  string `yaml:"proxy"`
}

type mihomoProvider struct {
	Type        string            `yaml:"type"`
	Path        string            `yaml:"path"`
	HealthCheck mihomoHealthCheck `yaml:"health-check"`
}

type mihomoHealthCheck struct {
	Enable   bool   `yaml:"enable"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval"`
}

type mihomoProxyGroup struct {
	Name      string   `yaml:"name"`
	Type      string   `yaml:"type"`
	Use       []string `yaml:"use,omitempty"`
	Proxies   []string `yaml:"proxies,omitempty"`
	URL       string   `yaml:"url,omitempty"`
	Interval  int      `yaml:"interval,omitempty"`
	Tolerance int      `yaml:"tolerance,omitempty"`
}

func NewProxySubscriptionMihomoRuntimeManager(cfg *appconfig.Config) service.ProxySubscriptionRuntimeManager {
	if cfg == nil || !cfg.ProxySubscriptionMihomo.Enabled {
		return nil
	}
	dataDir := strings.TrimSpace(cfg.ProxySubscriptionMihomo.DataDir)
	if dataDir == "" {
		dataDir = filepath.Join(resolveRuntimeDataDir(), "proxy-subscription-mihomo")
	}
	return &proxySubscriptionMihomoRuntimeManager{
		cfg:         cfg.ProxySubscriptionMihomo,
		dataDir:     dataDir,
		listenerDir: filepath.Join(dataDir, "runtimes"),
		runtimes:    make(map[string]*mihomoRuntimeState),
	}
}

func (m *proxySubscriptionMihomoRuntimeManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.listenerDir, 0o755); err != nil {
		return err
	}
	if m.initialized {
		return nil
	}
	if err := m.rehydrateExistingRuntimesLocked(ctx); err != nil {
		return err
	}
	if err := m.applyEmbeddedConfigLocked(ctx); err != nil {
		return err
	}
	m.initialized = true
	return nil
}

func (m *proxySubscriptionMihomoRuntimeManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	closeEmbeddedMihomoConnections()
	m.runtimes = make(map[string]*mihomoRuntimeState)
	m.initialized = false
	err := m.applyEmbeddedConfigLocked(ctx)
	executor.Shutdown()
	return err
}

func (m *proxySubscriptionMihomoRuntimeManager) UpsertRuntime(ctx context.Context, req service.ProxySubscriptionRuntimeUpsertRequest) (*service.ProxySubscriptionRuntimeUpsertResponse, error) {
	if err := m.Start(ctx); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.runtimes[req.RuntimeID]

	listenerHost := strings.TrimSpace(req.ListenerHost)
	if listenerHost == "" {
		listenerHost = strings.TrimSpace(m.cfg.ListenerHost)
	}
	if listenerHost == "" {
		listenerHost = "127.0.0.1"
	}

	preferredPort := req.ListenerPort
	if current != nil && current.ListenerPort > 0 && preferredPort == 0 {
		preferredPort = current.ListenerPort
	}
	listenerPort, err := m.allocatePortLocked(req.RuntimeID, preferredPort, listenerHost)
	if err != nil {
		return nil, err
	}

	providerPath := filepath.Join(m.listenerDir, req.RuntimeID+".provider.yaml")
	configPath := filepath.Join(m.listenerDir, req.RuntimeID+".yaml")
	oldProvider, providerExisted := readRollbackFile(providerPath)
	oldConfig, configExisted := readRollbackFile(configPath)
	if err := os.WriteFile(providerPath, []byte(buildProviderContent(req)), 0o644); err != nil {
		return nil, err
	}
	cfgBytes, err := buildMihomoConfig(req.RuntimeID, providerPath, listenerHost, listenerPort)
	if err != nil {
		restoreRollbackFile(providerPath, oldProvider, providerExisted)
		return nil, err
	}
	if err := os.WriteFile(configPath, cfgBytes, 0o644); err != nil {
		restoreRollbackFile(providerPath, oldProvider, providerExisted)
		return nil, err
	}

	state := &mihomoRuntimeState{
		RuntimeID:    req.RuntimeID,
		ConfigPath:   configPath,
		ProviderPath: providerPath,
		ListenerHost: listenerHost,
		ListenerPort: listenerPort,
	}

	m.runtimes[req.RuntimeID] = state
	if err := m.applyEmbeddedConfigLocked(ctx); err != nil {
		m.restoreRuntimeUpsertLocked(req.RuntimeID, current, providerPath, oldProvider, providerExisted, configPath, oldConfig, configExisted)
		return nil, err
	}
	if err := waitForTCP(listenerHost, listenerPort, 12*time.Second); err != nil {
		m.restoreRuntimeUpsertLocked(req.RuntimeID, current, providerPath, oldProvider, providerExisted, configPath, oldConfig, configExisted)
		rollbackErr := m.applyEmbeddedConfigLocked(ctx)
		return nil, errors.Join(fmt.Errorf("embedded mihomo listener not ready: %w", err), rollbackErr)
	}

	return &service.ProxySubscriptionRuntimeUpsertResponse{
		RuntimeID:    req.RuntimeID,
		ListenerHost: listenerHost,
		ListenerPort: listenerPort,
		Protocol:     service.ProxyNodeTypeSOCKS5H,
	}, nil
}

func (m *proxySubscriptionMihomoRuntimeManager) DeleteRuntime(ctx context.Context, runtimeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.runtimes[runtimeID]
	if state == nil {
		return nil
	}
	delete(m.runtimes, runtimeID)
	if err := m.applyEmbeddedConfigLocked(ctx); err != nil {
		m.runtimes[runtimeID] = state
		return err
	}
	_ = os.Remove(filepath.Join(m.listenerDir, runtimeID+".provider.yaml"))
	_ = os.Remove(filepath.Join(m.listenerDir, runtimeID+".yaml"))
	return nil
}

func (m *proxySubscriptionMihomoRuntimeManager) CheckRuntime(_ context.Context, runtimeID string) error {
	m.mu.Lock()
	state := m.runtimes[runtimeID]
	m.mu.Unlock()
	if state == nil {
		return fmt.Errorf("runtime not found")
	}
	return waitForTCP(state.ListenerHost, state.ListenerPort, 2*time.Second)
}

func (m *proxySubscriptionMihomoRuntimeManager) applyEmbeddedConfigLocked(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := os.MkdirAll(m.dataDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(m.listenerDir, 0o755); err != nil {
		return err
	}
	route.SetEmbedMode(true)
	log.SetLevel(log.WARNING)
	mihomoConstant.SetHomeDir(m.dataDir)
	mihomoConstant.SetConfig(filepath.Join(m.listenerDir, "_embedded.yaml"))
	if err := mihomoConfig.Init(m.dataDir); err != nil {
		return err
	}
	cfgBytes, err := m.buildEmbeddedConfigLocked()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(m.listenerDir, "_embedded.yaml"), cfgBytes, 0o644); err != nil {
		return err
	}
	cfg, err := executor.ParseWithBytes(cfgBytes)
	if err != nil {
		return fmt.Errorf("parse embedded mihomo config: %w", err)
	}
	executor.ApplyConfig(cfg, true)
	return nil
}

func closeEmbeddedMihomoConnections() {
	statistic.DefaultManager.Range(func(c statistic.Tracker) bool {
		_ = c.Close()
		return true
	})
}

func (m *proxySubscriptionMihomoRuntimeManager) restoreRuntimeUpsertLocked(
	runtimeID string,
	current *mihomoRuntimeState,
	providerPath string,
	oldProvider []byte,
	providerExisted bool,
	configPath string,
	oldConfig []byte,
	configExisted bool,
) {
	if current != nil {
		m.runtimes[runtimeID] = current
	} else {
		delete(m.runtimes, runtimeID)
	}
	restoreRollbackFile(providerPath, oldProvider, providerExisted)
	restoreRollbackFile(configPath, oldConfig, configExisted)
}

func readRollbackFile(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func restoreRollbackFile(path string, data []byte, existed bool) {
	if !existed {
		_ = os.Remove(path)
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func (m *proxySubscriptionMihomoRuntimeManager) buildEmbeddedConfigLocked() ([]byte, error) {
	cfg := mihomoConfigFile{
		AllowLAN:       false,
		Mode:           "rule",
		LogLevel:       "warning",
		IPv6:           true,
		Listeners:      []mihomoListener{},
		ProxyProviders: map[string]mihomoProvider{},
		ProxyGroups:    []mihomoProxyGroup{},
		Rules:          []string{"MATCH,DIRECT"},
	}
	runtimeIDs := make([]string, 0, len(m.runtimes))
	for runtimeID := range m.runtimes {
		runtimeIDs = append(runtimeIDs, runtimeID)
	}
	sort.Strings(runtimeIDs)
	for _, runtimeID := range runtimeIDs {
		state := m.runtimes[runtimeID]
		if state == nil || state.Stopped || state.ProviderPath == "" || state.ListenerPort <= 0 {
			continue
		}
		providerPath := state.ProviderPath
		if abs, err := filepath.Abs(providerPath); err == nil {
			providerPath = abs
		}
		providerName := mihomoRuntimeProviderName(runtimeID)
		groupName := mihomoRuntimeGroupName(runtimeID)
		cfg.ProxyProviders[providerName] = mihomoProvider{
			Type: "file",
			Path: providerPath,
			HealthCheck: mihomoHealthCheck{
				Enable:   true,
				URL:      "https://chatgpt.com/cdn-cgi/trace",
				Interval: 90,
			},
		}
		cfg.ProxyGroups = append(cfg.ProxyGroups, mihomoProxyGroup{
			Name:      groupName,
			Type:      "url-test",
			Use:       []string{providerName},
			URL:       "https://chatgpt.com/cdn-cgi/trace",
			Interval:  90,
			Tolerance: 40,
		})
		cfg.Listeners = append(cfg.Listeners, mihomoListener{
			Name:   mihomoRuntimeListenerName(runtimeID),
			Type:   "socks",
			Listen: state.ListenerHost,
			Port:   state.ListenerPort,
			UDP:    true,
			Proxy:  groupName,
		})
	}
	return yaml.Marshal(cfg)
}

func (m *proxySubscriptionMihomoRuntimeManager) allocatePortLocked(runtimeID string, preferredPort int, listenerHost string) (int, error) {
	if current := m.runtimes[runtimeID]; current != nil && current.ListenerPort > 0 {
		return current.ListenerPort, nil
	}
	if preferredPort > 0 && isTCPPortFree(listenerHost, preferredPort) {
		return preferredPort, nil
	}
	start, end := parsePortRange(strings.TrimSpace(m.cfg.ListenerPortRange))
	used := make(map[int]struct{}, len(m.runtimes))
	for _, item := range m.runtimes {
		used[item.ListenerPort] = struct{}{}
	}
	for port := start; port <= end; port++ {
		if _, ok := used[port]; ok {
			continue
		}
		if isTCPPortFree(listenerHost, port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free mihomo listener ports available in %d-%d", start, end)
}

func (m *proxySubscriptionMihomoRuntimeManager) rehydrateExistingRuntimesLocked(ctx context.Context) error {
	entries, err := os.ReadDir(m.listenerDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".provider.yaml") {
			continue
		}
		runtimeID := strings.TrimSuffix(name, ".yaml")
		if !strings.HasPrefix(runtimeID, "src-") {
			continue
		}
		if _, exists := m.runtimes[runtimeID]; exists {
			continue
		}
		configPath := filepath.Join(m.listenerDir, name)
		providerPath := filepath.Join(m.listenerDir, runtimeID+".provider.yaml")
		if _, err := os.Stat(providerPath); err != nil {
			continue
		}
		if err := m.startPersistedRuntimeLocked(ctx, runtimeID, configPath, providerPath); err != nil {
			slog.Warn("proxy_subscription_mihomo_runtime.rehydrate_runtime_failed", "runtime_id", runtimeID, "config_path", configPath, "error", err)
			continue
		}
	}
	return nil
}

func (m *proxySubscriptionMihomoRuntimeManager) startPersistedRuntimeLocked(ctx context.Context, runtimeID, configPath, providerPath string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfgFile mihomoConfigFile
	if err := yaml.Unmarshal(raw, &cfgFile); err != nil {
		return err
	}
	listenerHost := strings.TrimSpace(cfgFile.BindAddress)
	if listenerHost == "" {
		listenerHost = "127.0.0.1"
	}
	listenerPort := cfgFile.SocksPort
	if listenerPort <= 0 {
		return fmt.Errorf("invalid socks port in config: %s", configPath)
	}

	_ = ctx
	m.runtimes[runtimeID] = &mihomoRuntimeState{
		RuntimeID:    runtimeID,
		ConfigPath:   configPath,
		ProviderPath: providerPath,
		ListenerHost: listenerHost,
		ListenerPort: listenerPort,
	}
	return nil
}

func mihomoRuntimeProviderName(runtimeID string) string {
	return "sub2api-" + runtimeID + "-provider"
}

func mihomoRuntimeGroupName(runtimeID string) string {
	return "sub2api-" + runtimeID + "-group"
}

func mihomoRuntimeListenerName(runtimeID string) string {
	return "sub2api-" + runtimeID
}

func buildProviderContent(req service.ProxySubscriptionRuntimeUpsertRequest) string {
	if content := strings.TrimSpace(req.ProviderContent); content != "" {
		return ensureTrailingNewline(content)
	}
	if subscription := strings.TrimSpace(req.Subscription); subscription != "" {
		return ensureTrailingNewline(subscription)
	}
	switch req.NodeType {
	case service.ProxyNodeTypeHTTP, service.ProxyNodeTypeHTTPS, service.ProxyNodeTypeSOCKS5, service.ProxyNodeTypeSOCKS5H:
		cfg := map[string]any{
			"name":   displayNameOrRuntimeID(req.DisplayName, req.RuntimeID),
			"type":   normalizeDirectNodeType(req.NodeType),
			"server": req.Server,
			"port":   req.Port,
		}
		if username := stringMapValue(req.Config, "username"); username != "" {
			cfg["username"] = username
		}
		if password := stringMapValue(req.Config, "password"); password != "" {
			cfg["password"] = password
		}
		out, _ := yaml.Marshal(map[string]any{"proxies": []map[string]any{cfg}})
		return string(out)
	default:
		if uri := strings.TrimSpace(stringMapValue(req.Config, "uri")); uri != "" {
			normalized, err := parseSubscriptionURI(uri)
			if err == nil {
				return normalized + "\n"
			}
			return uri + "\n"
		}
		nodeMap := cloneAnyMap(req.Config)
		nodeMap["name"] = displayNameOrRuntimeID(req.DisplayName, req.RuntimeID)
		nodeMap["type"] = req.NodeType
		nodeMap["server"] = req.Server
		nodeMap["port"] = req.Port
		out, _ := yaml.Marshal(map[string]any{"proxies": []map[string]any{toStringAnyMap(nodeMap)}})
		return string(out)
	}
}

func buildMihomoConfig(runtimeID, providerPath, listenerHost string, listenerPort int) ([]byte, error) {
	cfg := mihomoConfigFile{
		AllowLAN:    false,
		BindAddress: listenerHost,
		SocksPort:   listenerPort,
		Mode:        "rule",
		LogLevel:    "warning",
		IPv6:        true,
		ProxyProviders: map[string]mihomoProvider{
			"subscription-node": {
				Type: "file",
				Path: providerPath,
				HealthCheck: mihomoHealthCheck{
					Enable:   true,
					URL:      "https://chatgpt.com/cdn-cgi/trace",
					Interval: 90,
				},
			},
		},
		ProxyGroups: []mihomoProxyGroup{
			{
				Name:      "runtime-group",
				Type:      "url-test",
				Use:       []string{"subscription-node"},
				URL:       "https://chatgpt.com/cdn-cgi/trace",
				Interval:  90,
				Tolerance: 40,
			},
		},
		Rules: []string{"MATCH,runtime-group"},
	}
	return yaml.Marshal(cfg)
}

func parsePortRange(raw string) (int, int) {
	if raw == "" {
		return 21080, 21180
	}
	parts := strings.SplitN(raw, "-", 2)
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end := start
	var err2 error
	if len(parts) == 2 {
		end, err2 = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	if err1 != nil || err2 != nil || start <= 0 || end < start {
		return 21080, 21180
	}
	return start, end
}

func isTCPPortFree(host string, port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(bindCheckHost(host), strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func waitForTCP(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	address := net.JoinHostPort(dialCheckHost(host), strconv.Itoa(port))
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout waiting for listener")
	}
	return lastErr
}

func bindCheckHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "127.0.0.1"
	}
	return host
}

func dialCheckHost(host string) string {
	host = strings.TrimSpace(host)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	default:
		return host
	}
}

func normalizeDirectNodeType(nodeType string) string {
	switch strings.ToLower(strings.TrimSpace(nodeType)) {
	case service.ProxyNodeTypeHTTPS:
		return service.ProxyNodeTypeHTTP
	case service.ProxyNodeTypeSOCKS5H:
		return service.ProxyNodeTypeSOCKS5
	default:
		return strings.ToLower(strings.TrimSpace(nodeType))
	}
}

func displayNameOrRuntimeID(displayName, runtimeID string) string {
	if trimmed := strings.TrimSpace(displayName); trimmed != "" {
		return trimmed
	}
	return runtimeID
}

func stringMapValue(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	if value, ok := input[key]; ok {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func cloneAnyMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func toStringAnyMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out[key] = normalizeProviderValue(input[key])
	}
	return out
}

func normalizeProviderValue(value any) any {
	switch typed := value.(type) {
	case string, bool, int, int64, float64:
		return typed
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeProviderValue(item))
		}
		return out
	case map[string]any:
		return toStringAnyMap(typed)
	default:
		return typed
	}
}

func parseSubscriptionURI(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("empty subscription uri")
	}
	if strings.HasPrefix(trimmed, "ss://") {
		return normalizeShadowsocksURI(trimmed)
	}
	if _, err := url.Parse(trimmed); err != nil {
		return "", err
	}
	return trimmed, nil
}

func normalizeShadowsocksURI(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Host != "" {
		return raw, nil
	}
	body := strings.TrimPrefix(raw, "ss://")
	fragment := ""
	if idx := strings.Index(body, "#"); idx >= 0 {
		fragment = body[idx:]
		body = body[:idx]
	}
	query := ""
	if idx := strings.Index(body, "?"); idx >= 0 {
		query = body[idx:]
		body = body[:idx]
	}
	decoded, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(body)
		if err != nil {
			return raw, nil
		}
	}
	return "ss://" + string(decoded) + query + fragment, nil
}

func ensureTrailingNewline(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}

func resolveRuntimeDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}
	if info, err := os.Stat("/app/data"); err == nil && info.IsDir() {
		return "/app/data"
	}
	return "."
}
