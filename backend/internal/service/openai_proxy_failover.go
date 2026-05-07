package service

import (
	"context"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

func (s *OpenAIGatewayService) resolveAutoFailoverProxyURL(ctx context.Context, account *Account) string {
	if s == nil || account == nil {
		return ""
	}
	if s.proxyPool != nil && s.proxyPool.SupportsAccount(account) {
		proxyURL, _, _, err := s.proxyPool.ResolveProxyURL(ctx, account)
		if err == nil {
			return proxyURL
		}
	}
	if account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func (s *OpenAIGatewayService) doUpstreamWithAutoFailover(
	account *Account,
	req *http.Request,
	profile *tlsfingerprint.Profile,
) (*http.Response, error) {
	if s == nil {
		return nil, nil
	}

	do := func(clonedReq *http.Request, proxyURL string) (*http.Response, error) {
		if profile != nil {
			return s.httpUpstream.DoWithTLS(clonedReq, proxyURL, account.ID, account.Concurrency, profile)
		}
		return s.httpUpstream.Do(clonedReq, proxyURL, account.ID, account.Concurrency)
	}

	if s.proxyPool != nil && s.proxyPool.SupportsAccount(account) {
		return s.proxyPool.DoHTTPRequest(req.Context(), account, req, do)
	}

	proxyURL := ""
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return do(req, proxyURL)
}
