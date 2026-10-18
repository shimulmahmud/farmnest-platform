package application

import (
        "context"

        "github.com/ecommerce/backend/internal/orders/domain"
        "github.com/ecommerce/backend/pkg/audit"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/ecommerce/backend/pkg/telemetry"
)

// Service implements order use cases.
type Service struct {
        orders  domain.OrderRepository
        events  domain.EventPublisher
        auditor audit.Auditor
        bizMetrics *telemetry.BusinessMetrics
        stock      domain.StockReserver
        coupons    domain.CouponApplier
}

// NewService creates an order application service.
func NewService(orders domain.OrderRepository, events domain.EventPublisher) *Service {
        return &Service{
                orders:  orders,
                events:  events,
                auditor: audit.NopAuditor,
        }
}

// SetAuditor injects an async auditor for cross-cutting audit logging.
func (s *Service) SetAuditor(a audit.Auditor) {
        s.auditor = a
}

// SetCoupons enables coupon codes: usage is counted when an order is placed and
// given back when it fails or is cancelled. Optional; nil disables coupons.
func (s *Service) SetCoupons(c domain.CouponApplier) {
	s.coupons = c
}

// SetStock links orders to inventory: stock is deducted when an order is placed
// and put back if it is cancelled. Optional; nil disables the link.
func (s *Service) SetStock(r domain.StockReserver) {
        s.stock = r
}

// SetBusinessMetrics injects business metrics counters for order events.
func (s *Service) SetBusinessMetrics(m *telemetry.BusinessMetrics) {
        s.bizMetrics = m
}

// --- Input structs ---

// CreateOrderInput holds the data needed to create a new order.
type CreateOrderInput struct {
        CustomerID      string                `json:"customer_id" validate:"required"`
        Items           []domain.OrderItem     `json:"items" validate:"required"`
        Pricing         domain.OrderPricing   `json:"pricing" validate:"required"`
        ShippingAddress *domain.Address       `json:"shipping_address"`
        BillingAddress  *domain.Address       `json:"billing_address"`
        // PaymentMethod is ONLINE (default) or COD. COD orders are confirmed and
        // take stock immediately; ONLINE orders take stock when payment completes.
        PaymentMethod   domain.PaymentMethod  `json:"payment_method"`
        // CouponCode / PromotionID: set by the handler after the coupon was validated (Pricing.Discount is already applied).
        CouponCode  string `json:"coupon_code"`
        PromotionID string `json:"promotion_id"`
        // Manual wallet payment details (ONLINE): provider, TrxID and sender number.
        PaymentProvider  string `json:"payment_provider"`
        PaymentReference string `json:"payment_reference"`
        PayerNumber      string `json:"payer_number"`
        ShippingMethod   string `json:"shipping_method"`
        DeliveryArea     string `json:"delivery_area"`
}

// UpdateStatusInput holds the data for an admin status transition.
type UpdateStatusInput struct {
        Status            string `json:"status" validate:"required"`
        Reason            string `json:"reason"`
        PaymentReference  string `json:"payment_reference"`
        ShipmentReference string `json:"shipment_reference"`
}

// --- Use cases ---

