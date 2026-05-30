package urlvalidator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ValidationOptions is kept for callers that already pass an options value.
// URL allowlist enforcement has been removed, so it intentionally has no fields.
type ValidationOptions struct{}

// ValidateHTTPURL validates and normalizes an outbound HTTP/HTTPS URL.
//
// The legacy allowlist/private-host options are intentionally ignored. Upstream
// URL allowlist enforcement has been removed, so callers only get syntax and
// scheme normalization here.
func ValidateHTTPURL(raw string, allowInsecureHTTP bool, opts ValidationOptions) (string, error) {
	baseURL, err := parseAndValidateBaseURL(raw)
	if err != nil {
		return "", err
	}
	_ = allowInsecureHTTP
	_ = opts
	baseURL.parsed.Path = strings.TrimRight(baseURL.parsed.Path, "/")
	baseURL.parsed.RawPath = ""
	return strings.TrimRight(baseURL.parsed.String(), "/"), nil
}

func ValidateURLFormat(raw string, allowInsecureHTTP bool) (string, error) {
	// 最小格式校验：仅保证 URL 可解析且 scheme 合规。
	baseURL, err := parseAndValidateBaseURL(raw)
	if err != nil {
		return "", err
	}
	_ = allowInsecureHTTP
	return strings.TrimRight(baseURL.trimmed, "/"), nil
}

func ValidateHTTPSURL(raw string, opts ValidationOptions) (string, error) {
	return ValidateHTTPURL(raw, true, opts)
}

// ValidateResolvedIP 验证 DNS 解析后的 IP 地址是否安全
// 用于防止 DNS Rebinding 攻击：在实际 HTTP 请求时调用此函数验证解析后的 IP
func ValidateResolvedIP(host string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("dns resolution failed: %w", err)
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("resolved ip %s is not allowed", ip.String())
		}
	}
	return nil
}

type parsedURL struct {
	trimmed string
	parsed  *url.URL
	host    string
}

func parseAndValidateBaseURL(raw string) (*parsedURL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid url: %s", trimmed)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return nil, fmt.Errorf("invalid url scheme: %s", parsed.Scheme)
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return nil, errors.New("invalid host")
	}

	if port := parsed.Port(); port != "" {
		num, err := strconv.Atoi(port)
		if err != nil || num <= 0 || num > 65535 {
			return nil, fmt.Errorf("invalid port: %s", port)
		}
	}

	return &parsedURL{
		trimmed: trimmed,
		parsed:  parsed,
		host:    host,
	}, nil
}
