package application

import (
        "context"
        "sync"
        "sync/atomic"
        "testing"

        "github.com/ecommerce/backend/internal/orders/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Mocks ---

func copyOrder(o *domain.Order) *domain.Order {
        cp := *o
        cp.Items = make([]domain.OrderItem, len(o.Items))
        copy(cp.Items, o.Items)
        cp.StatusHistory = make([]domain.StatusHistoryEntry, len(o.StatusHistory))
        copy(cp.StatusHistory, o.StatusHistory)
        return &cp
}

type mockOrderRepo struct {
        mu     sync.Mutex
        orders map[string]*domain.Order
}

func newMockOrderRepo() *mockOrderRepo {
        return &mockOrderRepo{orders: make(map[string]*domain.Order)}
}

func (r *mockOrderRepo) Create(_ context.Context, order *domain.Order) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        if _, ok := r.orders[order.ID]; ok {
                return errors.New("CONFLICT", "order already exists", 409)
        }
        cp := copyOrder(order)
        r.orders[order.ID] = cp
        return nil
}

func (r *mockOrderRepo) FindByID(_ context.Context, id string) (*domain.Order, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        o, ok := r.orders[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return copyOrder(o), nil
}

func (r *mockOrderRepo) FindByOrderNumber(_ context.Context, orderNumber string) (*domain.Order, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        for _, o := range r.orders {
                if o.OrderNumber == orderNumber {
                        return copyOrder(o), nil
                }
        }
        return nil, errors.ErrNotFound
}

func (r *mockOrderRepo) List(_ context.Context, filter *domain.OrderFilter, p pagination.Params) ([]*domain.Order, int64, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        var result []*domain.Order
        for _, o := range r.orders {
                if filter != nil {
                        if filter.CustomerID != "" && o.CustomerID != filter.CustomerID {
                                continue
                        }
                        if filter.Status != "" && string(o.Status) != filter.Status {
                                continue
                        }
                }
                result = append(result, copyOrder(o))
        }
        total := int64(len(result))
        start := p.Skip
        if start > len(result) {
                start = len(result)
        }
        end := start + p.Limit
        if end > len(result) {
                end = len(result)
        }
        return result[start:end], total, nil
}

func (r *mockOrderRepo) UpdateStatus(_ context.Context, orderID string, expectedStatus, newStatus domain.OrderStatus, reason string) (*domain.Order, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        o, ok := r.orders[orderID]
        if !ok {
                return nil, errors.New("CONFLICT", "order not found or status has changed", 409)
        }
        if o.Status != expectedStatus {
                return nil, errors.New("CONFLICT", "order not found or status has changed", 409)
        }
        if err := o.TransitionTo(newStatus, reason); err != nil {
                return nil, errors.New("BAD_REQUEST", err.Error(), 400)
        }
        return copyOrder(o), nil
}

func (r *mockOrderRepo) SetReferences(_ context.Context, orderID, paymentRef, shipmentRef string) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        o, ok := r.orders[orderID]
        if !ok {
                return errors.ErrNotFound
        }
        if paymentRef != "" {
                o.PaymentReference = paymentRef
        }
        if shipmentRef != "" {
                o.ShipmentReference = shipmentRef
        }
        return nil
}

func (r *mockOrderRepo) SetStockAllocations(_ context.Context, orderID string, allocs []domain.StockAllocation) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        o, ok := r.orders[orderID]
        if !ok {
                return errors.ErrNotFound
        }
        o.StockAllocations = allocs
        return nil
}

func (r *mockOrderRepo) SetPaymentStatus(_ context.Context, orderID string, st domain.PaymentStatus) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        o, ok := r.orders[orderID]
        if !ok {
                return errors.ErrNotFound
        }
        o.PaymentStatus = st
        return nil
}

func (r *mockOrderRepo) MarkStockRestored(_ context.Context, orderID string) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        o, ok := r.orders[orderID]
        if !ok {
                return errors.ErrNotFound
        }
        o.StockRestored = true
        return nil
}

func (r *mockOrderRepo) SetShipment(_ context.Context, orderID, carrier, tracking string, courierCost int) (*domain.Order, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        o, ok := r.orders[orderID]
        if !ok {
                return nil, errors.ErrNotFound
        }
        o.Carrier, o.TrackingNumber, o.ShipmentReference, o.CourierCost = carrier, tracking, tracking, courierCost
        return copyOrder(o), nil
}

