package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/httputil"
)

const (
	autoFailoverProxyPoolHealthCheckInterval = 90 * time.Second
	autoFailoverProxyCooldownDuration        = 2 * time.Minute
	autoFailoverProxyProbeTimeout            = 12 * time.Second
)

type ProxyFailoverCandidate struct {
	ProxyID  *int64
	Proxy    *Proxy
	ProxyURL string
	Source   string
}

type AutoFailoverProxyPoolService struct {
	proxyRepo         ProxyRepository
	accountRepo       AccountRepository
	settingService    *SettingService
	proxyLatencyCache ProxyLatencyCache
	proxyProber       ProxyExitInfoProber

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewAutoFailoverProxyPoolService(
	proxyRepo ProxyRepository,
	accountRepo AccountRepository,
	settingService *SettingService,
	proxyLatencyCache ProxyLatencyCache,
	proxyProber ProxyExitInfoProber,
) *AutoFailoverProxyPoolService {
	return &AutoFailoverProxyPoolService{
		proxyRepo:         proxyRepo,
		accountRepo:       accountRepo,
		settingService:    settingService,
		proxyLatencyCache: proxyLatencyCache,
		proxyProber:       proxyProber,
		stopCh:            make(chan struct{}),
	}
}

func (s *AutoFailoverProxyPoolService) Start() {
	if s == nil || s.proxyRepo == nil || s.settingService == nil || s.proxyLatencyCache == nil || s.proxyProber == nil {
		return
	}

	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.refreshPoolHealth(context.Background())

			ticker := time.NewTicker(autoFailoverProxyPoolHealthCheckInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					s.refreshPoolHealth(context.Background())
				case <-s.stopCh:
					return
				}
			}
		}()
	})
}

func (s *AutoFailoverProxyPoolService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *AutoFailoverProxyPoolService) SupportsAccount(account *Account) bool {
	if account == nil {
		return false
	}
	switch account.Platform {
	case PlatformOpenAI, PlatformAntigravity:
		return true
	default:
		return false
	}
}

func (s *AutoFailoverProxyPoolService) BuildCandidates(ctx context.Context, account *Account) ([]ProxyFailoverCandidate, error) {
	if account == nil {
		return nil, nil
	}

	currentProxy, currentPoolEnabled, err := s.loadCurrentProxy(ctx, account)
	if err != nil {
		return nil, err
	}

	poolProxies, err := s.loadPoolProxies(ctx)
	if err != nil {
		return nil, err
	}

	runtimeTargets := make([]*Proxy, 0, len(poolProxies)+1)
	runtimeTargets = append(runtimeTargets, poolProxies...)
	if currentProxy != nil {
		runtimeTargets = append(runtimeTargets, currentProxy)
	}
	latencies := s.loadProxyRuntimeInfo(ctx, runtimeTargets...)
	candidates := make([]ProxyFailoverCandidate, 0, len(poolProxies)+1)

	if currentProxy != nil {
		currentCandidate := ProxyFailoverCandidate{
			ProxyID:  &currentProxy.ID,
			Proxy:    cloneProxy(currentProxy),
			ProxyURL: currentProxy.URL(),
			Source:   "account_proxy",
		}
		// 兼容历史账号：显式绑定了非池代理时仍优先尝试；显式绑定的是池代理时，
		// 仅在其未处于冷却期时作为首选。
		if !currentPoolEnabled || !isProxyCoolingDown(latencies[currentProxy.ID]) {
			candidates = append(candidates, currentCandidate)
		}
	}

	sortPoolProxyCandidates(poolProxies, latencies)
	for _, proxy := range poolProxies {
		if proxy == nil {
			continue
		}
		if proxy.Status != StatusActive {
			continue
		}
		if currentProxy != nil && proxy.ID == currentProxy.ID {
			continue
		}
		if isProxyCoolingDown(latencies[proxy.ID]) {
			continue
		}
		id := proxy.ID
		candidates = append(candidates, ProxyFailoverCandidate{
			ProxyID:  &id,
			Proxy:    cloneProxy(proxy),
			ProxyURL: proxy.URL(),
			Source:   "auto_failover_pool",
		})
	}

	// 对不使用代理的旧账号保持兼容：只有在没有显式代理且没有池成员时，才退回直连。
	if len(candidates) == 0 && account.ProxyID == nil {
		candidates = append(candidates, ProxyFailoverCandidate{
			ProxyID:  nil,
			Proxy:    nil,
			ProxyURL: "",
			Source:   "direct",
		})
	}

	return candidates, nil
}

