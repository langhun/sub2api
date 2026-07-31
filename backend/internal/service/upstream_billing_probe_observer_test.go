package service

import (
	"context"
	"testing"
)

type upstreamBillingProbeSuccessObserverStub struct {
	called bool
}

func (s *upstreamBillingProbeSuccessObserverStub) OnUpstreamBillingProbeSuccess(_ context.Context, _ *Account, _ *UpstreamBillingProbeSnapshot) error {
	s.called = true
	return nil
}

func TestNotifyUpstreamBillingProbeSuccess(t *testing.T) {
	t.Cleanup(func() { SetUpstreamBillingProbeSuccessObserver(nil) })
	observer := &upstreamBillingProbeSuccessObserverStub{}
	SetUpstreamBillingProbeSuccessObserver(observer)
	if err := notifyUpstreamBillingProbeSuccess(context.Background(), &Account{}, &UpstreamBillingProbeSnapshot{}); err != nil {
		t.Fatalf("notify observer: %v", err)
	}
	if !observer.called {
		t.Fatal("observer was not called")
	}
}
