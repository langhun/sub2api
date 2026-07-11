package service

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

const (
	redPacketExpiryLeaderLockKey = "balance:redpacket:expiry:leader"
	redPacketExpiryLeaderLockTTL = 5 * time.Minute
)

type RedPacketExpiryService struct {
	transfer *BalanceTransferService
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
	once     sync.Once
	lock     LeaderLockCache
	db       *sql.DB
	owner    string
}

func NewRedPacketExpiryService(transfer *BalanceTransferService, interval time.Duration) *RedPacketExpiryService {
	if interval <= 0 {
		interval = time.Minute
	}
	return &RedPacketExpiryService{transfer: transfer, interval: interval, owner: uuid.NewString()}
}

func (s *RedPacketExpiryService) SetLeaderLock(lock LeaderLockCache, db *sql.DB) {
	if s == nil {
		return
	}
	s.lock = lock
	s.db = db
}

func (s *RedPacketExpiryService) Start() {
	if s == nil || s.transfer == nil || s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		s.run(ctx)
	}()
}

func (s *RedPacketExpiryService) run(ctx context.Context) {
	s.runOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *RedPacketExpiryService) runOnce(ctx context.Context) {
	lockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	release, ok := tryAcquireSingletonLeaderLock(lockCtx, s.lock, s.db, redPacketExpiryLeaderLockKey, s.owner, redPacketExpiryLeaderLockTTL)
	cancel()
	if !ok {
		return
	}
	defer release()
	if err := s.transfer.ExpireRedPackets(ctx); err != nil {
		logger.LegacyPrintf("service.redpacket_expiry", "expire red packets: %v", err)
	}
}

func (s *RedPacketExpiryService) Stop() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		if s.done != nil {
			<-s.done
		}
	})
}

func ProvideRedPacketExpiryService(transfer *BalanceTransferService, lock LeaderLockCache, db *sql.DB) *RedPacketExpiryService {
	service := NewRedPacketExpiryService(transfer, time.Minute)
	service.SetLeaderLock(lock, db)
	service.Start()
	return service
}
