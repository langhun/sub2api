package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"gopkg.in/yaml.v3"
)

type ProxySubscriptionService struct {
	sourceRepo   ProxySubscriptionSourceRepository
	nodeRepo     ProxySubscriptionNodeRepository
	proxyRepo    ProxyRepository
	settingSvc   *SettingService
	sidecar      ProxySubscriptionSidecarClient
	httpClient   *http.Client
	mu           sync.Map
	defaultHours int
}

func NewProxySubscriptionService(
	sourceRepo ProxySubscriptionSourceRepository,
	nodeRepo ProxySubscriptionNodeRepository,
	proxyRepo ProxyRepository,
	settingSvc *SettingService,
	sidecar ProxySubscriptionSidecarClient,
	cfg *config.Config,
) *ProxySubscriptionService {
	defaultHours := 6
	if cfg != nil && cfg.ProxySubscriptions.DefaultRefreshIntervalHours > 0 {
		defaultHours = cfg.ProxySubscriptions.DefaultRefreshIntervalHours
	}
	return &ProxySubscriptionService{
		sourceRepo:   sourceRepo,
		nodeRepo:     nodeRepo,
		proxyRepo:    proxyRepo,
		settingSvc:   settingSvc,
		sidecar:      sidecar,
		defaultHours: defaultHours,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (s *ProxySubscriptionService) ListSources(ctx context.Context, page, pageSize int, search string, enabled *bool) ([]ProxySubscriptionSource, int64, error) {
	return s.sourceRepo.List(ctx, page, pageSize, search, enabled)
}

func (s *ProxySubscriptionService) GetSource(ctx context.Context, id int64) (*ProxySubscriptionSource, error) {
	return s.sourceRepo.GetByID(ctx, id)
}

func (s *ProxySubscriptionService) CreateSource(ctx context.Context, input *CreateProxySubscriptionSourceInput) (*ProxySubscriptionSource, error) {
	source := &ProxySubscriptionSource{
		Name:                 strings.TrimSpace(input.Name),
		URL:                  strings.TrimSpace(input.URL),
		SourceFormat:         normalizeProxySubscriptionSourceFormat(input.SourceFormat),
		Enabled:              input.Enabled,
		RefreshIntervalHours: input.RefreshIntervalHours,
		AutoAddToPool:        input.AutoAddToPool,
	}
	if source.RefreshIntervalHours <= 0 {
		source.RefreshIntervalHours = s.defaultHours
	}
	if source.SourceFormat == "" {
		source.SourceFormat = ProxySubscriptionSourceFormatAuto
	}
	if err := s.sourceRepo.Create(ctx, source); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *ProxySubscriptionService) UpdateSource(ctx context.Context, id int64, input *UpdateProxySubscriptionSourceInput) (*ProxySubscriptionSource, error) {
	source, err := s.sourceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		source.Name = strings.TrimSpace(*input.Name)
	}
	if input.URL != nil {
		source.URL = strings.TrimSpace(*input.URL)
	}
	if input.SourceFormat != nil {
		source.SourceFormat = normalizeProxySubscriptionSourceFormat(*input.SourceFormat)
	}
	if input.Enabled != nil {
		source.Enabled = *input.Enabled
	}
	if input.RefreshIntervalHours != nil && *input.RefreshIntervalHours > 0 {
		source.RefreshIntervalHours = *input.RefreshIntervalHours
	}
	if input.AutoAddToPool != nil {
		source.AutoAddToPool = *input.AutoAddToPool
	}
	if err := s.sourceRepo.Update(ctx, source); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *ProxySubscriptionService) DeleteSource(ctx context.Context, id int64) error {
	return s.sourceRepo.Delete(ctx, id)
}

func (s *ProxySubscriptionService) ListNodes(ctx context.Context, sourceID int64) ([]ProxySubscriptionNode, error) {
	return s.nodeRepo.ListBySourceID(ctx, sourceID)
}

func (s *ProxySubscriptionService) ListMaterializedProxies(ctx context.Context, sourceID int64) ([]Proxy, error) {
	return s.proxyRepo.ListBySubscriptionSourceID(ctx, sourceID)
}

func (s *ProxySubscriptionService) RefreshSource(ctx context.Context, id int64) (*ProxySubscriptionRefreshResult, error) {
	unlock := s.lock(id)
	defer unlock()

	source, err := s.sourceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	payload, err := s.fetchSubscriptionContent(ctx, source.URL)
	if err != nil {
		source.LastError = err.Error()
		now := time.Now()
		source.LastRefreshedAt = &now
		_ = s.sourceRepo.Update(ctx, source)
		return nil, err
	}

	nodes, detectedFormat, parseErrors := parseProxySubscriptionPayload(payload, source.SourceFormat)
	if source.SourceFormat == ProxySubscriptionSourceFormatAuto {
		source.SourceFormat = detectedFormat
	}

	result := &ProxySubscriptionRefreshResult{
		SourceID:    id,
		RefreshedAt: time.Now(),
		Errors:      parseErrors,
		NodeCount:   len(nodes),
	}

	activeKeys := make([]string, 0, len(nodes))
	for i := range nodes {
		nodes[i].SourceID = id
		nodes[i].LastSeenAt = result.RefreshedAt
		activeKeys = append(activeKeys, nodes[i].NodeKey)

		existing, err := s.nodeRepo.GetBySourceAndNodeKey(ctx, id, nodes[i].NodeKey)
		if err != nil && err != sql.ErrNoRows {
			result.Errors = append(result.Errors, ProxySubscriptionError{Name: nodes[i].DisplayName, NodeKey: nodes[i].NodeKey, Message: err.Error()})
			continue
		}
		if existing != nil {
			existing.DisplayName = nodes[i].DisplayName
			existing.NodeType = nodes[i].NodeType
			existing.Server = nodes[i].Server
			existing.Port = nodes[i].Port
			existing.ConfigJSON = nodes[i].ConfigJSON
			existing.LastSeenAt = nodes[i].LastSeenAt
			existing.LandingStatus = ProxySubscriptionLandingStatusPending
			existing.LastError = ""
			if err := s.nodeRepo.Update(ctx, existing); err != nil {
				result.Errors = append(result.Errors, ProxySubscriptionError{Name: nodes[i].DisplayName, NodeKey: nodes[i].NodeKey, Message: err.Error()})
				continue
			}
			nodes[i].ID = existing.ID
			nodes[i].CreatedAt = existing.CreatedAt
			nodes[i].UpdatedAt = existing.UpdatedAt
		} else {
			nodes[i].LandingStatus = ProxySubscriptionLandingStatusPending
			if err := s.nodeRepo.Create(ctx, &nodes[i]); err != nil {
				result.Errors = append(result.Errors, ProxySubscriptionError{Name: nodes[i].DisplayName, NodeKey: nodes[i].NodeKey, Message: err.Error()})
				continue
			}
		}

		if err := s.materializeNode(ctx, source, &nodes[i], result); err != nil {
			result.Errors = append(result.Errors, ProxySubscriptionError{Name: nodes[i].DisplayName, NodeKey: nodes[i].NodeKey, Message: err.Error()})
		}
	}

	if err := s.nodeRepo.SoftDeleteMissingBySourceID(ctx, id, activeKeys, result.RefreshedAt); err != nil {
		result.Errors = append(result.Errors, ProxySubscriptionError{Message: err.Error()})
	}
	if err := s.reconcileMissingMaterializedProxies(ctx, source, activeKeys, result); err != nil {
		result.Errors = append(result.Errors, ProxySubscriptionError{Message: err.Error()})
	}

	source.LastRefreshedAt = &result.RefreshedAt
	if len(result.Errors) == 0 {
		source.LastSuccessAt = &result.RefreshedAt
		source.LastError = ""
	} else {
		source.LastError = result.Errors[len(result.Errors)-1].Message
	}
	source.LastNodeCount = result.NodeCount
	source.LastMaterializedProxyCount = result.MaterializedProxyCount
	_ = s.sourceRepo.Update(ctx, source)

	return result, nil
}

func (s *ProxySubscriptionService) fetchSubscriptionContent(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription fetch failed: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
}

func (s *ProxySubscriptionService) materializeNode(ctx context.Context, source *ProxySubscriptionSource, node *ProxySubscriptionNode, result *ProxySubscriptionRefreshResult) error {
	if isDirectProxyNode(node.NodeType) {
		return s.materializeDirectProxy(ctx, source, node, result)
	}
	return s.materializeSidecarProxy(ctx, source, node, result)
}

func (s *ProxySubscriptionService) materializeDirectProxy(ctx context.Context, source *ProxySubscriptionSource, node *ProxySubscriptionNode, result *ProxySubscriptionRefreshResult) error {
	existing, err := s.proxyRepo.FindBySubscriptionNodeID(ctx, node.ID)
	if err != nil {
		return err
	}
	name := node.DisplayName
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s-%s", source.Name, node.NodeType)
	}
	if existing == nil {
		created := &Proxy{
			Name:                  name,
			Protocol:              node.NodeType,
			Host:                  node.Server,
			Port:                  node.Port,
			Username:              stringValue(node.ConfigJSON["username"]),
			Password:              stringValue(node.ConfigJSON["password"]),
			Status:                StatusActive,
			ManagedBySubscription: true,
			SubscriptionSourceID:  &source.ID,
			SubscriptionNodeID:    &node.ID,
		}
		if err := s.proxyRepo.Create(ctx, created); err != nil {
			node.LandingStatus = ProxySubscriptionLandingStatusFailed
			node.LastError = err.Error()
			_ = s.nodeRepo.Update(ctx, node)
			return err
		}
		if source.AutoAddToPool && s.settingSvc != nil {
			_ = s.settingSvc.SetAutoFailoverProxyEnabled(ctx, created.ID, true)
		}
		result.CreatedProxyCount++
		result.MaterializedProxyCount++
		node.LandingStatus = ProxySubscriptionLandingStatusActive
		node.LastError = ""
		_ = s.nodeRepo.Update(ctx, node)
		return nil
	}

	existing.Name = name
	existing.Protocol = node.NodeType
	existing.Host = node.Server
	existing.Port = node.Port
	existing.Username = stringValue(node.ConfigJSON["username"])
	existing.Password = stringValue(node.ConfigJSON["password"])
	existing.ManagedBySubscription = true
	existing.SubscriptionSourceID = &source.ID
	existing.SubscriptionNodeID = &node.ID
	if err := s.proxyRepo.Update(ctx, existing); err != nil {
		node.LandingStatus = ProxySubscriptionLandingStatusFailed
		node.LastError = err.Error()
		_ = s.nodeRepo.Update(ctx, node)
		return err
	}
	result.UpdatedProxyCount++
	result.MaterializedProxyCount++
	node.LandingStatus = ProxySubscriptionLandingStatusActive
	node.LastError = ""
	_ = s.nodeRepo.Update(ctx, node)
	return nil
}

