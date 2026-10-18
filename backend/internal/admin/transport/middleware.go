package transport

import (
	adminDomain "github.com/ecommerce/backend/internal/admin/domain"
	"github.com/ecommerce/backend/internal/admin/application"
	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// RequirePermission returns middleware that checks if the authenticated user
// has the specified admin permission based on their role.
// This enforces RBAC at the server level and does not rely on frontend restrictions.
func RequirePermission(perm adminDomain.Permission) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth, ok := c.Locals("auth").(*authTransport.AuthContext)
		if !ok || auth == nil {
			return response.Error(c, errors.ErrUnauthorized)
		}

		if !adminDomain.HasPermission(auth.Role, perm) {
			return response.Error(c, errors.ErrForbidden)
		}

		return c.Next()
	}
}

// buildAuditContext extracts auth info and request metadata.
func buildAuditContext(c *fiber.Ctx) *auditContextInner {
	auth, _ := c.Locals("auth").(*authTransport.AuthContext)
	ac := &auditContextInner{
		ActorID:   "",
		ActorRole: "",
	}
	if auth != nil {
		ac.ActorID = auth.UserID
		ac.ActorRole = string(auth.Role)
	}
	ac.IPAddress = c.IP()
	ac.UserAgent = c.Get("User-Agent")
	ac.RequestID, _ = c.Locals("request_id").(string)
	return ac
}

// auditContextInner mirrors application.AuditContext.
type auditContextInner struct {
	ActorID   string
	ActorRole string
	IPAddress string
	UserAgent string
	RequestID string
}

// toAppAuditContext converts the transport-level audit context to the
// application-layer AuditContext used by the service.
func toAppAuditContext(ac *auditContextInner) *application.AuditContext {
	return &application.AuditContext{
		ActorID:   ac.ActorID,
		ActorRole: ac.ActorRole,
		IPAddress: ac.IPAddress,
		UserAgent: ac.UserAgent,
		RequestID: ac.RequestID,
	}
}
