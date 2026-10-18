package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ==================== Wishlist Tests ====================

type WishlistTestSuite struct {
	IntegrationSuite
	customerAuth *AuthResult
	adminAuth    *AuthResult
}

func TestWishlistSuite(t *testing.T) {
	suite.Run(t, new(WishlistTestSuite))
}

func (s *WishlistTestSuite) SetupTest() {
	s.IntegrationSuite.SetupTest()
	s.customerAuth = s.RegisterUser("wl-customer@example.com", "CustomerPass1!abc", "Jane", "Smith")
	s.adminAuth = s.RegisterUser("wl-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

func (s *WishlistTestSuite) createTestProductForWishlist(name, slug string) string {
	resp := s.POST("/api/v1/products", map[string]interface{}{
		"name":              name,
		"slug":              slug,
		"short_description": "test product for wishlist",
		"description":       "detailed description",
		"brand":             "TestBrand",
		"status":            "ACTIVE",
		"category_id":      "",
		"variants": []map[string]interface{}{{
			"sku": "SKU-" + slug, "price": 1999, "stock": 50, "status": "ACTIVE",
		}},
		"tags": []string{"wishlist-test"},
	}, s.adminAuth.AccessToken)
	s.Require().Equal(201, resp.StatusCode, "create product %s: %v", name, resp.Body)
	return s.dataField(resp)["id"].(string)
}

func (s *WishlistTestSuite) TestAddAndGetWishlist() {
	productID := s.createTestProductForWishlist("Wishlist Item 1", "wl-item-1")

	// Add item
	resp := s.POST("/api/v1/wishlist/items", map[string]string{
		"product_id": productID,
	}, s.customerAuth.AccessToken)
	s.Require().Equal(201, resp.StatusCode, "add wishlist item: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal(productID, data["product_id"])
	s.NotEmpty(data["id"])

	// Get wishlist
	getResp := s.GET("/api/v1/wishlist/", s.customerAuth.AccessToken)
	s.Equal(200, getResp.StatusCode)
	wlData := s.dataField(getResp)
	s.Equal(s.customerAuth.UserID, wlData["customer_id"])
	items := wlData["items"].([]interface{})
	s.Equal(1, len(items))
	s.Equal(productID, items[0].(map[string]interface{})["product_id"])
}

func (s *WishlistTestSuite) TestAddDuplicateItem() {
	productID := s.createTestProductForWishlist("Dup WL Item", "dup-wl-item")

	s.POST("/api/v1/wishlist/items", map[string]string{"product_id": productID}, s.customerAuth.AccessToken)

	// Second add should fail
	resp := s.POST("/api/v1/wishlist/items", map[string]string{"product_id": productID}, s.customerAuth.AccessToken)
	s.Equal(409, resp.StatusCode)
	s.assertErrorCode(resp, "CONFLICT")
}

func (s *WishlistTestSuite) TestRemoveItem() {
	productID := s.createTestProductForWishlist("Removable Item", "removable-item")

	addResp := s.POST("/api/v1/wishlist/items", map[string]string{"product_id": productID}, s.customerAuth.AccessToken)
	itemID := s.dataField(addResp)["id"].(string)

	// Remove
	delResp := s.DELETE("/api/v1/wishlist/items/"+itemID, s.customerAuth.AccessToken)
	s.Equal(200, delResp.StatusCode)

	// Verify empty wishlist
	getResp := s.GET("/api/v1/wishlist/", s.customerAuth.AccessToken)
	wlData := s.dataField(getResp)
	items := wlData["items"].([]interface{})
	s.Equal(0, len(items))
}

func (s *WishlistTestSuite) TestWishlistIsolation() {
	// Two different customers have separate wishlists
	otherCustomer := s.RegisterUser("wl-other@example.com", "Pass1!abc", "Other", "User")
	productID := s.createTestProductForWishlist("Isolation Item", "isolation-item")

	// Customer 1 adds item
	s.POST("/api/v1/wishlist/items", map[string]string{"product_id": productID}, s.customerAuth.AccessToken)

	// Customer 2 should have empty wishlist
	resp := s.GET("/api/v1/wishlist/", otherCustomer.AccessToken)
	s.Equal(200, resp.StatusCode)
	wlData := s.dataField(resp)
	items := wlData["items"].([]interface{})
	s.Equal(0, len(items))
}

func (s *WishlistTestSuite) TestWishlist_RequiresAuth() {
	resp := s.GET("/api/v1/wishlist/", "")
	s.Equal(401, resp.StatusCode)
}