func (s *ProxySubscriptionService) materializeSidecarProxy(ctx context.Context, source *ProxySubscriptionSource, node *ProxySubscriptionNode, result *ProxySubscriptionRefreshResult) error {
	if s.sidecar == nil {
		node.LandingStatus = ProxySubscriptionLandingStatusUnsupported
		node.LastError = "proxy subscription sidecar is not configured"
		_ = s.nodeRepo.Update(ctx, node)
		result.UnsupportedNodeCount++
		return nil
	}
	resp, err := s.sidecar.UpsertRuntime(ctx, ProxySubscriptionSidecarUpsertRequest{
		RuntimeID:    sidecarRuntimeID(source.ID, node.NodeKey),
		NodeType:     node.NodeType,
		DisplayName:  node.DisplayName,
		Server:       node.Server,
		Port:         node.Port,
		Config:       node.ConfigJSON,
		ListenerHost: "",
	})
	if err != nil {
		node.LandingStatus = ProxySubscriptionLandingStatusFailed
		node.LastError = err.Error()
		_ = s.nodeRepo.Update(ctx, node)
		return err
	}

	existing, err := s.proxyRepo.FindBySubscriptionNodeID(ctx, node.ID)
	if err != nil {
		return err
	}
	name := node.DisplayName
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s-%s", source.Name, node.NodeType)
	}
	if existing == nil {
		created := &Proxy{
			Name:                  name,
			Protocol:              ProxyNodeTypeSOCKS5H,
			Host:                  resp.ListenerHost,
			Port:                  resp.ListenerPort,
			Status:                StatusActive,
			ManagedBySubscription: true,
			SubscriptionSourceID:  &source.ID,
			SubscriptionNodeID:    &node.ID,
		}
		if err := s.proxyRepo.Create(ctx, created); err != nil {
			node.LandingStatus = ProxySubscriptionLandingStatusFailed
			node.LastError = err.Error()
			_ = s.nodeRepo.Update(ctx, node)
			return err
		}
		if source.AutoAddToPool && s.settingSvc != nil {
			_ = s.settingSvc.SetAutoFailoverProxyEnabled(ctx, created.ID, true)
		}
		result.CreatedProxyCount++
	} else {
		existing.Name = name
		existing.Protocol = ProxyNodeTypeSOCKS5H
		existing.Host = resp.ListenerHost
		existing.Port = resp.ListenerPort
		existing.ManagedBySubscription = true
		existing.SubscriptionSourceID = &source.ID
		existing.SubscriptionNodeID = &node.ID
		if err := s.proxyRepo.Update(ctx, existing); err != nil {
			node.LandingStatus = ProxySubscriptionLandingStatusFailed
			node.LastError = err.Error()
			_ = s.nodeRepo.Update(ctx, node)
			return err
		}
		result.UpdatedProxyCount++
	}
	result.MaterializedProxyCount++
	node.LandingStatus = ProxySubscriptionLandingStatusActive
	node.LastError = ""
	_ = s.nodeRepo.Update(ctx, node)
	return nil
}

