package service

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type proxyRepoStubForPool struct {
	proxies map[int64]Proxy
}

func (s *proxyRepoStubForPool) Create(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Create")
}
func (s *proxyRepoStubForPool) Update(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Update")
}
func (s *proxyRepoStubForPool) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete")
}
func (s *proxyRepoStubForPool) List(ctx context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected List")
}
func (s *proxyRepoStubForPool) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters")
}
func (s *proxyRepoStubForPool) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFiltersAndAccountCount")
}
func (s *proxyRepoStubForPool) ListActive(ctx context.Context) ([]Proxy, error) {
	panic("unexpected ListActive")
}
func (s *proxyRepoStubForPool) ListActiveWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error) {
	panic("unexpected ListActiveWithAccountCount")
}
func (s *proxyRepoStubForPool) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	panic("unexpected ExistsByHostPortAuth")
}
func (s *proxyRepoStubForPool) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	panic("unexpected CountAccountsByProxyID")
}
func (s *proxyRepoStubForPool) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	panic("unexpected ListAccountSummariesByProxyID")
}
func (s *proxyRepoStubForPool) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	proxy, ok := s.proxies[id]
	if !ok {
		return nil, ErrProxyNotFound
	}
	cloned := proxy
	return &cloned, nil
}
func (s *proxyRepoStubForPool) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	result := make([]Proxy, 0, len(ids))
	for _, id := range ids {
		if proxy, ok := s.proxies[id]; ok {
			result = append(result, proxy)
		}
	}
	return result, nil
}

type settingRepoStubForPool struct {
	values map[string]string
}

func (s *settingRepoStubForPool) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get")
}
func (s *settingRepoStubForPool) GetValue(ctx context.Context, key string) (string, error) {
	return s.values[key], nil
}
func (s *settingRepoStubForPool) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}
func (s *settingRepoStubForPool) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}
func (s *settingRepoStubForPool) SetMultiple(ctx context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}
func (s *settingRepoStubForPool) GetAll(ctx context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *settingRepoStubForPool) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type proxyLatencyCacheStubForPool struct {
	mu   sync.Mutex
	data map[int64]*ProxyLatencyInfo
}

func (s *proxyLatencyCacheStubForPool) GetProxyLatencies(ctx context.Context, proxyIDs []int64) (map[int64]*ProxyLatencyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[int64]*ProxyLatencyInfo, len(proxyIDs))
	for _, id := range proxyIDs {
		if info, ok := s.data[id]; ok {
			cloned := *info
			out[id] = &cloned
		}
	}
	return out, nil
}

func (s *proxyLatencyCacheStubForPool) SetProxyLatency(ctx context.Context, proxyID int64, info *ProxyLatencyInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[int64]*ProxyLatencyInfo)
	}
	cloned := *info
	s.data[proxyID] = &cloned
	return nil
}

type timeoutNetError struct{}

func (e *timeoutNetError) Error() string   { return "dial tcp timeout" }
func (e *timeoutNetError) Timeout() bool   { return true }
func (e *timeoutNetError) Temporary() bool { return true }

func TestAutoFailoverProxyPoolServiceBuildCandidatesSkipsCoolingProxy(t *testing.T) {
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[1,2,3]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	cache := &proxyLatencyCacheStubForPool{
		data: map[int64]*ProxyLatencyInfo{
			2: {
				HealthStatus:      "cooldown",
				CooldownUntilUnix: ptrInt64(time.Now().Add(2 * time.Minute).Unix()),
			},
			1: {
				HealthStatus: "healthy",
				LatencyMs:    ptrInt64(120),
				QualityScore: intPtrPoolTest(90),
			},
			3: {
				HealthStatus: "healthy",
				LatencyMs:    ptrInt64(80),
				QualityScore: intPtrPoolTest(70),
			},
		},
	}
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			1: {ID: 1, Name: "p1", Protocol: "http", Host: "p1.example", Port: 8080, Status: StatusActive},
			2: {ID: 2, Name: "p2", Protocol: "http", Host: "p2.example", Port: 8080, Status: StatusActive},
			3: {ID: 3, Name: "p3", Protocol: "http", Host: "p3.example", Port: 8080, Status: StatusActive},
		},
	}

	svc := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, cache, nil)
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"proxy_mode": AccountProxyModePool,
		},
	}

	candidates, err := svc.BuildCandidates(context.Background(), account)
	if err != nil {
		t.Fatalf("BuildCandidates() error = %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("BuildCandidates() len = %d, want 2", len(candidates))
	}
	if candidates[0].ProxyID == nil || *candidates[0].ProxyID != 1 {
		t.Fatalf("first candidate = %+v, want proxy 1", candidates[0])
	}
	if candidates[1].ProxyID == nil || *candidates[1].ProxyID != 3 {
		t.Fatalf("second candidate = %+v, want proxy 3", candidates[1])
	}
}