type mockEventPublisher struct {
        mu     sync.Mutex
        events map[string]int
}

func newMockEventPublisher() *mockEventPublisher {
        return &mockEventPublisher{events: make(map[string]int)}
}

func (p *mockEventPublisher) Publish(_ context.Context, subject string, _ *domain.OrderEvent) error {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.events[subject]++
        return nil
}

func (p *mockEventPublisher) Count(subject string) int {
        p.mu.Lock()
        defer p.mu.Unlock()
        return p.events[subject]
}

// --- Helpers ---

func newTestService(t *testing.T) (*Service, *mockOrderRepo, *mockEventPublisher) {
        t.Helper()
        repo := newMockOrderRepo()
        pub := newMockEventPublisher()
        svc := NewService(repo, pub)
        return svc, repo, pub
}

func testItems() []domain.OrderItem {
        return []domain.OrderItem{
                {
                        ProductID: "prod-1",
                        VariantID: "var-1",
                        Name:      "Test Product",
                        SKU:       "SKU-001",
                        Quantity:  2,
                        Pricing: domain.OrderItemPricing{
                                UnitPrice: 1000,
                                Discount:  0,
                                Total:     2000,
                        },
                },
        }
}

func testPricing() domain.OrderPricing {
        return domain.OrderPricing{
                Subtotal: 2000,
                Discount: 0,
                Tax:      200,
                Shipping: 500,
                Total:    2700,
        }
}

func testAddress() *domain.Address {
        return &domain.Address{
                Line1:      "123 Main St",
                City:       "Springfield",
                State:      "IL",
                PostalCode: "62701",
                Country:    "US",
        }
}

func createTestOrder(t *testing.T, svc *Service, customerID string) *domain.Order {
        t.Helper()
        order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
                CustomerID:      customerID,
                Items:           testItems(),
                Pricing:         testPricing(),
                ShippingAddress: testAddress(),
                BillingAddress:  testAddress(),
        })
        require.NoError(t, err)
        require.NotNil(t, order)
        return order
}

// walkToStatus transitions an order through the known path to reach a target status.
// Orders start at PAYMENT_PENDING after creation.
func walkToStatus(t *testing.T, svc *Service, orderID string, target domain.OrderStatus) *domain.Order {
        t.Helper()
        var order *domain.Order
        var err error

        switch target {
        case domain.OrderPaymentPending:
                return createTestOrder(t, svc, "cust-1")
        case domain.OrderPaid:
                order = createTestOrder(t, svc, "cust-1")
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "PAID", Reason: "test"})
        case domain.OrderProcessing:
                order = createTestOrder(t, svc, "cust-1")
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "PAID", Reason: "test"})
                require.NoError(t, err)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "PROCESSING", Reason: "test"})
        case domain.OrderShipped:
                order = createTestOrder(t, svc, "cust-1")
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "PAID", Reason: "test"})
                require.NoError(t, err)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "PROCESSING", Reason: "test"})
                require.NoError(t, err)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "SHIPPED", Reason: "test"})
        case domain.OrderDelivered:
                order = createTestOrder(t, svc, "cust-1")
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "PAID", Reason: "test"})
                require.NoError(t, err)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "PROCESSING", Reason: "test"})
                require.NoError(t, err)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "SHIPPED", Reason: "test"})
                require.NoError(t, err)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "DELIVERED", Reason: "test"})
        case domain.OrderRefundPending:
                order = walkToStatus(t, svc, orderID, domain.OrderDelivered)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "REFUND_PENDING", Reason: "test"})
        case domain.OrderCancelled:
                order = createTestOrder(t, svc, "cust-1")
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "CANCELLED", Reason: "test"})
        case domain.OrderRefunded:
                order = walkToStatus(t, svc, orderID, domain.OrderRefundPending)
                order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: "REFUNDED", Reason: "test"})
        default:
                t.Fatalf("unsupported target status: %s", target)
        }
        require.NoError(t, err)
        return order
}

// --- CreateOrder Tests ---

