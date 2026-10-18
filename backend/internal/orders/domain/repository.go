package domain

import (
	"context"

	"github.com/ecommerce/backend/pkg/pagination"
)

// OrderRepository defines order persistence operations.
type OrderRepository interface {
	// Create inserts a new order.
	Create(ctx context.Context, order *Order) error

	// FindByID retrieves an order by ID.
	FindByID(ctx context.Context, id string) (*Order, error)

	// FindByOrderNumber retrieves an order by its human-readable number.
	FindByOrderNumber(ctx context.Context, orderNumber string) (*Order, error)

	// List returns orders matching the filter with pagination.
	// Returns (orders, total count, error).
	List(ctx context.Context, filter *OrderFilter, p pagination.Params) ([]*Order, int64, error)

	// UpdateStatus atomically transitions an order's status.
	// Only succeeds if the current status matches expectedStatus.
	// Returns the updated order or an error if the transition is not possible.
	UpdateStatus(ctx context.Context, orderID string, expectedStatus, newStatus OrderStatus, reason string) (*Order, error)

	// SetReferences persists payment/shipment references (empty values are ignored).
	SetReferences(ctx context.Context, orderID, paymentRef, shipmentRef string) error

	// SetStockAllocations stores where the order's stock was taken from.
	SetStockAllocations(ctx context.Context, orderID string, allocs []StockAllocation) error

	// SetPaymentStatus records whether the money has been received.
	SetPaymentStatus(ctx context.Context, orderID string, status PaymentStatus) error

	// MarkStockRestored flags that a cancelled order's stock was put back.
	MarkStockRestored(ctx context.Context, orderID string) error

	// SetShipment records the carrier, tracking number and shipped time.
	SetShipment(ctx context.Context, orderID, carrier, trackingNumber string, courierCost int) (*Order, error)
}

// EventPublisher publishes order lifecycle events to NATS.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, event *OrderEvent) error
}
