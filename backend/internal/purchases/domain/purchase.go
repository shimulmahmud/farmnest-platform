package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ecommerce/backend/pkg/pagination"
)

// Item is one SKU received in a purchase. Money is whole BDT taka.
type Item struct {
	SKU         string `bson:"sku" json:"sku"`
	Quantity    int    `bson:"quantity" json:"quantity"`
	UnitCost    int    `bson:"unit_cost" json:"unit_cost"`
	LineTotal   int    `bson:"line_total" json:"line_total"`
	StockBefore int    `bson:"stock_before" json:"stock_before"`
	StockAfter  int    `bson:"stock_after" json:"stock_after"`
	CostBefore  int    `bson:"cost_before" json:"cost_before"` // variant cost before this purchase
	CostAfter   int    `bson:"cost_after" json:"cost_after"`   // weighted-average cost after
}

// Purchase records stock bought from a supplier and received into a warehouse.
type Purchase struct {
	ID           string    `bson:"_id" json:"id"`
	Number       string    `bson:"number" json:"number"`
	SupplierName string    `bson:"supplier_name" json:"supplier_name"`
	Reference    string    `bson:"reference" json:"reference"` // supplier invoice number
	WarehouseID  string    `bson:"warehouse_id" json:"warehouse_id"`
	Items        []Item    `bson:"items" json:"items"`
	TotalCost    int       `bson:"total_cost" json:"total_cost"`
	Note         string    `bson:"note,omitempty" json:"note,omitempty"`
	ReceivedAt   time.Time `bson:"received_at" json:"received_at"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
}

// NewPurchase creates a purchase with a generated ID and number.
func NewPurchase(supplier, reference, warehouseID, note string, items []Item) *Purchase {
	now := time.Now().UTC()
	id := uuid.New().String()
	total := 0
	for _, it := range items {
		total += it.LineTotal
	}
	return &Purchase{
		ID:           id,
		Number:       fmt.Sprintf("PUR-%s-%s", now.Format("20060102"), id[:6]),
		SupplierName: supplier,
		Reference:    reference,
		WarehouseID:  warehouseID,
		Items:        items,
		TotalCost:    total,
		Note:         note,
		ReceivedAt:   now,
		CreatedAt:    now,
	}
}

// WeightedAverageCost blends existing stock at oldCost with new units at newCost.
// With no existing stock the new cost wins. Result is whole BDT, rounded.
func WeightedAverageCost(oldQty, oldCost, newQty, newCost int) int {
	if oldQty <= 0 {
		return newCost
	}
	total := oldQty + newQty
	return (oldQty*oldCost + newQty*newCost + total/2) / total
}

// Repository persists purchases.
type Repository interface {
	Create(ctx context.Context, p *Purchase) error
	FindByID(ctx context.Context, id string) (*Purchase, error)
	List(ctx context.Context, supplier string, p pagination.Params) ([]*Purchase, int64, error)
}

// StockReceiver adds received units to inventory.
type StockReceiver interface {
	// Receive adds qty to the SKU's stock in the warehouse (creating the stock
	// record if needed) and returns the total stock before the receipt.
	Receive(ctx context.Context, sku, warehouseID string, qty int, note string) (before int, err error)
	// WarehouseExists reports whether the warehouse exists.
	WarehouseExists(ctx context.Context, warehouseID string) error
}

// CostCatalog reads and updates variant cost prices.
type CostCatalog interface {
	// CurrentCost returns the variant's cost; NOT_FOUND if the SKU does not exist.
	CurrentCost(ctx context.Context, sku string) (int, error)
	SetCost(ctx context.Context, sku string, cost int) error
}
