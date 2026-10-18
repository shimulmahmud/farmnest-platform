package transport

import (
	"context"
	"strings"

	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	"github.com/ecommerce/backend/internal/orders/domain"
	settingsDomain "github.com/ecommerce/backend/internal/settings/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// WalletConfig is a mobile-wallet number customers pay to (manual verification).
type WalletConfig struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Number string `json:"number"`
	Type   string `json:"type"` // e.g. "Personal", "Merchant"
}

// CheckoutConfig holds the checkout settings that come from the environment (payment
// options). Delivery charges are NOT here: the shop edits them in the admin panel and
// they are read from the database (see DeliveryRates).
type CheckoutConfig struct {
	Currency   string         `json:"currency"`
	CODEnabled bool           `json:"cod_enabled"`
	Wallets    []WalletConfig `json:"wallets"`
}

// DefaultCheckoutConfig returns sensible defaults for a Bangladesh store.
func DefaultCheckoutConfig() CheckoutConfig {
	return CheckoutConfig{Currency: "BDT", CODEnabled: true, Wallets: []WalletConfig{}}
}

// DeliveryRates provides the shop's current delivery charges (set in the admin panel).
type DeliveryRates interface {
	Delivery(ctx context.Context) (settingsDomain.Delivery, error)
}

// SetDelivery connects the editable delivery charges. Without it, built-in defaults are used.
func (h *Handler) SetDelivery(d DeliveryRates) { h.delivery = d }

// deliveryNow returns the current delivery charges, or the defaults if they cannot be read.
func (h *Handler) deliveryNow(ctx context.Context) settingsDomain.Delivery {
	if h.delivery != nil {
		if d, err := h.delivery.Delivery(ctx); err == nil {
			return d
		}
	}
	return settingsDomain.DefaultDelivery()
}

// normalizeArea returns the delivery area in the form stored on the order.
func normalizeArea(area string) string {
	area = strings.ToLower(strings.TrimSpace(area))
	if area == "" {
		return settingsDomain.AreaInsideDhaka
	}
	return area
}

// DeliveryMethodView is a delivery option as customers see it at checkout.
type DeliveryMethodView struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	Detail          string `json:"detail"`
	FeeInsideDhaka  int    `json:"fee_inside_dhaka"`
	FeeOutsideDhaka int    `json:"fee_outside_dhaka"`
}

// CheckoutConfigResponse is what the storefront needs to build the checkout screen.
type CheckoutConfigResponse struct {
	CheckoutConfig
	ShippingMethods       []DeliveryMethodView  `json:"shipping_methods"`
	DeliveryAreas         []settingsDomain.Area `json:"delivery_areas"`
	FreeShippingThreshold int                   `json:"free_shipping_threshold"` // standard delivery is free from this item total; 0 = never
}

// SetCheckoutConfig overrides the default checkout configuration.
func (h *Handler) SetCheckoutConfig(cfg CheckoutConfig) {
	h.checkout = cfg
}

// CheckoutConfig handles GET /api/v1/checkout/config (public).
func (h *Handler) CheckoutConfig(c *fiber.Ctx) error {
	d := h.deliveryNow(c.Context())
	methods := make([]DeliveryMethodView, 0, len(d.Methods))
	for _, m := range d.EnabledMethods() {
		methods = append(methods, DeliveryMethodView{
			ID: m.ID, Label: m.Label, Detail: m.Detail,
			FeeInsideDhaka: m.FeeInsideDhaka, FeeOutsideDhaka: m.FeeOutsideDhaka,
		})
	}
	return response.OK(c, CheckoutConfigResponse{
		CheckoutConfig:        h.checkout,
		ShippingMethods:       methods,
		DeliveryAreas:         settingsDomain.Areas(),
		FreeShippingThreshold: d.FreeShippingThreshold,
	})
}

