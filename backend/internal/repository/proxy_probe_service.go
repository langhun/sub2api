package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func NewProxyExitInfoProber(cfg *config.Config) service.ProxyExitInfoProber {
	insecure := false
	allowPrivate := false
	validateResolvedIP := true
	maxResponseBytes := defaultProxyProbeResponseMaxBytes
	if cfg != nil {
		insecure = cfg.Security.ProxyProbe.InsecureSkipVerify
		allowPrivate = cfg.Security.URLAllowlist.AllowPrivateHosts
		validateResolvedIP = cfg.Security.URLAllowlist.Enabled
		if cfg.Gateway.ProxyProbeResponseReadMaxBytes > 0 {
			maxResponseBytes = cfg.Gateway.ProxyProbeResponseReadMaxBytes
		}
	}
	if insecure {
		log.Printf("[ProxyProbe] Warning: insecure_skip_verify is not allowed and will cause probe failure.")
	}
	return &proxyProbeService{
		insecureSkipVerify: insecure,
		allowPrivateHosts:  allowPrivate,
		validateResolvedIP: validateResolvedIP,
		maxResponseBytes:   maxResponseBytes,
		serverPublicIP:     strings.TrimSpace(os.Getenv("SUB2API_SERVER_PUBLIC_IP")),
	}
}

const (
	defaultProxyProbeTimeout          = 10 * time.Second
	defaultProxyProbeResponseMaxBytes = int64(1024 * 1024)
)

// probeURLs 按优先级排列的探测 URL 列表
// 某些 AI API 专用代理只允许访问特定域名，因此需要多个备选
type proxyProbeURL struct {
	url    string
	parser string // "ip-api" or "httpbin"
}

var probeURLs = []proxyProbeURL{
	{"https://chatgpt.com/cdn-cgi/trace", "cf-trace"},
	{"https://www.cloudflare.com/cdn-cgi/trace", "cf-trace"},
}

type proxyProbeService struct {
	insecureSkipVerify bool
	allowPrivateHosts  bool
	validateResolvedIP bool
	maxResponseBytes   int64
	serverPublicIP     string
}

func (s *proxyProbeService) ProbeProxy(ctx context.Context, proxyURL string) (*service.ProxyExitInfo, int64, error) {
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:           proxyURL,
		Timeout:            defaultProxyProbeTimeout,
		InsecureSkipVerify: s.insecureSkipVerify,
		ValidateResolvedIP: s.validateResolvedIP,
		AllowPrivateHosts:  s.allowPrivateHosts,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create proxy client: %w", err)
	}

	var lastErr error
	for _, probe := range probeURLs {
		exitInfo, latencyMs, err := s.probeWithURL(ctx, client, probe.url, probe.parser)
		if err == nil {
			return exitInfo, latencyMs, nil
		}
		lastErr = err
	}

	return nil, 0, fmt.Errorf("all probe URLs failed, last error: %w", lastErr)
}

func (s *proxyProbeService) probeWithURL(ctx context.Context, client *http.Client, url string, parser string) (*service.ProxyExitInfo, int64, error) {
	startTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("proxy connection failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	latencyMs := time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		return nil, latencyMs, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	maxResponseBytes := s.maxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultProxyProbeResponseMaxBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, latencyMs, fmt.Errorf("failed to read response: %w", err)
	}
	if int64(len(body)) > maxResponseBytes {
		return nil, latencyMs, fmt.Errorf("proxy probe response exceeds limit: %d", maxResponseBytes)
	}

	switch parser {
	case "cf-trace":
		return s.parseCFTrace(body, latencyMs)
	case "ip-api":
		return s.parseIPAPI(body, latencyMs)
	case "httpbin":
		return s.parseHTTPBin(body, latencyMs)
	default:
		return nil, latencyMs, fmt.Errorf("unknown parser: %s", parser)
	}
}

func (s *proxyProbeService) parseIPAPI(body []byte, latencyMs int64) (*service.ProxyExitInfo, int64, error) {
	var ipInfo struct {
		Status      string `json:"status"`
		Message     string `json:"message"`
		Query       string `json:"query"`
		City        string `json:"city"`
		Region      string `json:"region"`
		RegionName  string `json:"regionName"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
	}

	if err := json.Unmarshal(body, &ipInfo); err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, latencyMs, fmt.Errorf("failed to parse response: %w (body: %s)", err, preview)
	}
	if strings.ToLower(ipInfo.Status) != "success" {
		if ipInfo.Message == "" {
			ipInfo.Message = "ip-api request failed"
		}
		return nil, latencyMs, fmt.Errorf("ip-api request failed: %s", ipInfo.Message)
	}

	region := ipInfo.RegionName
	if region == "" {
		region = ipInfo.Region
	}
	return &service.ProxyExitInfo{
		IP:          ipInfo.Query,
		City:        ipInfo.City,
		Region:      region,
		Country:     ipInfo.Country,
		CountryCode: ipInfo.CountryCode,
	}, latencyMs, s.validateExitInfo(ipInfo.Query, ipInfo.CountryCode)
}

func (s *proxyProbeService) parseCFTrace(body []byte, latencyMs int64) (*service.ProxyExitInfo, int64, error) {
	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
	values := make(map[string]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	ip := strings.TrimSpace(values["ip"])
	countryCode := strings.TrimSpace(values["loc"])
	info := &service.ProxyExitInfo{
		IP:          ip,
		Country:     countryCode,
		CountryCode: countryCode,
	}
	return info, latencyMs, s.validateExitInfo(ip, countryCode)
}

func (s *proxyProbeService) parseHTTPBin(body []byte, latencyMs int64) (*service.ProxyExitInfo, int64, error) {
	// httpbin.org/ip 返回格式: {"origin": "1.2.3.4"}
	var result struct {
		Origin string `json:"origin"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, latencyMs, fmt.Errorf("failed to parse httpbin response: %w", err)
	}
	if result.Origin == "" {
		return nil, latencyMs, fmt.Errorf("httpbin: no IP found in response")
	}
	return &service.ProxyExitInfo{
		IP: result.Origin,
	}, latencyMs, s.validateExitInfo(result.Origin, "")
}

func (s *proxyProbeService) validateExitInfo(ip, countryCode string) error {
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("proxy probe returned empty exit ip")
	}
	if s.serverPublicIP != "" && strings.TrimSpace(ip) == s.serverPublicIP {
		return fmt.Errorf("proxy probe leaked server public ip: %s", ip)
	}
	if strings.TrimSpace(countryCode) == "" {
		return fmt.Errorf("proxy probe country is unknown")
	}
	return nil
}
