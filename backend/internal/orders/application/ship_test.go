package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/orders/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

type fakeStock struct {
	deducted, restored []string
	checked            int
	failDeduct         bool
	failCheck          bool
}

func (f *fakeStock) CheckAvailable(_ context.Context, _ []domain.StockLine) error {
	f.checked++
	if f.failCheck {
		return errors.New("INSUFFICIENT_STOCK", "not enough stock", 409)
	}
	return nil
}

func (f *fakeStock) DeductForOrder(_ context.Context, id string, lines []domain.StockLine) ([]domain.StockAllocation, error) {
	if f.failDeduct {
		return nil, errors.New("INSUFFICIENT_STOCK", "not enough stock", 409)
	}
	f.deducted = append(f.deducted, id)
	out := make([]domain.StockAllocation, len(lines))
	for i, l := range lines {
		out[i] = domain.StockAllocation{SKU: l.SKU, WarehouseID: "wh-1", Quantity: l.Quantity}
	}
	return out, nil
}

func (f *fakeStock) RestoreForOrder(_ context.Context, id string, _ []domain.StockAllocation) error {
	f.restored = append(f.restored, id)
	return nil
}

func newCODOrder(t *testing.T, svc *Service) (*domain.Order, error) {
	t.Helper()
	return svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerID: "cust-cod", Items: testItems(), Pricing: testPricing(),
		ShippingAddress: testAddress(), BillingAddress: testAddress(),
		PaymentMethod: domain.PaymentCOD,
	})
}

func TestOnlineOrder_PlacingDoesNotTouchStock_PaymentDoes(t *testing.T) {
	svc, repo, _ := newTestService(t)
	fs := &fakeStock{}
	svc.SetStock(fs)

	order := createTestOrder(t, svc, "cust-1") // ONLINE by default
	assert.Equal(t, domain.OrderPaymentPending, order.Status)
	assert.Equal(t, domain.PaymentUnpaid, order.PaymentStatus)
	assert.Equal(t, 1, fs.checked, "availability is checked at checkout submit")
	assert.Empty(t, fs.deducted, "nothing is deducted before payment")

	paid, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: string(domain.OrderPaid), PaymentReference: "pay-1"})
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentPaid, paid.PaymentStatus)
	assert.Equal(t, []string{order.ID}, fs.deducted, "stock is taken when payment completes")
	assert.Len(t, repo.orders[order.ID].StockAllocations, 1)
}

func TestOnlineOrder_RejectedAtCheckoutWhenOutOfStock(t *testing.T) {
	svc, repo, _ := newTestService(t)
	svc.SetStock(&fakeStock{failCheck: true})

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerID: "c", Items: testItems(), Pricing: testPricing(),
		ShippingAddress: testAddress(), BillingAddress: testAddress(),
	})
	require.Error(t, err)
	assert.Empty(t, repo.orders)
}

func TestOnlineOrder_PaidButSoldOutMeanwhile_GoesToRefundPending(t *testing.T) {
	svc, repo, _ := newTestService(t)
	fs := &fakeStock{}
	svc.SetStock(fs)
	order := createTestOrder(t, svc, "cust-1")

	fs.failDeduct = true // someone else took the last units while this customer was paying
	got, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: string(domain.OrderPaid), PaymentReference: "pay-1"})
	require.NoError(t, err, "no error, so the payment worker does not retry")
	assert.Equal(t, domain.OrderRefundPending, got.Status)
	assert.Equal(t, domain.PaymentPaid, repo.orders[order.ID].PaymentStatus)
	assert.Empty(t, fs.deducted)
}

func TestCODOrder_TakesStockAtOnce_Confirmed_Unpaid(t *testing.T) {
	svc, repo, _ := newTestService(t)
	fs := &fakeStock{}
	svc.SetStock(fs)

	order, err := newCODOrder(t, svc)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderConfirmed, order.Status)
	assert.Equal(t, domain.PaymentUnpaid, order.PaymentStatus)
	assert.Equal(t, domain.PaymentCOD, order.PaymentMethod)
	assert.Equal(t, []string{order.ID}, fs.deducted)
	assert.Len(t, repo.orders[order.ID].StockAllocations, 1)
}

func TestCODOrder_RejectedWhenOutOfStock(t *testing.T) {
	svc, repo, _ := newTestService(t)
	svc.SetStock(&fakeStock{failDeduct: true})

	_, err := newCODOrder(t, svc)
	require.Error(t, err)
	assert.Empty(t, repo.orders, "zero stock cannot be ordered")
}

func TestCancelCODOrder_RestoresStockOnce(t *testing.T) {
	svc, repo, _ := newTestService(t)
	fs := &fakeStock{}
	svc.SetStock(fs)
	order, err := newCODOrder(t, svc)
	require.NoError(t, err)

	_, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: string(domain.OrderCancelled), Reason: "customer request"})
	require.NoError(t, err)
	assert.Equal(t, []string{order.ID}, fs.restored)
	assert.True(t, repo.orders[order.ID].StockRestored)
}

func TestCancelUnpaidOnlineOrder_RestoresNothing(t *testing.T) {
	svc, _, _ := newTestService(t)
	fs := &fakeStock{}
	svc.SetStock(fs)
	order := createTestOrder(t, svc, "cust-1")

	_, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: string(domain.OrderCancelled)})
	require.NoError(t, err)
	assert.Empty(t, fs.restored, "no stock was taken, so none is put back")
}

func TestShipCODOrder_ThenMarkPaid(t *testing.T) {
	svc, _, _ := newTestService(t)
	fs := &fakeStock{}
	svc.SetStock(fs)
	order, err := newCODOrder(t, svc)
	require.NoError(t, err)

	shipped, err := svc.ShipOrder(context.Background(), order.ID, ShipInput{Carrier: "Steadfast", TrackingNumber: "SF123"})
	require.NoError(t, err)
	assert.Equal(t, domain.OrderShipped, shipped.Status)
	assert.Equal(t, "SF123", shipped.TrackingNumber)
	assert.Equal(t, 1, len(fs.deducted), "stock was taken once, when the order was placed")

	paid, err := svc.MarkPaid(context.Background(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentPaid, paid.PaymentStatus)
}

func TestMarkPaid_RejectsOnlineOrders(t *testing.T) {
	svc, _, _ := newTestService(t)
	order := createTestOrder(t, svc, "cust-1")
	_, err := svc.MarkPaid(context.Background(), order.ID)
	require.Error(t, err)
}

func TestShipOrder_UnpaidOnlineOrderCannotShip_AndNeedsTracking(t *testing.T) {
	svc, _, _ := newTestService(t)
	order := createTestOrder(t, svc, "cust-1")

	_, err := svc.ShipOrder(context.Background(), order.ID, ShipInput{Carrier: "Steadfast", TrackingNumber: "SF1"})
	require.Error(t, err)

	_, err = svc.ShipOrder(context.Background(), order.ID, ShipInput{Carrier: "Steadfast"})
	require.Error(t, err)
}

func TestCreateOrder_RejectsUnknownPaymentMethod(t *testing.T) {
	svc, _, _ := newTestService(t)
	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerID: "c", Items: testItems(), Pricing: testPricing(),
		ShippingAddress: testAddress(), BillingAddress: testAddress(), PaymentMethod: "BARTER",
	})
	require.Error(t, err)
}
