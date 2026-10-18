package domain

import (
	"testing"

	userDomain "github.com/ecommerce/backend/internal/users/domain"
)

func TestHasPermission_SuperAdmin(t *testing.T) {
	// Super admin has all permissions
	perms := []Permission{
		PermUserRead, PermUserWrite,
		PermProductRead, PermProductWrite,
		PermCategoryRead, PermCategoryWrite,
		PermOrderRead, PermOrderWrite,
		PermInventoryRead, PermInventoryWrite,
		PermCouponRead, PermCouponWrite,
		PermReviewRead, PermReviewWrite,
		PermAnalyticsRead, PermAuditRead,
	}
	for _, p := range perms {
		if !HasPermission(userDomain.RoleSuperAdmin, p) {
			t.Errorf("SUPER_ADMIN should have %s", p)
		}
	}
}

func TestHasPermission_Admin(t *testing.T) {
	// Admin has all same permissions as super admin
	perms := []Permission{
		PermUserRead, PermUserWrite,
		PermProductRead, PermProductWrite,
		PermCouponRead, PermCouponWrite,
		PermAuditRead,
	}
	for _, p := range perms {
		if !HasPermission(userDomain.RoleAdmin, p) {
			t.Errorf("ADMIN should have %s", p)
		}
	}
}

func TestHasPermission_Manager_NoUserWrite(t *testing.T) {
	// Manager can read users but not write
	if !HasPermission(userDomain.RoleManager, PermUserRead) {
		t.Error("MANAGER should have PermUserRead")
	}
	if HasPermission(userDomain.RoleManager, PermUserWrite) {
		t.Error("MANAGER should NOT have PermUserWrite")
	}
}

func TestHasPermission_Manager_NoCouponWrite(t *testing.T) {
	// Manager cannot manage coupons
	if HasPermission(userDomain.RoleManager, PermCouponWrite) {
		t.Error("MANAGER should NOT have PermCouponWrite")
	}
	if HasPermission(userDomain.RoleManager, PermCouponRead) {
		t.Error("MANAGER should NOT have PermCouponRead")
	}
}

func TestHasPermission_Manager_NoAuditRead(t *testing.T) {
	// Manager cannot view audit logs
	if HasPermission(userDomain.RoleManager, PermAuditRead) {
		t.Error("MANAGER should NOT have PermAuditRead")
	}
}

func TestHasPermission_Support(t *testing.T) {
	// Support can read users, products, categories, orders, reviews
	if !HasPermission(userDomain.RoleSupport, PermUserRead) {
		t.Error("SUPPORT should have PermUserRead")
	}
	if !HasPermission(userDomain.RoleSupport, PermOrderWrite) {
		t.Error("SUPPORT should have PermOrderWrite")
	}
	// But cannot manage users, products, categories, or coupons
	if HasPermission(userDomain.RoleSupport, PermUserWrite) {
		t.Error("SUPPORT should NOT have PermUserWrite")
	}
	if HasPermission(userDomain.RoleSupport, PermProductWrite) {
		t.Error("SUPPORT should NOT have PermProductWrite")
	}
	if HasPermission(userDomain.RoleSupport, PermCouponRead) {
		t.Error("SUPPORT should NOT have PermCouponRead")
	}
}

func TestHasPermission_Customer_None(t *testing.T) {
	// Customer role has no admin permissions
	perms := []Permission{
		PermUserRead, PermUserWrite,
		PermProductRead, PermProductWrite,
		PermAuditRead, PermAnalyticsRead,
	}
	for _, p := range perms {
		if HasPermission(userDomain.RoleCustomer, p) {
			t.Errorf("CUSTOMER should NOT have %s", p)
		}
	}
}

func TestHasPermission_UnknownRole(t *testing.T) {
	// Unknown roles have no permissions
	if HasPermission(userDomain.Role("UNKNOWN"), PermUserRead) {
		t.Error("unknown role should have no permissions")
	}
}

func TestNewAuditLog(t *testing.T) {
	log := NewAuditLog("admin-1", "SUPER_ADMIN", AuditActionUserRoleUpdate, "user", "u1", "role changed")
	if log.ID == "" {
		t.Error("expected generated ID")
	}
	if log.ActorID != "admin-1" {
		t.Errorf("expected actor admin-1, got %s", log.ActorID)
	}
	if log.Action != AuditActionUserRoleUpdate {
		t.Errorf("expected action %s, got %s", AuditActionUserRoleUpdate, log.Action)
	}
	if log.ResourceType != "user" {
		t.Errorf("expected resource_type user, got %s", log.ResourceType)
	}
	if log.IPAddress != "" {
		t.Error("expected empty IP before SetRequestMetadata")
	}

	log.SetRequestMetadata("10.0.0.1", "test-agent")
	if log.IPAddress != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", log.IPAddress)
	}
	if log.UserAgent != "test-agent" {
		t.Errorf("expected agent test-agent, got %s", log.UserAgent)
	}
}

func TestSensitiveActions(t *testing.T) {
	// Verify all declared sensitive actions are marked
	if !SensitiveActions[AuditActionUserRoleUpdate] {
		t.Error("user.role.update should be sensitive")
	}
	if !SensitiveActions[AuditActionReviewApprove] {
		t.Error("review.approve should be sensitive")
	}
	if !SensitiveActions[AuditActionOrderStatusUpdate] {
		t.Error("order.status.update should be sensitive")
	}
	// Non-destructive actions should not be sensitive
	if SensitiveActions[AuditActionProductCreate] {
		t.Error("product.create should not be in SensitiveActions")
	}
	if SensitiveActions[AuditActionCategoryCreate] {
		t.Error("category.create should not be in SensitiveActions")
	}
}
