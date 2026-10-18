package transport

import (
	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	"github.com/ecommerce/backend/internal/settings/application"
	"github.com/ecommerce/backend/internal/settings/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// Handler serves the admin settings endpoints.
type Handler struct{ service *application.Service }

// NewHandler creates a settings handler.
func NewHandler(s *application.Service) *Handler { return &Handler{service: s} }

// DeliveryResponse is the delivery setting plus the list of areas (for the admin form).
type DeliveryResponse struct {
	domain.Delivery
	Areas []domain.Area `json:"areas"`
}

// UpdateDeliveryRequest is the body for PUT /api/v1/admin/settings/delivery.
type UpdateDeliveryRequest struct {
	FreeShippingThreshold int             `json:"free_shipping_threshold"`
	Methods               []domain.Method `json:"methods"`
}

// GetDelivery handles GET /api/v1/admin/settings/delivery.
func (h *Handler) GetDelivery(c *fiber.Ctx) error {
	d, err := h.service.Delivery(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, DeliveryResponse{Delivery: d, Areas: domain.Areas()})
}

// UpdateDelivery handles PUT /api/v1/admin/settings/delivery.
func (h *Handler) UpdateDelivery(c *fiber.Ctx) error {
	var req UpdateDeliveryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	by := ""
	if a, ok := c.Locals(authTransport.AuthContextKey).(*authTransport.AuthContext); ok && a != nil {
		by = a.UserID
	}
	d, err := h.service.UpdateDelivery(c.Context(), domain.Delivery{
		FreeShippingThreshold: req.FreeShippingThreshold,
		Methods:               req.Methods,
	}, by)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, DeliveryResponse{Delivery: *d, Areas: domain.Areas()})
}

// RegisterRoutes mounts the admin settings routes.
func RegisterRoutes(router fiber.Router, h *Handler, authMW, adminMW fiber.Handler) {
	g := router.Group("/api/v1/admin/settings", authMW, adminMW)
	g.Get("/delivery", h.GetDelivery)
	g.Put("/delivery", h.UpdateDelivery)
}
