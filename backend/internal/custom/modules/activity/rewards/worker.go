package rewards

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 20
	defaultMaxAttempts  = 5
	defaultRetryDelay   = 30 * time.Second
	defaultStaleAfter   = 5 * time.Minute
)

type Processor interface {
	ProcessDelivery(ctx context.Context, delivery Delivery) (string, error)
}

type WorkerOptions struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	RetryDelay   time.Duration
	StaleAfter   time.Duration
}

// Worker owns no business transaction. It delegates the claimed delivery's
// transaction to Outbox.ExecuteClaimed, which atomically wraps core effects,
// activity audit/history, and the delivered state transition.
type Worker struct {
	outbox    Outbox
	processor Processor
	opts      WorkerOptions
	now       func() time.Time
}

func NewWorker(outbox Outbox, processor Processor, options WorkerOptions) *Worker {
	if options.PollInterval <= 0 {
		options.PollInterval = defaultPollInterval
	}
	if options.BatchSize <= 0 {
		options.BatchSize = defaultBatchSize
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = defaultMaxAttempts
	}
	if options.RetryDelay <= 0 {
		options.RetryDelay = defaultRetryDelay
	}
	if options.StaleAfter <= 0 {
		options.StaleAfter = defaultStaleAfter
	}
	return &Worker{outbox: outbox, processor: processor, opts: options, now: time.Now}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.outbox == nil || w.processor == nil {
		return
	}
	w.recoverStale(ctx)
	if err := w.RunOnce(ctx); err != nil && ctx.Err() == nil {
		slog.Warn("activity reward delivery worker run failed", "error", err)
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
				slog.Warn("activity reward delivery worker run failed", "error", err)
			}
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if w == nil || w.outbox == nil || w.processor == nil {
		return nil
	}
	deliveries, err := w.outbox.ClaimDue(ctx, w.now(), w.opts.BatchSize)
	if err != nil {
		return fmt.Errorf("claim activity reward deliveries: %w", err)
	}
	var persistenceErrors []error
	for _, delivery := range deliveries {
		if err := w.processClaimed(ctx, delivery); err != nil {
			var persistenceErr persistenceError
			if errors.As(err, &persistenceErr) {
				persistenceErrors = append(persistenceErrors, err)
			}
		}
	}
	return errors.Join(persistenceErrors...)
}

// RunByID is a best-effort low-latency delivery attempt. A domain failure is
// persisted and retried later; callers must not roll back completed check-in.
func (w *Worker) RunByID(ctx context.Context, id int64) error {
	if w == nil || w.outbox == nil || w.processor == nil || id <= 0 {
		return nil
	}
	delivery, err := w.outbox.ClaimByID(ctx, id, w.now())
	if err != nil || delivery == nil {
		return err
	}
	return w.processClaimed(ctx, *delivery)
}

func (w *Worker) processClaimed(ctx context.Context, delivery Delivery) error {
	err := w.outbox.ExecuteClaimed(ctx, delivery.ID, func(txCtx context.Context, claimed Delivery) (string, error) {
		return w.processor.ProcessDelivery(txCtx, claimed)
	})
	if err == nil {
		return nil
	}
	var nextRetryAt *time.Time
	if delivery.Attempts < w.opts.MaxAttempts {
		next := w.now().Add(w.retryDelay(delivery.Attempts))
		nextRetryAt = &next
	}
	if markErr := w.outbox.MarkFailed(ctx, delivery.ID, err.Error(), nextRetryAt); markErr != nil {
		return errors.Join(err, persistenceError{err: fmt.Errorf("mark activity reward delivery %d failed: %w", delivery.ID, markErr)})
	}
	return err
}

func (w *Worker) recoverStale(ctx context.Context) {
	if w == nil || w.outbox == nil {
		return
	}
	now := w.now()
	if _, err := w.outbox.RecoverStale(ctx, now.Add(-w.opts.StaleAfter), now); err != nil && ctx.Err() == nil {
		slog.Warn("recover stale activity reward deliveries failed", "error", err)
	}
}

func (w *Worker) retryDelay(attempts int) time.Duration {
	exponent := max(attempts-1, 0)
	multiplier := math.Pow(2, float64(min(exponent, 10)))
	return time.Duration(float64(w.opts.RetryDelay) * multiplier)
}

type persistenceError struct{ err error }

func (e persistenceError) Error() string { return e.err.Error() }
func (e persistenceError) Unwrap() error { return e.err }

type workerRunner interface {
	Run(ctx context.Context)
}

// Runtime is an opt-in lifecycle adapter for application composition. Root
// runtime/Wire owns constructing it and must call Stop during service shutdown.
type Runtime struct {
	worker workerRunner

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

func NewRuntime(worker workerRunner) *Runtime {
	return &Runtime{worker: worker}
}

func (r *Runtime) Start(parent context.Context) {
	if r == nil || r.worker == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	r.done = make(chan struct{})
	r.started = true
	done := r.done
	r.mu.Unlock()
	go func() {
		defer close(done)
		r.worker.Run(ctx)
	}()
}

func (r *Runtime) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return
	}
	cancel, done := r.cancel, r.done
	r.mu.Unlock()
	cancel()
	<-done
}