// CreateOrder creates a new order and immediately transitions to PAYMENT_PENDING.
func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
        if input.CustomerID == "" {
                return nil, errors.New("VALIDATION_ERROR", "customer_id is required", 422)
        }
        if len(input.Items) == 0 {
                return nil, errors.New("VALIDATION_ERROR", "order must have at least one item", 422)
        }

        method := input.PaymentMethod
        if method == "" {
                method = domain.PaymentOnline
        }
        if !method.IsValid() {
                return nil, errors.New("VALIDATION_ERROR", "payment_method must be ONLINE or COD", 422)
        }

        order := domain.NewOrder(
                input.CustomerID,
                input.Items,
                input.Pricing,
                input.ShippingAddress,
                input.BillingAddress,
        )
        order.PaymentMethod = method
        order.CouponCode = input.CouponCode
        order.PromotionID = input.PromotionID
        order.PaymentProvider = input.PaymentProvider
        order.PaymentReference = input.PaymentReference
        order.PayerNumber = input.PayerNumber
        order.ShippingMethod = input.ShippingMethod
        order.DeliveryArea = input.DeliveryArea
        order.VendorTotals = domain.ComputeVendorTotals(order.Items)

        fromStatus := order.Status
        if err := order.TransitionTo(domain.OrderPaymentPending, "order created"); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to set initial status", 500, err)
        }

        // Coupon: count the use now (atomic; fails if the limit was just reached).
        // It is given back below if anything else fails.
        couponUsed := false
        if s.coupons != nil && order.PromotionID != "" {
                if err := s.coupons.Redeem(ctx, order.PromotionID); err != nil {
                        return nil, err
                }
                couponUsed = true
        }
        releaseCoupon := func() {
                if couponUsed {
                        _ = s.coupons.Release(ctx, order.PromotionID)
                }
        }

        // Stock rules:
        //  - COD: the order is final once placed, so stock is taken now (all-or-nothing).
        //  - ONLINE: nothing is taken until payment completes; here we only check
        //    that the goods are available so customers fail fast.
        // Checkout itself (cart, the 1-hour frontend session) never touches stock.
        if s.stock != nil {
                lines := stockLines(order.Items)
                if method == domain.PaymentCOD {
                        allocs, err := s.stock.DeductForOrder(ctx, order.ID, lines)
                        if err != nil {
                                releaseCoupon()
                                return nil, err
                        }
                        order.StockAllocations = allocs
                } else if err := s.stock.CheckAvailable(ctx, lines); err != nil {
                        releaseCoupon()
                        return nil, err
                }
        }
        if method == domain.PaymentCOD {
                if err := order.TransitionTo(domain.OrderConfirmed, "cash on delivery order confirmed"); err != nil {
                        return nil, errors.Wrap("INTERNAL_ERROR", "failed to confirm COD order", 500, err)
                }
        }

        if err := s.orders.Create(ctx, order); err != nil {
                releaseCoupon()
                if s.stock != nil && len(order.StockAllocations) > 0 {
                        _ = s.stock.RestoreForOrder(ctx, order.ID, order.StockAllocations)
                }
                return nil, err
        }

        // Business metric: order created
        if s.bizMetrics != nil {
                s.bizMetrics.IncrementOrdersCreated()
        }

        s.publishEvent(ctx, "order.created", &domain.OrderEvent{
                EventType:   "order.created",
                OrderID:     order.ID,
                OrderNumber: order.OrderNumber,
                CustomerID:  order.CustomerID,
                FromStatus:  string(fromStatus),
                ToStatus:    string(domain.OrderPaymentPending),
        })

        return order, nil
}

// GetOrderByID retrieves an order by ID.
// GetOrderByNumber returns an order by its human-readable number.
func (s *Service) GetOrderByNumber(ctx context.Context, number string) (*domain.Order, error) {
	return s.orders.FindByOrderNumber(ctx, number)
}

func (s *Service) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
        return s.orders.FindByID(ctx, id)
}

// ListOrders returns paginated orders for a specific customer.
func (s *Service) ListOrders(ctx context.Context, customerID, status string, p pagination.Params) ([]*domain.Order, int64, error) {
        if customerID == "" {
                return nil, 0, errors.New("VALIDATION_ERROR", "customer_id is required", 422)
        }
        filter := &domain.OrderFilter{
                CustomerID: customerID,
                Status:     status,
        }
        return s.orders.List(ctx, filter, p)
}

// ListAllOrders returns paginated orders for admin.
func (s *Service) ListAllOrders(ctx context.Context, status, customerID string, p pagination.Params) ([]*domain.Order, int64, error) {
        filter := &domain.OrderFilter{
                CustomerID: customerID,
                Status:     status,
        }
        return s.orders.List(ctx, filter, p)
}

