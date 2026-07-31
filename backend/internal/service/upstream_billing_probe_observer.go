package service

import (
	"context"
	"sync"
)

// UpstreamBillingProbeSuccessObserver is the fixed Overlay mount called only
// after a validated upstream billing snapshot has been persisted. It keeps the
// probe protocol independent from optional downstream accounting behavior.
type UpstreamBillingProbeSuccessObserver interface {
	OnUpstreamBillingProbeSuccess(ctx context.Context, account *Account, snapshot *UpstreamBillingProbeSnapshot) error
}

var upstreamBillingProbeSuccessObserver struct {
	sync.RWMutex
	value UpstreamBillingProbeSuccessObserver
}

// SetUpstreamBillingProbeSuccessObserver mounts or removes the optional
// observer during application composition.
func SetUpstreamBillingProbeSuccessObserver(observer UpstreamBillingProbeSuccessObserver) {
	upstreamBillingProbeSuccessObserver.Lock()
	defer upstreamBillingProbeSuccessObserver.Unlock()
	upstreamBillingProbeSuccessObserver.value = observer
}

func notifyUpstreamBillingProbeSuccess(ctx context.Context, account *Account, snapshot *UpstreamBillingProbeSnapshot) error {
	upstreamBillingProbeSuccessObserver.RLock()
	observer := upstreamBillingProbeSuccessObserver.value
	upstreamBillingProbeSuccessObserver.RUnlock()
	if observer == nil {
		return nil
	}
	return observer.OnUpstreamBillingProbeSuccess(ctx, account, snapshot)
}
