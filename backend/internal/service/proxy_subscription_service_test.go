package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type proxySubscriptionSourceRepoStub struct {
	sources map[int64]*ProxySubscriptionSource
	nextID  int64
}

func (s *proxySubscriptionSourceRepoStub) Create(_ context.Context, source *ProxySubscriptionSource) error {
	if s.sources == nil {
		s.sources = map[int64]*ProxySubscriptionSource{}
	}
	if s.nextID == 0 {
		s.nextID = 1
	}
	source.ID = s.nextID
	s.nextID++
	now := time.Now()
	source.CreatedAt = now
	source.UpdatedAt = now
	cloned := *source
	s.sources[source.ID] = &cloned
	return nil
}
func (s *proxySubscriptionSourceRepoStub) GetByID(_ context.Context, id int64) (*ProxySubscriptionSource, error) {
	item, ok := s.sources[id]
	if !ok {
		return nil, ErrProxyNotFound
	}
	cloned := *item
	return &cloned, nil
}
func (s *proxySubscriptionSourceRepoStub) List(_ context.Context, page, pageSize int, search string, enabled *bool) ([]ProxySubscriptionSource, int64, error) {
	out := make([]ProxySubscriptionSource, 0, len(s.sources))
	for _, item := range s.sources {
		out = append(out, *item)
	}
	return out, int64(len(out)), nil
}
func (s *proxySubscriptionSourceRepoStub) ListEnabled(_ context.Context) ([]ProxySubscriptionSource, error) {
	out := make([]ProxySubscriptionSource, 0, len(s.sources))
	for _, item := range s.sources {
		if item.Enabled {
			out = append(out, *item)
		}
	}
	return out, nil
}
func (s *proxySubscriptionSourceRepoStub) ListDueForRefresh(_ context.Context, now time.Time, limit int) ([]ProxySubscriptionSource, error) {
	out := make([]ProxySubscriptionSource, 0, len(s.sources))
	for _, item := range s.sources {
		if item.Enabled {
			out = append(out, *item)
		}
	}
	return out, nil
}
func (s *proxySubscriptionSourceRepoStub) Update(_ context.Context, source *ProxySubscriptionSource) error {
	cloned := *source
	s.sources[source.ID] = &cloned
	return nil
}
func (s *proxySubscriptionSourceRepoStub) Delete(_ context.Context, id int64) error {
	delete(s.sources, id)
	return nil
}

type proxySubscriptionNodeRepoStub struct {
	nodes  map[int64]*ProxySubscriptionNode
	nextID int64
}

func (s *proxySubscriptionNodeRepoStub) Create(_ context.Context, node *ProxySubscriptionNode) error {
	if s.nodes == nil {
		s.nodes = map[int64]*ProxySubscriptionNode{}
	}
	if s.nextID == 0 {
		s.nextID = 1
	}
	node.ID = s.nextID
	s.nextID++
	now := time.Now()
	node.CreatedAt = now
	node.UpdatedAt = now
	cloned := *node
	s.nodes[node.ID] = &cloned
	return nil
}
func (s *proxySubscriptionNodeRepoStub) Update(_ context.Context, node *ProxySubscriptionNode) error {
	cloned := *node
	s.nodes[node.ID] = &cloned
	return nil
}
func (s *proxySubscriptionNodeRepoStub) GetByID(_ context.Context, id int64) (*ProxySubscriptionNode, error) {
	item, ok := s.nodes[id]
	if !ok {
		return nil, nil
	}
	cloned := *item
	return &cloned, nil
}
func (s *proxySubscriptionNodeRepoStub) ListBySourceID(_ context.Context, sourceID int64) ([]ProxySubscriptionNode, error) {
	out := make([]ProxySubscriptionNode, 0)
	for _, item := range s.nodes {
		if item.SourceID == sourceID {
			out = append(out, *item)
		}
	}
	return out, nil
}
func (s *proxySubscriptionNodeRepoStub) GetBySourceAndNodeKey(_ context.Context, sourceID int64, nodeKey string) (*ProxySubscriptionNode, error) {
	for _, item := range s.nodes {
		if item.SourceID == sourceID && item.NodeKey == nodeKey {
			cloned := *item
			return &cloned, nil
		}
	}
	return nil, nil
}
func (s *proxySubscriptionNodeRepoStub) SoftDeleteMissingBySourceID(_ context.Context, sourceID int64, activeNodeKeys []string, now time.Time) error {
	active := make(map[string]struct{}, len(activeNodeKeys))
	for _, key := range activeNodeKeys {
		active[key] = struct{}{}
	}
	for _, item := range s.nodes {
		if item.SourceID != sourceID {
			continue
		}
		if _, ok := active[item.NodeKey]; ok {
			continue
		}
		item.LandingStatus = ProxySubscriptionLandingStatusStale
		item.UpdatedAt = now
	}
	return nil
}

