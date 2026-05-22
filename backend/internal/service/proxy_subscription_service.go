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
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"gopkg.in/yaml.v3"
)

type ProxySubscriptionService struct {
	sourceRepo   ProxySubscriptionSourceRepository
	nodeRepo     ProxySubscriptionNodeRepository
	proxyRepo    ProxyRepository
	accountRepo  proxySubscriptionAccountMover
	settingSvc   *SettingService
	runtime      ProxySubscriptionRuntimeManager
	proxyProber  ProxyExitInfoProber
	openAIProbe  func(context.Context, string) (bool, error)
	httpClient   *http.Client
	mu           sync.Map
	defaultHours int
}

func NewProxySubscriptionService(
	sourceRepo ProxySubscriptionSourceRepository,
	nodeRepo ProxySubscriptionNodeRepository,
	proxyRepo ProxyRepository,
	accountRepo proxySubscriptionAccountMover,
	settingSvc *SettingService,
	runtime ProxySubscriptionRuntimeManager,
	cfg *config.Config,
) *ProxySubscriptionService {
	defaultHours := 6
	if cfg != nil && cfg.ProxySubscriptions.DefaultRefreshIntervalHours > 0 {
		defaultHours = cfg.ProxySubscriptions.DefaultRefreshIntervalHours
	}
	httpClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:               45 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	})
	if err != nil {
		httpClient = &http.Client{
			Timeout: 45 * time.Second,
		}
	}
	return &ProxySubscriptionService{
		sourceRepo:   sourceRepo,
		nodeRepo:     nodeRepo,
		proxyRepo:    proxyRepo,
		accountRepo:  accountRepo,
		settingSvc:   settingSvc,
		runtime:      runtime,
		openAIProbe:  probeOpenAIReachable,
		defaultHours: defaultHours,
		httpClient:   httpClient,
	}
}

func (s *ProxySubscriptionService) SetProxyProber(prober ProxyExitInfoProber) {
	if s == nil {
		return
	}
	s.proxyProber = prober
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
		TargetEntryCount:     input.TargetEntryCount,
		AutoAddToPool:        input.AutoAddToPool,
	}
	if source.RefreshIntervalHours <= 0 {
		source.RefreshIntervalHours = s.defaultHours
	}
	if source.TargetEntryCount <= 0 {
		source.TargetEntryCount = 3
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
	if input.TargetEntryCount != nil && *input.TargetEntryCount > 0 {
		source.TargetEntryCount = *input.TargetEntryCount
	}
	if input.AutoAddToPool != nil {
		source.AutoAddToPool = *input.AutoAddToPool
	}
	if err := s.sourceRepo.Update(ctx, source); err != nil {
		return nil, err
	}
	if _, err := s.RefreshSource(ctx, id); err != nil {
		return source, err
	}
	return s.sourceRepo.GetByID(ctx, id)
}

func (s *ProxySubscriptionService) DeleteSource(ctx context.Context, id int64) error {
	proxies, err := s.proxyRepo.ListBySubscriptionSourceID(ctx, id)
	if err != nil {
		return err
	}
	if len(proxies) > 0 {
		if err := s.deleteManagedRuntime(ctx, id); err != nil {
			return err
		}
	}
	return s.sourceRepo.Delete(ctx, id)
}

func (s *ProxySubscriptionService) ListNodes(ctx context.Context, sourceID int64) ([]ProxySubscriptionNode, error) {
	return s.nodeRepo.ListBySourceID(ctx, sourceID)
}

func (s *ProxySubscriptionService) ListMaterializedProxies(ctx context.Context, sourceID int64) ([]Proxy, error) {
	return s.proxyRepo.ListBySubscriptionSourceID(ctx, sourceID)
}

