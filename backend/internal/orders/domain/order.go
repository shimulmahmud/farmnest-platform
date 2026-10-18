package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	OrderPending      OrderStatus = "PENDING"
	OrderPaymentPending OrderStatus = "PAYMENT_PENDING"
	OrderPaid         OrderStatus = "PAID"
	OrderConfirmed    OrderStatus = "CONFIRMED" // cash-on-delivery order accepted, payment still due
	OrderProcessing   OrderStatus = "PROCESSING"
	OrderShipped      OrderStatus = "SHIPPED"
	OrderDelivered    OrderStatus = "DELIVERED"
	OrderCancelled    OrderStatus = "CANCELLED"
	OrderRefundPending OrderStatus = "REFUND_PENDING"
	OrderRefunded     OrderStatus = "REFUNDED"
)

// validTransitions defines the only allowed status transitions.
// A missing key or missing target means the transition is invalid.
var validTransitions = map[OrderStatus][]OrderStatus{
	OrderPending:       {OrderPaymentPending, OrderCancelled},
	OrderPaymentPending: {OrderPaid, OrderConfirmed, OrderCancelled},
	OrderConfirmed:     {OrderProcessing, OrderCancelled},
	OrderPaid:          {OrderProcessing, OrderCancelled, OrderRefundPending},
	OrderProcessing:    {OrderShipped, OrderCancelled},
	OrderShipped:       {OrderDelivered},
	OrderDelivered:     {OrderRefundPending},
	OrderRefundPending: {OrderRefunded, OrderPaid},
	OrderRefunded:      {},
	OrderCancelled:     {},
}

// transitionEvents maps (from, to) status pairs to NATS event subjects.
var transitionEvents = map[[2]OrderStatus]string{
	{OrderPending, OrderPaymentPending}: "order.created",
	{OrderPaymentPending, OrderPaid}:     "order.paid",
	{OrderProcessing, OrderShipped}:       "order.shipped",
	{OrderShipped, OrderDelivered}:        "order.delivered",
	{OrderPaid, OrderCancelled}:           "order.cancelled",
	{OrderPaymentPending, OrderCancelled}: "order.cancelled",
	{OrderPending, OrderCancelled}:        "order.cancelled",
	{OrderProcessing, OrderCancelled}:     "order.cancelled",
}

// PaymentMethod is how the customer pays.
type PaymentMethod string

const (
	PaymentOnline PaymentMethod = "ONLINE"
	PaymentCOD    PaymentMethod = "COD"
)

// IsValid reports whether m is a known payment method.
func (m PaymentMethod) IsValid() bool { return m == PaymentOnline || m == PaymentCOD }

// PaymentStatus tells whether the money has been received.
type PaymentStatus string

const (
	PaymentUnpaid PaymentStatus = "UNPAID"
	PaymentPaid   PaymentStatus = "PAID"
)

// IsTerminal returns true if the order is in a final state.
func (s OrderStatus) IsTerminal() bool {
	return s == OrderDelivered || s == OrderCancelled || s == OrderRefunded
}

// CanTransitionTo checks whether transitioning to the target status is allowed.
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	if targets, ok := validTransitions[s]; ok {
		for _, t := range targets {
			if t == target {
				return true
			}
		}
	}
	return false
}

// TransitionEvent returns the NATS event subject for this transition, or empty if none.
func TransitionEvent(from, to OrderStatus) string {
	return transitionEvents[[2]OrderStatus{from, to}]
}

// Address represents a shipping or billing address.
type Address struct {
	Name       string `bson:"name,omitempty" json:"name,omitempty"`   // recipient
	Phone      string `bson:"phone,omitempty" json:"phone,omitempty"` // recipient mobile
	Line1      string `bson:"line1" json:"line1"`
	Line2      string `bson:"line2" json:"line2,omitempty"`
	City       string `bson:"city" json:"city"`
	State      string `bson:"state" json:"state"`
	PostalCode string `bson:"postal_code" json:"postal_code"`
	Country    string `bson:"country" json:"country"`
}

// OrderItemPricing holds the per-item pricing snapshot.
type OrderItemPricing struct {
	UnitPrice int `bson:"unit_price" json:"unit_price"` // whole BDT
	Discount  int `bson:"discount" json:"discount"`     // whole BDT
	Total     int `bson:"total" json:"total"`           // (unit_price - discount) * quantity
}

