package service

import (
	"context"
	"sync"
)

// RewardDeliveryWorkerRuntime owns the background delivery loop lifecycle.
type RewardDeliveryWorkerRuntime struct {
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func ProvideRewardDeliveryWorkerRuntime(store RewardDeliveryStore, processor *BlindBoxService) *RewardDeliveryWorkerRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	runtime := &RewardDeliveryWorkerRuntime{cancel: cancel, done: make(chan struct{})}
	worker := NewRewardDeliveryWorker(store, processor, RewardDeliveryWorkerOptions{})
	go func() {
		defer close(runtime.done)
		worker.Run(ctx)
	}()
	return runtime
}

func (r *RewardDeliveryWorkerRuntime) Stop() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		r.cancel()
		<-r.done
	})
}
