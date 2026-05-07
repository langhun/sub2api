package service

import "net/http"

func (s *AntigravityGatewayService) doUpstreamWithAutoFailover(account *Account, req *http.Request) (*http.Response, error) {
	if s == nil {
		return nil, nil
	}

	do := func(clonedReq *http.Request, proxyURL string) (*http.Response, error) {
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