func (s *AutoFailoverProxyPoolService) ResolveProxyURL(ctx context.Context, account *Account) (string, *int64, string, error) {
	if s == nil || !s.SupportsAccount(account) {
		proxyURL, proxyID, err := s.currentAccountProxyURL(ctx, account)
		return proxyURL, proxyID, "account_proxy", err
	}

	candidates, err := s.BuildCandidates(ctx, account)
	if err != nil {
		return "", nil, "", err
	}
	if len(candidates) == 0 {
		return "", nil, "", nil
	}
	first := candidates[0]
	return first.ProxyURL, first.ProxyID, first.Source, nil
}

func (s *AutoFailoverProxyPoolService) DoHTTPRequest(
	ctx context.Context,
	account *Account,
	req *http.Request,
	do func(*http.Request, string) (*http.Response, error),
) (*http.Response, error) {
	if req == nil || do == nil || s == nil || !s.SupportsAccount(account) {
		proxyURL, _, err := s.currentAccountProxyURL(ctx, account)
		if err != nil {
			return nil, err
		}
		return do(req, proxyURL)
	}

	candidates, err := s.BuildCandidates(ctx, account)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return do(req, "")
	}

	for idx, candidate := range candidates {
		attemptReq, cloneErr := cloneRequestForRetry(req)
		if cloneErr != nil {
			return nil, cloneErr
		}

		startedAt := time.Now()
		resp, callErr := do(attemptReq, candidate.ProxyURL)
		latencyMs := time.Since(startedAt).Milliseconds()

		if callErr != nil {
			retry, reason := s.ShouldRetryError(callErr)
			if retry {
				s.RecordFailure(ctx, candidate.ProxyID, reason, true)
				if idx+1 < len(candidates) {
					continue
				}
			}
			return nil, callErr
		}

		retry, reason, body, bodyErr := s.shouldRetryHTTPResponse(resp)
		if bodyErr != nil {
			return nil, bodyErr
		}
		if retry {
			s.RecordFailure(ctx, candidate.ProxyID, reason, true)
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			if idx+1 < len(candidates) {
				continue
			}
			if resp != nil {
				resp.Body = io.NopCloser(bytes.NewReader(body))
			}
			return resp, nil
		}

		s.RecordSuccess(ctx, candidate.ProxyID, &latencyMs)
		if err := s.PersistSelectedProxy(ctx, account, candidate, "proxy_auto_failover_success", nil, ""); err != nil {
			slog.Warn("proxy_auto_failover.persist_selected_failed", "account_id", account.ID, "error", err)
		}
		return resp, nil
	}

	return do(req, "")
}

func (s *AutoFailoverProxyPoolService) ShouldRetryError(err error) (bool, string) {
	if err == nil {
		return false, ""
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true, sanitizeProxyFailureReason(err.Error())
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true, sanitizeProxyFailureReason(err.Error())
	}
	var tlsErr *tls.RecordHeaderError
	if errors.As(err, &tlsErr) {
		return true, sanitizeProxyFailureReason(err.Error())
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "proxy") ||
		strings.Contains(msg, "socks") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "tls handshake") ||
		strings.Contains(msg, "certificate") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "context deadline exceeded") {
		return true, sanitizeProxyFailureReason(err.Error())
	}

	return false, ""
}

func (s *AutoFailoverProxyPoolService) RecordFailure(ctx context.Context, proxyID *int64, reason string, enterCooldown bool) {
	if s == nil || proxyID == nil || *proxyID <= 0 {
		return
	}
	now := time.Now()
	nowUnix := now.Unix()
	updated := s.mergeProxyRuntimeInfo(ctx, *proxyID, func(info *ProxyLatencyInfo) {
		info.Success = false
		info.Message = reason
		info.LastFailReason = reason
		info.LastFailAtUnix = ptrInt64(nowUnix)
		if enterCooldown {
			until := now.Add(autoFailoverProxyCooldownDuration).Unix()
			info.CooldownUntilUnix = ptrInt64(until)
			info.HealthStatus = "cooldown"
		} else {
			info.CooldownUntilUnix = nil
			info.HealthStatus = "failed"
		}
		info.UpdatedAt = now
	})
	if updated == nil {
		return
	}
}

