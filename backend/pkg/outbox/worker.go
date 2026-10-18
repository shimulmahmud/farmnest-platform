package outbox

import (
        "context"
        "fmt"
        "sync"
        "sync/atomic"
        "time"

        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/ecommerce/backend/pkg/logger"
)

// RawPublisher publishes a pre-built envelope. This interface allows
// the worker to be tested with a mock publisher.
type RawPublisher interface {
        PublishRaw(ctx context.Context, env *eventstore.Envelope) error
}

// Worker polls the outbox table and publishes pending events to NATS via JetStream.
// It handles retry with exponential backoff, crash recovery (stale event reset),
// and cleanup of old published events.
type Worker struct {
        repo   OutboxRepository
        pub    RawPublisher
        log    *logger.Logger

        // Config
        PollInterval      time.Duration
        BatchSize         int
        StaleThreshold    time.Duration
        CleanupInterval   time.Duration
        CleanupRetention  time.Duration

        // State
        running   atomic.Bool
        cancelFn  context.CancelFunc
        wg        sync.WaitGroup

        // Metrics
        metrics Metrics
}

// Metrics holds outbox worker counters.
type Metrics struct {
        Published  atomic.Int64
        Failed     atomic.Int64
        Retries    atomic.Int64
        ResetStale atomic.Int64
        CleanedUp  atomic.Int64
}

// WorkerConfig holds configuration for the outbox worker.
type WorkerConfig struct {
        PollInterval     time.Duration // how often to poll for pending events (default: 500ms)
        BatchSize        int           // max events per poll cycle (default: 50)
        StaleThreshold   time.Duration // how long before a PUBLISHING event is considered stale (default: 2min)
        CleanupInterval  time.Duration // how often to clean up old published events (default: 5min)
        CleanupRetention time.Duration // how long to keep published events before cleanup (default: 24h)
}

// DefaultWorkerConfig returns sensible defaults.
func DefaultWorkerConfig() WorkerConfig {
        return WorkerConfig{
                PollInterval:     500 * time.Millisecond,
                BatchSize:        50,
                StaleThreshold:   2 * time.Minute,
                CleanupInterval:  5 * time.Minute,
                CleanupRetention: 24 * time.Hour,
        }
}

// NewWorker creates a new outbox worker.
func NewWorker(repo OutboxRepository, pub RawPublisher, log *logger.Logger, cfg WorkerConfig) *Worker {
        if cfg.PollInterval <= 0 {
                cfg.PollInterval = DefaultWorkerConfig().PollInterval
        }
        if cfg.BatchSize <= 0 {
                cfg.BatchSize = DefaultWorkerConfig().BatchSize
        }
        if cfg.StaleThreshold <= 0 {
                cfg.StaleThreshold = DefaultWorkerConfig().StaleThreshold
        }
        if cfg.CleanupInterval <= 0 {
                cfg.CleanupInterval = DefaultWorkerConfig().CleanupInterval
        }
        if cfg.CleanupRetention <= 0 {
                cfg.CleanupRetention = DefaultWorkerConfig().CleanupRetention
        }

        return &Worker{
                repo:             repo,
                pub:              pub,
                log:              log.Named("outbox-worker"),
                PollInterval:     cfg.PollInterval,
                BatchSize:        cfg.BatchSize,
                StaleThreshold:   cfg.StaleThreshold,
                CleanupInterval:  cfg.CleanupInterval,
                CleanupRetention: cfg.CleanupRetention,
        }
}

// Start begins the outbox worker loop. It is safe to call Start only once.
func (w *Worker) Start(ctx context.Context) {
        if !w.running.CompareAndSwap(false, true) {
                return
        }

        ctx, w.cancelFn = context.WithCancel(ctx)

        // Recover any stale PUBLISHING events from a previous crash.
        w.recoverStale(ctx)

        // Main poll loop.
        w.wg.Add(1)
        go w.pollLoop(ctx)

        // Cleanup loop.
        w.wg.Add(1)
        go w.cleanupLoop(ctx)

        w.log.Infow("outbox worker started",
                "poll_interval", w.PollInterval,
                "batch_size", w.BatchSize,
                "stale_threshold", w.StaleThreshold,
                "cleanup_interval", w.CleanupInterval,
        )
}

// Stop gracefully shuts down the outbox worker.
func (w *Worker) Stop() {
        if !w.running.CompareAndSwap(true, false) {
                return
        }
        if w.cancelFn != nil {
                w.cancelFn()
        }
        w.wg.Wait()
        w.log.Infow("outbox worker stopped",
                "published", w.metrics.Published.Load(),
                "failed", w.metrics.Failed.Load(),
                "retries", w.metrics.Retries.Load(),
                "stale_recovered", w.metrics.ResetStale.Load(),
                "cleaned_up", w.metrics.CleanedUp.Load(),
        )
}

