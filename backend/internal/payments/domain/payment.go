package domain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the lifecycle state of a payment.
type PaymentStatus string

const (
	PaymentPending      PaymentStatus = "PENDING"
	PaymentAuthorized   PaymentStatus = "AUTHORIZED"
	PaymentPaid         PaymentStatus = "PAID"
	PaymentFailed       PaymentStatus = "FAILED"
	PaymentRefundPending PaymentStatus = "REFUND_PENDING"
	PaymentRefunded     PaymentStatus = "REFUNDED"
)

// Payment represents a payment transaction linked to an order.
type Payment struct {
	ID              string            `bson:"_id" json:"id"`
	OrderID         string            `bson:"order_id" json:"order_id"`
	Provider        string            `bson:"provider" json:"provider"`
	ProviderRef     string            `bson:"provider_reference" json:"provider_reference"`
	Amount          int               `bson:"amount" json:"amount"`           // in cents
	Currency        string            `bson:"currency" json:"currency"`
	Status          PaymentStatus     `bson:"status" json:"status"`
	Metadata        map[string]string `bson:"metadata" json:"metadata,omitempty"`
	CreatedAt       time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time         `bson:"updated_at" json:"updated_at"`
}

// IsFinal returns true if the payment is in a terminal state.
func (s PaymentStatus) IsFinal() bool {
	return s == PaymentPaid || s == PaymentFailed || s == PaymentRefunded
}

// NewPayment creates a new payment in PENDING status.
func NewPayment(orderID, provider, currency string, amount int) *Payment {
	now := time.Now().UTC()
	return &Payment{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		Provider:  provider,
		Amount:    amount,
		Currency:  currency,
		Status:    PaymentPending,
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// PaymentEvent is published on state changes.
type PaymentEvent struct {
	EventType  string `json:"event_type"`
	PaymentID  string `json:"payment_id"`
	OrderID    string `json:"order_id"`
	Provider   string `json:"provider"`
	Amount     int    `json:"amount"`
	Currency   string `json:"currency"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	ProviderRef string `json:"provider_reference,omitempty"`
}
