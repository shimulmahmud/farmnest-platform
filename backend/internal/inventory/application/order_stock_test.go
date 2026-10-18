package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/inventory/domain"
)

func setupWarehouseStock(t *testing.T, svc *Service, stock map[string]int) string {
	t.Helper()
	ctx := context.Background()
	w := domain.NewWarehouse("Main", "MAIN", "Dhaka")
	require.NoError(t, svc.warehouses.Create(ctx, w))
	for sku, qty := range stock {
		_, err := svc.CreateStock(ctx, CreateStockInput{SKU: sku, WarehouseID: w.ID, TotalStock: qty, LowStockThreshold: 1})
		require.NoError(t, err)
	}
	return w.ID
}

func TestDeductForOrder_LowersTotal_AndRestoreAddsBack(t *testing.T) {
	svc, _, _, _, _ := newTestService(t)
	ctx := context.Background()
	wh := setupWarehouseStock(t, svc, map[string]int{"A": 10, "B": 5})

	allocs, err := svc.DeductForOrder(ctx, "order-1", []OrderLine{{SKU: "A", Quantity: 2}, {SKU: "B", Quantity: 1}})
	require.NoError(t, err)
	require.Len(t, allocs, 2)

	a, _ := svc.GetStock(ctx, "A", wh)
	assert.Equal(t, 8, a.TotalStock, "total drops when the order is placed")
	assert.Equal(t, 0, a.ReservedStock, "nothing is left 'held'")

	require.NoError(t, svc.RestoreForOrder(ctx, "order-1", allocs))
	a, _ = svc.GetStock(ctx, "A", wh)
	b, _ := svc.GetStock(ctx, "B", wh)
	assert.Equal(t, 10, a.TotalStock, "cancel puts the units back")
	assert.Equal(t, 5, b.TotalStock)
}

func TestDeductForOrder_AllOrNothing_WhenOneLineIsShort(t *testing.T) {
	svc, _, _, _, _ := newTestService(t)
	ctx := context.Background()
	wh := setupWarehouseStock(t, svc, map[string]int{"A": 10, "B": 1})

	_, err := svc.DeductForOrder(ctx, "order-2", []OrderLine{{SKU: "A", Quantity: 3}, {SKU: "B", Quantity: 5}})
	require.Error(t, err)

	a, _ := svc.GetStock(ctx, "A", wh)
	assert.Equal(t, 10, a.TotalStock, "first line is rolled back when the second is short")
}

func TestDeductForOrder_ZeroStockCannotBeOrdered(t *testing.T) {
	svc, _, _, _, _ := newTestService(t)
	setupWarehouseStock(t, svc, map[string]int{"A": 0})

	_, err := svc.DeductForOrder(context.Background(), "order-3", []OrderLine{{SKU: "A", Quantity: 1}})
	require.Error(t, err)
}
