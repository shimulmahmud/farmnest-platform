package domain

import userDomain "github.com/ecommerce/backend/internal/users/domain"

// Permission represents a granular admin capability.
type Permission string

const (
	PermUserRead      Permission = "admin.users.read"
	PermUserWrite     Permission = "admin.users.write"
	PermProductRead   Permission = "admin.products.read"
	PermProductWrite  Permission = "admin.products.write"
	PermCategoryRead  Permission = "admin.categories.read"
	PermCategoryWrite Permission = "admin.categories.write"
	PermOrderRead     Permission = "admin.orders.read"
	PermOrderWrite    Permission = "admin.orders.write"
	PermInventoryRead  Permission = "admin.inventory.read"
	PermInventoryWrite Permission = "admin.inventory.write"
	PermCouponRead    Permission = "admin.coupons.read"
	PermCouponWrite   Permission = "admin.coupons.write"
	PermReviewRead    Permission = "admin.reviews.read"
	PermReviewWrite   Permission = "admin.reviews.write"
	PermAnalyticsRead Permission = "admin.analytics.read"
	PermAuditRead     Permission = "admin.audit.read"
)

// RolePermissions maps each admin role to its set of granted permissions.
// SUPER_ADMIN has all permissions. ADMIN and MANAGER have subsets.
var RolePermissions = map[userDomain.Role]map[Permission]bool{
	userDomain.RoleSuperAdmin: {
		PermUserRead: true, PermUserWrite: true,
		PermProductRead: true, PermProductWrite: true,
		PermCategoryRead: true, PermCategoryWrite: true,
		PermOrderRead: true, PermOrderWrite: true,
		PermInventoryRead: true, PermInventoryWrite: true,
		PermCouponRead: true, PermCouponWrite: true,
		PermReviewRead: true, PermReviewWrite: true,
		PermAnalyticsRead: true,
		PermAuditRead: true,
	},
	userDomain.RoleAdmin: {
		PermUserRead: true, PermUserWrite: true,
		PermProductRead: true, PermProductWrite: true,
		PermCategoryRead: true, PermCategoryWrite: true,
		PermOrderRead: true, PermOrderWrite: true,
		PermInventoryRead: true, PermInventoryWrite: true,
		PermCouponRead: true, PermCouponWrite: true,
		PermReviewRead: true, PermReviewWrite: true,
		PermAnalyticsRead: true,
		PermAuditRead: true,
	},
	userDomain.RoleManager: {
		PermUserRead: true,
		PermProductRead: true, PermProductWrite: true,
		PermCategoryRead: true, PermCategoryWrite: true,
		PermOrderRead: true, PermOrderWrite: true,
		PermInventoryRead: true, PermInventoryWrite: true,
		PermReviewRead: true, PermReviewWrite: true,
		PermAnalyticsRead: true,
	},
	userDomain.RoleSupport: {
		PermUserRead: true,
		PermProductRead: true,
		PermCategoryRead: true,
		PermOrderRead: true, PermOrderWrite: true,
		PermReviewRead: true,
	},
}

// HasPermission checks whether a role has a specific permission.
// Returns true if the role's permission set includes the requested permission.
func HasPermission(role userDomain.Role, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}
