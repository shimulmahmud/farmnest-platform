package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventType identifies the type of analytics event tracked by the analytics module.
// These use a shorter naming convention (e.g. "product.viewed") compared to the
// marketing module's "analytics.product_viewed" prefix, because the analytics
// stream subjects already scoped them under ECOMMERCE_ANALYTICS.
type EventType string

const (
	EventProductViewed   EventType = "product.viewed"
	EventCartCreated     EventType = "cart.created"
	EventCheckoutStarted EventType = "checkout.started"
	EventOrderCompleted  EventType = "order.completed" // an order that was DELIVERED
	EventOrderCreated    EventType = "order.created"   // an order that was placed
	EventPaymentCompleted EventType = "payment.completed"
)

// ProductViewedPayload is the payload for product.viewed events.
type ProductViewedPayload struct {
	CustomerID string `json:"customer_id"`
	ProductID  string `json:"product_id"`
	VariantID  string `json:"variant_id,omitempty"`
	CategoryID string `json:"category_id,omitempty"`
	Source     string `json:"source,omitempty"`
}

// CartCreatedPayload is the payload for cart.created events.
type CartCreatedPayload struct {
	CustomerID string   `json:"customer_id"`
	CartID     string   `json:"cart_id"`
	ItemIDs    []string `json:"item_ids,omitempty"`
}

// CheckoutStartedPayload is the payload for checkout.started events.
type CheckoutStartedPayload struct {
	CustomerID string `json:"customer_id"`
	CartID     string `json:"cart_id"`
	ItemCount  int    `json:"item_count,omitempty"`
	TotalCents int    `json:"total_cents,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

// OrderCompletedPayload is the payload for order.completed events.
type OrderCompletedPayload struct {
	CustomerID  string   `json:"customer_id"`
	OrderID     string   `json:"order_id"`
	OrderNumber string   `json:"order_number"`
	TotalCents  int      `json:"total_cents"`
	Currency    string   `json:"currency"`
	ItemIDs     []string `json:"item_ids,omitempty"`
	CategoryIDs []string `json:"category_ids,omitempty"`
}

// PaymentCompletedPayload is the payload for payment.completed events.
type PaymentCompletedPayload struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
	PaymentID  string `json:"payment_id"`
	AmountCents int  `json:"amount_cents"`
	Currency   string `json:"currency"`
}

// AnalyticsEvent is a persisted analytics record processed from JetStream.
// This is the analytics module's own storage — separate from marketing's
// analytics_events collection — to keep concerns decoupled and allow
// independent migration strategies.
type AnalyticsEvent struct {
	ID            string            `bson:"_id" json:"id"`
	EventID       string            `bson:"event_id" json:"event_id"`
	EventType     EventType         `bson:"event_type" json:"event_type"`
	CustomerID    string            `bson:"customer_id" json:"customer_id"`
	AggregateID   string            `bson:"aggregate_id" json:"aggregate_id"`
	AggregateType string            `bson:"aggregate_type" json:"aggregate_type"`
	Payload       map[string]any    `bson:"payload" json:"payload"`
	CorrelationID string            `bson:"correlation_id" json:"correlation_id"`
	OccurredAt    time.Time         `bson:"occurred_at" json:"occurred_at"`
	ProcessedAt   time.Time         `bson:"processed_at" json:"processed_at"`
	CreatedAt     time.Time         `bson:"created_at" json:"created_at"`
}

// NewAnalyticsEvent creates a new analytics event record.
func NewAnalyticsEvent(eventID string, eventType EventType, customerID, aggregateID, aggregateType, correlationID string, occurredAt time.Time, payload map[string]any) *AnalyticsEvent {
	now := time.Now().UTC()
	return &AnalyticsEvent{
		ID:            uuid.New().String(),
		EventID:       eventID,
		EventType:     eventType,
		CustomerID:    customerID,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Payload:       payload,
		CorrelationID: correlationID,
		OccurredAt:    occurredAt,
		ProcessedAt:   now,
		CreatedAt:     now,
	}
}

// AnalyticsError represents a domain-level error in the analytics module.
type AnalyticsError struct {
	Code    string
	Message string
}

func (e *AnalyticsError) Error() string {
	return e.Code + ": " + e.Message
}

// Domain errors.
var (
	ErrInvalidEventType   = &AnalyticsError{Code: "INVALID_EVENT_TYPE", Message: "unrecognized analytics event type"}
	ErrInvalidDateRange   = &AnalyticsError{Code: "INVALID_DATE_RANGE", Message: "start date must be before end date"}
	ErrAnalyticsNotFound  = &AnalyticsError{Code: "ANALYTICS_NOT_FOUND", Message: "no analytics data found for the given parameters"}
)

// TimeRange defines a start/end window for analytics queries.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Validate checks that the time range is valid.
func (tr TimeRange) Validate() error {
	if tr.Start.After(tr.End) {
		return ErrInvalidDateRange
	}
	return nil
}