func (s *ProxySubscriptionService) reconcileMissingMaterializedProxies(ctx context.Context, source *ProxySubscriptionSource, activeNodeKeys []string, result *ProxySubscriptionRefreshResult) error {
	proxies, err := s.proxyRepo.ListBySubscriptionSourceID(ctx, source.ID)
	if err != nil {
		return err
	}
	activeSet := make(map[string]struct{}, len(activeNodeKeys))
	for _, key := range activeNodeKeys {
		activeSet[key] = struct{}{}
	}
	for i := range proxies {
		if proxies[i].SubscriptionNodeID == nil {
			continue
		}
		node, err := s.nodeRepo.GetByID(ctx, *proxies[i].SubscriptionNodeID)
		if err != nil || node == nil {
			continue
		}
		if _, ok := activeSet[node.NodeKey]; ok {
			continue
		}
		proxies[i].Status = StatusDisabled
		if err := s.proxyRepo.Update(ctx, &proxies[i]); err != nil {
			return err
		}
		if s.settingSvc != nil {
			_ = s.settingSvc.SetAutoFailoverProxyEnabled(ctx, proxies[i].ID, false)
		}
		result.DisabledProxyCount++
		count, err := s.proxyRepo.CountAccountsByProxyID(ctx, proxies[i].ID)
		if err == nil && count == 0 {
			if err := s.proxyRepo.Delete(ctx, proxies[i].ID); err == nil {
				result.DeletedProxyCount++
			}
		}
	}
	return nil
}

