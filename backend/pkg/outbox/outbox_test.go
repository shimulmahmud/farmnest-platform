package outbox

import (
        "context"
        "encoding/json"
        "fmt"
        "sync"
        "testing"
        "time"

        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/ecommerce/backend/pkg/logger"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- In-memory test double ---

type inMemoryOutboxRepo struct {
        mu     sync.Mutex
        events map[string]*OutboxEvent
}

func newInMemoryOutboxRepo() *inMemoryOutboxRepo {
        return &inMemoryOutboxRepo{events: make(map[string]*OutboxEvent)}
}

func (r *inMemoryOutboxRepo) Insert(_ context.Context, event *OutboxEvent) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        if _, exists := r.events[event.EventID]; exists {
                return nil
        }
        r.events[event.EventID] = event
        return nil
}

func (r *inMemoryOutboxRepo) ClaimNext(_ context.Context) (*OutboxEvent, error) {
        r.mu.Lock()
        defer r.mu.Unlock()

        var oldest *OutboxEvent
        for _, e := range r.events {
                if e.Status == StatusPending {
                        if oldest == nil || e.CreatedAt.Before(oldest.CreatedAt) {
                                oldest = e
                        }
                }
        }
        if oldest == nil {
                return nil, nil
        }
        oldest.Status = StatusPublishing
        oldest.Attempts++
        return oldest, nil
}

func (r *inMemoryOutboxRepo) ResetToPending(_ context.Context, eventID string) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        e, ok := r.events[eventID]
        if !ok || e.Status != StatusPublishing {
                return fmt.Errorf("event %s not in PUBLISHING state", eventID)
        }
        e.Status = StatusPending
        return nil
}

func (r *inMemoryOutboxRepo) MarkPublished(_ context.Context, eventID string) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        e, ok := r.events[eventID]
        if !ok || e.Status != StatusPublishing {
                return fmt.Errorf("event %s not in PUBLISHING state", eventID)
        }
        now := time.Now().UTC()
        e.Status = StatusPublished
        e.PublishedAt = &now
        return nil
}

func (r *inMemoryOutboxRepo) MarkFailed(_ context.Context, eventID string, errMsg string) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        e, ok := r.events[eventID]
        if !ok || e.Status != StatusPublishing {
                return fmt.Errorf("event %s not in PUBLISHING state", eventID)
        }
        e.Status = StatusFailed
        e.LastError = errMsg
        return nil
}

func (r *inMemoryOutboxRepo) ResetStale(_ context.Context, staleThreshold time.Duration) (int64, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        cutoff := time.Now().UTC().Add(-staleThreshold)
        var count int64
        for _, e := range r.events {
                if e.Status == StatusPublishing && e.CreatedAt.Before(cutoff) {
                        e.Status = StatusPending
                        count++
                }
        }
        return count, nil
}

func (r *inMemoryOutboxRepo) DeletePublished(_ context.Context, olderThan time.Duration) (int64, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        cutoff := time.Now().UTC().Add(-olderThan)
        var count int64
        for id, e := range r.events {
                if e.Status == StatusPublished && e.PublishedAt != nil && e.PublishedAt.Before(cutoff) {
                        delete(r.events, id)
                        count++
                }
        }
        return count, nil
}

func (r *inMemoryOutboxRepo) CountByStatus(_ context.Context) (map[OutboxStatus]int64, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        counts := map[OutboxStatus]int64{}
        for _, e := range r.events {
                counts[e.Status]++
        }
        return counts, nil
}

// --- Mock Publisher ---

type mockEventPublisher struct {
        mu        sync.Mutex
        published []*eventstore.Envelope
        err       error
}

func (m *mockEventPublisher) PublishRaw(_ context.Context, env *eventstore.Envelope) error {
        m.mu.Lock()
        defer m.mu.Unlock()
        if m.err != nil {
                return m.err
        }
        m.published = append(m.published, env)
        return nil
}

func (m *mockEventPublisher) publishedCount() int {
        m.mu.Lock()
        defer m.mu.Unlock()
        return len(m.published)
}