// MetricsSnapshot returns a copy of the current metric values.
func (w *Worker) MetricsSnapshot() map[string]int64 {
        return map[string]int64{
                "published":      w.metrics.Published.Load(),
                "failed":         w.metrics.Failed.Load(),
                "retries":        w.metrics.Retries.Load(),
                "stale_recovered": w.metrics.ResetStale.Load(),
                "cleaned_up":     w.metrics.CleanedUp.Load(),
        }
}

func (w *Worker) pollLoop(ctx context.Context) {
        defer w.wg.Done()
        ticker := time.NewTicker(w.PollInterval)
        defer ticker.Stop()

        for {
                select {
                case <-ctx.Done():
                        return
                case <-ticker.C:
                        w.processBatch(ctx)
                }
        }
}

func (w *Worker) cleanupLoop(ctx context.Context) {
        defer w.wg.Done()
        ticker := time.NewTicker(w.CleanupInterval)
        defer ticker.Stop()

        for {
                select {
                case <-ctx.Done():
                        return
                case <-ticker.C:
                        w.cleanup(ctx)
                }
        }
}

func (w *Worker) processBatch(ctx context.Context) {
        for i := 0; i < w.BatchSize; i++ {
                // Check context before each claim to avoid unnecessary work during shutdown.
                select {
                case <-ctx.Done():
                        return
                default:
                }

                event, err := w.repo.ClaimNext(ctx)
                if err != nil {
                        w.log.Errorw("failed to claim next outbox event", "error", err)
                        return
                }
                if event == nil {
                        // No more pending events.
                        return
                }

                w.processEvent(ctx, event)
        }
}

func (w *Worker) processEvent(ctx context.Context, event *OutboxEvent) {
        // Check if max attempts exceeded.
        if event.Attempts > MaxPublishAttempts {
                err := w.repo.MarkFailed(ctx, event.EventID, fmt.Sprintf("exceeded max attempts (%d)", MaxPublishAttempts))
                if err != nil {
                        w.log.Errorw("failed to mark outbox event as failed",
                                "event_id", event.EventID,
                                "event_type", event.EventType,
                                "attempts", event.Attempts,
                                "error", err,
                        )
                }
                w.metrics.Failed.Add(1)
                w.log.Errorw("outbox event exhausted retry attempts",
                        "event_id", event.EventID,
                        "event_type", event.EventType,
                        "attempts", event.Attempts,
                )
                return
        }

        // Backoff if this is a retry.
        if event.Attempts > 1 {
                duration := BackoffDuration(event.Attempts - 1)
                w.metrics.Retries.Add(1)
                w.log.Infow("retrying outbox event",
                        "event_id", event.EventID,
                        "event_type", event.EventType,
                        "attempt", event.Attempts,
                        "backoff", duration,
                )

                select {
                case <-ctx.Done():
                        return
                case <-time.After(duration):
                }
        }

        // Convert to envelope and publish.
        env := event.ToEnvelope()

        publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
        defer cancel()

        err := w.pub.PublishRaw(publishCtx, env)
        if err != nil {
                w.log.Errorw("failed to publish outbox event",
                        "event_id", event.EventID,
                        "event_type", event.EventType,
                        "attempt", event.Attempts,
                        "error", err,
                )
                // Reset to PENDING so the next poll cycle picks it up.
                if resetErr := w.repo.ResetToPending(ctx, event.EventID); resetErr != nil {
                        w.log.Errorw("failed to reset outbox event to pending",
                                "event_id", event.EventID,
                                "reset_error", resetErr,
                        )
                }
                return
        }

        // Mark as published.
        if err := w.repo.MarkPublished(ctx, event.EventID); err != nil {
                w.log.Errorw("failed to mark outbox event as published",
                        "event_id", event.EventID,
                        "error", err,
                )
                return
        }

        w.metrics.Published.Add(1)
        w.log.Debugw("outbox event published",
                "event_id", event.EventID,
                "event_type", event.EventType,
                "aggregate_id", event.AggregateID,
                "attempt", event.Attempts,
        )
}

func (w *Worker) recoverStale(ctx context.Context) {
        count, err := w.repo.ResetStale(ctx, w.StaleThreshold)
        if err != nil {
                w.log.Errorw("failed to recover stale outbox events", "error", err)
                return
        }
        if count > 0 {
                w.metrics.ResetStale.Add(count)
                w.log.Infow("recovered stale outbox events", "count", count)
        }
}

func (w *Worker) cleanup(ctx context.Context) {
        count, err := w.repo.DeletePublished(ctx, w.CleanupRetention)
        if err != nil {
                w.log.Errorw("failed to cleanup published outbox events", "error", err)
                return
        }
        if count > 0 {
                w.metrics.CleanedUp.Add(count)
                w.log.Infow("cleaned up published outbox events", "count", count)
        }
}