type proxyRepoSubscriptionStub struct {
	proxies       map[int64]*Proxy
	nextID        int64
	accountCounts map[int64]int64
}

func (s *proxyRepoSubscriptionStub) Create(_ context.Context, proxy *Proxy) error {
	if s.proxies == nil {
		s.proxies = map[int64]*Proxy{}
	}
	if s.nextID == 0 {
		s.nextID = 1
	}
	proxy.ID = s.nextID
	s.nextID++
	now := time.Now()
	proxy.CreatedAt = now
	proxy.UpdatedAt = now
	cloned := *proxy
	s.proxies[proxy.ID] = &cloned
	return nil
}
func (s *proxyRepoSubscriptionStub) GetByID(_ context.Context, id int64) (*Proxy, error) {
	if item, ok := s.proxies[id]; ok {
		cloned := *item
		return &cloned, nil
	}
	return nil, ErrProxyNotFound
}
func (s *proxyRepoSubscriptionStub) ListByIDs(_ context.Context, ids []int64) ([]Proxy, error) {
	out := make([]Proxy, 0, len(ids))
	for _, id := range ids {
		if item, ok := s.proxies[id]; ok {
			out = append(out, *item)
		}
	}
	return out, nil
}
func (s *proxyRepoSubscriptionStub) ListBySubscriptionSourceID(_ context.Context, sourceID int64) ([]Proxy, error) {
	out := make([]Proxy, 0)
	for _, item := range s.proxies {
		if item.SubscriptionSourceID != nil && *item.SubscriptionSourceID == sourceID {
			out = append(out, *item)
		}
	}
	return out, nil
}
func (s *proxyRepoSubscriptionStub) FindBySubscriptionNodeID(_ context.Context, nodeID int64) (*Proxy, error) {
	for _, item := range s.proxies {
		if item.SubscriptionNodeID != nil && *item.SubscriptionNodeID == nodeID {
			cloned := *item
			return &cloned, nil
		}
	}
	return nil, nil
}
func (s *proxyRepoSubscriptionStub) FindByHostPortAuth(_ context.Context, host string, port int, username, password string) (*Proxy, error) {
	for _, item := range s.proxies {
		if item.Host == host && item.Port == port && item.Username == username && item.Password == password {
			cloned := *item
			return &cloned, nil
		}
	}
	return nil, nil
}
func (s *proxyRepoSubscriptionStub) Update(_ context.Context, proxy *Proxy) error {
	cloned := *proxy
	s.proxies[proxy.ID] = &cloned
	return nil
}
func (s *proxyRepoSubscriptionStub) Delete(_ context.Context, id int64) error {
	delete(s.proxies, id)
	return nil
}
func (s *proxyRepoSubscriptionStub) List(_ context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *proxyRepoSubscriptionStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, protocol, status, search string) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *proxyRepoSubscriptionStub) ListWithFiltersAndAccountCount(_ context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *proxyRepoSubscriptionStub) ListActive(_ context.Context) ([]Proxy, error) { return nil, nil }
func (s *proxyRepoSubscriptionStub) ListActiveWithAccountCount(_ context.Context) ([]ProxyWithAccountCount, error) {
	return nil, nil
}
func (s *proxyRepoSubscriptionStub) ExistsByHostPortAuth(_ context.Context, host string, port int, username, password string) (bool, error) {
	item, err := s.FindByHostPortAuth(context.Background(), host, port, username, password)
	return item != nil, err
}
func (s *proxyRepoSubscriptionStub) CountAccountsByProxyID(_ context.Context, proxyID int64) (int64, error) {
	if s.accountCounts != nil {
		return s.accountCounts[proxyID], nil
	}
	return 0, nil
}
func (s *proxyRepoSubscriptionStub) ListAccountSummariesByProxyID(_ context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	count := s.accountCounts[proxyID]
	items := make([]ProxyAccountSummary, 0, count)
	for i := int64(0); i < count; i++ {
		items = append(items, ProxyAccountSummary{
			ID:   proxyID*100 + i + 1,
			Name: fmt.Sprintf("account-%d", proxyID*100+i+1),
		})
	}
	return items, nil
}