func TestCreateOrder_Success(t *testing.T) {
        svc, _, pub := newTestService(t)

        order := createTestOrder(t, svc, "cust-1")

        assert.NotEmpty(t, order.ID)
        assert.NotEmpty(t, order.OrderNumber)
        assert.Equal(t, "cust-1", order.CustomerID)
        assert.Len(t, order.Items, 1)
        assert.Equal(t, domain.OrderPaymentPending, order.Status)
        assert.Len(t, order.StatusHistory, 2)
        assert.Equal(t, 2700, order.Pricing.Total)
        assert.Equal(t, 1, pub.Count("order.created"))
}

func TestCreateOrder_EmptyCustomerID(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, err := svc.CreateOrder(context.Background(), CreateOrderInput{
                CustomerID: "",
                Items:      testItems(),
                Pricing:    testPricing(),
        })

        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
        assert.Equal(t, 422, appErr.Status)
}

func TestCreateOrder_NoItems(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, err := svc.CreateOrder(context.Background(), CreateOrderInput{
                CustomerID: "cust-1",
                Items:      []domain.OrderItem{},
                Pricing:    testPricing(),
        })

        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

// --- GetOrderByID Tests ---

func TestGetOrderByID_Success(t *testing.T) {
        svc, _, _ := newTestService(t)
        created := createTestOrder(t, svc, "cust-1")

        found, err := svc.GetOrderByID(context.Background(), created.ID)
        require.NoError(t, err)
        assert.Equal(t, created.ID, found.ID)
        assert.Equal(t, created.OrderNumber, found.OrderNumber)
}

func TestGetOrderByID_NotFound(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, err := svc.GetOrderByID(context.Background(), "nonexistent")
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "NOT_FOUND", appErr.Code)
}

// --- ListOrders Tests ---

func TestListOrders_Success(t *testing.T) {
        svc, _, _ := newTestService(t)

        createTestOrder(t, svc, "cust-1")
        createTestOrder(t, svc, "cust-1")
        createTestOrder(t, svc, "cust-2")

        orders, total, err := svc.ListOrders(context.Background(), "cust-1", "", pagination.Params{Page: 1, PerPage: 20, Skip: 0, Limit: 20})
        require.NoError(t, err)
        assert.Equal(t, int64(2), total)
        assert.Len(t, orders, 2)
}

func TestListOrders_EmptyCustomerID(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, _, err := svc.ListOrders(context.Background(), "", "", pagination.Params{})
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestListOrders_ByStatus(t *testing.T) {
        svc, _, _ := newTestService(t)
        order := createTestOrder(t, svc, "cust-1")

        orders, total, err := svc.ListOrders(context.Background(), "cust-1", "PAYMENT_PENDING", pagination.Params{Page: 1, PerPage: 20, Skip: 0, Limit: 20})
        require.NoError(t, err)
        assert.Equal(t, int64(1), total)
        assert.Equal(t, order.ID, orders[0].ID)

        orders2, total2, err := svc.ListOrders(context.Background(), "cust-1", "DELIVERED", pagination.Params{Page: 1, PerPage: 20, Skip: 0, Limit: 20})
        require.NoError(t, err)
        assert.Equal(t, int64(0), total2)
        assert.Len(t, orders2, 0)
}

func TestListAllOrders_Success(t *testing.T) {
        svc, _, _ := newTestService(t)

        createTestOrder(t, svc, "cust-1")
        createTestOrder(t, svc, "cust-2")

        orders, total, err := svc.ListAllOrders(context.Background(), "", "", pagination.Params{Page: 1, PerPage: 20, Skip: 0, Limit: 20})
        require.NoError(t, err)
        assert.Equal(t, int64(2), total)
        assert.Len(t, orders, 2)
}

func TestListAllOrders_WithFilters(t *testing.T) {
        svc, _, _ := newTestService(t)

        createTestOrder(t, svc, "cust-1")
        createTestOrder(t, svc, "cust-2")

        orders, total, err := svc.ListAllOrders(context.Background(), "PAYMENT_PENDING", "cust-1", pagination.Params{Page: 1, PerPage: 20, Skip: 0, Limit: 20})
        require.NoError(t, err)
        assert.Equal(t, int64(1), total)
        assert.Len(t, orders, 1)
}

// --- Valid Transition Tests ---
//
// Known valid transitions (from service perspective, starting from PAYMENT_PENDING):
//   PAYMENT_PENDING -> PAID
//   PAYMENT_PENDING -> CANCELLED
//   PAID -> PROCESSING
//   PAID -> CANCELLED
//   PAID -> REFUND_PENDING
//   PROCESSING -> SHIPPED
//   PROCESSING -> CANCELLED
//   SHIPPED -> DELIVERED
//   DELIVERED -> REFUND_PENDING
//   REFUND_PENDING -> REFUNDED
//   REFUND_PENDING -> PAID

func TestValidTransition_PaymentPending_To_Paid(t *testing.T) {
        svc, _, pub := newTestService(t)
        order := createTestOrder(t, svc, "cust-1")

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PAID",
                Reason: "payment confirmed",
                PaymentReference: "PAY-123",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderPaid, updated.Status)
        assert.Equal(t, 1, pub.Count("order.paid"))
}

