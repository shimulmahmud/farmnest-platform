package transport

import (
	"fmt"

	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	"github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/pkg/audit"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for user profile endpoints.
type Handler struct {
	userRepo domain.UserRepository
	auditor  audit.Auditor
}

// NewHandler creates a user transport handler.
func NewHandler(userRepo domain.UserRepository) *Handler {
	return &Handler{userRepo: userRepo, auditor: audit.NopAuditor}
}

// SetAuditor injects an async auditor for profile change audit logging.
func (h *Handler) SetAuditor(a audit.Auditor) {
	h.auditor = a
}

// GetMe handles GET /api/v1/me.
func (h *Handler) GetMe(c *fiber.Ctx) error {
	auth, ok := c.Locals(authTransport.AuthContextKey).(*authTransport.AuthContext)
	if !ok || auth == nil {
		return c.SendStatus(401)
	}

	user, err := h.userRepo.FindByID(c.Context(), auth.UserID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toUserResponse(user))
}

// UpdateMe handles PATCH /api/v1/me.
// Records an audit entry when the customer changes their profile.
func (h *Handler) UpdateMe(c *fiber.Ctx) error {
	auth, ok := c.Locals(authTransport.AuthContextKey).(*authTransport.AuthContext)
	if !ok || auth == nil {
		return c.SendStatus(401)
	}

	var req struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	user, err := h.userRepo.FindByID(c.Context(), auth.UserID)
	if err != nil {
		return response.Error(c, err)
	}

	// Capture old values for audit
	oldProfile := fmt.Sprintf("first_name=%s,last_name=%s", user.FirstName, user.LastName)

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}

	if err := h.userRepo.Update(c.Context(), user); err != nil {
		return response.Error(c, err)
	}

	// Audit: record customer profile update
	newProfile := fmt.Sprintf("first_name=%s,last_name=%s", user.FirstName, user.LastName)
	if oldProfile != newProfile {
		h.auditor.Record(auth.UserID, audit.ActionCustomerProfileUpdate, "customer_profile", auth.UserID, oldProfile, newProfile, "customer updated own profile")
	}

	return response.OK(c, toUserResponse(user))
}

type userResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     string `json:"created_at"`
}

func toUserResponse(u *domain.User) *userResponse {
	return &userResponse{
		ID:            u.ID,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Role:          string(u.Role),
		Status:        string(u.Status),
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