func (s *ProxySubscriptionService) lock(sourceID int64) func() {
	ch, _ := s.mu.LoadOrStore(sourceID, &sync.Mutex{})
	mu := ch.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func normalizeProxySubscriptionSourceFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ProxySubscriptionSourceFormatDirectList:
		return ProxySubscriptionSourceFormatDirectList
	case ProxySubscriptionSourceFormatURIList:
		return ProxySubscriptionSourceFormatURIList
	case ProxySubscriptionSourceFormatClashYAML:
		return ProxySubscriptionSourceFormatClashYAML
	default:
		return ProxySubscriptionSourceFormatAuto
	}
}

func parseProxySubscriptionPayload(payload []byte, sourceFormat string) ([]ProxySubscriptionNode, string, []ProxySubscriptionError) {
	text := strings.TrimSpace(string(payload))
	if text == "" {
		return nil, normalizeProxySubscriptionSourceFormat(sourceFormat), []ProxySubscriptionError{{Message: "empty subscription content"}}
	}

	if sourceFormat == "" || sourceFormat == ProxySubscriptionSourceFormatAuto {
		if nodes, errs := parseClashYAMLNodes(text); len(nodes) > 0 {
			return nodes, ProxySubscriptionSourceFormatClashYAML, errs
		}
		if nodes, errs := parseURIListNodes(text); len(nodes) > 0 {
			return nodes, ProxySubscriptionSourceFormatURIList, errs
		}
		nodes, errs := parseDirectProxyNodes(text)
		return nodes, ProxySubscriptionSourceFormatDirectList, errs
	}

	switch sourceFormat {
	case ProxySubscriptionSourceFormatClashYAML:
		nodes, errs := parseClashYAMLNodes(text)
		return nodes, sourceFormat, errs
	case ProxySubscriptionSourceFormatURIList:
		nodes, errs := parseURIListNodes(text)
		return nodes, sourceFormat, errs
	default:
		nodes, errs := parseDirectProxyNodes(text)
		return nodes, ProxySubscriptionSourceFormatDirectList, errs
	}
}