type proxySubscriptionAccountRepoStub struct {
	bulkUpdates []struct {
		IDs     []int64
		ProxyID *int64
	}
}

func (s *proxySubscriptionAccountRepoStub) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	record := struct {
		IDs     []int64
		ProxyID *int64
	}{
		IDs:     append([]int64(nil), ids...),
		ProxyID: updates.ProxyID,
	}
	s.bulkUpdates = append(s.bulkUpdates, record)
	return int64(len(ids)), nil
}

type proxySubscriptionRuntimeStub struct {
	lastRequest ProxySubscriptionRuntimeUpsertRequest
	upsertErr   error
}

func (s *proxySubscriptionRuntimeStub) Start(ctx context.Context) error { return nil }
func (s *proxySubscriptionRuntimeStub) Stop(ctx context.Context) error  { return nil }
func (s *proxySubscriptionRuntimeStub) UpsertRuntime(_ context.Context, req ProxySubscriptionRuntimeUpsertRequest) (*ProxySubscriptionRuntimeUpsertResponse, error) {
	s.lastRequest = req
	if s.upsertErr != nil {
		return nil, s.upsertErr
	}
	return &ProxySubscriptionRuntimeUpsertResponse{
		RuntimeID:    req.RuntimeID,
		ListenerHost: "127.0.0.1",
		ListenerPort: 21080,
		Protocol:     "socks5h",
	}, nil
}
func (s *proxySubscriptionRuntimeStub) DeleteRuntime(ctx context.Context, runtimeID string) error {
	return nil
}
func (s *proxySubscriptionRuntimeStub) CheckRuntime(ctx context.Context, runtimeID string) error {
	return nil
}

type proxySubscriptionProberStub struct {
	exitInfo  *ProxyExitInfo
	exitInfos []*ProxyExitInfo
	latency   int64
	latencies []int64
	err       error
	calls     int
}

func (s *proxySubscriptionProberStub) ProbeProxy(ctx context.Context, proxyURL string) (*ProxyExitInfo, int64, error) {
	s.calls++
	if s.err != nil {
		return nil, 0, s.err
	}
	idx := s.calls - 1
	info := s.exitInfo
	if idx >= 0 && idx < len(s.exitInfos) && s.exitInfos[idx] != nil {
		info = s.exitInfos[idx]
	}
	if info == nil {
		info = &ProxyExitInfo{IP: "203.0.113.11", Country: "JP", CountryCode: "JP"}
	}
	latency := s.latency
	if idx >= 0 && idx < len(s.latencies) && s.latencies[idx] > 0 {
		latency = s.latencies[idx]
	}
	if latency == 0 {
		latency = 20
	}
	return info, latency, nil
}

func TestParseProxySubscriptionPayload_DirectList(t *testing.T) {
	nodes, format, errs := parseProxySubscriptionPayload([]byte("http,1.2.3.4,8080,user,pass,proxy-a"), ProxySubscriptionSourceFormatDirectList)
	if format != ProxySubscriptionSourceFormatDirectList {
		t.Fatalf("format = %s", format)
	}
	if len(errs) != 0 {
		t.Fatalf("errs = %#v", errs)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d", len(nodes))
	}
	if nodes[0].NodeType != ProxyNodeTypeHTTP || nodes[0].Server != "1.2.3.4" || nodes[0].Port != 8080 {
		t.Fatalf("unexpected node = %#v", nodes[0])
	}
}

func TestParseProxySubscriptionPayload_URIList(t *testing.T) {
	nodes, format, errs := parseProxySubscriptionPayload([]byte("socks5://user:pass@127.0.0.1:1080#proxy"), ProxySubscriptionSourceFormatURIList)
	if format != ProxySubscriptionSourceFormatURIList {
		t.Fatalf("format = %s", format)
	}
	if len(errs) != 0 {
		t.Fatalf("errs = %#v", errs)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d", len(nodes))
	}
	if nodes[0].NodeType != ProxyNodeTypeSOCKS5 {
		t.Fatalf("unexpected node type = %s", nodes[0].NodeType)
	}
}

