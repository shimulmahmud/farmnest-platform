package integration

import (
        "testing"

        "github.com/stretchr/testify/suite"
)

// ==================== Category Tests ====================

type CategoryTestSuite struct {
        IntegrationSuite
        adminAuth *AuthResult
}

func TestCategorySuite(t *testing.T) {
        suite.Run(t, new(CategoryTestSuite))
}

func (s *CategoryTestSuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.adminAuth = s.RegisterUser("cat-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

func (s *CategoryTestSuite) TestCreateAndGetCategory() {
        resp := s.POST("/api/v1/admin/categories/", map[string]interface{}{
                "name":        "Electronics",
                "slug":        "electronics",
                "description": "Electronic devices and accessories",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "create category: %v", resp.Body)
        data := s.dataField(resp)
        catID := data["id"].(string)
        s.Equal("Electronics", data["name"])
        s.Equal("electronics", data["slug"])
        s.Equal("ACTIVE", data["status"])

        // Get by slug (public)
        getResp := s.GET("/api/v1/categories/electronics", "")
        s.Equal(200, getResp.StatusCode)
        getCat := s.dataField(getResp)
        s.Equal("Electronics", getCat["name"])
        s.Equal("electronics", getCat["slug"])

        // Get by ID (admin)
        getResp2 := s.GET("/api/v1/admin/categories/"+catID, s.adminAuth.AccessToken)
        s.Equal(200, getResp2.StatusCode)
        s.dataField(getResp2)
}

func (s *CategoryTestSuite) TestListCategories() {
        for _, name := range []string{"Books", "Clothing", "Home"} {
                s.POST("/api/v1/admin/categories/", map[string]interface{}{
                        "name": name,
                        "slug": name,
                }, s.adminAuth.AccessToken)
        }

        resp := s.GET("/api/v1/categories/?page=1&per_page=10", "")
        s.Equal(200, resp.StatusCode)
        data := s.dataField(resp)
        items := data["data"].([]interface{})
        s.Equal(3, len(items))
}

func (s *CategoryTestSuite) TestUpdateCategory() {
        createResp := s.POST("/api/v1/admin/categories/", map[string]interface{}{
                "name": "Old Name",
                "slug": "old-name",
        }, s.adminAuth.AccessToken)
        catID := s.dataField(createResp)["id"].(string)

        updateResp := s.PATCH("/api/v1/admin/categories/"+catID, map[string]interface{}{
                "name":        "New Name",
                "description": "Updated description",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, updateResp.StatusCode, "update category: %v", updateResp.Body)
        data := s.dataField(updateResp)
        s.Equal("New Name", data["name"])
}

func (s *CategoryTestSuite) TestDeleteCategory() {
        createResp := s.POST("/api/v1/admin/categories/", map[string]interface{}{
                "name": "To Delete",
                "slug": "to-delete",
        }, s.adminAuth.AccessToken)
        catID := s.dataField(createResp)["id"].(string)

        delResp := s.DELETE("/api/v1/admin/categories/"+catID, s.adminAuth.AccessToken)
        s.Equal(200, delResp.StatusCode)

        // Verify deleted
        getResp := s.GET("/api/v1/categories/to-delete", "")
        s.Equal(404, getResp.StatusCode)
}

func (s *CategoryTestSuite) TestCategory_RequiresAdmin() {
        // Non-admin user
        userAuth := s.RegisterUser("cat-user@example.com", "UserPass1!abc", "Regular", "User")

        resp := s.POST("/api/v1/admin/categories/", map[string]interface{}{
                "name": "Forbidden", "slug": "forbidden",
        }, userAuth.AccessToken)
        s.Equal(403, resp.StatusCode)
}