func TestValidTransition_PaymentPending_To_Cancelled(t *testing.T) {
        svc, _, pub := newTestService(t)
        order := createTestOrder(t, svc, "cust-1")

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "CANCELLED",
                Reason: "customer cancelled",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderCancelled, updated.Status)
        assert.Equal(t, 1, pub.Count("order.cancelled"))
}

func TestValidTransition_Paid_To_Processing(t *testing.T) {
        svc, _, _ := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderPaid)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PROCESSING",
                Reason: "fulfillment started",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderProcessing, updated.Status)
}

func TestValidTransition_Paid_To_Cancelled(t *testing.T) {
        svc, _, pub := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderPaid)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "CANCELLED",
                Reason: "refund before processing",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderCancelled, updated.Status)
        assert.Equal(t, 1, pub.Count("order.cancelled"))
}

func TestValidTransition_Paid_To_RefundPending(t *testing.T) {
        svc, _, _ := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderPaid)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "REFUND_PENDING",
                Reason: "customer requested refund",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderRefundPending, updated.Status)
}

func TestValidTransition_Processing_To_Shipped(t *testing.T) {
        svc, _, pub := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderProcessing)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "SHIPPED",
                Reason: "shipped via courier",
                ShipmentReference: "SHIP-001",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderShipped, updated.Status)
        assert.Equal(t, "SHIP-001", updated.ShipmentReference)
        assert.Equal(t, 1, pub.Count("order.shipped"))
}

func TestValidTransition_Processing_To_Cancelled(t *testing.T) {
        svc, _, pub := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderProcessing)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "CANCELLED",
                Reason: "out of stock",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderCancelled, updated.Status)
        assert.Equal(t, 1, pub.Count("order.cancelled"))
}

func TestValidTransition_Shipped_To_Delivered(t *testing.T) {
        svc, _, pub := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderShipped)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "DELIVERED",
                Reason: "delivered to customer",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderDelivered, updated.Status)
        assert.Equal(t, 1, pub.Count("order.delivered"))
}

func TestValidTransition_Delivered_To_RefundPending(t *testing.T) {
        svc, _, _ := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderDelivered)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "REFUND_PENDING",
                Reason: "return requested",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderRefundPending, updated.Status)
}

func TestValidTransition_RefundPending_To_Refunded(t *testing.T) {
        svc, _, _ := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderRefundPending)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "REFUNDED",
                Reason: "refund completed",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderRefunded, updated.Status)
}

func TestValidTransition_RefundPending_To_Paid(t *testing.T) {
        svc, _, _ := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderRefundPending)

        updated, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PAID",
                Reason: "refund denied",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderPaid, updated.Status)
}

// --- Invalid Transition Tests ---

type invalidCase struct {
        name string
        from domain.OrderStatus
        to   domain.OrderStatus
}