func (s *AutoFailoverProxyPoolService) RecordSuccess(ctx context.Context, proxyID *int64, latencyMs *int64) {
	if s == nil || proxyID == nil || *proxyID <= 0 {
		return
	}
	now := time.Now()
	nowUnix := now.Unix()
	updated := s.mergeProxyRuntimeInfo(ctx, *proxyID, func(info *ProxyLatencyInfo) {
		info.Success = true
		info.Message = "Proxy is accessible"
		info.LatencyMs = latencyMs
		info.HealthStatus = "healthy"
		info.CooldownUntilUnix = nil
		info.LastRecoveredAtUnix = ptrInt64(nowUnix)
		info.UpdatedAt = now
	})
	if updated == nil {
		return
	}
}

func (s *AutoFailoverProxyPoolService) ClearCooldowns(ctx context.Context, proxyIDs []int64) error {
	if s == nil || s.proxyLatencyCache == nil {
		return nil
	}
	for _, proxyID := range dedupePositiveInt64s(proxyIDs) {
		s.mergeProxyRuntimeInfo(ctx, proxyID, func(info *ProxyLatencyInfo) {
			info.CooldownUntilUnix = nil
			if info.HealthStatus == "cooldown" {
				if info.Success {
					info.HealthStatus = "healthy"
				} else {
					info.HealthStatus = ""
				}
			}
			info.UpdatedAt = time.Now()
		})
	}
	return nil
}

func (s *AutoFailoverProxyPoolService) PersistSelectedProxy(
	ctx context.Context,
	account *Account,
	candidate ProxyFailoverCandidate,
	switchReason string,
	failedProxyID *int64,
	failedReason string,
) error {
	if s == nil || account == nil || candidate.ProxyID == nil || *candidate.ProxyID <= 0 {
		return nil
	}
	if account.ProxyID != nil && *account.ProxyID == *candidate.ProxyID {
		return nil
	}

	now := time.Now().Format(time.RFC3339)
	state := map[string]any{
		"active_proxy_id":      *candidate.ProxyID,
		"active_proxy_source":  candidate.Source,
		"last_switch_at":       now,
		"last_switch_reason":   switchReason,
		"last_failed_proxy_id": nil,
		"last_failed_at":       "",
		"last_failed_reason":   "",
	}
	if failedProxyID != nil && *failedProxyID > 0 {
		state["last_failed_proxy_id"] = *failedProxyID
		state["last_failed_at"] = now
		state["last_failed_reason"] = failedReason
	}

	account.ProxyID = candidate.ProxyID
	account.Proxy = cloneProxy(candidate.Proxy)
	if account.Extra == nil {
		account.Extra = make(map[string]any)
	}
	account.Extra["proxy_failover_state"] = state

	if account.ID <= 0 || s.accountRepo == nil {
		s.incrementSwitchCount(ctx, candidate.ProxyID)
		return nil
	}

	fresh, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil {
		return err
	}

	fresh.ProxyID = candidate.ProxyID
	if candidate.Proxy != nil {
		fresh.Proxy = cloneProxy(candidate.Proxy)
	}
	if fresh.Extra == nil {
		fresh.Extra = make(map[string]any)
	}
	fresh.Extra["proxy_failover_state"] = state

	if err := s.accountRepo.Update(ctx, fresh); err != nil {
		return err
	}

	s.incrementSwitchCount(ctx, candidate.ProxyID)
	return nil
}

func (s *AutoFailoverProxyPoolService) refreshPoolHealth(ctx context.Context) {
	proxies, err := s.loadPoolProxies(ctx)
	if err != nil {
		slog.Warn("proxy_auto_failover.refresh_pool_failed", "error", err)
		return
	}

	for _, proxy := range proxies {
		if proxy == nil || proxy.Status != StatusActive {
			continue
		}

		probeCtx, cancel := context.WithTimeout(ctx, autoFailoverProxyProbeTimeout)
		exitInfo, latencyMs, err := s.proxyProber.ProbeProxy(probeCtx, proxy.URL())
		cancel()
		if err != nil {
			s.RecordFailure(ctx, ptrInt64(proxy.ID), sanitizeProxyFailureReason(err.Error()), true)
			continue
		}

		s.mergeProxyRuntimeInfo(ctx, proxy.ID, func(info *ProxyLatencyInfo) {
			info.Success = true
			info.Message = "Proxy is accessible"
			info.LatencyMs = ptrInt64(latencyMs)
			info.IPAddress = exitInfo.IP
			info.Country = exitInfo.Country
			info.CountryCode = exitInfo.CountryCode
			info.Region = exitInfo.Region
			info.City = exitInfo.City
			info.HealthStatus = "healthy"
			info.CooldownUntilUnix = nil
			nowUnix := time.Now().Unix()
			info.LastRecoveredAtUnix = ptrInt64(nowUnix)
			info.UpdatedAt = time.Now()
		})
	}
}

