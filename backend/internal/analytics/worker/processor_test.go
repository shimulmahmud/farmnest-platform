package worker

import (
        "context"
        "encoding/json"
        "sync/atomic"
        "testing"
        "time"

        domain "github.com/ecommerce/backend/internal/analytics/domain"
        "github.com/ecommerce/backend/pkg/eventstore"
)

// mockRepo for worker tests.
type mockRepo struct {
        stored      []*domain.AnalyticsEvent
        existsMap   map[string]bool
        existsErr   error
        insertErr   error
        storedCount atomic.Int64
}

func newMockRepo() *mockRepo {
        return &mockRepo{
                existsMap: make(map[string]bool),
        }
}

func (m *mockRepo) Insert(_ context.Context, event *domain.AnalyticsEvent) error {
        if m.insertErr != nil {
                return m.insertErr
        }
        m.stored = append(m.stored, event)
        m.storedCount.Add(1)
        return nil
}

func (m *mockRepo) ExistsByEventID(_ context.Context, eventID string) (bool, error) {
        if m.existsErr != nil {
                return false, m.existsErr
        }
        return m.existsMap[eventID], nil
}

func (m *mockRepo) CountEvents(_ context.Context, _ domain.EventType, _ time.Time, _ time.Time) (int64, error) {
        return 0, nil
}

func (m *mockRepo) CountUniqueCustomers(_ context.Context, _ domain.EventType, _ time.Time, _ time.Time) (int64, error) {
        return 0, nil
}

func (m *mockRepo) SumRevenue(_ context.Context, _ time.Time, _ time.Time) (int64, error) {
        return 0, nil
}

func (m *mockRepo) TopProducts(_ context.Context, _ time.Time, _ time.Time, _ int) ([]*domain.ProductViewStat, error) {
        return nil, nil
}

func (m *mockRepo) ConversionRate(_ context.Context, _ time.Time, _ time.Time) (int64, int64, float64, error) {
        return 0, 0, 0, nil
}

func (m *mockRepo) DailyCounts(_ context.Context, _ domain.EventType, _ time.Time, _ time.Time) ([]*domain.DailyCount, error) {
        return nil, nil
}

func makeEnvelope(eventType string, payload interface{}) *eventstore.Envelope {
        payloadBytes, _ := json.Marshal(payload)
        return &eventstore.Envelope{
                EventID:      "evt-" + eventType,
                EventType:    eventType,
                AggregateID:  "agg-1",
                AggregateType: "product",
                OccurredAt:   time.Now().UTC(),
                CorrelationID: "corr-1",
                Payload:      payloadBytes,
        }
}

func TestResolveEventType_Direct(t *testing.T) {
        tests := []struct{
                input    string
                expected domain.EventType
        }{
                {"product.viewed", domain.EventProductViewed},
                {"cart.created", domain.EventCartCreated},
                {"checkout.started", domain.EventCheckoutStarted},
                {"order.completed", domain.EventOrderCompleted},
                {"payment.completed", domain.EventPaymentCompleted},
                {"unknown.event", ""},
                {"", ""},
        }
        for _, tt := range tests {
                result := resolveEventType(tt.input)
                if result != tt.expected {
                        t.Errorf("resolveEventType(%q) = %q, want %q", tt.input, result, tt.expected)
                }
        }
}

func TestResolveEventType_MarketingStyle(t *testing.T) {
        tests := []struct{
                input    string
                expected domain.EventType
        }{
                {"analytics.product_viewed", domain.EventProductViewed},
                {"analytics.cart_created", domain.EventCartCreated},
                {"analytics.checkout_started", domain.EventCheckoutStarted},
                {"analytics.order_completed", domain.EventOrderCompleted},
                {"analytics.cart_abandoned", ""},
        }
        for _, tt := range tests {
                result := resolveEventType(tt.input)
                if result != tt.expected {
                        t.Errorf("resolveEventType(%q) = %q, want %q", tt.input, result, tt.expected)
                }
        }
}