func TestInvalidTransitions(t *testing.T) {
        cases := []invalidCase{
                // Terminal states cannot transition
                {"DELIVERED -> SHIPPED", domain.OrderDelivered, domain.OrderShipped},
                {"CANCELLED -> PENDING", domain.OrderCancelled, domain.OrderPending},
        {"REFUNDED -> PAID", domain.OrderRefunded, domain.OrderPaid},
                {"REFUNDED -> REFUND_PENDING", domain.OrderRefunded, domain.OrderRefundPending},

                // Cannot go backwards (except REFUND_PENDING -> PAID which is valid)
                {"SHIPPED -> PAID", domain.OrderShipped, domain.OrderPaid},
                {"SHIPPED -> PENDING", domain.OrderShipped, domain.OrderPending},
                {"PAID -> PENDING", domain.OrderPaid, domain.OrderPending},
                {"PAID -> PAYMENT_PENDING", domain.OrderPaid, domain.OrderPaymentPending},
                {"PROCESSING -> PENDING", domain.OrderProcessing, domain.OrderPending},
                {"PROCESSING -> PAID", domain.OrderProcessing, domain.OrderPaid},

                // Cannot skip states
                {"PAYMENT_PENDING -> PROCESSING", domain.OrderPaymentPending, domain.OrderProcessing},
                {"PAYMENT_PENDING -> SHIPPED", domain.OrderPaymentPending, domain.OrderShipped},

                // Cannot go between unrelated states
                {"SHIPPED -> REFUND_PENDING", domain.OrderShipped, domain.OrderRefundPending},
                {"DELIVERED -> CANCELLED", domain.OrderDelivered, domain.OrderCancelled},

                // Self-transitions
                {"PAID -> PAID", domain.OrderPaid, domain.OrderPaid},
                {"PROCESSING -> PROCESSING", domain.OrderProcessing, domain.OrderProcessing},
                {"CANCELLED -> CANCELLED", domain.OrderCancelled, domain.OrderCancelled},
        }

        for _, tc := range cases {
                t.Run(tc.name, func(t *testing.T) {
                        svc, _, _ := newTestService(t)
                        order := walkToStatus(t, svc, "", tc.from)

                        _, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                                Status: string(tc.to),
                                Reason: "invalid test",
                        })
                        assert.Error(t, err)
                        appErr, ok := errors.IsAppError(err)
                        require.True(t, ok)
                        assert.Equal(t, "BAD_REQUEST", appErr.Code)
                        assert.Contains(t, appErr.Message, "invalid status transition")
                })
        }
}

// --- Duplicate Event Tests ---

func TestDuplicateEvent_NotPublishedOnCreation(t *testing.T) {
        svc, _, pub := newTestService(t)

        order := createTestOrder(t, svc, "cust-1") // publishes order.created once

        // Transition to PAID - publishes order.paid
        _, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PAID",
                Reason: "payment received",
        })
        require.NoError(t, err)

        assert.Equal(t, 1, pub.Count("order.created"))
        assert.Equal(t, 1, pub.Count("order.paid"))
}

func TestDuplicateTransition_Rejected(t *testing.T) {
        svc, _, pub := newTestService(t)

        order := createTestOrder(t, svc, "cust-1")

        // First transition succeeds
        _, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PAID",
                Reason: "payment received",
        })
        require.NoError(t, err)

        // Second attempt with stale status fails
        _, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PAID",
                Reason: "duplicate payment",
        })
        assert.Error(t, err)

        // Only one order.paid event
        assert.Equal(t, 1, pub.Count("order.paid"))
}

// --- Concurrent Tests ---

func TestConcurrentStatusTransition_OnlyOneSucceeds(t *testing.T) {
        svc, _, _ := newTestService(t)

        order := createTestOrder(t, svc, "cust-1")

        var wg sync.WaitGroup
        var successCount int64
        var failCount int64

        const goroutines = 10
        wg.Add(goroutines)
        for i := 0; i < goroutines; i++ {
                go func() {
                        defer wg.Done()
                        _, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                                Status: "PAID",
                                Reason: "concurrent payment",
                        })
                        if err != nil {
                                atomic.AddInt64(&failCount, 1)
                        } else {
                                atomic.AddInt64(&successCount, 1)
                        }
                }()
        }
        wg.Wait()

        assert.Equal(t, int64(1), successCount, "exactly one transition should succeed")
        assert.Equal(t, int64(goroutines-1), failCount, "all others should fail")
}

// --- Full Flow Tests ---

