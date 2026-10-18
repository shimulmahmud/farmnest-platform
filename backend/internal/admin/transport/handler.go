package transport

import (
        "github.com/ecommerce/backend/internal/admin/application"
        "github.com/ecommerce/backend/internal/admin/domain"
        authTransport "github.com/ecommerce/backend/internal/auth/transport"
        revDomain "github.com/ecommerce/backend/internal/reviews/domain"
        userDomain "github.com/ecommerce/backend/internal/users/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/ecommerce/backend/pkg/response"
        "github.com/ecommerce/backend/pkg/validator"
        "github.com/gofiber/fiber/v2"
)

type Handler struct {
        service   *application.Service
        validator *validator.Validator
}

func NewHandler(service *application.Service) *Handler {
        return &Handler{service: service, validator: validator.New()}
}

func getAuth(c *fiber.Ctx) (*authTransport.AuthContext, error) {
        auth, ok := c.Locals("auth").(*authTransport.AuthContext)
        if !ok || auth == nil {
                return nil, errors.ErrUnauthorized
        }
        return auth, nil
}

// --- User Management ---

// ListUsers returns a paginated, filterable list of users.
func (h *Handler) ListUsers(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        filter := &userDomain.UserFilter{
                Role:   userDomain.Role(c.Query("role")),
                Status: userDomain.AccountStatus(c.Query("status")),
                Search: c.Query("search"),
        }

        users, total, err := h.service.ListUsers(c.Context(), filter, p)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OKWithMeta(c, toUserResponses(users), response.PaginatedMeta(p.Page, p.PerPage, total))
}

// GetUser returns a single user by ID.
func (h *Handler) GetUser(c *fiber.Ctx) error {
        id := c.Params("id")
        user, err := h.service.GetUser(c.Context(), id)
        if err != nil {
                return response.Error(c, err)
        }
        return response.OK(c, toUserResponse(user))
}

// UpdateUserRole changes a user's role and creates an audit record.
func (h *Handler) UpdateUserRole(c *fiber.Ctx) error {
        if _, err := getAuth(c); err != nil {
                return response.Error(c, err)
        }

        var req UpdateRoleRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.New("VALIDATION_ERROR", "invalid request body", 422))
        }
        if err := h.validator.ValidateRequired(map[string]string{"role": req.Role}); err != nil {
                return response.Error(c, err)
        }

        ac := toAppAuditContext(buildAuditContext(c))
        if err := h.service.UpdateUserRole(c.Context(), ac, c.Params("id"), userDomain.Role(req.Role)); err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, fiber.Map{"message": "role updated"})
}

// UpdateUserStatus changes a user's account status and creates an audit record.
func (h *Handler) UpdateUserStatus(c *fiber.Ctx) error {
        if _, err := getAuth(c); err != nil {
                return response.Error(c, err)
        }

        var req UpdateStatusRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.New("VALIDATION_ERROR", "invalid request body", 422))
        }
        if err := h.validator.ValidateRequired(map[string]string{"status": req.Status}); err != nil {
                return response.Error(c, err)
        }

        ac := toAppAuditContext(buildAuditContext(c))
        if err := h.service.UpdateUserStatus(c.Context(), ac, c.Params("id"), userDomain.AccountStatus(req.Status)); err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, fiber.Map{"message": "status updated"})
}

// --- Customer Management ---

// ListCustomers returns a paginated list of all customers.
func (h *Handler) ListCustomers(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        customers, total, err := h.service.ListCustomers(c.Context(), p)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OKWithMeta(c, toCustomerResponses(customers), response.PaginatedMeta(p.Page, p.PerPage, total))
}

// GetCustomer returns a single customer by ID.
func (h *Handler) GetCustomer(c *fiber.Ctx) error {
        id := c.Params("id")
        customer, err := h.service.GetCustomer(c.Context(), id)
        if err != nil {
                return response.Error(c, err)
        }
        return response.OK(c, toCustomerResponse(customer))
}

// --- Review Moderation ---

// ListReviews returns a paginated list of reviews with optional status filter.
func (h *Handler) ListReviews(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        status := c.Query("status")

        result, err := h.service.ListReviews(c.Context(), revDomain.ModerationStatus(status), p)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OKWithMeta(c, toReviewModerationResponses(result.Reviews), response.PaginatedMeta(result.Page, result.PerPage, result.Total))
}

// ModerateReview applies an admin moderation action to a review and creates an audit record.
func (h *Handler) ModerateReview(c *fiber.Ctx) error {
        if _, err := getAuth(c); err != nil {
                return response.Error(c, err)
        }

        var req ModerateReviewRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.New("VALIDATION_ERROR", "invalid request body", 422))
        }
        if err := h.validator.ValidateRequired(map[string]string{"action": req.Action}); err != nil {
                return response.Error(c, err)
        }

        reviewID := c.Params("id")
        action := &domain.ReviewModerationAction{
                Action: req.Action,
                Reason: req.Reason,
        }

        ac := toAppAuditContext(buildAuditContext(c))
        review, err := h.service.ModerateReview(c.Context(), ac, reviewID, action)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toReviewModerationResponse(review))
}

// --- Audit Log ---

// ListAuditLogs returns a paginated list of audit log entries.
func (h *Handler) ListAuditLogs(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        filter := &application.AuditFilter{
                ActorID:      c.Query("actor_id"),
                Action:       c.Query("action"),
                ResourceType: c.Query("resource_type"),
                ResourceID:   c.Query("resource_id"),
                RequestID:    c.Query("request_id"),
        }

        entries, total, err := h.service.ListAuditLogs(c.Context(), filter, p)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OKWithMeta(c, toAuditEntryResponses(entries), response.PaginatedMeta(p.Page, p.PerPage, total))
}

// --- Dashboard ---

// GetDashboardCounts returns platform-wide counts.
func (h *Handler) GetDashboardCounts(c *fiber.Ctx) error {
        counts, err := h.service.GetDashboardCounts(c.Context())
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, DashboardCountsResponse{
                TotalUsers:     counts.TotalUsers,
                TotalCustomers: counts.TotalCustomers,
                PendingReviews: counts.PendingReviews,
        })
}
