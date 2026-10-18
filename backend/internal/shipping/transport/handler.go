package transport

import (
        "github.com/ecommerce/backend/internal/shipping/application"
        "github.com/ecommerce/backend/internal/shipping/domain"
        authTransport "github.com/ecommerce/backend/internal/auth/transport"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/ecommerce/backend/pkg/response"
        "github.com/ecommerce/backend/pkg/validator"
        "github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for shipping endpoints.
type Handler struct {
        service   *application.Service
        validator *validator.Validator
}

// NewHandler creates a shipping transport handler.
func NewHandler(service *application.Service) *Handler {
        return &Handler{
                service:   service,
                validator: validator.New(),
        }
}

// CalculateRate handles POST /api/v1/shipping/rates.
func (h *Handler) CalculateRate(c *fiber.Ctx) error {
        var req CalculateRateRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if req.Destination == nil {
                return response.Error(c, errors.New("VALIDATION_ERROR", "destination is required", 422))
        }
        if err := h.validator.ValidateRequired(map[string]string{
                "destination_line1":      req.Destination.Line1,
                "destination_city":       req.Destination.City,
                "destination_state":      req.Destination.State,
                "destination_postal_code": req.Destination.PostalCode,
                "destination_country":    req.Destination.Country,
        }); err != nil {
                return response.Error(c, err)
        }

        if len(req.Packages) == 0 {
                return response.Error(c, errors.New("VALIDATION_ERROR", "at least one package is required", 422))
        }

        rates, err := h.service.CalculateRate(c.Context(), application.RateRequestInput{
                Destination: toDomainShipmentAddress(req.Destination),
                Packages:    toDomainShipmentPackages(req.Packages),
                Currency:    req.Currency,
        })
        if err != nil {
                return response.Error(c, err)
        }

        resp := make([]ShippingRateResponse, len(rates))
        for i, r := range rates {
                resp[i] = toRateResponse(r)
        }

        return response.OK(c, resp)
}

// CreateShipment handles POST /api/v1/admin/shipments.
func (h *Handler) CreateShipment(c *fiber.Ctx) error {
        var req CreateShipmentRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "order_id":    req.OrderID,
                "customer_id": req.CustomerID,
                "method":      req.Method,
                "carrier":     req.Carrier,
        }); err != nil {
                return response.Error(c, err)
        }

        if !domain.ValidShippingMethods[domain.ShippingMethod(req.Method)] {
                return response.Error(c, errors.New("VALIDATION_ERROR", "invalid shipping method", 422))
        }

        if req.Destination == nil {
                return response.Error(c, errors.New("VALIDATION_ERROR", "destination is required", 422))
        }

        if len(req.Packages) == 0 {
                return response.Error(c, errors.New("VALIDATION_ERROR", "at least one package is required", 422))
        }

        shipment, err := h.service.CreateShipment(c.Context(), application.CreateShipmentInput{
                OrderID:     req.OrderID,
                CustomerID:  req.CustomerID,
                Method:      domain.ShippingMethod(req.Method),
                Carrier:     req.Carrier,
                Destination: toDomainShipmentAddress(req.Destination),
                Packages:    toDomainShipmentPackages(req.Packages),
                Currency:    req.Currency,
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.Created(c, toShipmentResponse(shipment))
}

// GetShipment handles GET /api/v1/shipments/:id.
func (h *Handler) GetShipment(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "shipment id is required", 422))
        }

        auth, ok := c.Locals("auth").(*authTransport.AuthContext)
        if !ok || auth == nil {
                return response.Error(c, errors.ErrUnauthorized)
        }

        shipment, err := h.service.GetShipment(c.Context(), id)
        if err != nil {
                return response.Error(c, err)
        }

        if shipment.CustomerID != auth.UserID {
                return response.Error(c, errors.ErrNotFound)
        }

        return response.OK(c, toShipmentResponse(shipment))
}

