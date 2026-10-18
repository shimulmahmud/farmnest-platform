package application

import (
	"context"
	"fmt"

	"github.com/ecommerce/backend/internal/purchases/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
)

// Service implements stock-receiving use cases.
type Service struct {
	repo  domain.Repository
	stock domain.StockReceiver
	costs domain.CostCatalog
}

// NewService creates a purchases service.
func NewService(repo domain.Repository, stock domain.StockReceiver, costs domain.CostCatalog) *Service {
	return &Service{repo: repo, stock: stock, costs: costs}
}

// ItemInput is one line to receive.
type ItemInput struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	UnitCost int    `json:"unit_cost"` // whole BDT per unit
}

// ReceiveInput is a stock purchase to receive.
type ReceiveInput struct {
	SupplierName string      `json:"supplier_name"`
	Reference    string      `json:"reference"`
	WarehouseID  string      `json:"warehouse_id"`
	Note         string      `json:"note"`
	Items        []ItemInput `json:"items"`
}

// Receive validates the purchase, adds the units to stock, updates each
// variant's cost to the weighted average, and records the purchase.
// All SKUs and the warehouse are validated before any stock is changed.
func (s *Service) Receive(ctx context.Context, in ReceiveInput) (*domain.Purchase, error) {
	if in.SupplierName == "" || in.WarehouseID == "" {
		return nil, errors.New("VALIDATION_ERROR", "supplier_name and warehouse_id are required", 422)
	}
	if len(in.Items) == 0 {
		return nil, errors.New("VALIDATION_ERROR", "at least one item is required", 422)
	}
	if err := s.stock.WarehouseExists(ctx, in.WarehouseID); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	oldCosts := make([]int, len(in.Items))
	for i, it := range in.Items {
		if it.SKU == "" || it.Quantity <= 0 || it.UnitCost <= 0 {
			return nil, errors.New("VALIDATION_ERROR",
				fmt.Sprintf("item[%d]: sku, quantity > 0 and unit_cost > 0 are required", i), 422)
		}
		if seen[it.SKU] {
			return nil, errors.New("VALIDATION_ERROR", "duplicate sku in purchase: "+it.SKU, 422)
		}
		seen[it.SKU] = true
		c, err := s.costs.CurrentCost(ctx, it.SKU)
		if err != nil {
			return nil, errors.New("VALIDATION_ERROR", "unknown SKU (create the product first): "+it.SKU, 422)
		}
		oldCosts[i] = c
	}

	items := make([]domain.Item, len(in.Items))
	for i, it := range in.Items {
		before, err := s.stock.Receive(ctx, it.SKU, in.WarehouseID, it.Quantity, "purchase from "+in.SupplierName)
		if err != nil {
			return nil, err
		}
		newCost := domain.WeightedAverageCost(before, oldCosts[i], it.Quantity, it.UnitCost)
		if err := s.costs.SetCost(ctx, it.SKU, newCost); err != nil {
			return nil, err
		}
		items[i] = domain.Item{
			SKU: it.SKU, Quantity: it.Quantity, UnitCost: it.UnitCost, LineTotal: it.Quantity * it.UnitCost,
			StockBefore: before, StockAfter: before + it.Quantity,
			CostBefore: oldCosts[i], CostAfter: newCost,
		}
	}

	purchase := domain.NewPurchase(in.SupplierName, in.Reference, in.WarehouseID, in.Note, items)
	if err := s.repo.Create(ctx, purchase); err != nil {
		return nil, err
	}
	return purchase, nil
}

// Get returns a purchase by ID.
func (s *Service) Get(ctx context.Context, id string) (*domain.Purchase, error) {
	return s.repo.FindByID(ctx, id)
}

// List returns purchases, newest first, optionally filtered by supplier name.
func (s *Service) List(ctx context.Context, supplier string, p pagination.Params) ([]*domain.Purchase, int64, error) {
	return s.repo.List(ctx, supplier, p)
}