func (s *ProxySubscriptionService) RefreshSource(ctx context.Context, id int64) (*ProxySubscriptionRefreshResult, error) {
	slog.Debug("proxy_subscription.refresh_source_start",
		"source_id", id)

	unlock := s.lock(id)
	defer unlock()

	source, err := s.sourceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	payload, err := s.fetchSubscriptionContent(ctx, source.URL)
	if err != nil {
		slog.Error("proxy_subscription.fetch_failed",
			"source_id", id,
			"source_url", source.URL,
			"error", err)
		source.LastError = err.Error()
		now := time.Now()
		source.LastRefreshedAt = &now
		_ = s.sourceRepo.Update(ctx, source)
		return nil, err
	}

	nodes, detectedFormat, parseErrors := parseProxySubscriptionPayload(payload, source.SourceFormat)
	slog.Debug("proxy_subscription.payload_parsed",
		"source_id", id,
		"detected_format", detectedFormat,
		"node_count", len(nodes),
		"parse_error_count", len(parseErrors))

	if source.SourceFormat == ProxySubscriptionSourceFormatAuto {
		source.SourceFormat = detectedFormat
	}
	runtimeNodes, filterErrors := filterRuntimeCandidateSubscriptionNodes(nodes)
	parseErrors = append(parseErrors, filterErrors...)

	slog.Debug("proxy_subscription.nodes_filtered",
		"source_id", id,
		"runtime_node_count", len(runtimeNodes),
		"filter_error_count", len(filterErrors))

	result := &ProxySubscriptionRefreshResult{
		SourceID:    id,
		RefreshedAt: time.Now(),
		Errors:      parseErrors,
		NodeCount:   len(runtimeNodes),
	}

	plan := s.selectRuntimeEntryNodesForMaterialization(ctx, source, runtimeNodes, source.TargetEntryCount)
	selectedRuntimeNodes := plan.selected
	if len(selectedRuntimeNodes) == 0 {
		result.Errors = append(result.Errors, plan.errors...)
	} else {
		result.SkippedNodeCount += len(plan.errors)
	}
	if len(runtimeNodes) > 0 && len(selectedRuntimeNodes) == 0 {
		if fallbackCount := s.countHealthyExistingRuntimeEntries(ctx, source); fallbackCount >= expectedRuntimeEntryCount(source) {
			result.MaterializedProxyCount = fallbackCount
			source.LastRefreshedAt = &result.RefreshedAt
			source.LastSuccessAt = &result.RefreshedAt
			source.LastError = ""
			source.LastNodeCount = result.NodeCount
			source.LastMaterializedProxyCount = fallbackCount
			_ = s.sourceRepo.Update(ctx, source)
			return result, nil
		}
		source.LastError = "no GPT-compatible subscription nodes available after runtime probe"
		now := time.Now()
		source.LastRefreshedAt = &now
		_ = s.sourceRepo.Update(ctx, source)
		return result, fmt.Errorf("%s", source.LastError)
	}
	expectedRuntimeNames := make(map[string]struct{}, len(selectedRuntimeNodes))
	expectedRuntimeIDs := make(map[string]struct{}, len(selectedRuntimeNodes))
	for idx := range selectedRuntimeNodes {
		expectedRuntimeNames[fmt.Sprintf("%s 自动选路入口 #%d", source.Name, idx+1)] = struct{}{}
		expectedRuntimeIDs[runtimeIDForSource(source.ID, idx+1)] = struct{}{}
	}
	commitAllowed := true
	activeKeys := make([]string, 0, len(runtimeNodes))
	for i := range runtimeNodes {
		runtimeNodes[i].SourceID = id
		runtimeNodes[i].LastSeenAt = result.RefreshedAt
		activeKeys = append(activeKeys, runtimeNodes[i].NodeKey)

		existing, err := s.nodeRepo.GetBySourceAndNodeKey(ctx, id, runtimeNodes[i].NodeKey)
		if err != nil && err != sql.ErrNoRows {
			commitAllowed = false
			result.Errors = append(result.Errors, ProxySubscriptionError{Name: runtimeNodes[i].DisplayName, NodeKey: runtimeNodes[i].NodeKey, Message: err.Error()})
			continue
		}
		if existing != nil {
			existing.DisplayName = runtimeNodes[i].DisplayName
			existing.NodeType = runtimeNodes[i].NodeType
			existing.Server = runtimeNodes[i].Server
			existing.Port = runtimeNodes[i].Port
			existing.ConfigJSON = runtimeNodes[i].ConfigJSON
			existing.LastSeenAt = runtimeNodes[i].LastSeenAt
			existing.LandingStatus = ProxySubscriptionLandingStatusPending
			existing.LastError = ""
			if err := s.nodeRepo.Update(ctx, existing); err != nil {
				commitAllowed = false
				result.Errors = append(result.Errors, ProxySubscriptionError{Name: runtimeNodes[i].DisplayName, NodeKey: runtimeNodes[i].NodeKey, Message: err.Error()})
				continue
			}
			runtimeNodes[i].ID = existing.ID
			runtimeNodes[i].CreatedAt = existing.CreatedAt
			runtimeNodes[i].UpdatedAt = existing.UpdatedAt
		} else {
			runtimeNodes[i].LandingStatus = ProxySubscriptionLandingStatusPending
			if err := s.nodeRepo.Create(ctx, &runtimeNodes[i]); err != nil {
				commitAllowed = false
				result.Errors = append(result.Errors, ProxySubscriptionError{Name: runtimeNodes[i].DisplayName, NodeKey: runtimeNodes[i].NodeKey, Message: err.Error()})
				continue
			}
		}

		if isDirectProxyNode(runtimeNodes[i].NodeType) {
			if err := s.materializeDirectProxy(ctx, source, &runtimeNodes[i], result); err != nil {
				commitAllowed = false
				result.Errors = append(result.Errors, ProxySubscriptionError{Name: runtimeNodes[i].DisplayName, NodeKey: runtimeNodes[i].NodeKey, Message: err.Error()})
			}
			continue
		}
		entryIndex := runtimeEntryIndex(selectedRuntimeNodes, runtimeNodes[i].NodeKey)
		if entryIndex < 0 {
			runtimeNodes[i].LandingStatus = ProxySubscriptionLandingStatusActive
			runtimeNodes[i].LastError = ""
			_ = s.nodeRepo.Update(ctx, &runtimeNodes[i])
			continue
		}
		groupNodes := plan.groups[entryIndex]
		if len(groupNodes) == 0 {
			groupNodes = []ProxySubscriptionNode{runtimeNodes[i]}
		}
		providerContent := buildRuntimeCandidateProviderContent(groupNodes)
		if err := s.materializeRuntimeProxy(ctx, source, payload, providerContent, entryIndex+1, &runtimeNodes[i], result); err != nil {
			commitAllowed = false
			result.Errors = append(result.Errors, ProxySubscriptionError{Name: runtimeNodes[i].DisplayName, NodeKey: runtimeNodes[i].NodeKey, Message: err.Error()})
		}
	}

	if !commitAllowed {
		source.LastRefreshedAt = &result.RefreshedAt
		source.LastError = result.Errors[len(result.Errors)-1].Message
		source.LastNodeCount = result.NodeCount
		source.LastMaterializedProxyCount = result.MaterializedProxyCount
		_ = s.sourceRepo.Update(ctx, source)
		return result, fmt.Errorf("%s", source.LastError)
	}

	if err := s.nodeRepo.SoftDeleteMissingBySourceID(ctx, id, activeKeys, result.RefreshedAt); err != nil {
		result.Errors = append(result.Errors, ProxySubscriptionError{Message: err.Error()})
	}
	if err := s.reconcileMissingMaterializedProxies(ctx, source, activeKeys, expectedRuntimeNames, expectedRuntimeIDs, result); err != nil {
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
			Username:              getStringValue(node.ConfigJSON["username"]),
			Password:              getStringValue(node.ConfigJSON["password"]),
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
	existing.Username = getStringValue(node.ConfigJSON["username"])
	existing.Password = getStringValue(node.ConfigJSON["password"])
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

func (s *ProxySubscriptionService) materializeRuntimeProxy(ctx context.Context, source *ProxySubscriptionSource, payload []byte, providerContent string, entryIndex int, node *ProxySubscriptionNode, result *ProxySubscriptionRefreshResult) error {
	slog.Debug("proxy_subscription.materialize_runtime_proxy_start",
		"source_id", source.ID,
		"entry_index", entryIndex,
		"node_key", node.NodeKey,
		"node_type", node.NodeType,
		"display_name", node.DisplayName)

	if s.runtime == nil {
		slog.Warn("proxy_subscription.runtime_not_configured",
			"source_id", source.ID,
			"node_key", node.NodeKey)
		node.LandingStatus = ProxySubscriptionLandingStatusUnsupported
		node.LastError = "proxy subscription mihomo runtime is not configured"
		_ = s.nodeRepo.Update(ctx, node)
		result.UnsupportedNodeCount++
		return nil
	}
	proxies, err := s.proxyRepo.ListBySubscriptionSourceID(ctx, source.ID)
	if err != nil {
		return err
	}
	var existing *Proxy
	preferredPort := 0
	for i := range proxies {
		if !proxies[i].ManagedBySubscription || proxies[i].Protocol != ProxyNodeTypeSOCKS5H {
			continue
		}
		if strings.Contains(proxies[i].Name, fmt.Sprintf("#%d", entryIndex)) || (entryIndex == 1 && !strings.Contains(proxies[i].Name, "#")) {
			existing = &proxies[i]
			preferredPort = proxies[i].Port
			break
		}
	}
	resp, err := s.runtime.UpsertRuntime(ctx, ProxySubscriptionRuntimeUpsertRequest{
		RuntimeID:       runtimeIDForSource(source.ID, entryIndex),
		SourceName:      source.Name,
		SourceFormat:    source.SourceFormat,
		Subscription:    string(payload),
		ProviderContent: providerContent,
		EntryIndex:      entryIndex,
		NodeType:        node.NodeType,
		DisplayName:     node.DisplayName,
		Server:          node.Server,
		Port:            node.Port,
		Config:          node.ConfigJSON,
		ListenerHost:    "",
		ListenerPort:    preferredPort,
	})
	if err != nil {
		slog.Error("proxy_subscription.runtime_upsert_failed",
			"source_id", source.ID,
			"runtime_id", runtimeIDForSource(source.ID, entryIndex),
			"entry_index", entryIndex,
			"node_key", node.NodeKey,
			"error", err)
		node.LandingStatus = ProxySubscriptionLandingStatusFailed
		node.LastError = err.Error()
		_ = s.nodeRepo.Update(ctx, node)
		return err
	}

	slog.Debug("proxy_subscription.runtime_upsert_success",
		"source_id", source.ID,
		"runtime_id", runtimeIDForSource(source.ID, entryIndex),
		"listener_host", resp.ListenerHost,
		"listener_port", resp.ListenerPort)
	name := node.DisplayName
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%s 自动选路入口 #%d", source.Name, entryIndex)
	} else {
		name = fmt.Sprintf("%s 自动选路入口 #%d", source.Name, entryIndex)
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
		existing.SubscriptionNodeID = nil
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

func (s *ProxySubscriptionService) reconcileMissingMaterializedProxies(ctx context.Context, source *ProxySubscriptionSource, activeNodeKeys []string, expectedRuntimeNames map[string]struct{}, expectedRuntimeIDs map[string]struct{}, result *ProxySubscriptionRefreshResult) error {
	proxies, err := s.proxyRepo.ListBySubscriptionSourceID(ctx, source.ID)
	if err != nil {
		return err
	}
	activeManaged := make([]Proxy, 0, len(proxies))
	for i := range proxies {
		if proxies[i].ManagedBySubscription && proxies[i].Protocol == ProxyNodeTypeSOCKS5H {
			if runtimeID, ok := runtimeIDFromProxyName(source.ID, source.Name, proxies[i].Name); ok {
				if _, expected := expectedRuntimeIDs[runtimeID]; expected {
					activeManaged = append(activeManaged, proxies[i])
				}
			}
		}
	}
	activeSet := make(map[string]struct{}, len(activeNodeKeys))
	for _, key := range activeNodeKeys {
		activeSet[key] = struct{}{}
	}
	for i := range proxies {
		if proxies[i].SubscriptionNodeID != nil {
			node, err := s.nodeRepo.GetByID(ctx, *proxies[i].SubscriptionNodeID)
			if err == nil && node != nil {
				if _, ok := activeSet[node.NodeKey]; ok {
					continue
				}
			}
		} else if proxies[i].ManagedBySubscription {
			if proxies[i].Protocol == ProxyNodeTypeSOCKS5H {
				if runtimeID, ok := runtimeIDFromProxyName(source.ID, source.Name, proxies[i].Name); ok {
					if _, expected := expectedRuntimeIDs[runtimeID]; expected {
						continue
					}
				}
				if _, ok := expectedRuntimeNames[proxies[i].Name]; ok {
					continue
				}
			}
		}
		if proxies[i].ManagedBySubscription && proxies[i].Protocol == ProxyNodeTypeSOCKS5H {
			if runtimeID, ok := runtimeIDFromProxyName(source.ID, source.Name, proxies[i].Name); ok {
				if err := s.deleteManagedRuntimeByID(ctx, runtimeID); err != nil {
					return err
				}
			}
		}
		proxies[i].Status = StatusDisabled
		if err := s.proxyRepo.Update(ctx, &proxies[i]); err != nil {
			return err
		}
		if err := s.reassignAccountsFromManagedProxy(ctx, proxies[i].ID, activeManaged); err != nil {
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

func (s *ProxySubscriptionService) reassignAccountsFromManagedProxy(ctx context.Context, oldProxyID int64, candidates []Proxy) error {
	if s == nil || s.accountRepo == nil || oldProxyID <= 0 || len(candidates) == 0 {
		return nil
	}
	summaries, err := s.proxyRepo.ListAccountSummariesByProxyID(ctx, oldProxyID)
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		return nil
	}
	loads := make(map[int64]int64, len(candidates))
	for _, candidate := range candidates {
		count, err := s.proxyRepo.CountAccountsByProxyID(ctx, candidate.ID)
		if err != nil {
			return err
		}
		loads[candidate.ID] = count
	}
	assignments := make(map[int64][]int64, len(candidates))
	for _, account := range summaries {
		target := pickManagedProxyMigrationTarget(candidates, loads)
		assignments[target] = append(assignments[target], account.ID)
		loads[target]++
	}
	for targetID, accountIDs := range assignments {
		target := targetID
		if _, err := s.accountRepo.BulkUpdate(ctx, accountIDs, AccountBulkUpdate{ProxyID: &target}); err != nil {
			return err
		}
	}
	return nil
}

func pickManagedProxyMigrationTarget(candidates []Proxy, loads map[int64]int64) int64 {
	bestID := candidates[0].ID
	bestLoad := loads[bestID]
	for _, candidate := range candidates[1:] {
		load := loads[candidate.ID]
		if load < bestLoad || (load == bestLoad && candidate.ID < bestID) {
			bestID = candidate.ID
			bestLoad = load
		}
	}
	return bestID
}

func (s *ProxySubscriptionService) deleteManagedRuntime(ctx context.Context, sourceID int64) error {
	if s.runtime == nil {
		return nil
	}
	for idx := 1; idx <= 10; idx++ {
		if err := s.deleteManagedRuntimeByID(ctx, runtimeIDForSource(sourceID, idx)); err != nil {
			return err
		}
	}
	return nil
}

func (s *ProxySubscriptionService) deleteManagedRuntimeByID(ctx context.Context, runtimeID string) error {
	if s.runtime == nil || strings.TrimSpace(runtimeID) == "" {
		return nil
	}
	return s.runtime.DeleteRuntime(ctx, runtimeID)
}

func (s *ProxySubscriptionService) lock(sourceID int64) func() {
	ch, _ := s.mu.LoadOrStore(sourceID, &sync.Mutex{})
	mu, ok := ch.(*sync.Mutex)
	if !ok {
		panic(fmt.Sprintf("proxy subscription lock type mismatch for source %d", sourceID))
	}
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
		nodeType := strings.ToLower(getStringValue(item["type"]))
		server := getStringValue(item["server"])
		port := getIntValue(item["port"])
		if nodeType == "" || server == "" || port <= 0 {
			errs = append(errs, ProxySubscriptionError{Name: getStringValue(item["name"]), Message: "invalid clash node"})
			continue
		}
		cfg := cloneMap(item)
		name := getStringValue(item["name"])
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

func filterRuntimeCandidateSubscriptionNodes(nodes []ProxySubscriptionNode) ([]ProxySubscriptionNode, []ProxySubscriptionError) {
	if len(nodes) == 0 {
		return nodes, nil
	}
	filtered := make([]ProxySubscriptionNode, 0, len(nodes))
	errs := make([]ProxySubscriptionError, 0)
	for _, node := range nodes {
		name := strings.TrimSpace(node.DisplayName)
		if isDirectProxyNode(node.NodeType) {
			if isNonNodeAnnouncementName(name) {
				errs = append(errs, ProxySubscriptionError{
					NodeKey: node.NodeKey,
					Name:    name,
					Message: "node skipped: unsupported or non-node entry for OpenAI/GPT routing",
				})
				continue
			}
			filtered = append(filtered, node)
			continue
		}
		if shouldSkipSubscriptionNodeByName(name) {
			errs = append(errs, ProxySubscriptionError{
				NodeKey: node.NodeKey,
				Name:    name,
				Message: "node skipped: unsupported or non-node entry for OpenAI/GPT routing",
			})
			continue
		}
		filtered = append(filtered, node)
	}
	return filtered, errs
}

func shouldSkipSubscriptionNodeByName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return false
	}
	return isNonNodeAnnouncementName(normalized)
}

func isNonNodeAnnouncementName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return false
	}
	nonNodeHints := []string{
		"剩余流量",
		"套餐到期",
		"永久官网",
		"官网",
		"tg频道",
		"流量",
		"到期",
	}
	for _, hint := range nonNodeHints {
		if strings.Contains(normalized, strings.ToLower(hint)) {
			return true
		}
	}
	return false
}

func buildRuntimeCandidateProviderContent(nodes []ProxySubscriptionNode) string {
	if len(nodes) == 0 {
		return "proxies: []\n"
	}
	items := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		entry := cloneMap(node.ConfigJSON)
		entry["name"] = node.DisplayName
		entry["type"] = node.NodeType
		entry["server"] = node.Server
		entry["port"] = node.Port
		items = append(items, entry)
	}
	raw, _ := yaml.Marshal(map[string]any{"proxies": items})
	return string(raw)
}

type runtimeMaterializationPlan struct {
	selected []ProxySubscriptionNode
	groups   map[int][]ProxySubscriptionNode
	errors   []ProxySubscriptionError
}

func (s *ProxySubscriptionService) selectRuntimeEntryNodesForMaterialization(ctx context.Context, source *ProxySubscriptionSource, nodes []ProxySubscriptionNode, targetCount int) runtimeMaterializationPlan {
	fallback := selectRuntimeEntryNodes(nodes, targetCount)
	if len(nodes) == 0 || targetCount <= 0 || s.runtime == nil || source == nil {
		return runtimeMaterializationPlan{
			selected: fallback,
			groups:   buildRuntimeEntryGroups(fallback, nodes),
		}
	}
	return runtimeMaterializationPlan{
		selected: fallback,
		groups:   buildRuntimeEntryGroups(fallback, nodes),
	}
}

func probeOpenAIReachable(ctx context.Context, proxyURL string) (bool, error) {
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:              proxyURL,
		Timeout:               8 * time.Second,
		ResponseHeaderTimeout: 6 * time.Second,
	})
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", proxyQualityClientUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return true, nil
	default:
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return true, nil
		}
		return false, nil
	}
}