func TestFullHappyPath(t *testing.T) {
        svc, _, pub := newTestService(t)

        order := createTestOrder(t, svc, "cust-1")
        assert.Equal(t, domain.OrderPaymentPending, order.Status)

        // PAYMENT_PENDING -> PAID
        order, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PAID", Reason: "payment confirmed", PaymentReference: "PAY-001",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderPaid, order.Status)

        // PAID -> PROCESSING
        order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PROCESSING", Reason: "fulfillment started",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderProcessing, order.Status)

        // PROCESSING -> SHIPPED
        order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "SHIPPED", Reason: "shipped", ShipmentReference: "SHIP-12345",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderShipped, order.Status)
        assert.Equal(t, "SHIP-12345", order.ShipmentReference)

        // SHIPPED -> DELIVERED
        order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "DELIVERED", Reason: "delivered",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderDelivered, order.Status)

        // Verify events
        assert.Equal(t, 1, pub.Count("order.created"))
        assert.Equal(t, 1, pub.Count("order.paid"))
        assert.Equal(t, 1, pub.Count("order.shipped"))
        assert.Equal(t, 1, pub.Count("order.delivered"))

        // Verify status history
        expectedHistory := []domain.OrderStatus{
                domain.OrderPending,
                domain.OrderPaymentPending,
                domain.OrderPaid,
                domain.OrderProcessing,
                domain.OrderShipped,
                domain.OrderDelivered,
        }
        require.Len(t, order.StatusHistory, len(expectedHistory))
        for i, expected := range expectedHistory {
                assert.Equal(t, expected, order.StatusHistory[i].Status, "history entry %d", i)
        }
}

func TestRefundFlow(t *testing.T) {
        svc, _, _ := newTestService(t)

        order := walkToStatus(t, svc, "", domain.OrderDelivered)

        // DELIVERED -> REFUND_PENDING
        order, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "REFUND_PENDING", Reason: "return requested",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderRefundPending, order.Status)

        // REFUND_PENDING -> REFUNDED
        order, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "REFUNDED", Reason: "refund processed",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderRefunded, order.Status)
}

func TestRefundReversal(t *testing.T) {
        svc, _, _ := newTestService(t)

        order := walkToStatus(t, svc, "", domain.OrderRefundPending)

        // REFUND_PENDING -> PAID (reversal)
        order, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "PAID", Reason: "refund denied",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.OrderPaid, order.Status)
}

// --- UpdateStatus Edge Cases ---

func TestUpdateStatus_OrderNotFound(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, err := svc.UpdateOrderStatus(context.Background(), "nonexistent", UpdateStatusInput{
                Status: "PAID", Reason: "test",
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestUpdateStatus_ConcurrentConflict(t *testing.T) {
        svc, repo, _ := newTestService(t)

        order := createTestOrder(t, svc, "cust-1") // PAYMENT_PENDING

        // Directly mutate the order status in the repo to simulate a concurrent
        // change that happened between the service's FindByID and UpdateStatus.
        // We use a separate transition that is valid from the fetched state but
        // the repo will reject due to status mismatch.
        //
        // To test the repo's CONFLICT path directly, we call the mock repo
        // UpdateStatus with a stale expectedStatus.
        _, err := repo.UpdateStatus(context.Background(), order.ID,
                domain.OrderPaymentPending, domain.OrderPaid, "test")
        require.NoError(t, err) // first transition succeeds

        // Now call again with the same expectedStatus — should get CONFLICT
        // because the order is already PAID.
        _, err = repo.UpdateStatus(context.Background(), order.ID,
                domain.OrderPaymentPending, domain.OrderPaid, "test")
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "CONFLICT", appErr.Code)
}

// Test that cancellation events are published from all valid cancel states.
func TestCancellation_EventsFromAllValidStates(t *testing.T) {
        cancelFrom := []domain.OrderStatus{
                domain.OrderPaymentPending,
                domain.OrderPaid,
                domain.OrderProcessing,
        }

        for _, from := range cancelFrom {
                t.Run("cancel from "+string(from), func(t *testing.T) {
                        svc, _, pub := newTestService(t)
                        order := walkToStatus(t, svc, "", from)

                        _, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                                Status: "CANCELLED", Reason: "customer requested",
                        })
                        require.NoError(t, err)
                        assert.Equal(t, 1, pub.Count("order.cancelled"))
                })
        }
}

func TestCancellation_FromDelivered_Fails(t *testing.T) {
        svc, _, _ := newTestService(t)
        order := walkToStatus(t, svc, "", domain.OrderDelivered)

        _, err := svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{
                Status: "CANCELLED", Reason: "too late",
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "BAD_REQUEST", appErr.Code)
}