// --- Domain Tests ---

func TestOutboxStatusConstants(t *testing.T) {
        assert.Equal(t, OutboxStatus("PENDING"), StatusPending)
        assert.Equal(t, OutboxStatus("PUBLISHING"), StatusPublishing)
        assert.Equal(t, OutboxStatus("PUBLISHED"), StatusPublished)
        assert.Equal(t, OutboxStatus("FAILED"), StatusFailed)
}

func TestMaxPublishAttempts(t *testing.T) {
        assert.Equal(t, 10, MaxPublishAttempts)
}

func TestBackoffDuration(t *testing.T) {
        tests := []struct {
                attempt  int
                expected time.Duration
        }{
                {0, 1 * time.Second},
                {1, 2 * time.Second},
                {2, 4 * time.Second},
                {3, 8 * time.Second},
                {4, 16 * time.Second},
                {5, 32 * time.Second},
                {8, 256 * time.Second},
                {9, 5 * time.Minute},
                {10, 5 * time.Minute}, // capped
                {99, 5 * time.Minute}, // capped
                {-1, 1 * time.Second}, // negative treated as 0
        }
        for _, tt := range tests {
                t.Run(tt.expected.String(), func(t *testing.T) {
                        got := BackoffDuration(tt.attempt)
                        assert.Equal(t, tt.expected, got)
                })
        }
}

func TestNewOutboxEventFromEnvelope(t *testing.T) {
        env, err := eventstore.NewEnvelope("order.created", "ord-123", "order", 1, map[string]string{"key": "val"})
        require.NoError(t, err)

        evt := NewOutboxEvent(env)

        assert.Equal(t, env.EventID, evt.EventID)
        assert.Equal(t, "order.created", evt.EventType)
        assert.Equal(t, "ord-123", evt.AggregateID)
        assert.Equal(t, "order", evt.AggregateType)
        assert.Equal(t, StatusPending, evt.Status)
        assert.Equal(t, 0, evt.Attempts)
        assert.Equal(t, env.CorrelationID, evt.CorrelationID)
        assert.Equal(t, env.Payload, evt.Payload)
        assert.Equal(t, "ECOMMERCE_ORDERS.order.created", evt.Subject)
        assert.False(t, evt.CreatedAt.IsZero())
        assert.Nil(t, evt.PublishedAt)
        assert.Empty(t, evt.LastError)
}

func TestNewOutboxEventDirect(t *testing.T) {
        evt, err := NewOutboxEventDirect("inventory.reserved", "sku-1", "inventory", "corr-1", 1, map[string]string{"sku": "ABC"})
        require.NoError(t, err)

        assert.NotEmpty(t, evt.EventID)
        assert.Equal(t, "inventory.reserved", evt.EventType)
        assert.Equal(t, "sku-1", evt.AggregateID)
        assert.Equal(t, "inventory", evt.AggregateType)
        assert.Equal(t, StatusPending, evt.Status)
        assert.Equal(t, "corr-1", evt.CorrelationID)
        assert.Equal(t, "ECOMMERCE_INVENTORY.inventory.reserved", evt.Subject)

        var payload map[string]string
        err = json.Unmarshal(evt.Payload, &payload)
        require.NoError(t, err)
        assert.Equal(t, "ABC", payload["sku"])
}

func TestNewOutboxEventDirect_EmptyCorrelationID(t *testing.T) {
        evt, err := NewOutboxEventDirect("order.created", "ord-1", "order", "", 1, nil)
        require.NoError(t, err)
        assert.NotEmpty(t, evt.CorrelationID) // auto-generated
}

func TestNewOutboxEventDirect_BadPayload(t *testing.T) {
        _, err := NewOutboxEventDirect("test", "id", "type", "", 1, func() {}) // functions can't be marshaled
        assert.Error(t, err)
}

