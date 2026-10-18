package transport

import (
	"time"

	"github.com/ecommerce/backend/internal/orders/domain"
)

// --- Request DTOs ---

// CreateOrderRequest is the JSON body for POST /api/v1/orders.
type CreateOrderRequest struct {
	Items []CreateOrderItemRequest `json:"items" validate:"required"`
	ShippingAddress *AddressRequest   `json:"shipping_address" validate:"required"`
	BillingAddress  *AddressRequest   `json:"billing_address" validate:"required"`
	// PaymentMethod is "ONLINE" (default) or "COD".
	PaymentMethod string `json:"payment_method"`
	// ShippingMethod is "standard" (default) or "express"; the fee is set by the server.
	ShippingMethod string `json:"shipping_method"`
	// DeliveryArea is "inside_dhaka" (default) or "outside_dhaka"; courier prices differ.
	DeliveryArea string `json:"delivery_area"`
	// Manual wallet payment (ONLINE): the customer sent money to our bKash/Nagad/Rocket
	// number and gives the provider, transaction ID and the number they paid from.
	PaymentProvider  string `json:"payment_provider"`
	PaymentReference string `json:"payment_reference"`
	PayerNumber      string `json:"payer_number"`
	// CouponCode is optional; the server validates it and applies the discount.
	CouponCode string `json:"coupon_code"`
}

// CreateOrderItemRequest is an item within a create order request.
type CreateOrderItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id" validate:"required"`
	// Name, SKU, UnitPrice and Discount are IGNORED when server-side pricing is
	// enabled (see Handler.SetCatalog); send only product_id, variant_id, quantity.
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity" validate:"required"`
	UnitPrice int    `json:"unit_price"`
	Discount  int    `json:"discount"`
}

// AddressRequest is an address in a request body.
type AddressRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Line1      string `json:"line1" validate:"required"`
	Line2      string `json:"line2"`
	City       string `json:"city" validate:"required"`
	State      string `json:"state" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required"`
	Country    string `json:"country" validate:"required"`
}

// UpdateStatusRequest is the JSON body for PATCH /api/v1/admin/orders/:id/status.
// ShipOrderRequest is the body for POST /api/v1/admin/orders/:id/ship.
type ShipOrderRequest struct {
	Carrier        string `json:"carrier"`
	TrackingNumber string `json:"tracking_number"`
	CourierCost    int    `json:"courier_cost"` // optional, whole BDT
}

type UpdateStatusRequest struct {
	Status           string `json:"status" validate:"required"`
	Reason           string `json:"reason"`
	PaymentReference  string `json:"payment_reference"`
	ShipmentReference string `json:"shipment_reference"`
}

// --- Response DTOs ---

// OrderResponse is the order data returned in API responses.
type OrderResponse struct {
	ID               string               `json:"id"`
	OrderNumber      string               `json:"order_number"`
	CustomerID       string               `json:"customer_id"`
	Items            []OrderItemResponse  `json:"items"`
	VendorTotals     []domain.VendorTotal `json:"vendor_totals"`
	Pricing          OrderPricingResponse `json:"pricing"`
	ShippingAddress  *AddressResponse     `json:"shipping_address"`
	BillingAddress   *AddressResponse     `json:"billing_address"`
	PaymentReference string               `json:"payment_reference,omitempty"`
	ShipmentReference string              `json:"shipment_reference,omitempty"`
	PaymentMethod     string              `json:"payment_method"`
	PaymentProvider   string              `json:"payment_provider,omitempty"`
	PayerNumber       string              `json:"payer_number,omitempty"`
	ShippingMethod    string              `json:"shipping_method,omitempty"`
	DeliveryArea      string              `json:"delivery_area,omitempty"`
	CouponCode        string              `json:"coupon_code,omitempty"`
	PaymentStatus     string              `json:"payment_status"`
	Carrier           string              `json:"carrier,omitempty"`
	TrackingNumber    string              `json:"tracking_number,omitempty"`
	ShippedAt         *time.Time          `json:"shipped_at,omitempty"`
	Status           string               `json:"status"`
	StatusHistory    []StatusHistoryResponse `json:"status_history"`
	CreatedAt        string               `json:"created_at"`
	UpdatedAt        string               `json:"updated_at"`
}

