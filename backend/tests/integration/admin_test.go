package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ==================== Admin RBAC Tests ====================

type AdminTestSuite struct {
	IntegrationSuite
	customerAuth *AuthResult
	adminAuth    *AuthResult
}

func TestAdminSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}

func (s *AdminTestSuite) SetupTest() {
	s.IntegrationSuite.SetupTest()
	s.customerAuth = s.RegisterUser("admin-cust@example.com", "CustomerPass1!abc", "Jane", "Smith")
	s.adminAuth = s.RegisterUser("admin-usr@example.com", "AdminPass1!abc", "Admin", "User")
}

func (s *AdminTestSuite) TestDashboardCounts() {
	resp := s.GET("/api/v1/admin/dashboard/counts", s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "dashboard counts: %v", resp.Body)
	data := s.dataField(resp)
	s.NotNil(data["total_users"])
	s.NotNil(data["total_orders"])
	s.NotNil(data["total_products"])
	s.NotNil(data["total_revenue"])
}

func (s *AdminTestSuite) TestDashboard_ForbiddenForCustomer() {
	resp := s.GET("/api/v1/admin/dashboard/counts", s.customerAuth.AccessToken)
	s.Equal(403, resp.StatusCode)
}

func (s *AdminTestSuite) TestListUsers() {
	// Create a few more users
	s.RegisterUser("listed1@example.com", "Pass1!abc", "Listed", "One")
	s.RegisterUser("listed2@example.com", "Pass1!abc", "Listed", "Two")

	resp := s.GET("/api/v1/admin/users?page=1&per_page=10", s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "list users: %v", resp.Body)
	data := s.dataField(resp)
	users := data["data"].([]interface{})
	// At least the 3 users (admin + 2 listed)
	s.GreaterOrEqual(len(users), 3)
}

func (s *AdminTestSuite) TestGetUser() {
	resp := s.GET("/api/v1/admin/users/"+s.customerAuth.UserID, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "get user: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal(s.customerAuth.UserID, data["id"])
	s.Equal("admin-cust@example.com", data["email"])
}

func (s *AdminTestSuite) TestUpdateUserRole() {
	user := s.RegisterUser("role-update@example.com", "Pass1!abc", "Role", "Update")

	resp := s.PATCH("/api/v1/admin/users/"+user.UserID+"/role", map[string]string{
		"role": "MANAGER",
	}, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "update role: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal("MANAGER", data["role"])
}

func (s *AdminTestSuite) TestListCustomers() {
	resp := s.GET("/api/v1/admin/customers?page=1&per_page=10", s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "list customers: %v", resp.Body)
	data := s.dataField(resp)
	customers := data["data"].([]interface{})
	s.GreaterOrEqual(len(customers), 1)
}

func (s *AdminTestSuite) TestListAuditLogs() {
	// Perform an action that generates an audit log (create a product)
	s.POST("/api/v1/products", map[string]interface{}{
		"name":              "Audit Test Product",
		"slug":              "audit-test-product",
		"short_description": "For audit testing",
		"description":       "Detailed description",
		"brand":             "TestBrand",
		"status":            "ACTIVE",
		"category_id":      "",
		"variants": []map[string]interface{}{{
			"sku": "AT-001", "price": 1000, "stock": 10, "status": "ACTIVE",
		}},
		"tags": []string{"audit-test"},
	}, s.adminAuth.AccessToken)

	resp := s.GET("/api/v1/admin/audit-logs?page=1&per_page=10", s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "list audit logs: %v", resp.Body)
	data := s.dataField(resp)
	logs := data["data"].([]interface{})
	s.GreaterOrEqual(len(logs), 1)

	// Check audit log structure
	firstLog := logs[0].(map[string]interface{})
	s.NotEmpty(firstLog["id"])
	s.NotEmpty(firstLog["action"])
	s.NotEmpty(firstLog["actor_id"])
	s.NotEmpty(firstLog["timestamp"])
}

func (s *AdminTestSuite) TestAdmin_Unauthenticated() {
	resp := s.GET("/api/v1/admin/dashboard/counts", "")
	s.Equal(401, resp.StatusCode)
}
