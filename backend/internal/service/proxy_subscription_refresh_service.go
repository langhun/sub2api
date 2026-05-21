package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type ProxySubscriptionRefreshService struct {
	sourceRepo ProxySubscriptionSourceRepository
	service    *ProxySubscriptionService
	cfg        *config.Config

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewProxySubscriptionRefreshService(
	sourceRepo ProxySubscriptionSourceRepository,
	service *ProxySubscriptionService,
	cfg *config.Config,
) *ProxySubscriptionRefreshService {
	return &ProxySubscriptionRefreshService{
		sourceRepo: sourceRepo,
		service:    service,
		cfg:        cfg,
		stopCh:     make(chan struct{}),
	}
}

func (s *ProxySubscriptionRefreshService) Start() {
	if s == nil || s.sourceRepo == nil || s.service == nil || s.cfg == nil || !s.cfg.ProxySubscriptions.Enabled {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.loop()
		}()
	})
}

func (s *ProxySubscriptionRefreshService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *ProxySubscriptionRefreshService) loop() {
	interval := time.Duration(s.cfg.ProxySubscriptions.RefreshScanIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.refreshDue(context.Background())
	for {
		select {
		case <-ticker.C:
			s.refreshDue(context.Background())
		case <-s.stopCh:
			return
		}
	}
}

func (s *ProxySubscriptionRefreshService) refreshDue(ctx context.Context) {
	limit := s.cfg.ProxySubscriptions.SyncConcurrency
	if limit <= 0 {
		limit = 2
	}
	items, err := s.sourceRepo.ListDueForRefresh(ctx, time.Now(), limit)
	if err != nil {
		slog.Warn("proxy_subscription_refresh.list_due_failed", "error", err)
		return
	}
	for i := range items {
		if _, err := s.service.RefreshSource(ctx, items[i].ID); err != nil {
			slog.Warn("proxy_subscription_refresh.refresh_failed", "source_id", items[i].ID, "error", err)
		}
	}
}
