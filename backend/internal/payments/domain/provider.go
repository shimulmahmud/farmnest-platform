package domain

import (
	"context"
	"time"
)

// ProviderResult is the response from a payment provider operation.
type ProviderResult struct {
	Success       bool
	ProviderRef   string
	ErrorMessage  string
}

// PaymentProvider abstracts third-party payment processing.
// Implementations must be safe for concurrent use.
type PaymentProvider interface {
	// CreatePayment initiates a payment with the external provider.
	// Returns a ProviderResult with the provider's reference on success.
	CreatePayment(ctx context.Context, payment *Payment) (*ProviderResult, error)

	// VerifyPayment checks the payment status with the external provider.
	VerifyPayment(ctx context.Context, payment *Payment) (*ProviderResult, error)

	// CapturePayment captures an authorized payment.
	CapturePayment(ctx context.Context, payment *Payment) (*ProviderResult, error)

	// RefundPayment initiates a refund with the external provider.
	RefundPayment(ctx context.Context, payment *Payment) (*ProviderResult, error)
}

// ProviderConfig holds configuration for a payment provider.
type ProviderConfig struct {
	Name      string
	Timeout   time.Duration
	APIKey    string
	SecretKey string
	Sandbox   bool
}
