package application

import (
	"context"

	"github.com/ecommerce/backend/internal/inventory/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
)

// OrderLine is a SKU and quantity an order needs.
type OrderLine struct {
	SKU      string
	Quantity int
}

// Allocation records where an order's units were taken from, so a cancellation
// can put them back in the same warehouse.
type Allocation struct {
	SKU         string
	WarehouseID string
	Quantity    int
}

// DeductForOrder removes the units from stock immediately when an order is
// placed. It is all-or-nothing: if any line cannot be covered, everything
// already deducted is put back and INSUFFICIENT_STOCK is returned. For each SKU
// the first active warehouse with enough available stock is used. Each step is
// atomic in the database, so two customers can never take the same unit.
func (s *Service) DeductForOrder(ctx context.Context, orderID string, lines []OrderLine) ([]Allocation, error) {
	if orderID == "" {
		return nil, errors.New("VALIDATION_ERROR", "order id is required", 422)
	}

	merged := map[string]int{}
	var order []string
	for _, l := range lines {
		if l.Quantity <= 0 {
			return nil, errors.New("VALIDATION_ERROR", "quantity must be positive for "+l.SKU, 422)
		}
		if _, ok := merged[l.SKU]; !ok {
			order = append(order, l.SKU)
		}
		merged[l.SKU] += l.Quantity
	}

	warehouses, _, err := s.warehouses.List(ctx, pagination.Params{Page: 1, PerPage: 100, Skip: 0, Limit: 100})
	if err != nil {
		return nil, err
	}

	var done []Allocation
	for _, sku := range order {
		qty := merged[sku]
		var taken *Allocation
		for _, w := range warehouses {
			if !w.IsActive {
				continue
			}
			// ReserveStock only succeeds if available >= qty (atomic); consuming
			// the hold right away lowers the total by the same amount.
			if _, err := s.stock.ReserveStock(ctx, sku, w.ID, qty); err != nil {
				continue
			}
			updated, err := s.stock.ConsumeStock(ctx, sku, w.ID, qty)
			if err != nil {
				_, _ = s.stock.ReleaseStock(ctx, sku, w.ID, qty)
				continue
			}
			_ = s.movements.CreateMovement(ctx, domain.NewMovement(
				sku, w.ID, domain.MovementConsumed, qty, orderID, "order placed: stock deducted"))
			if updated.IsLowStock() {
				s.publishEvent(ctx, "inventory.low_stock", &domain.InventoryEvent{
					EventType: "inventory.low_stock", SKU: sku, WarehouseID: w.ID,
					Quantity: updated.TotalStock, Stock: stockSnap(updated),
				})
			}
			taken = &Allocation{SKU: sku, WarehouseID: w.ID, Quantity: qty}
			break
		}
		if taken == nil {
			_ = s.RestoreForOrder(ctx, orderID, done) // roll back earlier lines
			return nil, errors.New("INSUFFICIENT_STOCK", "not enough stock for SKU "+sku, 409)
		}
		done = append(done, *taken)
	}
	return done, nil
}

// RestoreForOrder puts an order's units back into the warehouses they came from
// (cancellation or failed order creation). Every allocation is attempted; the
// first error, if any, is returned.
func (s *Service) RestoreForOrder(ctx context.Context, orderID string, allocs []Allocation) error {
	var firstErr error
	for _, a := range allocs {
		if _, err := s.stock.AdjustStock(ctx, a.SKU, a.WarehouseID, a.Quantity, "order cancelled: stock restored"); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		_ = s.movements.CreateMovement(ctx, domain.NewMovement(
			a.SKU, a.WarehouseID, domain.MovementInbound, a.Quantity, orderID, "order cancelled: stock restored"))
	}
	return firstErr
}

// CheckAvailability verifies every line could be covered by a single active
// warehouse right now, without taking any stock. Returns INSUFFICIENT_STOCK
// naming the first SKU that cannot be covered.
func (s *Service) CheckAvailability(ctx context.Context, lines []OrderLine) error {
	merged := map[string]int{}
	var order []string
	for _, l := range lines {
		if l.Quantity <= 0 {
			return errors.New("VALIDATION_ERROR", "quantity must be positive for "+l.SKU, 422)
		}
		if _, ok := merged[l.SKU]; !ok {
			order = append(order, l.SKU)
		}
		merged[l.SKU] += l.Quantity
	}
	warehouses, _, err := s.warehouses.List(ctx, pagination.Params{Page: 1, PerPage: 100, Skip: 0, Limit: 100})
	if err != nil {
		return err
	}
	for _, sku := range order {
		ok := false
		for _, w := range warehouses {
			if !w.IsActive {
				continue
			}
			item, err := s.stock.GetStock(ctx, sku, w.ID)
			if err == nil && item.AvailableStock() >= merged[sku] {
				ok = true
				break
			}
		}
		if !ok {
			return errors.New("INSUFFICIENT_STOCK", "not enough stock for SKU "+sku, 409)
		}
	}
	return nil
}
