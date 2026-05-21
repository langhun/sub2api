package service

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type ProxySubscriptionRuntimeRehydrateService struct {
	subscriptionService *ProxySubscriptionService
	sourceRepo          ProxySubscriptionSourceRepository
	cfg                 *config.Config

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewProxySubscriptionRuntimeRehydrateService(
	subscriptionService *ProxySubscriptionService,
	sourceRepo ProxySubscriptionSourceRepository,
	cfg *config.Config,
) *ProxySubscriptionRuntimeRehydrateService {
	return &ProxySubscriptionRuntimeRehydrateService{
		subscriptionService: subscriptionService,
		sourceRepo:          sourceRepo,
		cfg:                 cfg,
		stopCh:              make(chan struct{}),
	}
}

func (s *ProxySubscriptionRuntimeRehydrateService) Start() {
	if s == nil || s.subscriptionService == nil || s.sourceRepo == nil || s.cfg == nil || !s.cfg.ProxySubscriptions.Enabled {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.rehydrate()
		}()
	})
}

func (s *ProxySubscriptionRuntimeRehydrateService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *ProxySubscriptionRuntimeRehydrateService) rehydrate() {
	sources, err := s.sourceRepo.ListEnabled(context.Background())
	if err != nil {
		slog.Warn("proxy_subscription_runtime_rehydrate.list_sources_failed", "error", err)
		return
	}
	for i := range sources {
		select {
		case <-s.stopCh:
			return
		default:
		}
		if _, err := s.subscriptionService.RefreshSource(context.Background(), sources[i].ID); err != nil {
			slog.Warn("proxy_subscription_runtime_rehydrate.refresh_failed", "source_id", sources[i].ID, "error", err)
		}
	}
}
