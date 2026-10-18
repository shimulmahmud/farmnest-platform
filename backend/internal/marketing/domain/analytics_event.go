package domain

import (
        "context"
        "time"

        "github.com/google/uuid"
)

// AnalyticsEventType identifies the type of analytics event.
type AnalyticsEventType string

const (
        AnalyticsProductViewed   AnalyticsEventType = "analytics.product_viewed"
        AnalyticsCartCreated    AnalyticsEventType = "analytics.cart_created"
        AnalyticsCartAbandoned  AnalyticsEventType = "analytics.cart_abandoned"
        AnalyticsCheckoutStarted AnalyticsEventType = "analytics.checkout_started"
        AnalyticsOrderCompleted AnalyticsEventType = "analytics.order_completed"
)

// ProductViewedPayload is the payload for analytics.product_viewed events.
type ProductViewedPayload struct {
        CustomerID string `json:"customer_id"`
        ProductID  string `json:"product_id"`
        VariantID  string `json:"variant_id,omitempty"`
        CategoryID string `json:"category_id,omitempty"`
        Source     string `json:"source,omitempty"` // e.g. "search", "browse", "recommendation"
}

// CartCreatedPayload is the payload for analytics.cart_created events.
type CartCreatedPayload struct {
        CustomerID string `json:"customer_id"`
        CartID     string `json:"cart_id"`
        ItemIDs    []string `json:"item_ids,omitempty"`
}

// CartAbandonedPayload is the payload for analytics.cart_abandoned events.
type CartAbandonedPayload struct {
        CustomerID string  `json:"customer_id"`
        CartID     string  `json:"cart_id"`
        ItemIDs    []string `json:"item_ids,omitempty"`
        TotalCents int     `json:"total_cents,omitempty"`
        Currency   string  `json:"currency,omitempty"`
        ItemCount  int     `json:"item_count,omitempty"`
}

// CheckoutStartedPayload is the payload for analytics.checkout_started events.
type CheckoutStartedPayload struct {
        CustomerID string `json:"customer_id"`
        CartID     string `json:"cart_id"`
        ItemCount  int    `json:"item_count,omitempty"`
        TotalCents int    `json:"total_cents,omitempty"`
        Currency   string `json:"currency,omitempty"`
}

// OrderCompletedPayload is the payload for analytics.order_completed events.
type OrderCompletedPayload struct {
        CustomerID  string `json:"customer_id"`
        OrderID     string `json:"order_id"`
        OrderNumber string `json:"order_number"`
        TotalCents  int    `json:"total_cents"`
        Currency    string `json:"currency"`
        ItemIDs     []string `json:"item_ids,omitempty"`
        CategoryIDs []string `json:"category_ids,omitempty"`
}

// AnalyticsEvent is a persisted record of a customer analytics event.
// Used for segment evaluation, campaign attribution, and reporting.
type AnalyticsEvent struct {
        ID           string             `bson:"_id" json:"id"`
        EventID      string             `bson:"event_id" json:"event_id"`          // from envelope
        EventType    AnalyticsEventType  `bson:"event_type" json:"event_type"`
        CustomerID   string             `bson:"customer_id" json:"customer_id"`
        AggregateID  string             `bson:"aggregate_id" json:"aggregate_id"`
        Payload      map[string]any     `bson:"payload" json:"payload"`
        CorrelationID string            `bson:"correlation_id" json:"correlation_id"`
        ProcessedAt  time.Time          `bson:"processed_at" json:"processed_at"`
        CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

// NewAnalyticsEvent creates a persisted analytics event record.
func NewAnalyticsEvent(eventID, eventType, customerID, aggregateID, correlationID string, payload map[string]any) *AnalyticsEvent {
        now := time.Now().UTC()
        return &AnalyticsEvent{
                ID:            uuid.New().String(),
                EventID:       eventID,
                EventType:     AnalyticsEventType(eventType),
                CustomerID:    customerID,
                AggregateID:   aggregateID,
                Payload:       payload,
                CorrelationID: correlationID,
                ProcessedAt:   now,
                CreatedAt:     now,
        }
}

// AnalyticsEventRepository persists analytics events.
type AnalyticsEventRepository interface {
        // Insert creates a new analytics event record.
        Insert(ctx context.Context, event *AnalyticsEvent) error

        // ExistsByEventID checks idempotency.
        ExistsByEventID(ctx context.Context, eventID string) (bool, error)

        // CountByCustomer counts events for a customer within a time range.
        CountByCustomer(ctx context.Context, customerID string, eventType AnalyticsEventType, since time.Time) (int64, error)

        // ListByCustomer returns recent events for a customer (for segment evaluation).
        ListByCustomer(ctx context.Context, customerID string, limit int) ([]*AnalyticsEvent, error)
}
