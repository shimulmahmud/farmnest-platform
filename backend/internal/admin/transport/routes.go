package transport

import (
	adminDomain "github.com/ecommerce/backend/internal/admin/domain"
	"github.com/gofiber/fiber/v2"
)

// RegisterAdminRoutes mounts all admin endpoints with RBAC enforcement.
// Every admin endpoint requires authentication + admin role + specific permission.
func RegisterAdminRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler) {
	// All admin routes require auth + admin-level role first
	admin := router.Group("/api/v1/admin", authMW, adminMW)

	// --- Dashboard (all admin roles can read) ---
	admin.Get("/dashboard/counts", handler.GetDashboardCounts)

	// --- User Management ---
	admin.Get("/users", RequirePermission(adminDomain.PermUserRead), handler.ListUsers)
	admin.Get("/users/:id", RequirePermission(adminDomain.PermUserRead), handler.GetUser)
	admin.Patch("/users/:id/role", RequirePermission(adminDomain.PermUserWrite), handler.UpdateUserRole)
	admin.Patch("/users/:id/status", RequirePermission(adminDomain.PermUserWrite), handler.UpdateUserStatus)

	// --- Customer Management ---
	admin.Get("/customers", RequirePermission(adminDomain.PermUserRead), handler.ListCustomers)
	admin.Get("/customers/:id", RequirePermission(adminDomain.PermUserRead), handler.GetCustomer)

	// --- Review Moderation ---
	admin.Get("/reviews", RequirePermission(adminDomain.PermReviewRead), handler.ListReviews)
	admin.Patch("/reviews/:id/moderate", RequirePermission(adminDomain.PermReviewWrite), handler.ModerateReview)

	// --- Audit Log (read-only, restricted to admin and above) ---
	admin.Get("/audit-logs", RequirePermission(adminDomain.PermAuditRead), handler.ListAuditLogs)
}