func TestToEnvelope(t *testing.T) {
        env, err := eventstore.NewEnvelope("payment.completed", "pay-1", "payment", 1, nil)
        require.NoError(t, err)

        evt := NewOutboxEvent(env)
        restored := evt.ToEnvelope()

        assert.Equal(t, env.EventID, restored.EventID)
        assert.Equal(t, env.EventType, restored.EventType)
        assert.Equal(t, env.AggregateID, restored.AggregateID)
        assert.Equal(t, env.AggregateType, restored.AggregateType)
        assert.Equal(t, env.CorrelationID, restored.CorrelationID)
        assert.Equal(t, env.Payload, restored.Payload)
        assert.Equal(t, env.Version, restored.Version)
}

// --- Repository Tests (in-memory) ---

func TestInMemoryRepo_Insert(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        env, _ := eventstore.NewEnvelope("order.created", "ord-1", "order", 1, nil)
        evt := NewOutboxEvent(env)

        err := repo.Insert(context.Background(), evt)
        require.NoError(t, err)

        // Duplicate insert is idempotent.
        err = repo.Insert(context.Background(), evt)
        require.NoError(t, err)
}

func TestInMemoryRepo_ClaimNext(t *testing.T) {
        repo := newInMemoryOutboxRepo()

        // No events -> nil.
        claimed, err := repo.ClaimNext(context.Background())
        require.NoError(t, err)
        assert.Nil(t, claimed)

        // Insert two events.
        env1, _ := eventstore.NewEnvelope("order.created", "ord-1", "order", 1, nil)
        env2, _ := eventstore.NewEnvelope("order.paid", "ord-2", "order", 1, nil)

        evt1 := NewOutboxEvent(env1)
        evt2 := NewOutboxEvent(env2)

        // Make evt2 older.
        evt2.CreatedAt = evt1.CreatedAt.Add(-1 * time.Hour)

        _ = repo.Insert(context.Background(), evt1)
        _ = repo.Insert(context.Background(), evt2)

        // Oldest (evt2) should be claimed first.
        claimed, err = repo.ClaimNext(context.Background())
        require.NoError(t, err)
        assert.Equal(t, "ord-2", claimed.AggregateID)
        assert.Equal(t, StatusPublishing, claimed.Status)
        assert.Equal(t, 1, claimed.Attempts)

        // Next claim gets evt1.
        claimed, err = repo.ClaimNext(context.Background())
        require.NoError(t, err)
        assert.Equal(t, "ord-1", claimed.AggregateID)

        // No more.
        claimed, err = repo.ClaimNext(context.Background())
        require.NoError(t, err)
        assert.Nil(t, claimed)
}

func TestInMemoryRepo_MarkPublished(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        env, _ := eventstore.NewEnvelope("order.created", "ord-1", "order", 1, nil)
        evt := NewOutboxEvent(env)
        _ = repo.Insert(context.Background(), evt)

        _, _ = repo.ClaimNext(context.Background())

        err := repo.MarkPublished(context.Background(), evt.EventID)
        require.NoError(t, err)

        e, ok := repo.events[evt.EventID]
        require.True(t, ok)
        assert.Equal(t, StatusPublished, e.Status)
        assert.NotNil(t, e.PublishedAt)

        // Mark published again should fail.
        err = repo.MarkPublished(context.Background(), evt.EventID)
        assert.Error(t, err)
}

func TestInMemoryRepo_MarkFailed(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        env, _ := eventstore.NewEnvelope("order.created", "ord-1", "order", 1, nil)
        evt := NewOutboxEvent(env)
        _ = repo.Insert(context.Background(), evt)

        _, _ = repo.ClaimNext(context.Background())

        err := repo.MarkFailed(context.Background(), evt.EventID, "NATS unavailable")
        require.NoError(t, err)

        e := repo.events[evt.EventID]
        assert.Equal(t, StatusFailed, e.Status)
        assert.Equal(t, "NATS unavailable", e.LastError)
}

