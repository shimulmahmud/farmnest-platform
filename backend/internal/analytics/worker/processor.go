package worker

import (
        "context"
        "strings"
        "sync/atomic"

        domain "github.com/ecommerce/backend/internal/analytics/domain"
        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/ecommerce/backend/pkg/logger"
)

// ProcessorDependencies holds the injected dependencies for the analytics worker.
type ProcessorDependencies struct {
        Repo domain.AnalyticsRepository
        Log  *logger.Logger
}

// Metrics tracks worker statistics using atomic counters for thread safety.
type Metrics struct {
        Processed         atomic.Int64
        IdempotentSkipped atomic.Int64
        Stored            atomic.Int64
        Errors            atomic.Int64
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *Metrics) Snapshot() map[string]int64 {
        return map[string]int64{
                "processed":          m.Processed.Load(),
                "idempotent_skipped": m.IdempotentSkipped.Load(),
                "stored":             m.Stored.Load(),
                "errors":             m.Errors.Load(),
        }
}

// Processor consumes analytics events from JetStream, validates them,
// performs idempotency checks, and persists them to analytics storage.
// It never blocks HTTP requests — it runs exclusively as JetStream consumers.
type Processor struct {
        deps    ProcessorDependencies
        metrics Metrics
}

// NewProcessor creates a new analytics processor.
func NewProcessor(deps ProcessorDependencies) *Processor {
        return &Processor{deps: deps}
}

// Metrics returns the processor metrics.
func (p *Processor) Metrics() *Metrics {
        return &p.metrics
}

// Handle processes any analytics event from the ECOMMERCE_ANALYTICS stream.
// It dispatches to the appropriate type-specific handler based on the event type.
// Unrecognized event types are logged and skipped (non-retryable).
func (p *Processor) Handle() eventstore.EventHandler {
        return func(ctx context.Context, env *eventstore.Envelope) error {
                p.metrics.Processed.Add(1)

                eventType := resolveEventType(env.EventType)
                if eventType == "" {
                        p.deps.Log.Warnw("analytics: unrecognized event type",
                                "event_id", env.EventID, "event_type", env.EventType)
                        return nil // non-retryable
                }

                // Idempotency check
                exists, err := p.deps.Repo.ExistsByEventID(ctx, env.EventID)
                if err != nil {
                        p.deps.Log.Warnw("analytics: idempotency check failed",
                                "event_id", env.EventID, "error", err)
                }
                if exists {
                        p.metrics.IdempotentSkipped.Add(1)
                        return nil
                }

                // Extract customer ID from payload
                customerID := extractCustomerID(env.Payload)

                // Decode payload into a generic map for flexible storage
                var payloadMap map[string]any
                if err := env.DecodePayload(&payloadMap); err != nil {
                        p.deps.Log.Errorw("analytics: failed to decode payload",
                                "event_id", env.EventID, "error", err)
                        return nil // non-retryable — bad payload format
                }

                // Create and persist the analytics event
                event := domain.NewAnalyticsEvent(
                        env.EventID,
                        eventType,
                        customerID,
                        env.AggregateID,
                        env.AggregateType,
                        env.CorrelationID,
                        env.OccurredAt,
                        payloadMap,
                )

                if storeErr := p.deps.Repo.Insert(ctx, event); storeErr != nil {
                        p.deps.Log.Errorw("analytics: failed to store event",
                                "event_id", env.EventID, "event_type", eventType, "error", storeErr)
                        p.metrics.Errors.Add(1)
                        return nil // non-retryable — event remains in JetStream for replay
                }

                p.metrics.Stored.Add(1)
                p.deps.Log.Debugw("analytics: event stored",
                        "event_id", env.EventID, "event_type", eventType, "customer_id", customerID)

                return nil
        }
}

// resolveEventType maps the NATS event type string to a domain EventType.
// Handles both the M21-style "product.viewed" and the marketing-style
// "analytics.product_viewed" naming conventions.
func resolveEventType(eventType string) domain.EventType {
        // Direct M21-style mapping
        switch domain.EventType(eventType) {
        case domain.EventProductViewed,
                domain.EventCartCreated,
                domain.EventCheckoutStarted,
                domain.EventOrderCompleted,
                domain.EventPaymentCompleted:
                return domain.EventType(eventType)
        }

        // Marketing-style mapping: "analytics.product_viewed" -> "product.viewed"
        if strings.HasPrefix(eventType, "analytics.") {
                normalized := strings.TrimPrefix(eventType, "analytics.")
                normalized = strings.ReplaceAll(normalized, "_", ".")
                switch domain.EventType(normalized) {
                case domain.EventProductViewed,
                        domain.EventCartCreated,
                        domain.EventCheckoutStarted,
                        domain.EventOrderCompleted,
                        domain.EventPaymentCompleted:
                        return domain.EventType(normalized)
                }
        }

        return ""
}

// extractCustomerID attempts to extract customer_id from the raw JSON payload.
// It performs a simple string search to avoid full unmarshalling overhead
// in the hot path. If not found, returns empty string.
func extractCustomerID(payload []byte) string {
        // Fast path: look for "customer_id" key in the JSON.
        key := `"customer_id"`
        idx := strings.Index(string(payload), key)
        if idx == -1 {
                return ""
        }

        // Find the colon after the key.
        rest := string(payload[idx+len(key):])
        rest = strings.TrimLeft(rest, " \t\n\r:")

        // If the value is a quoted string, extract it.
        if len(rest) > 0 && rest[0] == '"' {
                end := strings.Index(rest[1:], `"`)
                if end > 0 {
                        return rest[1 : end+1]
                }
        }

        return ""
}