func TestParseProxySubscriptionPayload_ClashYAML(t *testing.T) {
	payload := []byte("proxies:\n  - name: hk\n    type: vmess\n    server: example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n")
	nodes, format, errs := parseProxySubscriptionPayload(payload, ProxySubscriptionSourceFormatClashYAML)
	if format != ProxySubscriptionSourceFormatClashYAML {
		t.Fatalf("format = %s", format)
	}
	if len(errs) != 0 {
		t.Fatalf("errs = %#v", errs)
	}
	if len(nodes) != 1 || nodes[0].NodeType != ProxyNodeTypeVMess {
		t.Fatalf("nodes = %#v", nodes)
	}
}

func TestProxySubscriptionService_RefreshSource_MaterializesDirectProxy(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			1: {
				ID:                   1,
				Name:                 "source-1",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatDirectList,
				Enabled:              true,
				RefreshIntervalHours: 6,
				AutoAddToPool:        false,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{}
	proxyRepo := &proxyRepoSubscriptionStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("http,1.2.3.4,8080,user,pass,proxy-a"))
	}))
	defer server.Close()
	sourceRepo.sources[1].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, nil, nil, nil, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	result, err := svc.RefreshSource(context.Background(), 1)
	if err != nil {
		t.Fatalf("RefreshSource error = %v", err)
	}
	if result.MaterializedProxyCount != 1 || result.CreatedProxyCount != 1 {
		t.Fatalf("unexpected result = %#v", result)
	}
	if len(proxyRepo.proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(proxyRepo.proxies))
	}
}

func TestProxySubscriptionService_RefreshSource_MaterializesRuntimeProxy(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			2: {
				ID:                   2,
				Name:                 "source-2",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{}
	proxyRepo := &proxyRepoSubscriptionStub{}
	runtime := &proxySubscriptionRuntimeStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n"))
	}))
	defer server.Close()
	sourceRepo.sources[2].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, nil, nil, runtime, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	result, err := svc.RefreshSource(context.Background(), 2)
	if err != nil {
		t.Fatalf("RefreshSource error = %v", err)
	}
	if result.MaterializedProxyCount != 1 {
		t.Fatalf("unexpected result = %#v", result)
	}
	if runtime.lastRequest.NodeType != ProxyNodeTypeVMess {
		t.Fatalf("unexpected runtime request = %#v", runtime.lastRequest)
	}
	if runtime.lastRequest.Subscription == "" {
		t.Fatal("expected subscription payload to be passed to runtime manager")
	}
	if !strings.Contains(runtime.lastRequest.ProviderContent, "example.com") {
		t.Fatalf("expected provider content to include the selected node, got %s", runtime.lastRequest.ProviderContent)
	}
	if len(proxyRepo.proxies) != 1 {
		t.Fatalf("expected one subscription-level proxy, got %d", len(proxyRepo.proxies))
	}
	if proxyRepo.proxies[1].SubscriptionNodeID != nil {
		t.Fatalf("expected subscription-level proxy without node binding, got %#v", proxyRepo.proxies[1].SubscriptionNodeID)
	}
}

func TestProxySubscriptionService_RefreshSource_MaterializesSingleRuntimeForMultipleNodes(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			5: {
				ID:                   5,
				Name:                 "source-5",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{}
	proxyRepo := &proxyRepoSubscriptionStub{}
	runtime := &proxySubscriptionRuntimeStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: jp.example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n  - name: sg\n    type: trojan\n    server: sg.example.com\n    port: 443\n    password: abc123\n"))
	}))
	defer server.Close()
	sourceRepo.sources[5].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, nil, nil, runtime, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	result, err := svc.RefreshSource(context.Background(), 5)
	if err != nil {
		t.Fatalf("RefreshSource error = %v", err)
	}
	if result.NodeCount != 2 {
		t.Fatalf("expected 2 parsed nodes, got %#v", result)
	}
	if len(proxyRepo.proxies) != 1 {
		t.Fatalf("expected 1 materialized proxy for the whole subscription, got %d", len(proxyRepo.proxies))
	}
	if !strings.Contains(runtime.lastRequest.ProviderContent, "jp.example.com") || !strings.Contains(runtime.lastRequest.ProviderContent, "sg.example.com") {
		t.Fatalf("expected runtime provider content to include all candidate nodes, got %s", runtime.lastRequest.ProviderContent)
	}
	for _, node := range nodeRepo.nodes {
		if node.LandingStatus != ProxySubscriptionLandingStatusActive {
			t.Fatalf("expected active node landing status, got %#v", node)
		}
	}
}

