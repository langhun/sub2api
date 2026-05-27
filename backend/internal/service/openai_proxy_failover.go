package service

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

func (s *OpenAIGatewayService) resolveAutoFailoverProxyURL(_ interface{}, account *Account) string {
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func (s *OpenAIGatewayService) doUpstreamWithAutoFailover(
	account *Account,
	req *http.Request,
	profile *tlsfingerprint.Profile,
) (*http.Response, error) {
	proxyURL := ""
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if profile != nil {
		return s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, profile)
	}
	return s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
}
