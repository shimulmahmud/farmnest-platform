package infrastructure

import (
	"context"

	invApp "github.com/ecommerce/backend/internal/inventory/application"
	"github.com/ecommerce/backend/internal/orders/domain"
)

type stockReserver struct {
	inv *invApp.Service
}

// NewStockReserver adapts the inventory service to domain.StockReserver.
func NewStockReserver(inv *invApp.Service) domain.StockReserver {
	return &stockReserver{inv: inv}
}

func (r *stockReserver) CheckAvailable(ctx context.Context, lines []domain.StockLine) error {
	in := make([]invApp.OrderLine, len(lines))
	for i, l := range lines {
		in[i] = invApp.OrderLine{SKU: l.SKU, Quantity: l.Quantity}
	}
	return r.inv.CheckAvailability(ctx, in)
}

func (r *stockReserver) DeductForOrder(ctx context.Context, orderID string, lines []domain.StockLine) ([]domain.StockAllocation, error) {
	in := make([]invApp.OrderLine, len(lines))
	for i, l := range lines {
		in[i] = invApp.OrderLine{SKU: l.SKU, Quantity: l.Quantity}
	}
	allocs, err := r.inv.DeductForOrder(ctx, orderID, in)
	if err != nil {
		return nil, err
	}
	out := make([]domain.StockAllocation, len(allocs))
	for i, a := range allocs {
		out[i] = domain.StockAllocation{SKU: a.SKU, WarehouseID: a.WarehouseID, Quantity: a.Quantity}
	}
	return out, nil
}

func (r *stockReserver) RestoreForOrder(ctx context.Context, orderID string, allocs []domain.StockAllocation) error {
	in := make([]invApp.Allocation, len(allocs))
	for i, a := range allocs {
		in[i] = invApp.Allocation{SKU: a.SKU, WarehouseID: a.WarehouseID, Quantity: a.Quantity}
	}
	return r.inv.RestoreForOrder(ctx, orderID, in)
}