func (s *AutoFailoverProxyPoolService) shouldRetryHTTPResponse(resp *http.Response) (bool, string, []byte, error) {
	if resp == nil {
		return false, "", nil, nil
	}
	if resp.StatusCode == http.StatusProxyAuthRequired {
		body, err := readAndRestoreBody(resp)
		return true, "proxy authentication required", body, err
	}
	if resp.StatusCode < 400 {
		return false, "", nil, nil
	}

	body, err := readAndRestoreBody(resp)
	if err != nil {
		return false, "", nil, err
	}
	if httputil.IsCloudflareChallengeResponse(resp.StatusCode, resp.Header, body) {
		return true, "cloudflare challenge", body, nil
	}

	bodyMsg := strings.ToLower(string(body))
	if strings.Contains(bodyMsg, "proxy connect") ||
		strings.Contains(bodyMsg, "proxy error") ||
		strings.Contains(bodyMsg, "tunnel connection failed") ||
		strings.Contains(bodyMsg, "dial tcp") ||
		strings.Contains(bodyMsg, "connection refused") ||
		strings.Contains(bodyMsg, "i/o timeout") {
		return true, sanitizeProxyFailureReason(strings.TrimSpace(string(body))), body, nil
	}

	return false, "", body, nil
}

func (s *AutoFailoverProxyPoolService) currentAccountProxyURL(ctx context.Context, account *Account) (string, *int64, error) {
	if account == nil || account.ProxyID == nil {
		return "", nil, nil
	}
	if account.Proxy != nil {
		id := account.Proxy.ID
		return account.Proxy.URL(), &id, nil
	}
	if s == nil || s.proxyRepo == nil {
		return "", nil, nil
	}

	proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil {
		return "", nil, err
	}
	if proxy == nil {
		return "", nil, nil
	}
	return proxy.URL(), &proxy.ID, nil
}

func (s *AutoFailoverProxyPoolService) loadCurrentProxy(ctx context.Context, account *Account) (*Proxy, bool, error) {
	if account == nil || account.ProxyID == nil || s.proxyRepo == nil {
		return nil, false, nil
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil {
		return nil, false, err
	}
	if proxy == nil {
		return nil, false, nil
	}

	poolSet, err := s.poolProxyIDSet(ctx)
	if err != nil {
		return proxy, false, err
	}
	_, enabled := poolSet[proxy.ID]
	proxy.AutoFailoverPoolEnabled = enabled
	return proxy, enabled, nil
}

func (s *AutoFailoverProxyPoolService) loadPoolProxies(ctx context.Context) ([]*Proxy, error) {
	if s == nil || s.proxyRepo == nil || s.settingService == nil {
		return nil, nil
	}
	poolIDs, err := s.settingService.GetAutoFailoverProxyPoolIDs(ctx)
	if err != nil || len(poolIDs) == 0 {
		return nil, err
	}

	proxies, err := s.proxyRepo.ListByIDs(ctx, poolIDs)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]*Proxy, len(proxies))
	for i := range proxies {
		proxy := proxies[i]
		proxy.AutoFailoverPoolEnabled = true
		proxyCopy := proxy
		byID[proxy.ID] = &proxyCopy
	}

	result := make([]*Proxy, 0, len(poolIDs))
	for _, id := range poolIDs {
		if proxy, ok := byID[id]; ok {
			result = append(result, proxy)
		}
	}
	return result, nil
}

func (s *AutoFailoverProxyPoolService) poolProxyIDSet(ctx context.Context) (map[int64]struct{}, error) {
	ids, err := s.settingService.GetAutoFailoverProxyPoolIDs(ctx)
	if err != nil {
		return nil, err
	}
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set, nil
}

func (s *AutoFailoverProxyPoolService) loadProxyRuntimeInfo(ctx context.Context, proxies ...*Proxy) map[int64]*ProxyLatencyInfo {
	infoMap := make(map[int64]*ProxyLatencyInfo)
	if s == nil || s.proxyLatencyCache == nil {
		return infoMap
	}

	ids := make([]int64, 0, len(proxies))
	seen := make(map[int64]struct{}, len(proxies))
	for _, proxy := range proxies {
		if proxy == nil {
			continue
		}
		if _, ok := seen[proxy.ID]; ok {
			continue
		}
		seen[proxy.ID] = struct{}{}
		ids = append(ids, proxy.ID)
	}
	if len(ids) == 0 {
		return infoMap
	}

	latencies, err := s.proxyLatencyCache.GetProxyLatencies(ctx, ids)
	if err != nil {
		slog.Warn("proxy_auto_failover.load_runtime_failed", "error", err)
		return infoMap
	}
	return latencies
}

