package service

import (
	"errors"
	"net/http"
)

func (s *AntigravityGatewayService) doUpstreamWithAutoFailover(account *Account, req *http.Request, override HTTPUpstream) (*http.Response, error) {
	if s == nil {
		return nil, nil
	}
	upstream := override
	if upstream == nil {
		upstream = s.httpUpstream
	}
	if upstream == nil {
		return nil, errors.New("http upstream not configured")
	}

	do := func(clonedReq *http.Request, proxyURL string) (*http.Response, error) {
		var accountID int64
		var concurrency int
		if account != nil {
			accountID = account.ID
			concurrency = account.Concurrency
		}
		return upstream.Do(clonedReq, proxyURL, accountID, concurrency)
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
