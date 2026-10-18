package domain

import (
	"context"
	"time"
)

// WebhookEventType represents the type of payment webhook event.
type WebhookEventType string

const (
	WebhookEventPaymentCompleted WebhookEventType = "payment.completed"
	WebhookEventPaymentFailed    WebhookEventType = "payment.failed"
	WebhookEventPaymentRefunded  WebhookEventType = "payment.refunded"
)

// WebhookEvent represents an incoming payment webhook from a provider.
type WebhookEvent struct {
	ID             string           `json:"id"`
	Type           WebhookEventType `json:"type"`
	Provider       string           `json:"provider"`
	PaymentRef     string           `json:"payment_ref"`     // provider's reference for the payment
	Amount         int              `json:"amount"`          // in cents
	Currency       string           `json:"currency"`
	Status         string           `json:"status"`         // provider's status string
	Timestamp      time.Time        `json:"timestamp"`
	RawPayload     []byte           `json:"-"` // original raw body for auditing
}

// ProcessedWebhook represents a persisted webhook event for idempotency.
type ProcessedWebhook struct {
	ID        string    `bson:"_id" json:"id"`
	WebhookID string    `bson:"webhook_id" json:"webhook_id"`
	Provider  string    `bson:"provider" json:"provider"`
	EventType string    `bson:"event_type" json:"event_type"`
	PaymentID string    `bson:"payment_id" json:"payment_id"`
	ProcessedAt time.Time `bson:"processed_at" json:"processed_at"`
}

// WebhookEventStore persists processed webhook events for idempotency.
type WebhookEventStore interface {
	// HasBeenProcessed checks whether a webhook with this ID has already been processed.
	HasBeenProcessed(ctx context.Context, webhookID string) (bool, error)

	// MarkProcessed records that a webhook has been processed.
	MarkProcessed(ctx context.Context, pw *ProcessedWebhook) error
}

// SignatureVerifier verifies webhook signatures from payment providers.
type SignatureVerifier interface {
	// Verify checks the signature of a webhook payload.
	// provider is the name of the payment provider (from the URL param).
	// signature is the value from the provider's signature header.
	// payload is the raw request body.
	Verify(ctx context.Context, provider, signature string, payload []byte) error
}

// PaymentEventPublisher publishes payment state change events.
type PaymentEventPublisher interface {
	// Publish sends a payment event to the message bus.
	Publish(ctx context.Context, event *PaymentEvent) error
}

// MapWebhookTypeToPaymentStatus maps a webhook event type to the target payment status.
func MapWebhookTypeToPaymentStatus(eventType WebhookEventType) PaymentStatus {
	switch eventType {
	case WebhookEventPaymentCompleted:
		return PaymentPaid
	case WebhookEventPaymentFailed:
		return PaymentFailed
	case WebhookEventPaymentRefunded:
		return PaymentRefunded
	default:
		return ""
	}
}
