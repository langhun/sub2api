//go:build unit

package service

import "context"

type proxyRepoStubForUnassign struct {
	proxyRepoStub

	summaries map[int64][]ProxyAccountSummary
	listCalls []int64
}

func (s *proxyRepoStubForUnassign) ListAccountSummariesByProxyID(_ context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	s.listCalls = append(s.listCalls, proxyID)
	return append([]ProxyAccountSummary(nil), s.summaries[proxyID]...), nil
}