// UpdateOrderStatus performs a status transition with validation and audit.
func (s *Service) UpdateOrderStatus(ctx context.Context, orderID string, input UpdateStatusInput) (*domain.Order, error) {
        newStatus := domain.OrderStatus(input.Status)

        current, err := s.orders.FindByID(ctx, orderID)
        if err != nil {
                return nil, err
        }

        if !current.CanTransitionTo(newStatus) {
                return nil, errors.New("BAD_REQUEST",
                        "invalid status transition: "+string(current.Status)+" -> "+string(newStatus), 400)
        }

        // Online order being paid: take the stock now, BEFORE recording the payment,
        // so a transient failure leaves the order PAYMENT_PENDING and is retried.
        var paidAllocs []domain.StockAllocation
        if newStatus == domain.OrderPaid && current.Status == domain.OrderPaymentPending &&
                s.stock != nil && current.PaymentMethod != domain.PaymentCOD && len(current.StockAllocations) == 0 {
                allocs, derr := s.stock.DeductForOrder(ctx, orderID, stockLines(current.Items))
                if derr != nil {
                        if appErr, ok := errors.IsAppError(derr); ok && appErr.Code == "INSUFFICIENT_STOCK" {
                                return s.paidButOutOfStock(ctx, current, input)
                        }
                        return nil, derr
                }
                paidAllocs = allocs
                if err := s.orders.SetStockAllocations(ctx, orderID, allocs); err != nil {
                        _ = s.stock.RestoreForOrder(ctx, orderID, allocs)
                        return nil, err
                }
        }

        updated, err := s.orders.UpdateStatus(ctx, orderID, current.Status, newStatus, input.Reason)
        if err != nil {
                if len(paidAllocs) > 0 {
                        _ = s.stock.RestoreForOrder(ctx, orderID, paidAllocs)
                        _ = s.orders.SetStockAllocations(ctx, orderID, nil)
                }
                return nil, err
        }
        if newStatus == domain.OrderPaid {
                if err := s.orders.SetPaymentStatus(ctx, orderID, domain.PaymentPaid); err != nil {
                        return nil, err
                }
                updated.PaymentStatus = domain.PaymentPaid
        }

        // Persist references (previously they were only set on the returned copy).
        if err := s.orders.SetReferences(ctx, orderID, input.PaymentReference, input.ShipmentReference); err != nil {
                return nil, err
        }
        if input.PaymentReference != "" {
                updated.PaymentReference = input.PaymentReference
        }
        if input.ShipmentReference != "" {
                updated.ShipmentReference = input.ShipmentReference
        }

        // A cancelled order gives its coupon use back.
        if newStatus == domain.OrderCancelled && s.coupons != nil && current.PromotionID != "" {
                _ = s.coupons.Release(ctx, current.PromotionID)
        }

        // A cancelled order puts its stock back (once).
        if newStatus == domain.OrderCancelled && s.stock != nil && len(current.StockAllocations) > 0 && !current.StockRestored {
                if err := s.stock.RestoreForOrder(ctx, orderID, current.StockAllocations); err != nil {
                        return nil, errors.Wrap("INTERNAL_ERROR",
                                "order cancelled but stock could not be restored; adjust stock manually", 500, err)
                }
                if err := s.orders.MarkStockRestored(ctx, orderID); err != nil {
                        return nil, err
                }
        }

        eventSubject := domain.TransitionEvent(current.Status, newStatus)
        if eventSubject != "" {
                s.publishEvent(ctx, eventSubject, &domain.OrderEvent{
                        EventType:        eventSubject,
                        OrderID:          updated.ID,
                        OrderNumber:      updated.OrderNumber,
                        CustomerID:       updated.CustomerID,
                        FromStatus:       string(current.Status),
                        ToStatus:         string(newStatus),
                        PaymentReference:  input.PaymentReference,
                        ShipmentReference: input.ShipmentReference,
                })
        }

        // Audit: track status changes, especially cancel and refund
        oldStatus := string(current.Status)
        newStatusStr := string(newStatus)
        action := audit.ActionOrderStatusChange
        details := input.Reason
        if newStatus == domain.OrderCancelled {
                action = audit.ActionOrderCancel
                if details == "" {
                        details = "order cancelled"
                }
        } else if newStatus == domain.OrderRefunded {
                action = audit.ActionOrderRefund
                if details == "" {
                        details = "order refunded"
                }
        }
        s.auditor.Record("", action, "order", orderID, oldStatus, newStatusStr, details)

        return updated, nil
}

// ShipInput holds the data for marking an order as shipped.
type ShipInput struct {
        Carrier        string
        TrackingNumber string
        CourierCost    int // optional: what the courier charged us, whole BDT
}

