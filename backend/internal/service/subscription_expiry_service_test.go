package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type subscriptionExpiryRepoStub struct {
	userSubRepoNoop
	mu         sync.Mutex
	updateRuns int
	blockCh    chan struct{}
}

func (s *subscriptionExpiryRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	s.mu.Lock()
	s.updateRuns++
	s.mu.Unlock()
	if s.blockCh != nil {
		<-s.blockCh
	}
	return 0, nil
}

func (s *subscriptionExpiryRepoStub) List(context.Context, pagination.PaginationParams, *int64, *int64, string, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{}, nil
}

func (s *subscriptionExpiryRepoStub) runs() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateRuns
}

func TestSubscriptionExpiryService_StartIsIdempotent(t *testing.T) {
	repo := &subscriptionExpiryRepoStub{blockCh: make(chan struct{})}
	svc := NewSubscriptionExpiryService(repo, time.Hour)

	svc.Start()
	require.Eventually(t, func() bool {
		return repo.runs() == 1
	}, time.Second, 10*time.Millisecond)

	svc.Start()
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 1, repo.runs(), "second Start should not launch another loop")

	close(repo.blockCh)
	svc.Stop()
}
