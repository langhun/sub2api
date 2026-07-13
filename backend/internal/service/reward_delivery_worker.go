package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"
)

const (
	defaultRewardDeliveryPollInterval = 5 * time.Second
	defaultRewardDeliveryBatchSize    = 20
	defaultRewardDeliveryMaxAttempts  = 5
	defaultRewardDeliveryRetryDelay   = 30 * time.Second
	defaultRewardDeliveryStaleAfter   = 5 * time.Minute
)

type RewardDeliveryProcessor interface {
	ProcessRewardDelivery(ctx context.Context, delivery RewardDelivery) (string, error)
}

type RewardDeliveryWorkerOptions struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	RetryDelay   time.Duration
	StaleAfter   time.Duration
}

type RewardDeliveryWorker struct {
	store     RewardDeliveryStore
	processor RewardDeliveryProcessor
	opts      RewardDeliveryWorkerOptions
	now       func() time.Time
}

func NewRewardDeliveryWorker(store RewardDeliveryStore, processor RewardDeliveryProcessor, opts RewardDeliveryWorkerOptions) *RewardDeliveryWorker {
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultRewardDeliveryPollInterval
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultRewardDeliveryBatchSize
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = defaultRewardDeliveryMaxAttempts
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = defaultRewardDeliveryRetryDelay
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = defaultRewardDeliveryStaleAfter
	}
	return &RewardDeliveryWorker{store: store, processor: processor, opts: opts, now: time.Now}
}

func (w *RewardDeliveryWorker) Run(ctx context.Context) {
	if w == nil || w.store == nil || w.processor == nil {
		return
	}
	w.recoverStale(ctx)
	if err := w.RunOnce(ctx); err != nil && ctx.Err() == nil {
		slog.Warn("reward delivery worker run failed", "error", err)
	}

	ticker := time.NewTicker(w.opts.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.recoverStale(ctx)
			if err := w.RunOnce(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("reward delivery worker run failed", "error", err)
			}
		}
	}
}

func (w *RewardDeliveryWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.store == nil || w.processor == nil {
		return nil
	}
	now := w.now()
	deliveries, err := w.store.ClaimDue(ctx, now, w.opts.BatchSize)
	if err != nil {
		return err
	}
	var deliveryErrors []error
	for i := range deliveries {
		if err := w.processClaimed(ctx, deliveries[i]); err != nil && isRewardDeliveryPersistenceError(err) {
			deliveryErrors = append(deliveryErrors, err)
		}
	}
	return errors.Join(deliveryErrors...)
}

// RunByID attempts an immediate delivery after the eligibility transaction commits.
// A failed attempt is returned to pending with the same retry policy as the worker.
func (w *RewardDeliveryWorker) RunByID(ctx context.Context, id int64) error {
	if w == nil || w.store == nil || w.processor == nil {
		return nil
	}
	delivery, err := w.store.ClaimByID(ctx, id, w.now())
	if err != nil || delivery == nil {
		return err
	}
	return w.processClaimed(ctx, *delivery)
}

func (w *RewardDeliveryWorker) processClaimed(ctx context.Context, delivery RewardDelivery) error {
	processErr := w.store.ProcessClaimed(ctx, delivery.ID, func(txCtx context.Context, claimed RewardDelivery) (string, error) {
		return w.processor.ProcessRewardDelivery(txCtx, claimed)
	})
	if processErr == nil {
		return nil
	}

	var nextRetryAt *time.Time
	if delivery.Attempts < w.opts.MaxAttempts {
		next := w.now().Add(w.retryDelay(delivery.Attempts))
		nextRetryAt = &next
	}
	if err := w.store.MarkFailed(ctx, delivery.ID, processErr.Error(), nextRetryAt); err != nil {
		return errors.Join(processErr, rewardDeliveryPersistenceError{err: fmt.Errorf("mark reward delivery %d failed: %w", delivery.ID, err)})
	}
	return processErr
}

type rewardDeliveryPersistenceError struct {
	err error
}

func (e rewardDeliveryPersistenceError) Error() string { return e.err.Error() }
func (e rewardDeliveryPersistenceError) Unwrap() error { return e.err }

func isRewardDeliveryPersistenceError(err error) bool {
	var target rewardDeliveryPersistenceError
	return errors.As(err, &target)
}

func (w *RewardDeliveryWorker) recoverStale(ctx context.Context) {
	now := w.now()
	if _, err := w.store.RecoverStale(ctx, now.Add(-w.opts.StaleAfter), now); err != nil && ctx.Err() == nil {
		slog.Warn("recover stale reward deliveries failed", "error", err)
	}
}

func (w *RewardDeliveryWorker) retryDelay(attempts int) time.Duration {
	exponent := max(attempts-1, 0)
	multiplier := math.Pow(2, float64(min(exponent, 10)))
	return time.Duration(float64(w.opts.RetryDelay) * multiplier)
}