func parseDirectProxyNodes(text string) ([]ProxySubscriptionNode, []ProxySubscriptionError) {
	lines := splitNonEmptyLines(text)
	nodes := make([]ProxySubscriptionNode, 0, len(lines))
	errs := make([]ProxySubscriptionError, 0)
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			errs = append(errs, ProxySubscriptionError{Name: line, Message: "invalid direct proxy line"})
			continue
		}
		protocol := strings.ToLower(strings.TrimSpace(parts[0]))
		if !isDirectProxyNode(protocol) {
			errs = append(errs, ProxySubscriptionError{Name: line, Message: "unsupported direct proxy protocol"})
			continue
		}
		port, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			errs = append(errs, ProxySubscriptionError{Name: line, Message: "invalid direct proxy port"})
			continue
		}
		cfg := map[string]any{}
		if len(parts) > 3 {
			cfg["username"] = strings.TrimSpace(parts[3])
		}
		if len(parts) > 4 {
			cfg["password"] = strings.TrimSpace(parts[4])
		}
		name := ""
		if len(parts) > 5 {
			name = strings.TrimSpace(parts[5])
		}
		nodes = append(nodes, ProxySubscriptionNode{
			NodeKey:     stableNodeKey(protocol, strings.TrimSpace(parts[1]), port, cfg),
			DisplayName: name,
			NodeType:    protocol,
			Server:      strings.TrimSpace(parts[1]),
			Port:        port,
			ConfigJSON:  cfg,
		})
	}
	return dedupeNodes(nodes), errs
}

func parseURIListNodes(text string) ([]ProxySubscriptionNode, []ProxySubscriptionError) {
	decoded := text
	if maybeBase64(text) {
		if raw, err := base64.StdEncoding.DecodeString(stripAllWhitespace(text)); err == nil {
			decoded = string(raw)
		}
	}
	lines := splitNonEmptyLines(decoded)
	nodes := make([]ProxySubscriptionNode, 0, len(lines))
	errs := make([]ProxySubscriptionError, 0)
	for _, line := range lines {
		node, err := parseProxyURI(line)
		if err != nil {
			errs = append(errs, ProxySubscriptionError{Name: line, Message: err.Error()})
			continue
		}
		nodes = append(nodes, *node)
	}
	return dedupeNodes(nodes), errs
}