// OrderItem is an immutable historical snapshot of a purchased item.
type OrderItem struct {
	VendorID  string           `bson:"vendor_id" json:"vendor_id,omitempty"`
	CommissionBps int          `bson:"commission_bps" json:"commission_bps"` // frozen at purchase
	ProductID string           `bson:"product_id" json:"product_id"`
	VariantID string           `bson:"variant_id" json:"variant_id"`
	Name      string           `bson:"name" json:"name"`
	SKU       string           `bson:"sku" json:"sku"`
	Barcode   string           `bson:"barcode,omitempty" json:"barcode,omitempty"`
	ImageURL  string           `bson:"image_url,omitempty" json:"image_url,omitempty"`
	Options   map[string]string `bson:"options,omitempty" json:"options,omitempty"`
	Currency  string           `bson:"currency,omitempty" json:"currency,omitempty"`
	WeightG   int              `bson:"weight_g" json:"weight_g"`
	RequiresShipping bool      `bson:"requires_shipping" json:"requires_shipping"`
	Quantity  int              `bson:"quantity" json:"quantity"`
	// UnitCost is what this unit cost us when the order was placed (whole BDT).
	// Used only for profit reports; never exposed to customers.
	UnitCost  int              `bson:"unit_cost" json:"-"`
	Pricing   OrderItemPricing `bson:"pricing" json:"pricing"`
}

// ItemTotal computes and returns the total for this item.
func (i *OrderItem) ItemTotal() int {
	return i.Pricing.Total
}

// OrderPricing holds the order-level pricing breakdown.
type OrderPricing struct {
	Subtotal int `bson:"subtotal" json:"subtotal"` // sum of item totals
	Discount int `bson:"discount" json:"discount"` // total discount whole BDT
	Tax      int `bson:"tax" json:"tax"`           // tax whole BDT
	Shipping int `bson:"shipping" json:"shipping"`  // shipping cost whole BDT
	Total    int `bson:"total" json:"total"`         // final amount, whole BDT
}

// VendorTotal is the frozen per-vendor money breakdown of an order (whole BDT).
type VendorTotal struct {
	VendorID   string `bson:"vendor_id" json:"vendor_id"`
	Subtotal   int    `bson:"subtotal" json:"subtotal"`
	Commission int    `bson:"commission" json:"commission"`
	Payout     int    `bson:"payout" json:"payout"` // subtotal - commission
}

// ComputeVendorTotals groups vendor-owned items and applies each item's frozen
// commission rate. Platform-owned items (empty VendorID) are skipped.
func ComputeVendorTotals(items []OrderItem) []VendorTotal {
	idx := map[string]int{}
	var out []VendorTotal
	for _, it := range items {
		if it.VendorID == "" {
			continue
		}
		i, ok := idx[it.VendorID]
		if !ok {
			i = len(out)
			idx[it.VendorID] = i
			out = append(out, VendorTotal{VendorID: it.VendorID})
		}
		commission := (it.Pricing.Total*it.CommissionBps + 5000) / 10000
		out[i].Subtotal += it.Pricing.Total
		out[i].Commission += commission
		out[i].Payout += it.Pricing.Total - commission
	}
	return out
}

// StatusHistoryEntry records a single status change.
type StatusHistoryEntry struct {
	Status    OrderStatus `bson:"status" json:"status"`
	ChangedAt time.Time   `bson:"changed_at" json:"changed_at"`
	Reason    string      `bson:"reason" json:"reason"`
}

