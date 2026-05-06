//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type proxyRepoStubForFailedList struct {
	proxyRepoStub

	pages        map[int][]ProxyWithAccountCount
	total        int64
	calls        []pagination.PaginationParams
	lastProtocol string
	lastStatus   string
	lastSearch   string
}

func (s *proxyRepoStubForFailedList) ListWithFiltersAndAccountCount(_ context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	s.calls = append(s.calls, params)
	s.lastProtocol = protocol
	s.lastStatus = status
	s.lastSearch = search

	rows := append([]ProxyWithAccountCount(nil), s.pages[params.Page]...)
	return rows, &pagination.PaginationResult{
		Total:    s.total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

type proxyLatencyCacheStub struct {
	data      map[int64]*ProxyLatencyInfo
	requested [][]int64
}

func (s *proxyLatencyCacheStub) GetProxyLatencies(_ context.Context, proxyIDs []int64) (map[int64]*ProxyLatencyInfo, error) {
	s.requested = append(s.requested, append([]int64(nil), proxyIDs...))
	result := make(map[int64]*ProxyLatencyInfo, len(proxyIDs))
	for _, proxyID := range proxyIDs {
		if info, ok := s.data[proxyID]; ok {
			result[proxyID] = info
		}
	}
	return result, nil
}

func (s *proxyLatencyCacheStub) SetProxyLatency(_ context.Context, _ int64, _ *ProxyLatencyInfo) error {
	return nil
}

func TestAdminService_ListProxiesWithAccountCount_FailedStatusScansAllPages(t *testing.T) {
	page1 := make([]ProxyWithAccountCount, 0, 1000)
	for i := 1; i <= 1000; i++ {
		page1 = append(page1, ProxyWithAccountCount{
			Proxy: Proxy{ID: int64(i), Name: "proxy"},
		})
	}
	page2 := []ProxyWithAccountCount{
		{Proxy: Proxy{ID: 1001, Name: "proxy-1001"}},
	}

	repo := &proxyRepoStubForFailedList{
		pages: map[int][]ProxyWithAccountCount{
			1: page1,
			2: page2,
		},
		total: 1001,
	}
	cache := &proxyLatencyCacheStub{
		data: map[int64]*ProxyLatencyInfo{
			1001: {Success: false},
		},
	}
	svc := &adminServiceImpl{
		proxyRepo:         repo,
		proxyLatencyCache: cache,
	}

	proxies, total, err := svc.ListProxiesWithAccountCount(context.Background(), 1, 20, "http", "failed", "needle", "created_at", "DESC")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, proxies, 1)
	require.Equal(t, int64(1001), proxies[0].ID)
	require.Equal(t, "failed", proxies[0].LatencyStatus)

	require.Len(t, repo.calls, 2)
	require.Equal(t, 1, repo.calls[0].Page)
	require.Equal(t, 2, repo.calls[1].Page)
	require.Equal(t, 1000, repo.calls[0].PageSize)
	require.Equal(t, "http", repo.lastProtocol)
	require.Equal(t, "", repo.lastStatus)
	require.Equal(t, "needle", repo.lastSearch)
	require.Len(t, cache.requested, 2)
}