func TestAutoFailoverProxyPoolServiceBuildCandidatesDefaultsToDirectWhenPoolNotSelected(t *testing.T) {
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[1,2]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			1: {ID: 1, Name: "p1", Protocol: "http", Host: "p1.example", Port: 8080, Status: StatusActive},
			2: {ID: 2, Name: "p2", Protocol: "http", Host: "p2.example", Port: 8080, Status: StatusActive},
		},
	}

	svc := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, &proxyLatencyCacheStubForPool{}, nil)
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	candidates, err := svc.BuildCandidates(context.Background(), account)
	if err != nil {
		t.Fatalf("BuildCandidates() error = %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("BuildCandidates() len = %d, want 1", len(candidates))
	}
	if candidates[0].Source != "direct" || candidates[0].ProxyID != nil || candidates[0].ProxyURL != "" {
		t.Fatalf("candidate = %+v, want direct fallback only", candidates[0])
	}
}

func TestAutoFailoverProxyPoolServiceBuildCandidatesUsesPoolWhenExplicitlySelected(t *testing.T) {
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[1,2]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			1: {ID: 1, Name: "p1", Protocol: "http", Host: "p1.example", Port: 8080, Status: StatusActive},
			2: {ID: 2, Name: "p2", Protocol: "http", Host: "p2.example", Port: 8080, Status: StatusActive},
		},
	}

	svc := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, &proxyLatencyCacheStubForPool{}, nil)
	account := &Account{
		Platform: PlatformAntigravity,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"proxy_mode": AccountProxyModePool,
		},
	}

	candidates, err := svc.BuildCandidates(context.Background(), account)
	if err != nil {
		t.Fatalf("BuildCandidates() error = %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("BuildCandidates() len = %d, want 2", len(candidates))
	}
	if candidates[0].Source != "auto_failover_pool" || candidates[1].Source != "auto_failover_pool" {
		t.Fatalf("candidates = %+v, want pool candidates", candidates)
	}
}

func TestAutoFailoverProxyPoolServiceDoHTTPRequestFailsOverAndMutatesTransientAccount(t *testing.T) {
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[1,2]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	cache := &proxyLatencyCacheStubForPool{data: map[int64]*ProxyLatencyInfo{}}
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			1: {ID: 1, Name: "bad", Protocol: "http", Host: "bad.example", Port: 8080, Status: StatusActive},
			2: {ID: 2, Name: "good", Protocol: "http", Host: "good.example", Port: 8080, Status: StatusActive},
		},
	}
	svc := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, cache, nil)

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		ProxyID:  ptrInt64(1),
	}
	req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", bytes.NewReader([]byte(`{"hello":"world"}`)))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	calls := make([]string, 0, 2)
	resp, err := svc.DoHTTPRequest(context.Background(), account, req, func(clonedReq *http.Request, proxyURL string) (*http.Response, error) {
		calls = append(calls, proxyURL)
		if len(calls) == 1 {
			return nil, &net.OpError{Op: "dial", Err: &timeoutNetError{}}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`ok`))),
		}, nil
	})
	if err != nil {
		t.Fatalf("DoHTTPRequest() error = %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("DoHTTPRequest() response = %#v, want 200", resp)
	}
	if len(calls) != 2 {
		t.Fatalf("proxy call count = %d, want 2", len(calls))
	}
	if calls[0] == calls[1] {
		t.Fatalf("expected failover to a different proxy, calls = %#v", calls)
	}
	if account.ProxyID == nil || *account.ProxyID != 2 {
		t.Fatalf("account.ProxyID = %v, want 2", account.ProxyID)
	}
	state, ok := account.Extra["proxy_failover_state"].(map[string]any)
	if !ok {
		t.Fatalf("proxy_failover_state missing from account.Extra: %#v", account.Extra)
	}
	if got, _ := state["active_proxy_id"].(int64); got != 0 {
		// JSON-like maps in runtime state use float64/int64 depending on source. Allow fallback below.
	}
	if activeID, ok := state["active_proxy_id"].(int64); ok && activeID != 2 {
		t.Fatalf("active_proxy_id = %d, want 2", activeID)
	}
	if activeID, ok := state["active_proxy_id"].(float64); ok && int64(activeID) != 2 {
		t.Fatalf("active_proxy_id = %v, want 2", activeID)
	}
	if cache.data[1] == nil || cache.data[1].HealthStatus != "cooldown" {
		t.Fatalf("proxy 1 runtime state = %#v, want cooldown", cache.data[1])
	}
	if cache.data[2] == nil || cache.data[2].HealthStatus != "healthy" {
		t.Fatalf("proxy 2 runtime state = %#v, want healthy", cache.data[2])
	}
}