// OrderItemResponse is an item snapshot in an order response.
type OrderItemResponse struct {
	ProductID string             `json:"product_id"`
	VariantID string             `json:"variant_id"`
	Name      string             `json:"name"`
	SKU       string             `json:"sku"`
	ImageURL  string             `json:"image_url,omitempty"`
	Options   map[string]string  `json:"options,omitempty"`
	Quantity  int                `json:"quantity"`
	Pricing   ItemPricingResponse `json:"pricing"`
}

// ItemPricingResponse is the per-item pricing in a response.
type ItemPricingResponse struct {
	UnitPrice int `json:"unit_price"`
	Discount  int `json:"discount"`
	Total     int `json:"total"`
}

// OrderPricingResponse is the order-level pricing in a response.
type OrderPricingResponse struct {
	Subtotal int `json:"subtotal"`
	Discount int `json:"discount"`
	Tax      int `json:"tax"`
	Shipping int `json:"shipping"`
	Total    int `json:"total"`
}

// AddressResponse is an address in a response.
type AddressResponse struct {
	Name       string `json:"name,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// StatusHistoryResponse is a status change entry in a response.
type StatusHistoryResponse struct {
	Status    string `json:"status"`
	ChangedAt string `json:"changed_at"`
	Reason    string `json:"reason"`
}

// --- Mappers ---

func toOrderResponse(o *domain.Order) OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Name:      item.Name,
			SKU:       item.SKU,
			ImageURL:  item.ImageURL,
			Options:   item.Options,
			Quantity:  item.Quantity,
			Pricing: ItemPricingResponse{
				UnitPrice: item.Pricing.UnitPrice,
				Discount:  item.Pricing.Discount,
				Total:     item.Pricing.Total,
			},
		}
	}

	history := make([]StatusHistoryResponse, len(o.StatusHistory))
	for i, h := range o.StatusHistory {
		history[i] = StatusHistoryResponse{
			Status:    string(h.Status),
			ChangedAt: h.ChangedAt.UTC().Format(time.RFC3339),
			Reason:    h.Reason,
		}
	}

	return OrderResponse{
		ID:               o.ID,
		OrderNumber:      o.OrderNumber,
		CustomerID:       o.CustomerID,
		Items:            items,
		VendorTotals:     o.VendorTotals,
		Pricing: OrderPricingResponse{
			Subtotal: o.Pricing.Subtotal,
			Discount: o.Pricing.Discount,
			Tax:      o.Pricing.Tax,
			Shipping: o.Pricing.Shipping,
			Total:    o.Pricing.Total,
		},
		ShippingAddress:  toAddressResponse(o.ShippingAddress),
		BillingAddress:   toAddressResponse(o.BillingAddress),
		PaymentReference: o.PaymentReference,
		ShipmentReference: o.ShipmentReference,
		PaymentMethod:     string(o.PaymentMethod),
		PaymentProvider:   o.PaymentProvider,
		PayerNumber:       o.PayerNumber,
		ShippingMethod:    o.ShippingMethod,
		DeliveryArea:      o.DeliveryArea,
		CouponCode:        o.CouponCode,
		PaymentStatus:     string(o.PaymentStatus),
		Carrier:           o.Carrier,
		TrackingNumber:    o.TrackingNumber,
		ShippedAt:         o.ShippedAt,
		Status:           string(o.Status),
		StatusHistory:    history,
		CreatedAt:        o.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        o.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toAddressResponse(a *domain.Address) *AddressResponse {
	if a == nil {
		return nil
	}
	return &AddressResponse{
		Name:       a.Name,
		Phone:      a.Phone,
		Line1:      a.Line1,
		Line2:      a.Line2,
		City:       a.City,
		State:      a.State,
		PostalCode: a.PostalCode,
		Country:    a.Country,
	}
}

func toDomainAddress(a *AddressRequest) *domain.Address {
	if a == nil {
		return nil
	}
	return &domain.Address{
		Name:       a.Name,
		Phone:      a.Phone,
		Line1:      a.Line1,
		Line2:      a.Line2,
		City:       a.City,
		State:      a.State,
		PostalCode: a.PostalCode,
		Country:    a.Country,
	}
}

// AdminOrderResponse is the order as the shop sees it: it also shows what the courier charged
// us (never shown to customers).
type AdminOrderResponse struct {
	OrderResponse
	CourierCost int `json:"courier_cost"`
}

func toAdminOrderResponse(o *domain.Order) AdminOrderResponse {
	return AdminOrderResponse{OrderResponse: toOrderResponse(o), CourierCost: o.CourierCost}
}
