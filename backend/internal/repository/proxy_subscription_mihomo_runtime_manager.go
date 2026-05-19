package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"gopkg.in/yaml.v3"
)

type proxySubscriptionMihomoRuntimeManager struct {
	cfg         config.ProxySubscriptionMihomoConfig
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
	Command      *exec.Cmd
	WaitCh       chan error
	Stopped      bool
}

type mihomoConfigFile struct {
	AllowLAN       bool                      `yaml:"allow-lan"`
	BindAddress    string                    `yaml:"bind-address,omitempty"`
	SocksPort      int                       `yaml:"socks-port"`
	Mode           string                    `yaml:"mode"`
	LogLevel       string                    `yaml:"log-level"`
	IPv6           bool                      `yaml:"ipv6"`
	ProxyProviders map[string]mihomoProvider `yaml:"proxy-providers,omitempty"`
	ProxyGroups    []mihomoProxyGroup        `yaml:"proxy-groups"`
	Rules          []string                  `yaml:"rules"`
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

func NewProxySubscriptionMihomoRuntimeManager(cfg *config.Config) service.ProxySubscriptionRuntimeManager {
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

func (m *proxySubscriptionMihomoRuntimeManager) Start(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.listenerDir, 0o755); err != nil {
		return err
	}
	if m.initialized {
		return nil
	}
	if err := m.rehydrateExistingRuntimesLocked(); err != nil {
		return err
	}
	m.initialized = true
	return nil
}

func (m *proxySubscriptionMihomoRuntimeManager) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []string
	for runtimeID, state := range m.runtimes {
		if err := m.stopRuntimeLocked(state); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", runtimeID, err))
		}
	}
	m.runtimes = make(map[string]*mihomoRuntimeState)
	m.initialized = false
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
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
	portKey := req.RuntimeID
	if current != nil {
		portKey = req.RuntimeID + "-candidate"
	}
	listenerPort, err := m.allocatePortLocked(portKey, preferredPort)
	if err != nil {
		return nil, err
	}

	providerPath := filepath.Join(m.listenerDir, req.RuntimeID+".provider.yaml")
	configPath := filepath.Join(m.listenerDir, req.RuntimeID+".yaml")
	if err := os.WriteFile(providerPath, []byte(buildProviderContent(req)), 0o644); err != nil {
		return nil, err
	}
	cfgBytes, err := buildMihomoConfig(req.RuntimeID, providerPath, listenerHost, listenerPort)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(configPath, cfgBytes, 0o644); err != nil {
		return nil, err
	}

	cmd, waitCh, err := m.startMihomoCommand(configPath)
	if err != nil {
		return nil, err
	}

	state := &mihomoRuntimeState{
		RuntimeID:    req.RuntimeID,
		ConfigPath:   configPath,
		ProviderPath: providerPath,
		ListenerHost: listenerHost,
		ListenerPort: listenerPort,
		Command:      cmd,
		WaitCh:       waitCh,
	}

	if err := waitForTCP(listenerHost, listenerPort, 12*time.Second); err != nil {
		_ = m.stopRuntimeLocked(state)
		return nil, fmt.Errorf("mihomo listener not ready: %w", err)
	}
	if current != nil {
		if err := m.stopRuntimeLocked(current); err != nil {
			_ = m.stopRuntimeLocked(state)
			return nil, err
		}
	}
	m.runtimes[req.RuntimeID] = state

	return &service.ProxySubscriptionRuntimeUpsertResponse{
		RuntimeID:    req.RuntimeID,
		ListenerHost: listenerHost,
		ListenerPort: listenerPort,
		Protocol:     service.ProxyNodeTypeSOCKS5H,
	}, nil
}

func (m *proxySubscriptionMihomoRuntimeManager) DeleteRuntime(_ context.Context, runtimeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.runtimes[runtimeID]
	if state != nil {
		if err := m.stopRuntimeLocked(state); err != nil {
			return err
		}
		delete(m.runtimes, runtimeID)
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

func (m *proxySubscriptionMihomoRuntimeManager) stopRuntimeLocked(state *mihomoRuntimeState) error {
	if state == nil || state.Stopped || state.Command == nil || state.Command.Process == nil {
		if state != nil {
			state.Stopped = true
			state.Command = nil
			state.WaitCh = nil
		}
		return nil
	}
	_ = state.Command.Process.Signal(syscall.SIGTERM)
	select {
	case err := <-state.WaitCh:
		if err != nil && !strings.Contains(err.Error(), "exit status") {
			return err
		}
	case <-time.After(5 * time.Second):
		if err := state.Command.Process.Kill(); err != nil {
			return err
		}
		err := <-state.WaitCh
		if err != nil && !strings.Contains(err.Error(), "exit status") {
			return err
		}
	}
	state.Stopped = true
	state.Command = nil
	state.WaitCh = nil
	return nil
}

func (m *proxySubscriptionMihomoRuntimeManager) startMihomoCommand(configPath string) (*exec.Cmd, chan error, error) {
	cmd := exec.Command(m.resolveBinary(), "-d", m.dataDir, "-f", configPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = buildRuntimeSysProcAttr()
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start mihomo: %w", err)
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	return cmd, waitCh, nil
}

func (m *proxySubscriptionMihomoRuntimeManager) allocatePortLocked(runtimeID string, preferredPort int) (int, error) {
	if current := m.runtimes[runtimeID]; current != nil && current.ListenerPort > 0 {
		return current.ListenerPort, nil
	}
	if preferredPort > 0 && isTCPPortFree(preferredPort) {
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
		if isTCPPortFree(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free mihomo listener ports available in %d-%d", start, end)
}

func (m *proxySubscriptionMihomoRuntimeManager) rehydrateExistingRuntimesLocked() error {
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
		if err := m.startPersistedRuntimeLocked(runtimeID, configPath, providerPath); err != nil {
			continue
		}
	}
	return nil
}

func (m *proxySubscriptionMihomoRuntimeManager) startPersistedRuntimeLocked(runtimeID, configPath, providerPath string) error {
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

	cmd, waitCh, err := m.startMihomoCommand(configPath)
	if err != nil {
		return fmt.Errorf("start persisted mihomo: %w", err)
	}

	state := &mihomoRuntimeState{
		RuntimeID:    runtimeID,
		ConfigPath:   configPath,
		ProviderPath: providerPath,
		ListenerHost: listenerHost,
		ListenerPort: listenerPort,
		Command:      cmd,
		WaitCh:       waitCh,
	}
	if err := waitForTCP(listenerHost, listenerPort, 12*time.Second); err != nil {
		_ = m.stopRuntimeLocked(state)
		return fmt.Errorf("persisted mihomo listener not ready: %w", err)
	}
	m.runtimes[runtimeID] = state
	return nil
}

func (m *proxySubscriptionMihomoRuntimeManager) resolveBinary() string {
	if strings.TrimSpace(m.cfg.MihomoBin) != "" {
		return strings.TrimSpace(m.cfg.MihomoBin)
	}
	return "mihomo"
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

func isTCPPortFree(port int) bool {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func waitForTCP(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	address := net.JoinHostPort(host, strconv.Itoa(port))
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

func buildRuntimeSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
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