func TestInMemoryRepo_ResetStale(t *testing.T) {
        repo := newInMemoryOutboxRepo()

        env, _ := eventstore.NewEnvelope("order.created", "ord-1", "order", 1, nil)
        evt := NewOutboxEvent(env)
        evt.CreatedAt = time.Now().UTC().Add(-5 * time.Minute)
        evt.Status = StatusPublishing
        repo.events[evt.EventID] = evt

        count, err := repo.ResetStale(context.Background(), 2*time.Minute)
        require.NoError(t, err)
        assert.Equal(t, int64(1), count)
        assert.Equal(t, StatusPending, repo.events[evt.EventID].Status)

        // Now not stale anymore.
        count, err = repo.ResetStale(context.Background(), 2*time.Minute)
        require.NoError(t, err)
        assert.Equal(t, int64(0), count)
}

func TestInMemoryRepo_DeletePublished(t *testing.T) {
        repo := newInMemoryOutboxRepo()

        env, _ := eventstore.NewEnvelope("order.created", "ord-1", "order", 1, nil)
        evt := NewOutboxEvent(env)
        oldTime := time.Now().UTC().Add(-48 * time.Hour)
        evt.Status = StatusPublished
        evt.PublishedAt = &oldTime
        repo.events[evt.EventID] = evt

        count, err := repo.DeletePublished(context.Background(), 24*time.Hour)
        require.NoError(t, err)
        assert.Equal(t, int64(1), count)
        assert.NotContains(t, repo.events, evt.EventID)
}

func TestInMemoryRepo_CountByStatus(t *testing.T) {
        repo := newInMemoryOutboxRepo()

        for i := 0; i < 3; i++ {
                env, _ := eventstore.NewEnvelope("order.created", "ord-"+string(rune('A'+i)), "order", 1, nil)
                evt := NewOutboxEvent(env)
                _ = repo.Insert(context.Background(), evt)
        }

        // Claim one.
        _, _ = repo.ClaimNext(context.Background())

        counts, err := repo.CountByStatus(context.Background())
        require.NoError(t, err)
        assert.Equal(t, int64(2), counts[StatusPending])
        assert.Equal(t, int64(1), counts[StatusPublishing])
}

// --- Worker Tests ---

func newTestLogger(t *testing.T) *logger.Logger {
        log, err := logger.New("info", "console")
        require.NoError(t, err)
        return log
}

func TestWorker_StartStop(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        mockPub := &mockEventPublisher{}
        log := newTestLogger(t)

        worker := NewWorker(repo, mockPub, log, WorkerConfig{
                PollInterval:     50 * time.Millisecond,
                BatchSize:        10,
                StaleThreshold:   1 * time.Minute,
                CleanupInterval:  10 * time.Minute,
                CleanupRetention: 24 * time.Hour,
        })

        ctx, cancel := context.WithCancel(context.Background())

        worker.Start(ctx)
        assert.True(t, worker.running.Load())

        time.Sleep(200 * time.Millisecond)

        cancel()
        worker.Stop()
        assert.False(t, worker.running.Load())
}

func TestWorker_PublishesPendingEvents(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        mockPub := &mockEventPublisher{}
        log := newTestLogger(t)

        env, _ := eventstore.NewEnvelope("order.created", "ord-1", "order", 1, map[string]string{"test": "data"})
        evt := NewOutboxEvent(env)
        _ = repo.Insert(context.Background(), evt)

        worker := NewWorker(repo, mockPub, log, WorkerConfig{
                PollInterval:     50 * time.Millisecond,
                BatchSize:        10,
                StaleThreshold:   1 * time.Minute,
                CleanupInterval:  10 * time.Minute,
                CleanupRetention: 24 * time.Hour,
        })

        ctx, cancel := context.WithCancel(context.Background())
        worker.Start(ctx)

        // Wait for the event to be processed.
        deadline := time.After(3 * time.Second)
        for {
                select {
                case <-deadline:
                        t.Fatal("timed out waiting for event to be published")
                default:
                }
                if mockPub.publishedCount() >= 1 {
                        break
                }
                time.Sleep(50 * time.Millisecond)
        }

        cancel()
        worker.Stop()

        assert.Equal(t, int64(1), worker.metrics.Published.Load())
        assert.Equal(t, int64(0), worker.metrics.Failed.Load())

        mockPub.mu.Lock()
        require.Len(t, mockPub.published, 1)
        assert.Equal(t, "ord-1", mockPub.published[0].AggregateID)
        assert.Equal(t, "order.created", mockPub.published[0].EventType)
        mockPub.mu.Unlock()
}