// GetShipmentByOrder handles GET /api/v1/shipments/order/:order_id.
func (h *Handler) GetShipmentByOrder(c *fiber.Ctx) error {
        orderID := c.Params("order_id")
        if orderID == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "order_id is required", 422))
        }

        auth, ok := c.Locals("auth").(*authTransport.AuthContext)
        if !ok || auth == nil {
                return response.Error(c, errors.ErrUnauthorized)
        }

        shipment, err := h.service.GetShipmentByOrder(c.Context(), orderID)
        if err != nil {
                return response.Error(c, err)
        }

        if shipment.CustomerID != auth.UserID {
                return response.Error(c, errors.ErrNotFound)
        }

        return response.OK(c, toShipmentResponse(shipment))
}

// ListShipments handles GET /api/v1/shipments.
func (h *Handler) ListShipments(c *fiber.Ctx) error {
        auth, ok := c.Locals("auth").(*authTransport.AuthContext)
        if !ok || auth == nil {
                return response.Error(c, errors.ErrUnauthorized)
        }

        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        shipments, total, err := h.service.ListShipments(c.Context(), auth.UserID, c.Query("status"), p)
        if err != nil {
                return response.Error(c, err)
        }

        resp := make([]ShipmentResponse, len(shipments))
        for i, s := range shipments {
                resp[i] = toShipmentResponse(s)
        }

        meta := response.PaginatedMeta(p.Page, p.PerPage, total)
        return response.OKWithMeta(c, resp, meta)
}

// TrackShipment handles POST /api/v1/shipments/:id/track.
func (h *Handler) TrackShipment(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "shipment id is required", 422))
        }

        auth, ok := c.Locals("auth").(*authTransport.AuthContext)
        if !ok || auth == nil {
                return response.Error(c, errors.ErrUnauthorized)
        }

        shipment, err := h.service.TrackShipment(c.Context(), id)
        if err != nil {
                return response.Error(c, err)
        }

        if shipment.CustomerID != auth.UserID {
                return response.Error(c, errors.ErrNotFound)
        }

        return response.OK(c, toShipmentResponse(shipment))
}

// AdminListShipments handles GET /api/v1/admin/shipments.
func (h *Handler) AdminListShipments(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        shipments, total, err := h.service.ListAllShipments(c.Context(), c.Query("status"), c.Query("customer_id"), c.Query("carrier"), p)
        if err != nil {
                return response.Error(c, err)
        }

        resp := make([]ShipmentResponse, len(shipments))
        for i, s := range shipments {
                resp[i] = toShipmentResponse(s)
        }

        meta := response.PaginatedMeta(p.Page, p.PerPage, total)
        return response.OKWithMeta(c, resp, meta)
}

// UpdateShipmentStatus handles PATCH /api/v1/admin/shipments/:id/status.
func (h *Handler) UpdateShipmentStatus(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "shipment id is required", 422))
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

        shipment, err := h.service.UpdateShipmentStatus(c.Context(), id, application.UpdateStatusInput{
                Status: req.Status,
                Reason: req.Reason,
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toShipmentResponse(shipment))
}

// CancelShipment handles POST /api/v1/shipments/:id/cancel.
func (h *Handler) CancelShipment(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "shipment id is required", 422))
        }

        auth, ok := c.Locals("auth").(*authTransport.AuthContext)
        if !ok || auth == nil {
                return response.Error(c, errors.ErrUnauthorized)
        }

        var req CancelShipmentRequest
        _ = c.BodyParser(&req) // reason is optional

        shipment, err := h.service.CancelShipment(c.Context(), id, req.Reason)
        if err != nil {
                return response.Error(c, err)
        }

        // Ownership check: only the customer or admin can cancel
        if shipment.CustomerID != auth.UserID {
                return response.Error(c, errors.ErrNotFound)
        }

        return response.OK(c, toShipmentResponse(shipment))
}