func TestAutoFailoverProxyPoolServiceDoHTTPRequestCapsSingleRequestAttempts(t *testing.T) {
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[1,2,3,4,5,6]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	cache := &proxyLatencyCacheStubForPool{data: map[int64]*ProxyLatencyInfo{}}
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			1: {ID: 1, Name: "p1", Protocol: "http", Host: "p1.example", Port: 8080, Status: StatusActive},
			2: {ID: 2, Name: "p2", Protocol: "http", Host: "p2.example", Port: 8080, Status: StatusActive},
			3: {ID: 3, Name: "p3", Protocol: "http", Host: "p3.example", Port: 8080, Status: StatusActive},
			4: {ID: 4, Name: "p4", Protocol: "http", Host: "p4.example", Port: 8080, Status: StatusActive},
			5: {ID: 5, Name: "p5", Protocol: "http", Host: "p5.example", Port: 8080, Status: StatusActive},
			6: {ID: 6, Name: "p6", Protocol: "http", Host: "p6.example", Port: 8080, Status: StatusActive},
		},
	}
	svc := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, cache, nil)
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		ProxyID:  ptrInt64(1),
	}
	req, err := http.NewRequest(http.MethodGet, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	calls := make([]string, 0, 6)
	_, err = svc.DoHTTPRequest(context.Background(), account, req, func(_ *http.Request, proxyURL string) (*http.Response, error) {
		calls = append(calls, proxyURL)
		return nil, &net.OpError{Op: "dial", Err: &timeoutNetError{}}
	})
	if err == nil {
		t.Fatal("DoHTTPRequest() error = nil, want exhausted failover error")
	}
	if len(calls) != 1+autoFailoverProxyMaxPoolAttempts {
		t.Fatalf("proxy call count = %d, want %d; calls = %#v", len(calls), 1+autoFailoverProxyMaxPoolAttempts, calls)
	}
	if !strings.Contains(err.Error(), "proxy failover exhausted after 4 attempts") {
		t.Fatalf("error = %v, want observable exhausted attempts", err)
	}
	if cache.data[4] == nil || cache.data[4].HealthStatus != "cooldown" {
		t.Fatalf("proxy 4 runtime state = %#v, want cooldown", cache.data[4])
	}
	if cache.data[5] != nil || cache.data[6] != nil {
		t.Fatalf("proxies beyond cap were touched: p5=%#v p6=%#v", cache.data[5], cache.data[6])
	}
}

func TestAutoFailoverProxyPoolServiceDoesNotRetryCredentialError(t *testing.T) {
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: "[1,2]",
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	cache := &proxyLatencyCacheStubForPool{data: map[int64]*ProxyLatencyInfo{}}
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			1: {ID: 1, Name: "p1", Protocol: "http", Host: "p1.example", Port: 8080, Status: StatusActive},
			2: {ID: 2, Name: "p2", Protocol: "http", Host: "p2.example", Port: 8080, Status: StatusActive},
		},
	}
	svc := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, cache, nil)

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		ProxyID:  ptrInt64(1),
	}
	req, err := http.NewRequest(http.MethodGet, "https://example.com/v1/responses", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	callCount := 0
	resp, err := svc.DoHTTPRequest(context.Background(), account, req, func(clonedReq *http.Request, proxyURL string) (*http.Response, error) {
		callCount++
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"invalid token"}`))),
		}, nil
	})
	if err != nil {
		t.Fatalf("DoHTTPRequest() unexpected error = %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("response = %#v, want 401", resp)
	}
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1", callCount)
	}
}

type trackingProxyProber struct {
	delay time.Duration

	mu       sync.Mutex
	inFlight int
	max      int
	calls    int
}

func (p *trackingProxyProber) ProbeProxy(ctx context.Context, proxyURL string) (*ProxyExitInfo, int64, error) {
	p.mu.Lock()
	p.inFlight++
	p.calls++
	if p.inFlight > p.max {
		p.max = p.inFlight
	}
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.inFlight--
		p.mu.Unlock()
	}()

	select {
	case <-time.After(p.delay):
		return &ProxyExitInfo{IP: "203.0.113.1", Country: "Test", CountryCode: "TS"}, 25, nil
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
}

func TestAutoFailoverProxyPoolServiceRefreshPoolHealthUsesBoundedConcurrency(t *testing.T) {
	poolIDs := "[1,2,3,4,5,6,7,8,9,10,11,12]"
	settingRepo := &settingRepoStubForPool{
		values: map[string]string{
			SettingKeyAutoFailoverProxyPool: poolIDs,
		},
	}
	settingSvc := NewSettingService(settingRepo, nil)
	proxies := make(map[int64]Proxy)
	for id := int64(1); id <= 12; id++ {
		proxies[id] = Proxy{ID: id, Name: "p", Protocol: "http", Host: "p.example", Port: int(8000 + id), Status: StatusActive}
	}

	prober := &trackingProxyProber{delay: 10 * time.Millisecond}
	svc := NewAutoFailoverProxyPoolService(
		&proxyRepoStubForPool{proxies: proxies},
		nil,
		settingSvc,
		&proxyLatencyCacheStubForPool{},
		prober,
	)

	svc.refreshPoolHealth(context.Background())

	if prober.calls != len(proxies) {
		t.Fatalf("probe calls = %d, want %d", prober.calls, len(proxies))
	}
	if prober.max <= 1 {
		t.Fatalf("max concurrency = %d, want concurrent probing", prober.max)
	}
	if prober.max > autoFailoverProxyHealthCheckWorkers {
		t.Fatalf("max concurrency = %d, want <= %d", prober.max, autoFailoverProxyHealthCheckWorkers)
	}
}

func intPtrPoolTest(v int) *int {
	return &v
}