func TestProxySubscriptionService_RefreshSource_UsesApprovedNodeGroupsForMultipleEntries(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			6: {
				ID:                   6,
				Name:                 "source-6",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
				TargetEntryCount:     2,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{}
	proxyRepo := &proxyRepoSubscriptionStub{}
	runtime := &proxySubscriptionRuntimeStub{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp-1\n    type: vmess\n    server: jp1.example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n  - name: sg-1\n    type: trojan\n    server: sg1.example.com\n    port: 443\n    password: abc123\n  - name: us-1\n    type: vless\n    server: us1.example.com\n    port: 443\n    uuid: 22222222-2222-2222-2222-222222222222\n  - name: jp-2\n    type: vmess\n    server: jp2.example.com\n    port: 443\n    uuid: 33333333-3333-3333-3333-333333333333\n"))
	}))
	defer server.Close()
	sourceRepo.sources[6].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, nil, nil, runtime, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	result, err := svc.RefreshSource(context.Background(), 6)
	if err != nil {
		t.Fatalf("RefreshSource error = %v", err)
	}
	if result.MaterializedProxyCount != 2 {
		t.Fatalf("expected 2 materialized proxies, got %#v", result)
	}
	if !strings.Contains(runtime.lastRequest.ProviderContent, "sg1.example.com") || !strings.Contains(runtime.lastRequest.ProviderContent, "jp2.example.com") {
		t.Fatalf("expected grouped provider content to include later nodes, got %s", runtime.lastRequest.ProviderContent)
	}
}

func TestProxySubscriptionService_RefreshSource_RuntimeMaterializationFailureDoesNotDisableExistingProxy(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			7: {
				ID:                   7,
				Name:                 "source-7",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
				TargetEntryCount:     1,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{
		nodes: map[int64]*ProxySubscriptionNode{
			71: {
				ID:            71,
				SourceID:      7,
				NodeKey:       "old-node",
				DisplayName:   "old-node",
				NodeType:      ProxyNodeTypeVMess,
				Server:        "old.example.com",
				Port:          443,
				ConfigJSON:    map[string]any{"uuid": "old"},
				LandingStatus: ProxySubscriptionLandingStatusActive,
			},
		},
	}
	proxyRepo := &proxyRepoSubscriptionStub{
		proxies: map[int64]*Proxy{
			81: {
				ID:                    81,
				Name:                  "source-7 自动选路入口 #1",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21080,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(7),
			},
		},
		nextID: 90,
	}
	runtime := &proxySubscriptionRuntimeStub{upsertErr: ErrProxyNotFound}
	prober := &proxySubscriptionProberStub{
		exitInfo: &ProxyExitInfo{IP: "203.0.113.21", Country: "JP", CountryCode: "JP"},
		latency:  16,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: new.example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n"))
	}))
	defer server.Close()
	sourceRepo.sources[7].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, nil, nil, runtime, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})
	svc.SetProxyProber(prober)
	svc.openAIProbe = func(context.Context, string) (bool, error) { return true, nil }

	result, err := svc.RefreshSource(context.Background(), 7)
	if err == nil {
		t.Fatalf("expected refresh error, got result %#v", result)
	}
	item := proxyRepo.proxies[81]
	if item == nil {
		t.Fatal("expected existing managed proxy to remain")
	}
	if item.Status != StatusActive {
		t.Fatalf("expected existing managed proxy to remain active, got %#v", item)
	}
	if len(proxyRepo.proxies) != 1 {
		t.Fatalf("expected no proxy cleanup on failed materialization, got %d", len(proxyRepo.proxies))
	}
}

func TestProxySubscriptionService_RefreshSource_DisablesStaleMaterializedProxy(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			3: {
				ID:                   3,
				Name:                 "source-3",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
				TargetEntryCount:     1,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{
		nodes: map[int64]*ProxySubscriptionNode{
			11: {
				ID:            11,
				SourceID:      3,
				NodeKey:       "old-node",
				DisplayName:   "old-node",
				NodeType:      ProxyNodeTypeHTTP,
				Server:        "9.9.9.9",
				Port:          8080,
				ConfigJSON:    map[string]any{},
				LandingStatus: ProxySubscriptionLandingStatusActive,
			},
		},
		nextID: 20,
	}
	proxyRepo := &proxyRepoSubscriptionStub{
		proxies: map[int64]*Proxy{
			21: {
				ID:                    21,
				Name:                  "old-subscription-proxy",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21080,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(3),
			},
		},
		nextID: 30,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n"))
	}))
	defer server.Close()
	sourceRepo.sources[3].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, nil, nil, &proxySubscriptionRuntimeStub{}, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	result, err := svc.RefreshSource(context.Background(), 3)
	if err != nil {
		t.Fatalf("RefreshSource error = %v", err)
	}
	if result.DisabledProxyCount != 0 {
		t.Fatalf("did not expect subscription-level proxy to be disabled when refreshed successfully, result = %#v", result)
	}
	if len(proxyRepo.proxies) != 1 {
		t.Fatalf("expected one subscription-level proxy to remain, got %d", len(proxyRepo.proxies))
	}
}