// ShipOrder is the one-step fulfillment action: PAID/PROCESSING -> SHIPPED.
// It moves the order to PROCESSING if needed, moves to SHIPPED and records the
// courier and tracking number. Stock is not touched (deducted at order time).
// It is safe to retry: an order left in PROCESSING can be shipped again.
func (s *Service) ShipOrder(ctx context.Context, orderID string, input ShipInput) (*domain.Order, error) {
        if input.CourierCost < 0 {
                return nil, errors.New("VALIDATION_ERROR", "courier_cost must not be negative", 422)
        }
        if input.Carrier == "" || input.TrackingNumber == "" {
                return nil, errors.New("VALIDATION_ERROR", "carrier and tracking_number are required", 422)
        }

        order, err := s.orders.FindByID(ctx, orderID)
        if err != nil {
                return nil, err
        }
        if order.Status != domain.OrderPaid && order.Status != domain.OrderConfirmed && order.Status != domain.OrderProcessing {
                return nil, errors.New("BAD_REQUEST",
                        "only PAID, CONFIRMED (COD) or PROCESSING orders can be shipped (current: "+string(order.Status)+")", 400)
        }

        if order.Status == domain.OrderPaid || order.Status == domain.OrderConfirmed {
                if _, err := s.UpdateOrderStatus(ctx, orderID, UpdateStatusInput{Status: string(domain.OrderProcessing), Reason: "packing"}); err != nil {
                        return nil, err
                }
        }

        // Stock was already deducted when the order was placed: shipping does not touch it.

        if _, err := s.UpdateOrderStatus(ctx, orderID, UpdateStatusInput{
                Status: string(domain.OrderShipped), Reason: "shipped via " + input.Carrier, ShipmentReference: input.TrackingNumber,
        }); err != nil {
                return nil, err
        }

        return s.orders.SetShipment(ctx, orderID, input.Carrier, input.TrackingNumber, input.CourierCost)
}

// paidButOutOfStock handles the rare case where a customer paid online but the
// goods ran out meanwhile: the payment is recorded and the order moves straight
// to REFUND_PENDING so it can be refunded. No error is returned, so the payment
// worker does not retry.
func (s *Service) paidButOutOfStock(ctx context.Context, current *domain.Order, input UpdateStatusInput) (*domain.Order, error) {
        if _, err := s.orders.UpdateStatus(ctx, current.ID, current.Status, domain.OrderPaid, input.Reason); err != nil {
                return nil, err
        }
        if err := s.orders.SetPaymentStatus(ctx, current.ID, domain.PaymentPaid); err != nil {
                return nil, err
        }
        if err := s.orders.SetReferences(ctx, current.ID, input.PaymentReference, ""); err != nil {
                return nil, err
        }
        refunding, err := s.orders.UpdateStatus(ctx, current.ID, domain.OrderPaid, domain.OrderRefundPending,
                "out of stock after payment: refund required")
        if err != nil {
                return nil, err
        }
        refunding.PaymentStatus = domain.PaymentPaid
        s.auditor.Record("", audit.ActionOrderStatusChange, "order", current.ID,
                string(domain.OrderPaid), string(domain.OrderRefundPending), "out of stock after payment")
        return refunding, nil
}

// MarkPaid records that a cash-on-delivery order's money was collected.
func (s *Service) MarkPaid(ctx context.Context, orderID string) (*domain.Order, error) {
        order, err := s.orders.FindByID(ctx, orderID)
        if err != nil {
                return nil, err
        }
        if order.PaymentMethod != domain.PaymentCOD {
                return nil, errors.New("BAD_REQUEST", "only cash-on-delivery orders are marked paid by hand; online orders are marked paid by the payment", 400)
        }
        if order.Status == domain.OrderCancelled || order.Status == domain.OrderRefunded {
                return nil, errors.New("BAD_REQUEST", "a "+string(order.Status)+" order cannot be marked paid", 400)
        }
        if order.PaymentStatus == domain.PaymentPaid {
                return order, nil
        }
        if err := s.orders.SetPaymentStatus(ctx, orderID, domain.PaymentPaid); err != nil {
                return nil, err
        }
        order.PaymentStatus = domain.PaymentPaid
        s.auditor.Record("", audit.ActionOrderStatusChange, "order", orderID, "UNPAID", "PAID", "cash collected")
        return order, nil
}

func stockLines(items []domain.OrderItem) []domain.StockLine {
        lines := make([]domain.StockLine, len(items))
        for i, it := range items {
                lines[i] = domain.StockLine{SKU: it.SKU, Quantity: it.Quantity}
        }
        return lines
}

// --- Helpers ---

func (s *Service) publishEvent(ctx context.Context, subject string, event *domain.OrderEvent) {
        if s.events != nil {
                _ = s.events.Publish(ctx, subject, event)
        }
}
