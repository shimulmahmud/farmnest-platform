package transport

import (
	"strings"

	"github.com/ecommerce/backend/internal/orders/application"
	"github.com/ecommerce/backend/internal/orders/domain"
	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/ecommerce/backend/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for order endpoints.
type Handler struct {
	service   *application.Service
	validator *validator.Validator
	catalog   domain.CatalogResolver
	coupons   domain.CouponApplier
	checkout  CheckoutConfig
	delivery  DeliveryRates
}

// SetCatalog enables server-side pricing. When set, name, sku, unit_price and
// discount sent by the client are ignored and re-read from the catalog.
// Production wiring must call this; it is optional only so existing tests
// that use fabricated product IDs keep working.
func (h *Handler) SetCatalog(c domain.CatalogResolver) {
	h.catalog = c
}

// SetCoupons enables coupon codes at checkout (optional).
func (h *Handler) SetCoupons(c domain.CouponApplier) {
	h.coupons = c
}

// NewHandler creates an order transport handler.
func NewHandler(service *application.Service) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
			checkout:  DefaultCheckoutConfig(),
	}
}

// CreateOrder handles POST /api/v1/orders.
func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	if req.ShippingAddress == nil {
		return response.Error(c, errors.New("VALIDATION_ERROR", "shipping_address is required", 422))
	}
	if err := h.validator.ValidateRequired(map[string]string{
		"shipping_address_line1":      req.ShippingAddress.Line1,
		"shipping_address_city":       req.ShippingAddress.City,
		"shipping_address_state":      req.ShippingAddress.State,
	}); err != nil {
		return response.Error(c, err)
	}

	if req.ShippingAddress.Country == "" {
		req.ShippingAddress.Country = "BD"
	}
	if req.BillingAddress == nil {
		req.BillingAddress = req.ShippingAddress // single delivery address
	}
	if strings.TrimSpace(req.ShippingAddress.Name) == "" || strings.TrimSpace(req.ShippingAddress.Phone) == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "shipping_address name and phone are required for delivery", 422))
	}

	paymentMethod := domain.PaymentMethod(strings.ToUpper(strings.TrimSpace(req.PaymentMethod)))
	if paymentMethod == "" {
		paymentMethod = domain.PaymentOnline
	}
	if err := h.validatePayment(paymentMethod, req); err != nil {
		return response.Error(c, err)
	}

	if len(req.Items) == 0 {
		return response.Error(c, errors.New("VALIDATION_ERROR", "items must have at least one entry", 422))
	}

	// Extract customer ID from auth context
	auth, ok := c.Locals("auth").(*authTransport.AuthContext)
	if !ok || auth == nil {
		return response.Error(c, errors.ErrUnauthorized)
	}

	// Build domain items and compute pricing
	domainItems := make([]domain.OrderItem, len(req.Items))
	var subtotal int
	couponLines := make([]domain.CouponLine, 0, len(req.Items))
	for i, item := range req.Items {
		if item.Quantity <= 0 {
			return response.Error(c, errors.New("VALIDATION_ERROR", "item quantity must be positive", 422))
		}
		if err := h.validator.ValidateRequired(map[string]string{
			"item_product_id": item.ProductID,
			"item_variant_id": item.VariantID,
		}); err != nil {
			return response.Error(c, err)
		}

		productID, variantID := item.ProductID, item.VariantID
		vendorID, name, sku := "", item.Name, item.SKU
		var snap domain.ResolvedItem
		unitPrice, discount := item.UnitPrice, item.Discount

		if h.catalog != nil {
			// Trusted path: everything price-related comes from the catalog.
			resolved, err := h.catalog.ResolveForOrder(c.Context(), item.ProductID, item.VariantID)
			if err != nil {
				return response.Error(c, err)
			}
			productID, variantID = resolved.ProductID, resolved.VariantID
			vendorID, name, sku = resolved.VendorID, resolved.Name, resolved.SKU
			unitPrice, discount = resolved.UnitPrice, 0
			snap = *resolved
		} else if err := h.validator.ValidateRequired(map[string]string{
			"item_name": item.Name,
			"item_sku":  item.SKU,
		}); err != nil {
			return response.Error(c, err)
		}

		if unitPrice <= 0 || discount < 0 || discount > unitPrice {
			return response.Error(c, errors.New("VALIDATION_ERROR", "invalid item price", 422))
		}

		itemTotal := (unitPrice - discount) * item.Quantity
		domainItems[i] = domain.OrderItem{
			VendorID:  vendorID,
			CommissionBps: snap.CommissionBps,
			ProductID: productID,
			VariantID: variantID,
			Name:      name,
			SKU:       sku,
			Barcode:   snap.Barcode,
			ImageURL:  snap.ImageURL,
			Options:   snap.Options,
			Currency:  snap.Currency,
			WeightG:   snap.WeightG,
			RequiresShipping: snap.RequiresShipping,
			Quantity:  item.Quantity,
			UnitCost:  snap.UnitCost,
			Pricing: domain.OrderItemPricing{
				UnitPrice: unitPrice,
				Discount:  discount,
				Total:     itemTotal,
			},
		}
		subtotal += itemTotal
		couponLines = append(couponLines, domain.CouponLine{ProductID: productID, CategoryID: snap.CategoryID, Total: itemTotal})
	}

	// For now, compute pricing on the server side from items.
	// A future checkout module would provide the full pricing breakdown.
	shippingMethod, shippingFee, err := h.deliveryNow(c.Context()).Fee(req.ShippingMethod, req.DeliveryArea, subtotal)
	if err != nil {
		return response.Error(c, err)
	}

	// Coupon: validated here against the real cart; its use is counted when the order is placed.
	var couponCode, promotionID string
	discount := 0
	if code := strings.TrimSpace(req.CouponCode); code != "" {
		if h.coupons == nil {
			return response.Error(c, errors.New("VALIDATION_ERROR", "coupons are not available", 422))
		}
		quote, err := h.coupons.Quote(c.Context(), code, auth.UserID, couponLines, subtotal)
		if err != nil {
			return response.Error(c, err)
		}
		discount, couponCode, promotionID = quote.Discount, quote.Code, quote.PromotionID
		if quote.FreeShipping {
			shippingFee = 0
		}
	}
	pricing := domain.OrderPricing{
		Subtotal: subtotal,
		Discount: discount,
		Shipping: shippingFee,
		Total:    subtotal - discount + shippingFee,
	}

	order, err := h.service.CreateOrder(c.Context(), application.CreateOrderInput{
		CustomerID:      auth.UserID,
		Items:           domainItems,
		Pricing:         pricing,
		ShippingAddress: toDomainAddress(req.ShippingAddress),
		BillingAddress:  toDomainAddress(req.BillingAddress),
		PaymentMethod:   paymentMethod,
		PaymentProvider:  strings.ToLower(strings.TrimSpace(req.PaymentProvider)),
		PaymentReference: strings.TrimSpace(req.PaymentReference),
		PayerNumber:      strings.TrimSpace(req.PayerNumber),
		ShippingMethod:   shippingMethod,
		DeliveryArea:     normalizeArea(req.DeliveryArea),
		CouponCode:       couponCode,
		PromotionID:      promotionID,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, toOrderResponse(order))
}