func TestProxySubscriptionService_RefreshSource_DisablesButKeepsBoundStaleProxy(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			4: {
				ID:                   4,
				Name:                 "source-4",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
				TargetEntryCount:     1,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{
		nodes: map[int64]*ProxySubscriptionNode{
			31: {
				ID:            31,
				SourceID:      4,
				NodeKey:       "old-node-bound",
				DisplayName:   "old-node-bound",
				NodeType:      ProxyNodeTypeHTTP,
				Server:        "9.9.9.9",
				Port:          8080,
				ConfigJSON:    map[string]any{},
				LandingStatus: ProxySubscriptionLandingStatusActive,
			},
		},
	}
	proxyRepo := &proxyRepoSubscriptionStub{
		proxies: map[int64]*Proxy{
			41: {
				ID:                    41,
				Name:                  "old-subscription-proxy-bound",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21080,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(4),
			},
		},
		accountCounts: map[int64]int64{
			41: 2,
		},
		nextID: 50,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: trojan\n    server: jp.example.com\n    port: 443\n    password: abc123\n"))
	}))
	defer server.Close()
	sourceRepo.sources[4].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, nil, nil, &proxySubscriptionRuntimeStub{}, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	result, err := svc.RefreshSource(context.Background(), 4)
	if err != nil {
		t.Fatalf("RefreshSource error = %v", err)
	}
	if result.DisabledProxyCount != 0 {
		t.Fatalf("did not expect refreshed subscription-level proxy to be disabled, result = %#v", result)
	}
	item, ok := proxyRepo.proxies[41]
	if !ok {
		t.Fatal("expected subscription-level proxy to remain")
	}
	if item.Status != StatusActive {
		t.Fatalf("expected refreshed subscription-level proxy to stay active, got %s", item.Status)
	}
}

func TestProxySubscriptionService_UpdateSource_TargetEntryCountRefreshesAndMigratesAccounts(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			8: {
				ID:                   8,
				Name:                 "source-8",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
				TargetEntryCount:     3,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{}
	proxyRepo := &proxyRepoSubscriptionStub{
		proxies: map[int64]*Proxy{
			81: {
				ID:                    81,
				Name:                  "source-8 自动选路入口 #1",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21081,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(8),
			},
			82: {
				ID:                    82,
				Name:                  "source-8 自动选路入口 #2",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21082,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(8),
			},
			83: {
				ID:                    83,
				Name:                  "source-8 自动选路入口 #3",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21083,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(8),
			},
		},
		accountCounts: map[int64]int64{
			81: 1,
			82: 2,
			83: 4,
		},
		nextID: 90,
	}
	accountRepo := &proxySubscriptionAccountRepoStub{}
	runtime := &proxySubscriptionRuntimeStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp-1\n    type: vmess\n    server: jp1.example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n  - name: jp-2\n    type: vmess\n    server: jp2.example.com\n    port: 443\n    uuid: 22222222-2222-2222-2222-222222222222\n"))
	}))
	defer server.Close()
	sourceRepo.sources[8].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, accountRepo, nil, runtime, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	targetCount := 2
	updated, err := svc.UpdateSource(context.Background(), 8, &UpdateProxySubscriptionSourceInput{
		TargetEntryCount: &targetCount,
	})
	if err != nil {
		t.Fatalf("UpdateSource error = %v", err)
	}
	if updated.TargetEntryCount != 2 {
		t.Fatalf("expected updated target_entry_count=2, got %#v", updated)
	}
	if len(accountRepo.bulkUpdates) != 2 {
		t.Fatalf("expected 2 migration bulk updates, got %#v", accountRepo.bulkUpdates)
	}
	if runtime.lastRequest.RuntimeID != "src-8-subscription-2" {
		t.Fatalf("expected refresh to materialize runtime entry #2, got %#v", runtime.lastRequest)
	}
	if stale := proxyRepo.proxies[83]; stale == nil || stale.Status != StatusDisabled {
		t.Fatalf("expected stale proxy #3 to be disabled, got %#v", stale)
	}
}

