package redpacket

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/google/uuid"
)

const (
	defaultExpiryInterval = time.Minute
	// Keep the old key until all application instances run the extracted worker.
	defaultExpiryLeaseKey = "balance:redpacket:expiry:leader"
	defaultExpiryLeaseTTL = 5 * time.Minute
)

type expiryWorker struct {
	expire contractExpiryRefunder
	leases contract.SingletonLeaseCoordinator
	config ExpiryWorkerConfig
	owner  string
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

// NewExpiryWorker creates the module-owned worker. Runtime must call Start
// after construction and Stop during application cleanup.
func NewExpiryWorker(deps ExpiryWorkerDependencies) ExpiryWorker {
	config := deps.Config
	if config.Interval <= 0 {
		config.Interval = defaultExpiryInterval
	}
	if config.LeaseKey == "" {
		config.LeaseKey = defaultExpiryLeaseKey
	}
	if config.LeaseTTL <= 0 {
		config.LeaseTTL = defaultExpiryLeaseTTL
	}
	return &expiryWorker{expire: deps.Expire, leases: deps.Leases, config: config, owner: uuid.NewString()}
}

func (w *expiryWorker) Start() {
	if w == nil || w.expire == nil || w.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		w.run(ctx)
	}()
}

func (w *expiryWorker) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		if w.done != nil {
			<-w.done
		}
	})
}

func (w *expiryWorker) run(ctx context.Context) {
	w.runOnce(ctx)
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *expiryWorker) runOnce(ctx context.Context) {
	if w == nil || w.expire == nil {
		return
	}
	lease := contract.Lease(noopLease{})
	if w.leases != nil {
		lockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		acquired, ok, err := w.leases.AcquireSingletonLease(lockCtx, w.config.LeaseKey, w.owner, w.config.LeaseTTL)
		cancel()
		if err != nil {
			slog.Error("acquire activity red-packet expiry lease", "error", err)
			return
		}
		if !ok {
			return
		}
		if acquired != nil {
			lease = acquired
		}
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := lease.Release(releaseCtx); err != nil {
			slog.Error("release activity red-packet expiry lease", "error", err)
		}
	}()
	if _, err := w.expire.RefundExpired(ctx); err != nil {
		slog.Error("refund expired activity red packets", "error", err)
	}
}

type noopLease struct{}

func (noopLease) Release(context.Context) error { return nil }