// walletByID returns the configured wallet with this id, if any.
func (c CheckoutConfig) walletByID(id string) (WalletConfig, bool) {
	for _, w := range c.Wallets {
		if w.ID == id {
			return w, true
		}
	}
	return WalletConfig{}, false
}

// validatePayment checks the payment fields of a create-order request.
func (h *Handler) validatePayment(method domain.PaymentMethod, req CreateOrderRequest) error {
	if !method.IsValid() {
		return errors.New("VALIDATION_ERROR", "payment_method must be ONLINE or COD", 422)
	}
	if method == domain.PaymentCOD {
		if !h.checkout.CODEnabled {
			return errors.New("VALIDATION_ERROR", "cash on delivery is not available", 422)
		}
		return nil
	}
	// ONLINE with no manual wallets configured: the payment arrives from a payment gateway
	// (webhook), so there is nothing to verify here.
	if len(h.checkout.Wallets) == 0 {
		return nil
	}
	// ONLINE with manual wallets = the customer already sent money to one of our wallets and gives the proof.
	provider := strings.ToLower(strings.TrimSpace(req.PaymentProvider))
	if _, ok := h.checkout.walletByID(provider); !ok {
		return errors.New("VALIDATION_ERROR", "payment_provider is not supported", 422)
	}
	if strings.TrimSpace(req.PaymentReference) == "" {
		return errors.New("VALIDATION_ERROR", "payment_reference (transaction ID) is required for online payment", 422)
	}
	if strings.TrimSpace(req.PayerNumber) == "" {
		return errors.New("VALIDATION_ERROR", "payer_number is required for online payment", 422)
	}
	return nil
}

// TrackResponse is the limited public view of an order for guest tracking.
type TrackResponse struct {
	OrderNumber    string                  `json:"order_number"`
	Status         string                  `json:"status"`
	PaymentMethod  string                  `json:"payment_method"`
	PaymentStatus  string                  `json:"payment_status"`
	CreatedAt      string                  `json:"created_at"`
	Pricing        OrderPricingResponse         `json:"pricing"`
	Items          []TrackItem             `json:"items"`
	Carrier        string                  `json:"carrier,omitempty"`
	TrackingNumber string                  `json:"tracking_number,omitempty"`
	City           string                  `json:"city,omitempty"`
	StatusHistory  []StatusHistoryResponse `json:"status_history"`
}

// TrackItem is an item line in the tracking view.
type TrackItem struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url,omitempty"`
	Quantity int    `json:"quantity"`
	Total    int    `json:"total"`
}

// digits keeps only the digits of s.
func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// samePhone compares phone numbers by their last 10 digits, so +8801712345678 and 01712345678 match.
func samePhone(a, b string) bool {
	da, db := digits(a), digits(b)
	if len(da) < 10 || len(db) < 10 {
		return false
	}
	return da[len(da)-10:] == db[len(db)-10:]
}

// TrackOrder handles GET /api/v1/track-order?order_number=...&phone=... (public).
// The phone must match the delivery phone on the order. Every failure returns the
// same NOT_FOUND so order numbers cannot be probed.
func (h *Handler) TrackOrder(c *fiber.Ctx) error {
	number := strings.TrimSpace(c.Query("order_number"))
	phone := strings.TrimSpace(c.Query("phone"))
	if number == "" || phone == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "order_number and phone are required", 422))
	}
	order, err := h.service.GetOrderByNumber(c.Context(), number)
	if err != nil || order.ShippingAddress == nil || !samePhone(order.ShippingAddress.Phone, phone) {
		return response.Error(c, errors.ErrNotFound)
	}

	full := toOrderResponse(order)
	items := make([]TrackItem, len(order.Items))
	for i, it := range order.Items {
		items[i] = TrackItem{Name: it.Name, ImageURL: it.ImageURL, Quantity: it.Quantity, Total: it.Pricing.Total}
	}
	return response.OK(c, TrackResponse{
		OrderNumber:    order.OrderNumber,
		Status:         string(order.Status),
		PaymentMethod:  string(order.PaymentMethod),
		PaymentStatus:  string(order.PaymentStatus),
		CreatedAt:      full.CreatedAt,
		Pricing:        full.Pricing,
		Items:          items,
		Carrier:        order.Carrier,
		TrackingNumber: order.TrackingNumber,
		City:           order.ShippingAddress.City,
		StatusHistory:  full.StatusHistory,
	})
}