func TestExpectedRuntimeEntryCount_DefaultsToThree(t *testing.T) {
	if got := expectedRuntimeEntryCount(nil); got != 3 {
		t.Fatalf("expected nil source default to 3, got %d", got)
	}
	if got := expectedRuntimeEntryCount(&ProxySubscriptionSource{}); got != 3 {
		t.Fatalf("expected empty source default to 3, got %d", got)
	}
}

func TestProxySubscriptionService_RefreshSource_ReassignsAccountsFromDisabledManagedProxies(t *testing.T) {
	sourceRepo := &proxySubscriptionSourceRepoStub{
		sources: map[int64]*ProxySubscriptionSource{
			8: {
				ID:                   8,
				Name:                 "source-8",
				URL:                  "http://example.test/sub",
				SourceFormat:         ProxySubscriptionSourceFormatClashYAML,
				Enabled:              true,
				RefreshIntervalHours: 6,
				TargetEntryCount:     2,
			},
		},
	}
	nodeRepo := &proxySubscriptionNodeRepoStub{}
	proxyRepo := &proxyRepoSubscriptionStub{
		proxies: map[int64]*Proxy{
			81: {
				ID:                    81,
				Name:                  "source-8 自动选路入口 #1",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21080,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(8),
			},
			82: {
				ID:                    82,
				Name:                  "source-8 自动选路入口 #2",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21081,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(8),
			},
			83: {
				ID:                    83,
				Name:                  "source-8 自动选路入口 #3",
				Protocol:              ProxyNodeTypeSOCKS5H,
				Host:                  "127.0.0.1",
				Port:                  21082,
				Status:                StatusActive,
				ManagedBySubscription: true,
				SubscriptionSourceID:  ptrInt64Sub(8),
			},
		},
		accountCounts: map[int64]int64{
			83: 3,
		},
		nextID: 90,
	}
	accountRepo := &proxySubscriptionAccountRepoStub{}
	runtime := &proxySubscriptionRuntimeStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies:\n  - name: jp\n    type: vmess\n    server: jp.example.com\n    port: 443\n    uuid: 11111111-1111-1111-1111-111111111111\n  - name: sg\n    type: trojan\n    server: sg.example.com\n    port: 443\n    password: abc123\n"))
	}))
	defer server.Close()
	sourceRepo.sources[8].URL = server.URL

	svc := NewProxySubscriptionService(sourceRepo, nodeRepo, proxyRepo, accountRepo, nil, runtime, &config.Config{
		ProxySubscriptions: config.ProxySubscriptionsConfig{DefaultRefreshIntervalHours: 6},
	})

	result, err := svc.RefreshSource(context.Background(), 8)
	if err != nil {
		t.Fatalf("RefreshSource error = %v", err)
	}
	if result.DisabledProxyCount == 0 {
		t.Fatalf("expected stale proxy to be disabled, result = %#v", result)
	}
	if len(accountRepo.bulkUpdates) == 0 {
		t.Fatal("expected bound accounts to be reassigned to active managed proxies")
	}
	targets := map[int64]struct{}{}
	for _, call := range accountRepo.bulkUpdates {
		if call.ProxyID == nil {
			t.Fatal("expected reassignment to a concrete active proxy")
		}
		targets[*call.ProxyID] = struct{}{}
	}
	if _, ok := targets[81]; !ok {
		t.Fatalf("expected accounts to be moved onto remaining active proxy 81, got %#v", targets)
	}
	if _, ok := targets[82]; !ok {
		t.Fatalf("expected accounts to be moved onto remaining active proxy 82, got %#v", targets)
	}
}