func TestExtractCustomerID(t *testing.T) {
        tests := []struct{
                name     string
                payload  string
                expected string
        }{
                {
                        name:     "standard json",
                        payload:  `{"customer_id": "cust-123", "product_id": "prod-1"}`,
                        expected: "cust-123",
                },
                {
                        name:     "no customer id",
                        payload:  `{"product_id": "prod-1"}`,
                        expected: "",
                },
                {
                        name:     "empty payload",
                        payload:  `{}`,
                        expected: "",
                },
        }
        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        result := extractCustomerID([]byte(tt.payload))
                        if result != tt.expected {
                                t.Errorf("extractCustomerID() = %q, want %q", result, tt.expected)
                        }
                })
        }
}

func TestProcessor_Handle_ProductViewed(t *testing.T) {
        repo := newMockRepo()
        p := NewProcessor(ProcessorDependencies{Repo: repo})
        handler := p.Handle()

        env := makeEnvelope("product.viewed", map[string]any{
                "customer_id": "cust-1",
                "product_id":  "prod-1",
        })

        err := handler(context.Background(), env)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if repo.storedCount.Load() != 1 {
                t.Errorf("expected 1 stored event, got %d", repo.storedCount.Load())
        }
        if p.Metrics().Processed.Load() != 1 {
                t.Errorf("expected 1 processed, got %d", p.Metrics().Processed.Load())
        }
}

func TestProcessor_Handle_Idempotency(t *testing.T) {
        repo := newMockRepo()
        repo.existsMap["evt-product.viewed"] = true
        p := NewProcessor(ProcessorDependencies{Repo: repo})
        handler := p.Handle()

        env := makeEnvelope("product.viewed", map[string]any{
                "customer_id": "cust-1",
                "product_id":  "prod-1",
        })

        err := handler(context.Background(), env)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if repo.storedCount.Load() != 0 {
                t.Errorf("expected 0 stored events (idempotent skip), got %d", repo.storedCount.Load())
        }
        if p.Metrics().IdempotentSkipped.Load() != 1 {
                t.Errorf("expected 1 skipped, got %d", p.Metrics().IdempotentSkipped.Load())
        }
}

func TestProcessor_Handle_UnrecognizedEventType(t *testing.T) {
        repo := newMockRepo()
        p := NewProcessor(ProcessorDependencies{Repo: repo})
        handler := p.Handle()

        env := makeEnvelope("unknown.event", map[string]any{"foo": "bar"})

        err := handler(context.Background(), env)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if repo.storedCount.Load() != 0 {
                t.Errorf("expected 0 stored events, got %d", repo.storedCount.Load())
        }
}

func TestProcessor_Handle_BadPayload(t *testing.T) {
        repo := newMockRepo()
        p := NewProcessor(ProcessorDependencies{Repo: repo})
        handler := p.Handle()

        env := &eventstore.Envelope{
                EventID:      "evt-bad",
                EventType:    "product.viewed",
                AggregateID:  "agg-1",
                AggregateType: "product",
                OccurredAt:   time.Now().UTC(),
                Payload:      []byte("not-json{{{"),
        }

        err := handler(context.Background(), env)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        // Bad payload should be non-retryable but counted as processed
        if p.Metrics().Processed.Load() != 1 {
                t.Errorf("expected 1 processed, got %d", p.Metrics().Processed.Load())
        }
        if p.Metrics().Errors.Load() != 0 {
                t.Errorf("expected 0 errors (non-retryable), got %d", p.Metrics().Errors.Load())
        }
}

func TestMetrics_Snapshot(t *testing.T) {
        var m Metrics
        m.Processed.Add(10)
        m.Stored.Add(8)
        m.IdempotentSkipped.Add(1)
        m.Errors.Add(1)

        snap := m.Snapshot()
        if snap["processed"] != 10 {
                t.Errorf("expected 10 processed, got %d", snap["processed"])
        }
        if snap["stored"] != 8 {
                t.Errorf("expected 8 stored, got %d", snap["stored"])
        }
}
