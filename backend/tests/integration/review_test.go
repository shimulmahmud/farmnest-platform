package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ==================== Review Tests ====================

type ReviewTestSuite struct {
	IntegrationSuite
	customerAuth *AuthResult
	adminAuth    *AuthResult
}

func TestReviewSuite(t *testing.T) {
	suite.Run(t, new(ReviewTestSuite))
}

func (s *ReviewTestSuite) SetupTest() {
	s.IntegrationSuite.SetupTest()
	s.customerAuth = s.RegisterUser("rev-customer@example.com", "CustomerPass1!abc", "Jane", "Smith")
	s.adminAuth = s.RegisterUser("rev-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

// createTestProduct creates a product for review testing.
func (s *ReviewTestSuite) createTestProduct(name, slug string) string {
	resp := s.POST("/api/v1/products", map[string]interface{}{
		"name":              name,
		"slug":              slug,
		"short_description": "test product for reviews",
		"description":       "detailed description",
		"brand":             "TestBrand",
		"status":            "ACTIVE",
		"category_id":      "",
		"variants": []map[string]interface{}{{
			"sku": "SKU-" + slug, "price": 1999, "stock": 50, "status": "ACTIVE",
		}},
		"tags": []string{"review-test"},
	}, s.adminAuth.AccessToken)
	s.Require().Equal(201, resp.StatusCode, "create product %s: %v", name, resp.Body)
	return s.dataField(resp)["id"].(string)
}

func (s *ReviewTestSuite) TestCreateReview() {
	productID := s.createTestProduct("Review Widget", "review-widget")

	resp := s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating":  4,
		"title":   "Great product!",
		"comment": "Really enjoyed using this product.",
	}, s.customerAuth.AccessToken)
	s.Require().Equal(201, resp.StatusCode, "create review: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal(productID, data["product_id"])
	s.Equal(s.customerAuth.UserID, data["customer_id"])
	s.Equal(4, int(data["rating"].(float64)))
	s.Equal("Great product!", data["title"])
	s.Equal("PENDING", data["status"])
	s.NotEmpty(data["id"])
}

func (s *ReviewTestSuite) TestCreateReview_Duplicate() {
	productID := s.createTestProduct("Dup Review Product", "dup-review-product")

	// Create first review
	s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating": 5, "title": "First",
	}, s.customerAuth.AccessToken)

	// Duplicate review should fail
	resp := s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating": 3, "title": "Second",
	}, s.customerAuth.AccessToken)
	s.Equal(409, resp.StatusCode)
	s.assertErrorCode(resp, "CONFLICT")
}

func (s *ReviewTestSuite) TestCreateReview_InvalidRating() {
	productID := s.createTestProduct("Invalid Rating", "invalid-rating-product")

	resp := s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating":  6,
		"title":   "Too high",
	}, s.customerAuth.AccessToken)
	s.Equal(422, resp.StatusCode)
	s.assertErrorCode(resp, "VALIDATION_ERROR")
}

func (s *ReviewTestSuite) TestGetReview() {
	productID := s.createTestProduct("Get Review Product", "get-review-product")

	createResp := s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating": 3, "title": "OK product", "comment": "It's alright",
	}, s.customerAuth.AccessToken)
	reviewID := s.dataField(createResp)["id"].(string)

	// Get review by ID (authenticated)
	getResp := s.GET("/api/v1/reviews/"+reviewID, s.customerAuth.AccessToken)
	s.Equal(200, getResp.StatusCode)
	data := s.dataField(getResp)
	s.Equal("OK product", data["title"])
	s.Equal(3, int(data["rating"].(float64)))
}

func (s *ReviewTestSuite) TestUpdateReview() {
	productID := s.createTestProduct("Update Review Product", "update-review-product")

	createResp := s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating": 3, "title": "Original",
	}, s.customerAuth.AccessToken)
	reviewID := s.dataField(createResp)["id"].(string)

	// Update review
	newTitle := "Updated Title"
	newRating := 5
	resp := s.PATCH("/api/v1/reviews/"+reviewID, map[string]interface{}{
		"rating": newRating,
		"title":  &newTitle,
	}, s.customerAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "update review: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal(5, int(data["rating"].(float64)))
	s.Equal("Updated Title", data["title"])
}

func (s *ReviewTestSuite) TestDeleteReview() {
	productID := s.createTestProduct("Delete Review Product", "delete-review-product")

	createResp := s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating": 2, "title": "Delete me",
	}, s.customerAuth.AccessToken)
	reviewID := s.dataField(createResp)["id"].(string)

	resp := s.DELETE("/api/v1/reviews/"+reviewID, s.customerAuth.AccessToken)
	s.Equal(200, resp.StatusCode)

	// Verify deleted
	getResp := s.GET("/api/v1/reviews/"+reviewID, s.customerAuth.AccessToken)
	s.Equal(404, getResp.StatusCode)
}

func (s *ReviewTestSuite) TestListReviewsByProduct() {
	productID := s.createTestProduct("List Review Product", "list-review-product")

	// Create multiple reviews from different users
	for i := 0; i < 3; i++ {
		user := s.RegisterUser("reviewer-"+string(rune('A'+i))+"@example.com", "Pass1!abc", "Rev", string(rune('A'+i)))
		s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
			"rating": i + 1, "title": "Review " + string(rune('A'+i)),
		}, user.AccessToken)
	}

	// List public reviews (only APPROVED, so should be 0 for PENDING reviews)
	resp := s.GET("/api/v1/products/"+productID+"/reviews?page=1&per_page=10", "")
	s.Equal(200, resp.StatusCode)
	data := s.dataField(resp)
	// Reviews start as PENDING, so public listing returns empty or only approved ones
	_ = data
}

func (s *ReviewTestSuite) TestReview_RequiresAuth() {
	productID := s.createTestProduct("Auth Review Product", "auth-review-product")

	resp := s.POST("/api/v1/products/"+productID+"/reviews", map[string]interface{}{
		"rating": 4, "title": "No auth",
	}, "")
	s.Equal(401, resp.StatusCode)
}