func TestFilterRuntimeCandidateSubscriptionNodes(t *testing.T) {
	nodes := []ProxySubscriptionNode{
		{NodeKey: "1", DisplayName: "香港HKT1", NodeType: ProxyNodeTypeVMess},
		{NodeKey: "2", DisplayName: "剩余流量：492GB", NodeType: ProxyNodeTypeVMess},
		{NodeKey: "3", DisplayName: "日本softbank", NodeType: ProxyNodeTypeVMess},
		{NodeKey: "4", DisplayName: "新加坡01", NodeType: ProxyNodeTypeTrojan},
		{NodeKey: "5", DisplayName: "DMIT-LACMIN2-1|3x", NodeType: ProxyNodeTypeVMess},
	}

	filtered, errs := filterRuntimeCandidateSubscriptionNodes(nodes)
	if len(filtered) != 4 {
		t.Fatalf("expected 4 nodes after filtering, got %d", len(filtered))
	}
	if filtered[0].DisplayName != "香港HKT1" || filtered[1].DisplayName != "日本softbank" || filtered[2].DisplayName != "新加坡01" || filtered[3].DisplayName != "DMIT-LACMIN2-1|3x" {
		t.Fatalf("unexpected filtered nodes = %#v", filtered)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 filter error, got %#v", errs)
	}
}

func TestRuntimeIDFromProxyName(t *testing.T) {
	runtimeID, ok := runtimeIDFromProxyName(12, "codex-test-sub", "codex-test-sub 自动选路入口 #2")
	if !ok {
		t.Fatal("expected runtime id to be parsed")
	}
	if runtimeID != "src-12-subscription-2" {
		t.Fatalf("unexpected runtime id = %s", runtimeID)
	}
}

func TestBuildRuntimeCandidateProviderContent(t *testing.T) {
	nodes := []ProxySubscriptionNode{
		{
			DisplayName: "日本01",
			NodeType:    ProxyNodeTypeVMess,
			Server:      "jp.example.com",
			Port:        443,
			ConfigJSON: map[string]any{
				"uuid": "11111111-1111-1111-1111-111111111111",
			},
		},
	}

	content := buildRuntimeCandidateProviderContent(nodes)
	if !strings.Contains(content, "name: 日本01") || !strings.Contains(content, "server: jp.example.com") {
		t.Fatalf("unexpected provider content = %s", content)
	}
	if !strings.Contains(content, "uuid: 11111111-1111-1111-1111-111111111111") {
		t.Fatalf("expected uuid to be preserved in provider content, got %s", content)
	}
}

func TestSelectRuntimeEntryNodes_DedupesAndRespectsTargetCount(t *testing.T) {
	nodes := []ProxySubscriptionNode{
		{NodeKey: "1", DisplayName: "日本01"},
		{NodeKey: "2", DisplayName: "日本01"},
		{NodeKey: "3", DisplayName: "新加坡01"},
		{NodeKey: "4", DisplayName: "德国01"},
	}

	selected := selectRuntimeEntryNodes(nodes, 3)
	if len(selected) != 3 {
		t.Fatalf("expected 3 selected nodes, got %#v", selected)
	}
	if selected[0].NodeKey != "1" || selected[1].NodeKey != "3" || selected[2].NodeKey != "4" {
		t.Fatalf("unexpected selection order = %#v", selected)
	}
}

func TestRuntimeIDForSource_WithEntryIndex(t *testing.T) {
	if got := runtimeIDForSource(12, 3); got != "src-12-subscription-3" {
		t.Fatalf("unexpected runtime id = %s", got)
	}
}

func TestGroupNodesForEntry_DistributesWithoutOverlap(t *testing.T) {
	all := []ProxySubscriptionNode{
		{NodeKey: "1", DisplayName: "n1"},
		{NodeKey: "2", DisplayName: "n2"},
		{NodeKey: "3", DisplayName: "n3"},
		{NodeKey: "4", DisplayName: "n4"},
		{NodeKey: "5", DisplayName: "n5"},
	}
	selected := []ProxySubscriptionNode{
		{NodeKey: "1"},
		{NodeKey: "2"},
	}

	groups := buildRuntimeEntryGroups(selected, all)
	group1 := groups[0]
	group2 := groups[1]

	if len(group1) != 3 || len(group2) != 2 {
		t.Fatalf("unexpected group sizes: %#v %#v", group1, group2)
	}
	if group1[0].NodeKey != "1" || group1[1].NodeKey != "3" || group1[2].NodeKey != "5" {
		t.Fatalf("unexpected group1 = %#v", group1)
	}
	if group2[0].NodeKey != "2" || group2[1].NodeKey != "4" {
		t.Fatalf("unexpected group2 = %#v", group2)
	}
}

func ptrInt64Sub(v int64) *int64 {
	return &v
}