// GetOrder handles GET /api/v1/orders/:id.
func (h *Handler) GetOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "order id is required", 422))
	}

	// Extract customer ID for ownership check
	auth, ok := c.Locals("auth").(*authTransport.AuthContext)
	if !ok || auth == nil {
		return response.Error(c, errors.ErrUnauthorized)
	}

	order, err := h.service.GetOrderByID(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	// Ensure the order belongs to the authenticated customer
	if order.CustomerID != auth.UserID {
		return response.Error(c, errors.ErrNotFound)
	}

	return response.OK(c, toOrderResponse(order))
}

// ListOrders handles GET /api/v1/orders.
func (h *Handler) ListOrders(c *fiber.Ctx) error {
	// Extract customer ID from auth context
	auth, ok := c.Locals("auth").(*authTransport.AuthContext)
	if !ok || auth == nil {
		return response.Error(c, errors.ErrUnauthorized)
	}

	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}

	orders, total, err := h.service.ListOrders(c.Context(), auth.UserID, c.Query("status"), p)
	if err != nil {
		return response.Error(c, err)
	}

	resp := make([]OrderResponse, len(orders))
	for i, o := range orders {
		resp[i] = toOrderResponse(o)
	}

	meta := response.PaginatedMeta(p.Page, p.PerPage, total)
	return response.OKWithMeta(c, resp, meta)
}

// AdminListOrders handles GET /api/v1/admin/orders.
func (h *Handler) AdminListOrders(c *fiber.Ctx) error {
	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}

	orders, total, err := h.service.ListAllOrders(c.Context(), c.Query("status"), c.Query("customer_id"), p)
	if err != nil {
		return response.Error(c, err)
	}

	resp := make([]AdminOrderResponse, len(orders))
	for i, o := range orders {
		resp[i] = toAdminOrderResponse(o)
	}

	meta := response.PaginatedMeta(p.Page, p.PerPage, total)
	return response.OKWithMeta(c, resp, meta)
}

// UpdateOrderStatus handles PATCH /api/v1/admin/orders/:id/status.
func (h *Handler) UpdateOrderStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "order id is required", 422))
	}

	var req UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	if err := h.validator.ValidateRequired(map[string]string{
		"status": req.Status,
	}); err != nil {
		return response.Error(c, err)
	}

	order, err := h.service.UpdateOrderStatus(c.Context(), id, application.UpdateStatusInput{
		Status:           req.Status,
		Reason:           req.Reason,
		PaymentReference:  req.PaymentReference,
		ShipmentReference: req.ShipmentReference,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toAdminOrderResponse(order))
}

// ShipOrder handles POST /api/v1/admin/orders/:id/ship.
// One step: marks the order SHIPPED and records the
// courier and tracking number.
func (h *Handler) ShipOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "order id is required", 422))
	}
	var req ShipOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	order, err := h.service.ShipOrder(c.Context(), id, application.ShipInput{
		Carrier:        req.Carrier,
		TrackingNumber: req.TrackingNumber,
		CourierCost:    req.CourierCost,
	})
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, toAdminOrderResponse(order))
}

// MarkPaid handles POST /api/v1/admin/orders/:id/mark-paid.
// Records that the cash for a cash-on-delivery order was collected.
func (h *Handler) MarkPaid(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "order id is required", 422))
	}
	order, err := h.service.MarkPaid(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, toAdminOrderResponse(order))
}