// QuoteRequest is the body for POST /api/v1/checkout/quote.
type QuoteRequest struct {
	Items          []CreateOrderItemRequest `json:"items"`
	ShippingMethod string                   `json:"shipping_method"`
	DeliveryArea   string                   `json:"delivery_area"`
	CouponCode     string                   `json:"coupon_code"`
}

// QuoteCoupon reports what happened to the coupon code, without failing the whole quote.
type QuoteCoupon struct {
	Code         string `json:"code"`
	Applied      bool   `json:"applied"`
	Message      string `json:"message"`
	FreeShipping bool   `json:"free_shipping"`
}

// QuoteResponse is the price breakdown the server would charge right now.
type QuoteResponse struct {
	Subtotal       int          `json:"subtotal"`
	Discount       int          `json:"discount"`
	Shipping       int          `json:"shipping"`
	Total          int          `json:"total"`
	ShippingMethod string       `json:"shipping_method"`
	Coupon         *QuoteCoupon `json:"coupon,omitempty"`
}

// Quote handles POST /api/v1/checkout/quote (signed-in customers).
// It prices the cart exactly like order creation (catalog prices, delivery fee, coupon)
// without touching stock, so the checkout screen shows what will really be charged.
func (h *Handler) Quote(c *fiber.Ctx) error {
	auth, ok := c.Locals("auth").(*authTransport.AuthContext)
	if !ok || auth == nil {
		return response.Error(c, errors.ErrUnauthorized)
	}
	if h.catalog == nil {
		return response.Error(c, errors.New("INTERNAL_ERROR", "pricing is not configured", 500))
	}
	var req QuoteRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	if len(req.Items) == 0 {
		return response.Error(c, errors.New("VALIDATION_ERROR", "items must have at least one entry", 422))
	}

	subtotal := 0
	lines := make([]domain.CouponLine, 0, len(req.Items))
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return response.Error(c, errors.New("VALIDATION_ERROR", "item quantity must be positive", 422))
		}
		r, err := h.catalog.ResolveForOrder(c.Context(), it.ProductID, it.VariantID)
		if err != nil {
			return response.Error(c, err)
		}
		total := r.UnitPrice * it.Quantity
		subtotal += total
		lines = append(lines, domain.CouponLine{ProductID: r.ProductID, CategoryID: r.CategoryID, Total: total})
	}

	method, shipping, err := h.deliveryNow(c.Context()).Fee(req.ShippingMethod, req.DeliveryArea, subtotal)
	if err != nil {
		return response.Error(c, err)
	}

	out := QuoteResponse{Subtotal: subtotal, ShippingMethod: method}
	if code := strings.TrimSpace(req.CouponCode); code != "" {
		out.Coupon = &QuoteCoupon{Code: strings.ToUpper(code)}
		if h.coupons == nil {
			out.Coupon.Message = "coupons are not available"
		} else if q, qerr := h.coupons.Quote(c.Context(), code, auth.UserID, lines, subtotal); qerr != nil {
			if appErr, ok := errors.IsAppError(qerr); ok && appErr.Code == "VALIDATION_ERROR" {
				out.Coupon.Message = appErr.Message
			} else {
				return response.Error(c, qerr)
			}
		} else {
			out.Coupon.Applied, out.Coupon.Message, out.Coupon.FreeShipping = true, q.Message, q.FreeShipping
			out.Discount = q.Discount
			if q.FreeShipping {
				shipping = 0
			}
		}
	}
	out.Shipping = shipping
	out.Total = subtotal - out.Discount + shipping
	return response.OK(c, out)
}