func (s *AutoFailoverProxyPoolService) mergeProxyRuntimeInfo(ctx context.Context, proxyID int64, apply func(*ProxyLatencyInfo)) *ProxyLatencyInfo {
	if s == nil || s.proxyLatencyCache == nil || proxyID <= 0 || apply == nil {
		return nil
	}

	latencies, err := s.proxyLatencyCache.GetProxyLatencies(ctx, []int64{proxyID})
	if err != nil {
		slog.Warn("proxy_auto_failover.load_runtime_failed", "proxy_id", proxyID, "error", err)
		return nil
	}

	info := &ProxyLatencyInfo{UpdatedAt: time.Now()}
	if existing := latencies[proxyID]; existing != nil {
		cloned := *existing
		info = &cloned
	}
	apply(info)

	if err := s.proxyLatencyCache.SetProxyLatency(ctx, proxyID, info); err != nil {
		slog.Warn("proxy_auto_failover.store_runtime_failed", "proxy_id", proxyID, "error", err)
		return nil
	}
	return info
}

func (s *AutoFailoverProxyPoolService) incrementSwitchCount(ctx context.Context, proxyID *int64) {
	if s == nil || proxyID == nil || *proxyID <= 0 {
		return
	}
	s.mergeProxyRuntimeInfo(ctx, *proxyID, func(info *ProxyLatencyInfo) {
		current := int64(0)
		if info.FailoverSwitchCount != nil {
			current = *info.FailoverSwitchCount
		}
		current++
		info.FailoverSwitchCount = ptrInt64(current)
		info.UpdatedAt = time.Now()
	})
}

func cloneRequestForRetry(req *http.Request) (*http.Request, error) {
	if req == nil {
		return nil, nil
	}

	if req.Body == nil {
		return req.Clone(req.Context()), nil
	}

	if req.GetBody == nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body for proxy retry: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}

	cloned := req.Clone(req.Context())
	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("clone request body for proxy retry: %w", err)
	}
	cloned.Body = body
	cloned.GetBody = req.GetBody
	return cloned, nil
}

func readAndRestoreBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func sanitizeProxyFailureReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "proxy unavailable"
	}
	if len(reason) > 240 {
		return reason[:240]
	}
	return reason
}

func sortPoolProxyCandidates(proxies []*Proxy, info map[int64]*ProxyLatencyInfo) {
	sort.SliceStable(proxies, func(i, j int) bool {
		left := info[proxies[i].ID]
		right := info[proxies[j].ID]

		leftRank := proxyHealthRank(left)
		rightRank := proxyHealthRank(right)
		if leftRank != rightRank {
			return leftRank < rightRank
		}

		leftScore := int64(-1)
		rightScore := int64(-1)
		if left != nil && left.QualityScore != nil {
			leftScore = int64(*left.QualityScore)
		}
		if right != nil && right.QualityScore != nil {
			rightScore = int64(*right.QualityScore)
		}
		if leftScore != rightScore {
			return leftScore > rightScore
		}

		leftLatency := int64(1<<62 - 1)
		rightLatency := int64(1<<62 - 1)
		if left != nil && left.LatencyMs != nil {
			leftLatency = *left.LatencyMs
		}
		if right != nil && right.LatencyMs != nil {
			rightLatency = *right.LatencyMs
		}
		if leftLatency != rightLatency {
			return leftLatency < rightLatency
		}

		return proxies[i].ID < proxies[j].ID
	})
}

func proxyHealthRank(info *ProxyLatencyInfo) int {
	if info == nil {
		return 1
	}
	if isProxyCoolingDown(info) {
		return 3
	}
	switch info.HealthStatus {
	case "healthy":
		return 0
	case "failed":
		return 2
	case "cooldown":
		return 3
	default:
		if info.Success {
			return 0
		}
		if info.Message != "" {
			return 2
		}
		return 1
	}
}

func isProxyCoolingDown(info *ProxyLatencyInfo) bool {
	if info == nil || info.CooldownUntilUnix == nil {
		return false
	}
	return *info.CooldownUntilUnix > time.Now().Unix()
}

func cloneProxy(proxy *Proxy) *Proxy {
	if proxy == nil {
		return nil
	}
	cloned := *proxy
	return &cloned
}

func ptrInt64(value int64) *int64 {
	return &value
}