func TestWorker_MetricsSnapshot(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        mockPub := &mockEventPublisher{}
        log := newTestLogger(t)

        worker := NewWorker(repo, mockPub, log, WorkerConfig{
                PollInterval: 50 * time.Millisecond,
                BatchSize:    10,
        })

        snapshot := worker.MetricsSnapshot()
        assert.Equal(t, int64(0), snapshot["published"])
        assert.Equal(t, int64(0), snapshot["failed"])
}

func TestWorker_RecoverStaleOnStart(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        mockPub := &mockEventPublisher{}
        log := newTestLogger(t)

        env, _ := eventstore.NewEnvelope("order.created", "ord-stale", "order", 1, nil)
        evt := NewOutboxEvent(env)
        evt.Status = StatusPublishing
        evt.CreatedAt = time.Now().UTC().Add(-10 * time.Minute)
        repo.events[evt.EventID] = evt

        worker := NewWorker(repo, mockPub, log, WorkerConfig{
                PollInterval:   50 * time.Millisecond,
                BatchSize:      10,
                StaleThreshold: 1 * time.Minute,
        })

        ctx, cancel := context.WithCancel(context.Background())
        worker.Start(ctx)

        deadline := time.After(3 * time.Second)
        for {
                select {
                case <-deadline:
                        t.Fatal("timed out waiting for stale recovery")
                default:
                }
                if mockPub.publishedCount() >= 1 {
                        break
                }
                time.Sleep(50 * time.Millisecond)
        }

        cancel()
        worker.Stop()

        assert.Equal(t, int64(1), worker.metrics.ResetStale.Load())
        assert.Equal(t, int64(1), worker.metrics.Published.Load())
}

func TestWorker_MaxAttemptsMarksFailed(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        mockPub := &mockEventPublisher{err: fmt.Errorf("NATS down")}
        log := newTestLogger(t)

        env, _ := eventstore.NewEnvelope("order.created", "ord-fail", "order", 1, nil)
        evt := NewOutboxEvent(env)
        evt.Status = StatusPublishing
        evt.Attempts = MaxPublishAttempts + 1
        repo.events[evt.EventID] = evt

        worker := NewWorker(repo, mockPub, log, WorkerConfig{
                PollInterval: 50 * time.Millisecond,
                BatchSize:    10,
        })

        worker.processEvent(context.Background(), evt)

        assert.Equal(t, int64(1), worker.metrics.Failed.Load())
        e := repo.events[evt.EventID]
        assert.Equal(t, StatusFailed, e.Status)
        assert.Contains(t, e.LastError, "exceeded max attempts")
}

func TestDefaultWorkerConfig(t *testing.T) {
        cfg := DefaultWorkerConfig()
        assert.Equal(t, 500*time.Millisecond, cfg.PollInterval)
        assert.Equal(t, 50, cfg.BatchSize)
        assert.Equal(t, 2*time.Minute, cfg.StaleThreshold)
        assert.Equal(t, 5*time.Minute, cfg.CleanupInterval)
        assert.Equal(t, 24*time.Hour, cfg.CleanupRetention)
}

func TestWorker_StartIdempotent(t *testing.T) {
        repo := newInMemoryOutboxRepo()
        mockPub := &mockEventPublisher{}
        log := newTestLogger(t)

        worker := NewWorker(repo, mockPub, log, WorkerConfig{
                PollInterval: 50 * time.Millisecond,
        })

        ctx, cancel := context.WithCancel(context.Background())

        worker.Start(ctx)
        worker.Start(ctx) // second call should be no-op
        assert.True(t, worker.running.Load())

        cancel()
        worker.Stop()
        assert.False(t, worker.running.Load())
}
