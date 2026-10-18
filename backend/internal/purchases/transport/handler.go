package transport

import (
	"github.com/ecommerce/backend/internal/purchases/application"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// Handler implements purchase (stock receiving) endpoints.
type Handler struct{ service *application.Service }

// NewHandler creates a purchases handler.
func NewHandler(s *application.Service) *Handler { return &Handler{service: s} }

// Receive handles POST /api/v1/admin/purchases.
func (h *Handler) Receive(c *fiber.Ctx) error {
	var req application.ReceiveInput
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	p, err := h.service.Receive(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, p)
}

// Get handles GET /api/v1/admin/purchases/:id.
func (h *Handler) Get(c *fiber.Ctx) error {
	p, err := h.service.Get(c.Context(), c.Params("id"))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, p)
}

// List handles GET /api/v1/admin/purchases.
func (h *Handler) List(c *fiber.Ctx) error {
	pg, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}
	list, total, err := h.service.List(c.Context(), c.Query("supplier_name"), pg)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OKWithMeta(c, list, response.PaginatedMeta(pg.Page, pg.PerPage, total))
}

// RegisterRoutes mounts the admin-only purchase routes.
func RegisterRoutes(router fiber.Router, h *Handler, authMW, adminMW fiber.Handler) {
	g := router.Group("/api/v1/admin/purchases", authMW, adminMW)
	g.Post("/", h.Receive)
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
}
