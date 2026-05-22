package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type testProxyRepo struct {
	getByIDFn func(ctx context.Context, id int64) (*Proxy, error)
}

func (r *testProxyRepo) Create(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Create call")
}

func (r *testProxyRepo) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	if r.getByIDFn == nil {
		panic("unexpected GetByID call")
	}
	return r.getByIDFn(ctx, id)
}

func (r *testProxyRepo) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	panic("unexpected ListByIDs call")
}

func (r *testProxyRepo) ListBySubscriptionSourceID(ctx context.Context, sourceID int64) ([]Proxy, error) {
	panic("unexpected ListBySubscriptionSourceID call")
}

func (r *testProxyRepo) FindBySubscriptionNodeID(ctx context.Context, nodeID int64) (*Proxy, error) {
	panic("unexpected FindBySubscriptionNodeID call")
}

func (r *testProxyRepo) FindByHostPortAuth(ctx context.Context, host string, port int, username, password string) (*Proxy, error) {
	panic("unexpected FindByHostPortAuth call")
}

func (r *testProxyRepo) Update(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Update call")
}

func (r *testProxyRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (r *testProxyRepo) List(ctx context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *testProxyRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *testProxyRepo) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFiltersAndAccountCount call")
}

func (r *testProxyRepo) ListActive(ctx context.Context) ([]Proxy, error) {
	panic("unexpected ListActive call")
}

func (r *testProxyRepo) ListActiveWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error) {
	panic("unexpected ListActiveWithAccountCount call")
}

func (r *testProxyRepo) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	panic("unexpected ExistsByHostPortAuth call")
}

func (r *testProxyRepo) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	panic("unexpected CountAccountsByProxyID call")
}

func (r *testProxyRepo) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	panic("unexpected ListAccountSummariesByProxyID call")
}

type testProxyConnectionTester struct {
	testFn func(ctx context.Context, proxyURL string) error
}

func (t *testProxyConnectionTester) TestConnection(ctx context.Context, proxyURL string) error {
	if t.testFn == nil {
		panic("unexpected TestConnection call")
	}
	return t.testFn(ctx, proxyURL)
}

func TestProxyService_TestConnection_Success(t *testing.T) {
	repo := &testProxyRepo{
		getByIDFn: func(ctx context.Context, id int64) (*Proxy, error) {
			return &Proxy{
				ID:       id,
				Protocol: "http",
				Host:     "127.0.0.1",
				Port:     7890,
				Username: "user",
				Password: "pass",
			}, nil
		},
	}
	var gotProxyURL string
	tester := &testProxyConnectionTester{
		testFn: func(ctx context.Context, proxyURL string) error {
			gotProxyURL = proxyURL
			return nil
		},
	}

	svc := NewProxyServiceWithTester(repo, tester)
	err := svc.TestConnection(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, "http://user:pass@127.0.0.1:7890", gotProxyURL)
}

func TestProxyService_TestConnection_GetByIDError(t *testing.T) {
	repoErr := errors.New("repo unavailable")
	repo := &testProxyRepo{
		getByIDFn: func(ctx context.Context, id int64) (*Proxy, error) {
			return nil, repoErr
		},
	}
	tester := &testProxyConnectionTester{
		testFn: func(ctx context.Context, proxyURL string) error {
			t.Fatalf("tester should not be called when repo fails")
			return nil
		},
	}

	svc := NewProxyServiceWithTester(repo, tester)
	err := svc.TestConnection(context.Background(), 11)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get proxy")
	require.ErrorIs(t, err, repoErr)
}

func TestProxyService_TestConnection_ProbeError(t *testing.T) {
	probeErr := errors.New("dial tcp timeout")
	repo := &testProxyRepo{
		getByIDFn: func(ctx context.Context, id int64) (*Proxy, error) {
			return &Proxy{
				ID:       id,
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     8080,
			}, nil
		},
	}
	tester := &testProxyConnectionTester{
		testFn: func(ctx context.Context, proxyURL string) error {
			require.Equal(t, "http://proxy.example.com:8080", proxyURL)
			return probeErr
		},
	}

	svc := NewProxyServiceWithTester(repo, tester)
	err := svc.TestConnection(context.Background(), 12)
	require.Error(t, err)
	require.Contains(t, err.Error(), "test proxy connection via http://proxy.example.com:8080")
	require.ErrorIs(t, err, probeErr)
}

func TestProxyService_TestConnection_NilProxyFromRepository(t *testing.T) {
	repo := &testProxyRepo{
		getByIDFn: func(ctx context.Context, id int64) (*Proxy, error) {
			return nil, nil
		},
	}
	tester := &testProxyConnectionTester{
		testFn: func(ctx context.Context, proxyURL string) error {
			t.Fatalf("tester should not be called when proxy is nil")
			return nil
		},
	}

	svc := NewProxyServiceWithTester(repo, tester)
	err := svc.TestConnection(context.Background(), 13)
	require.Error(t, err)
	require.Contains(t, err.Error(), "proxy is nil")
}