func parseClashYAMLNodes(text string) ([]ProxySubscriptionNode, []ProxySubscriptionError) {
	var root struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(text), &root); err != nil {
		return nil, []ProxySubscriptionError{{Message: "invalid clash yaml"}}
	}
	nodes := make([]ProxySubscriptionNode, 0, len(root.Proxies))
	errs := make([]ProxySubscriptionError, 0)
	for _, item := range root.Proxies {
		nodeType := strings.ToLower(stringValue(item["type"]))
		server := stringValue(item["server"])
		port := intValue(item["port"])
		if nodeType == "" || server == "" || port <= 0 {
			errs = append(errs, ProxySubscriptionError{Name: stringValue(item["name"]), Message: "invalid clash node"})
			continue
		}
		cfg := cloneMap(item)
		name := stringValue(item["name"])
		nodes = append(nodes, ProxySubscriptionNode{
			NodeKey:     stableNodeKey(nodeType, server, port, cfg),
			DisplayName: name,
			NodeType:    nodeType,
			Server:      server,
			Port:        port,
			ConfigJSON:  cfg,
		})
	}
	return dedupeNodes(nodes), errs
}

func parseProxyURI(raw string) (*ProxySubscriptionNode, error) {
	trimmed := strings.TrimSpace(raw)
	u, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy uri")
	}
	switch strings.ToLower(u.Scheme) {
	case ProxyNodeTypeHTTP, ProxyNodeTypeHTTPS, ProxyNodeTypeSOCKS5, ProxyNodeTypeSOCKS5H:
		port, _ := strconv.Atoi(u.Port())
		cfg := map[string]any{}
		if u.User != nil {
			cfg["username"] = u.User.Username()
			if pwd, ok := u.User.Password(); ok {
				cfg["password"] = pwd
			}
		}
		return &ProxySubscriptionNode{
			NodeKey:     stableNodeKey(u.Scheme, u.Hostname(), port, cfg),
			DisplayName: u.Fragment,
			NodeType:    strings.ToLower(u.Scheme),
			Server:      u.Hostname(),
			Port:        port,
			ConfigJSON:  cfg,
		}, nil
	case ProxyNodeTypeSS, ProxyNodeTypeTrojan, ProxyNodeTypeVMess, ProxyNodeTypeVLess, ProxyNodeTypeHysteria, ProxyNodeTypeHysteria2:
		port, _ := strconv.Atoi(u.Port())
		cfg := map[string]any{
			"uri": trimmed,
		}
		return &ProxySubscriptionNode{
			NodeKey:     stableNodeKey(strings.ToLower(u.Scheme), u.Hostname(), port, cfg),
			DisplayName: u.Fragment,
			NodeType:    strings.ToLower(u.Scheme),
			Server:      u.Hostname(),
			Port:        port,
			ConfigJSON:  cfg,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported uri scheme")
	}
}

func stableNodeKey(nodeType, server string, port int, cfg map[string]any) string {
	payload, _ := json.Marshal(cfg)
	sum := sha256.Sum256([]byte(strings.ToLower(nodeType) + "|" + strings.ToLower(server) + "|" + strconv.Itoa(port) + "|" + string(payload)))
	return hex.EncodeToString(sum[:])
}

func splitNonEmptyLines(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func stripAllWhitespace(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if r != '\n' && r != '\r' && r != '\t' && r != ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func maybeBase64(text string) bool {
	trimmed := stripAllWhitespace(text)
	if len(trimmed) < 16 || len(trimmed)%4 != 0 {
		return false
	}
	for _, r := range trimmed {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '+' || r == '/' || r == '=') {
			return false
		}
	}
	return true
}

func dedupeNodes(nodes []ProxySubscriptionNode) []ProxySubscriptionNode {
	seen := make(map[string]struct{}, len(nodes))
	out := make([]ProxySubscriptionNode, 0, len(nodes))
	for _, item := range nodes {
		if _, ok := seen[item.NodeKey]; ok {
			continue
		}
		seen[item.NodeKey] = struct{}{}
		out = append(out, item)
	}
	return out
}

func isDirectProxyNode(nodeType string) bool {
	switch nodeType {
	case ProxyNodeTypeHTTP, ProxyNodeTypeHTTPS, ProxyNodeTypeSOCKS5, ProxyNodeTypeSOCKS5H:
		return true
	default:
		return false
	}
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func intValue(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		out, _ := strconv.Atoi(strings.TrimSpace(n))
		return out
	default:
		return 0
	}
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sidecarRuntimeID(sourceID int64, nodeKey string) string {
	return fmt.Sprintf("src-%d-%s", sourceID, nodeKey[:12])
}
