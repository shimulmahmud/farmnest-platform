package infrastructure

import (
	"context"

	invApp "github.com/ecommerce/backend/internal/inventory/application"
	"github.com/ecommerce/backend/internal/purchases/domain"
	prodDomain "github.com/ecommerce/backend/internal/products/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

// stockReceiver adapts the inventory service.
type stockReceiver struct{ inv *invApp.Service }

// NewStockReceiver adapts the inventory service to domain.StockReceiver.
func NewStockReceiver(inv *invApp.Service) domain.StockReceiver { return &stockReceiver{inv: inv} }

func (r *stockReceiver) WarehouseExists(ctx context.Context, id string) error {
	_, err := r.inv.GetWarehouse(ctx, id)
	return err
}

func (r *stockReceiver) Receive(ctx context.Context, sku, warehouseID string, qty int, note string) (int, error) {
	item, err := r.inv.GetStock(ctx, sku, warehouseID)
	if err != nil {
		if appErr, ok := errors.IsAppError(err); ok && appErr.Code == "NOT_FOUND" {
			if _, cErr := r.inv.CreateStock(ctx, invApp.CreateStockInput{
				SKU: sku, WarehouseID: warehouseID, TotalStock: qty, LowStockThreshold: 5,
			}); cErr != nil {
				return 0, cErr
			}
			return 0, nil
		}
		return 0, err
	}
	before := item.TotalStock
	if _, err := r.inv.AdjustStock(ctx, invApp.AdjustStockInput{
		SKU: sku, WarehouseID: warehouseID, Quantity: qty, Reason: note,
	}); err != nil {
		return 0, err
	}
	return before, nil
}

// costCatalog adapts the product repository.
type costCatalog struct{ products prodDomain.ProductRepository }

// NewCostCatalog adapts the product repository to domain.CostCatalog.
func NewCostCatalog(p prodDomain.ProductRepository) domain.CostCatalog { return &costCatalog{products: p} }

func (c *costCatalog) CurrentCost(ctx context.Context, sku string) (int, error) {
	p, err := c.products.FindBySKU(ctx, sku)
	if err != nil {
		return 0, err
	}
	for _, v := range p.Variants {
		if v.SKU == sku {
			return v.CostPrice, nil
		}
	}
	return 0, errors.ErrNotFound
}

func (c *costCatalog) SetCost(ctx context.Context, sku string, cost int) error {
	return c.products.SetVariantCost(ctx, sku, cost)
}
