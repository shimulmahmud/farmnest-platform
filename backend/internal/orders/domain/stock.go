package domain

import "context"

// StockLine is a SKU and quantity of an order.
type StockLine struct {
	SKU      string
	Quantity int
}

// StockAllocation records which warehouse an order's units were taken from.
type StockAllocation struct {
	SKU         string `bson:"sku" json:"sku"`
	WarehouseID string `bson:"warehouse_id" json:"warehouse_id"`
	Quantity    int    `bson:"quantity" json:"quantity"`
}

// StockReserver ties orders to inventory. Optional: when nil, orders do not
// touch stock (used by tests).
type StockReserver interface {
	// CheckAvailable verifies the lines could be fulfilled right now WITHOUT
	// taking any stock (used when an online order is created, before payment).
	CheckAvailable(ctx context.Context, lines []StockLine) error
	// DeductForOrder removes the units from stock when the order is placed,
	// all-or-nothing. Returns INSUFFICIENT_STOCK (409) when a line cannot be covered.
	DeductForOrder(ctx context.Context, orderID string, lines []StockLine) ([]StockAllocation, error)
	// RestoreForOrder puts the units back (order cancelled).
	RestoreForOrder(ctx context.Context, orderID string, allocs []StockAllocation) error
}
