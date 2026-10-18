package transport

import (
	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/internal/vendors/application"
	"github.com/ecommerce/backend/internal/vendors/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// Handler implements vendor HTTP endpoints.
type Handler struct{ service *application.Service }

// NewHandler creates a vendor handler.
func NewHandler(s *application.Service) *Handler { return &Handler{service: s} }

// CreateVendorRequest is the body for POST /api/v1/admin/vendors.
type CreateVendorRequest struct {
	OwnerUserID   string `json:"owner_user_id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	CommissionBps int    `json:"commission_bps"`
}

// SetStatusRequest is the body for PATCH /api/v1/admin/vendors/:id/status.
type SetStatusRequest struct {
	Status string `json:"status"`
}

// CreateVendor handles POST /api/v1/admin/vendors.
func (h *Handler) CreateVendor(c *fiber.Ctx) error {
	var req CreateVendorRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	v, err := h.service.Create(c.Context(), req.OwnerUserID, req.Name, req.Slug, req.CommissionBps)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, v)
}

// ListVendors handles GET /api/v1/admin/vendors.
func (h *Handler) ListVendors(c *fiber.Ctx) error {
	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}
	list, total, err := h.service.List(c.Context(), c.Query("status"), p)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OKWithMeta(c, list, response.PaginatedMeta(p.Page, p.PerPage, total))
}

// GetVendor handles GET /api/v1/admin/vendors/:id.
func (h *Handler) GetVendor(c *fiber.Ctx) error {
	v, err := h.service.Get(c.Context(), c.Params("id"))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, v)
}

// SetStatus handles PATCH /api/v1/admin/vendors/:id/status.
func (h *Handler) SetStatus(c *fiber.Ctx) error {
	var req SetStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	v, err := h.service.SetStatus(c.Context(), c.Params("id"), domain.Status(req.Status))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, v)
}

// Me handles GET /api/v1/vendor/me.
func (h *Handler) Me(c *fiber.Ctx) error {
	id, _ := c.Locals("vendor_id").(string)
	v, err := h.service.Get(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, v)
}

// VendorMiddleware requires role VENDOR with an APPROVED vendor account and
// stores its ID in Locals("vendor_id"). Vendor-scoped handlers must read the
// vendor ID from here, never from the request.
func VendorMiddleware(s *application.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth, ok := c.Locals(authTransport.AuthContextKey).(*authTransport.AuthContext)
		if !ok || auth == nil {
			return response.Error(c, errors.ErrUnauthorized)
		}
		if auth.Role != userDomain.RoleVendor {
			return response.Error(c, errors.ErrForbidden)
		}
		v, err := s.GetByOwner(c.Context(), auth.UserID)
		if err != nil || v.Status != domain.StatusApproved {
			return response.Error(c, errors.ErrForbidden)
		}
		c.Locals("vendor_id", v.ID)
		return c.Next()
	}
}

// RegisterVendorRoutes mounts vendor endpoints.
func RegisterVendorRoutes(router fiber.Router, h *Handler, authMW, adminMW, vendorMW fiber.Handler) {
	admin := router.Group("/api/v1/admin/vendors", authMW, adminMW)
	admin.Post("/", h.CreateVendor)
	admin.Get("/", h.ListVendors)
	admin.Get("/:id", h.GetVendor)
	admin.Patch("/:id/status", h.SetStatus)

	router.Get("/api/v1/vendor/me", authMW, vendorMW, h.Me)
}