func buildRuntimeEntryGroups(selectedNodes []ProxySubscriptionNode, candidateNodes []ProxySubscriptionNode) map[int][]ProxySubscriptionNode {
	groups := make(map[int][]ProxySubscriptionNode)
	if len(selectedNodes) == 0 || len(candidateNodes) == 0 {
		return groups
	}
	groupCount := len(selectedNodes)
	seen := make([]map[string]struct{}, groupCount)
	selectedKeys := make(map[string]struct{}, len(selectedNodes))
	for idx := 0; idx < groupCount; idx++ {
		seen[idx] = map[string]struct{}{}
		groups[idx] = []ProxySubscriptionNode{selectedNodes[idx]}
		seen[idx][selectedNodes[idx].NodeKey] = struct{}{}
		selectedKeys[selectedNodes[idx].NodeKey] = struct{}{}
	}
	extraIndex := 0
	for _, node := range candidateNodes {
		if _, ok := selectedKeys[node.NodeKey]; ok {
			continue
		}
		groupIndex := extraIndex % groupCount
		extraIndex++
		if _, ok := seen[groupIndex][node.NodeKey]; ok {
			continue
		}
		groups[groupIndex] = append(groups[groupIndex], node)
		seen[groupIndex][node.NodeKey] = struct{}{}
	}
	return groups
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
			_, _ = b.WriteRune(r)
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
		if (r < 'A' || r > 'Z') &&
			(r < 'a' || r > 'z') &&
			(r < '0' || r > '9') &&
			r != '+' && r != '/' && r != '=' {
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

// getStringValue extracts a string value from an any type.
// Returns empty string if the value is not a string.
func getStringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// getIntValue extracts an integer value from an any type.
// Supports conversion from int, int64, float64, and string types.
// Returns 0 if the value cannot be converted to an integer.
func getIntValue(v any) int {
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

func runtimeIDForSource(sourceID int64, entryIndex int) string {
	return fmt.Sprintf("src-%d-subscription-%d", sourceID, entryIndex)
}

func selectRuntimeEntryNodes(nodes []ProxySubscriptionNode, targetCount int) []ProxySubscriptionNode {
	if targetCount <= 0 {
		targetCount = 1
	}
	if len(nodes) == 0 {
		return nil
	}
	selected := make([]ProxySubscriptionNode, 0, targetCount)
	seen := make(map[string]struct{}, targetCount)
	for _, node := range nodes {
		if len(selected) >= targetCount {
			break
		}
		key := strings.ToLower(strings.TrimSpace(node.DisplayName))
		if key == "" {
			key = node.NodeKey
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, node)
	}
	return selected
}

func runtimeEntryIndex(nodes []ProxySubscriptionNode, nodeKey string) int {
	for i := range nodes {
		if nodes[i].NodeKey == nodeKey {
			return i
		}
	}
	return -1
}

func runtimeIDFromProxyName(sourceID int64, sourceName, proxyName string) (string, bool) {
	trimmedName := strings.TrimSpace(proxyName)
	prefix := strings.TrimSpace(sourceName) + " 自动选路入口 #"
	if !strings.HasPrefix(trimmedName, prefix) {
		return "", false
	}
	indexText := strings.TrimSpace(strings.TrimPrefix(trimmedName, prefix))
	index, err := strconv.Atoi(indexText)
	if err != nil || index <= 0 {
		return "", false
	}
	return runtimeIDForSource(sourceID, index), true
}

func expectedRuntimeEntryCount(source *ProxySubscriptionSource) int {
	if source == nil || source.TargetEntryCount <= 0 {
		return 3
	}
	return source.TargetEntryCount
}

func (s *ProxySubscriptionService) countHealthyExistingRuntimeEntries(ctx context.Context, source *ProxySubscriptionSource) int {
	if s == nil || s.proxyRepo == nil || s.runtime == nil || source == nil {
		return 0
	}
	proxies, err := s.proxyRepo.ListBySubscriptionSourceID(ctx, source.ID)
	if err != nil {
		return 0
	}
	count := 0
	for i := range proxies {
		if !proxies[i].ManagedBySubscription || proxies[i].Protocol != ProxyNodeTypeSOCKS5H {
			continue
		}
		runtimeID, ok := runtimeIDFromProxyName(source.ID, source.Name, proxies[i].Name)
		if !ok {
			continue
		}
		if err := s.runtime.CheckRuntime(ctx, runtimeID); err != nil {
			continue
		}
		count++
	}
	return count
}
