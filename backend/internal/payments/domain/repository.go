package domain

import (
        "context"
)

// PaymentRepository defines payment persistence operations.
type PaymentRepository interface {
        // Create inserts a new payment. Returns the payment on success or
        // ErrDuplicatePayment if a payment already exists for this order+provider.
        Create(ctx context.Context, payment *Payment) error

        // FindByID retrieves a payment by ID.
        FindByID(ctx context.Context, id string) (*Payment, error)

        // FindByOrderID retrieves the payment for an order with a given provider.
        FindByOrderID(ctx context.Context, orderID, provider string) (*Payment, error)

        // UpdateStatus atomically sets the payment status and provider reference
        // if the current status matches expectedStatus.
        // Returns the updated payment or an error on mismatch.
        UpdateStatus(ctx context.Context, id string, expectedStatus, newStatus PaymentStatus, providerRef string) (*Payment, error)

        // FindByProviderRef retrieves a payment by its provider-specific reference.
        FindByProviderRef(ctx context.Context, provider, providerRef string) (*Payment, error)
}

// ErrDuplicatePayment is returned when a payment already exists for an order+provider.
// It is a sentinel that the application layer checks to implement idempotency.
var ErrDuplicatePayment = &duplicatePaymentError{}

type duplicatePaymentError struct{}

func (e *duplicatePaymentError) Error() string { return "payment already exists for this order and provider" }