// Order represents a customer order with historical item snapshots.
type Order struct {
	ID               string              `bson:"_id" json:"id"`
	OrderNumber      string              `bson:"order_number" json:"order_number"`
	CustomerID       string              `bson:"customer_id" json:"customer_id"`
	Items            []OrderItem         `bson:"items" json:"items"`
	VendorTotals     []VendorTotal       `bson:"vendor_totals" json:"vendor_totals"`
	StockAllocations []StockAllocation   `bson:"stock_allocations,omitempty" json:"-"`
	StockRestored    bool                `bson:"stock_restored" json:"-"`
	CouponCode       string              `bson:"coupon_code,omitempty" json:"coupon_code,omitempty"`
	PromotionID      string              `bson:"promotion_id,omitempty" json:"promotion_id,omitempty"`
	PaymentMethod    PaymentMethod       `bson:"payment_method" json:"payment_method"`
	PaymentProvider  string              `bson:"payment_provider,omitempty" json:"payment_provider,omitempty"` // bkash, nagad, rocket (manual wallet payment)
	PayerNumber      string              `bson:"payer_number,omitempty" json:"payer_number,omitempty"`
	ShippingMethod   string              `bson:"shipping_method,omitempty" json:"shipping_method,omitempty"`
	DeliveryArea     string              `bson:"delivery_area,omitempty" json:"delivery_area,omitempty"`
	PaymentStatus    PaymentStatus       `bson:"payment_status" json:"payment_status"`
	Pricing          OrderPricing        `bson:"pricing" json:"pricing"`
	ShippingAddress  *Address            `bson:"shipping_address" json:"shipping_address"`
	BillingAddress   *Address            `bson:"billing_address" json:"billing_address"`
	PaymentReference string              `bson:"payment_reference" json:"payment_reference,omitempty"`
	ShipmentReference string             `bson:"shipment_reference" json:"shipment_reference,omitempty"`
	Carrier          string              `bson:"carrier,omitempty" json:"carrier,omitempty"`
	CourierCost      int                 `bson:"courier_cost" json:"-"` // what the courier charged us (whole BDT)
	TrackingNumber   string              `bson:"tracking_number,omitempty" json:"tracking_number,omitempty"`
	ShippedAt        *time.Time          `bson:"shipped_at,omitempty" json:"shipped_at,omitempty"`
	Status           OrderStatus         `bson:"status" json:"status"`
	StatusHistory    []StatusHistoryEntry `bson:"status_history" json:"status_history"`
	CreatedAt        time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time           `bson:"updated_at" json:"updated_at"`
}

// CanTransitionTo delegates to the OrderStatus method.
func (o *Order) CanTransitionTo(target OrderStatus) bool {
	return o.Status.CanTransitionTo(target)
}

// TransitionTo validates and applies a status transition, appending to history.
// Returns an error if the transition is invalid.
func (o *Order) TransitionTo(target OrderStatus, reason string) error {
	if !o.CanTransitionTo(target) {
		return fmt.Errorf("invalid transition: %s -> %s", o.Status, target)
	}
	now := time.Now().UTC()
	o.Status = target
	o.StatusHistory = append(o.StatusHistory, StatusHistoryEntry{
		Status:    target,
		ChangedAt: now,
		Reason:    reason,
	})
	o.UpdatedAt = now
	return nil
}

// NewOrder creates a new order in PENDING status.
func NewOrder(customerID string, items []OrderItem, pricing OrderPricing, shipping, billing *Address) *Order {
	now := time.Now().UTC()
	return &Order{
		ID:              uuid.New().String(),
		OrderNumber:     generateOrderNumber(now),
		CustomerID:      customerID,
		Items:           items,
		Pricing:         pricing,
		ShippingAddress: shipping,
		BillingAddress:  billing,
		Status:          OrderPending,
		PaymentMethod:   PaymentOnline,
		PaymentStatus:   PaymentUnpaid,
		StatusHistory: []StatusHistoryEntry{
			{
				Status:    OrderPending,
				ChangedAt: now,
				Reason:    "order created",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// generateOrderNumber creates a human-readable order number: ORD-YYYYMMDD-XXXXXXXX.
func generateOrderNumber(t time.Time) string {
	date := t.UTC().Format("20060102")
	suffix := uuid.New().String()[:8]
	return fmt.Sprintf("ORD-%s-%s", date, suffix)
}

// OrderEvent is published to NATS on order state changes.
type OrderEvent struct {
	EventType       string `json:"event_type"`
	OrderID         string `json:"order_id"`
	OrderNumber     string `json:"order_number"`
	CustomerID      string `json:"customer_id"`
	FromStatus      string `json:"from_status"`
	ToStatus        string `json:"to_status"`
	PaymentReference string `json:"payment_reference,omitempty"`
	ShipmentReference string `json:"shipment_reference,omitempty"`
}

// OrderFilter holds optional filters for listing orders.
type OrderFilter struct {
	CustomerID string
	Status     string
	From       *time.Time
	To         *time.Time
}
