package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ==================== Marketing Tests ====================

type MarketingTestSuite struct {
	IntegrationSuite
	adminAuth *AuthResult
}

func TestMarketingSuite(t *testing.T) {
	suite.Run(t, new(MarketingTestSuite))
}

func (s *MarketingTestSuite) SetupTest() {
	s.IntegrationSuite.SetupTest()
	s.adminAuth = s.RegisterUser("mkt-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

// --- Segment Tests ---

func (s *MarketingTestSuite) TestCreateAndGetSegment() {
	resp := s.POST("/api/v1/admin/marketing/segments", map[string]interface{}{
		"name":        "High Value Customers",
		"type":        "STATIC",
		"description": "Customers who spent more than $500",
		"rules": []map[string]interface{}{{
			"field":    "total_spent",
			"operator": "gte",
			"value":    50000,
		}},
	}, s.adminAuth.AccessToken)
	s.Require().Equal(201, resp.StatusCode, "create segment: %v", resp.Body)
	data := s.dataField(resp)
	segID := data["id"].(string)
	s.Equal("High Value Customers", data["name"])
	s.Equal("STATIC", data["type"])

	// Get by ID
	getResp := s.GET("/api/v1/admin/marketing/segments/"+segID, s.adminAuth.AccessToken)
	s.Equal(200, getResp.StatusCode)
	getSeg := s.dataField(getResp)
	s.Equal("High Value Customers", getSeg["name"])
}

func (s *MarketingTestSuite) TestListSegments() {
	for i := 0; i < 3; i++ {
		s.POST("/api/v1/admin/marketing/segments", map[string]interface{}{
			"name": "Segment " + string(rune('A'+i)),
			"type": "STATIC",
		}, s.adminAuth.AccessToken)
	}

	resp := s.GET("/api/v1/admin/marketing/segments?page=1&per_page=10", s.adminAuth.AccessToken)
	s.Equal(200, resp.StatusCode)
	data := s.dataField(resp)
	items := data["data"].([]interface{})
	s.Equal(3, len(items))
}

func (s *MarketingTestSuite) TestUpdateSegment() {
	createResp := s.POST("/api/v1/admin/marketing/segments", map[string]interface{}{
		"name": "Old Segment", "type": "DYNAMIC",
	}, s.adminAuth.AccessToken)
	segID := s.dataField(createResp)["id"].(string)

	resp := s.PUT("/api/v1/admin/marketing/segments/"+segID, map[string]interface{}{
		"name":        "Updated Segment",
		"description": "Updated description",
	}, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "update segment: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal("Updated Segment", data["name"])
}

func (s *MarketingTestSuite) TestDeleteSegment() {
	createResp := s.POST("/api/v1/admin/marketing/segments", map[string]interface{}{
		"name": "Delete Me", "type": "STATIC",
	}, s.adminAuth.AccessToken)
	segID := s.dataField(createResp)["id"].(string)

	resp := s.DELETE("/api/v1/admin/marketing/segments/"+segID, s.adminAuth.AccessToken)
	s.Equal(200, resp.StatusCode)

	// Verify deleted
	getResp := s.GET("/api/v1/admin/marketing/segments/"+segID, s.adminAuth.AccessToken)
	s.Equal(404, getResp.StatusCode)
}

// --- Campaign Tests ---

func (s *MarketingTestSuite) TestCreateCampaign() {
	// Create segment first
	segResp := s.POST("/api/v1/admin/marketing/segments", map[string]interface{}{
		"name": "Campaign Seg", "type": "STATIC",
	}, s.adminAuth.AccessToken)
	segID := s.dataField(segResp)["id"].(string)

	resp := s.POST("/api/v1/admin/marketing/campaigns", map[string]interface{}{
		"name": "Summer Sale Campaign",
		"type": "ABANDONED_CART",
		"description": "Recover abandoned carts",
		"rules": map[string]interface{}{
			"segment_ids":           []string{segID},
			"event_types":           []string{"cart.abandoned"},
			"min_abandoned_minutes": 30,
			"max_emails_per_user":    3,
		},
	}, s.adminAuth.AccessToken)
	s.Require().Equal(201, resp.StatusCode, "create campaign: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal("Summer Sale Campaign", data["name"])
	s.Equal("ABANDONED_CART", data["type"])
	s.Equal("DRAFT", data["status"])
	s.NotEmpty(data["id"])

	rules := data["rules"].(map[string]interface{})
	s.Equal(1, len(rules["segment_ids"].([]interface{})))
	s.Equal(30, int(rules["min_abandoned_minutes"].(float64)))
}

func (s *MarketingTestSuite) TestTransitionCampaign() {
	createResp := s.POST("/api/v1/admin/marketing/campaigns", map[string]interface{}{
		"name": "Transition Test", "type": "ABANDONED_CART",
	}, s.adminAuth.AccessToken)
	campaignID := s.dataField(createResp)["id"].(string)

	// DRAFT -> ACTIVE
	resp := s.PATCH("/api/v1/admin/marketing/campaigns/"+campaignID+"/status", map[string]string{
		"status": "ACTIVE",
	}, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "activate campaign: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal("ACTIVE", data["status"])

	// ACTIVE -> ARCHIVED
	resp2 := s.PATCH("/api/v1/admin/marketing/campaigns/"+campaignID+"/status", map[string]string{
		"status": "ARCHIVED",
	}, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp2.StatusCode, "archive campaign: %v", resp2.Body)
	data2 := s.dataField(resp2)
	s.Equal("ARCHIVED", data2["status"])
}

func (s *MarketingTestSuite) TestListCampaigns() {
	for i := 0; i < 2; i++ {
		s.POST("/api/v1/admin/marketing/campaigns", map[string]interface{}{
			"name": "Campaign " + string(rune('A'+i)), "type": "ABANDONED_CART",
		}, s.adminAuth.AccessToken)
	}

	resp := s.GET("/api/v1/admin/marketing/campaigns?page=1&per_page=10", s.adminAuth.AccessToken)
	s.Equal(200, resp.StatusCode)
	data := s.dataField(resp)
	items := data["data"].([]interface{})
	s.Equal(2, len(items))
}

// --- Promotion Tests ---

func (s *MarketingTestSuite) TestCreatePromotion() {
	resp := s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
		"name":             "10% Off Electronics",
		"code":             "SAVE10",
		"description":      "Get 10% off all electronics",
		"discount_type":    "PERCENTAGE",
		"discount_value":   10,
		"min_order_cents":  5000,
		"max_discount_cents": 10000,
		"usage_limit":      100,
	}, s.adminAuth.AccessToken)
	s.Require().Equal(201, resp.StatusCode, "create promotion: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal("10% Off Electronics", data["name"])
	s.Equal("SAVE10", data["code"])
	s.Equal("PERCENTAGE", data["discount_type"])
	s.Equal(10, int(data["discount_value"].(float64)))
	s.Equal("DRAFT", data["status"])
	s.Equal(0, int(data["usage_count"].(float64)))
	s.NotEmpty(data["id"])
}

func (s *MarketingTestSuite) TestGetPromotionByCode() {
	s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
		"name":           "Code Lookup Promo",
		"code":           "LOOKUP20",
		"discount_type":  "PERCENTAGE",
		"discount_value": 20,
	}, s.adminAuth.AccessToken)

	resp := s.GET("/api/v1/admin/marketing/promotions/code/LOOKUP20", s.adminAuth.AccessToken)
	s.Equal(200, resp.StatusCode)
	data := s.dataField(resp)
	s.Equal("LOOKUP20", data["code"])
	s.Equal("DRAFT", data["status"])
}

func (s *MarketingTestSuite) TestTransitionPromotion() {
	createResp := s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
		"name":           "Transition Promo",
		"code":           "TRANS15",
		"discount_type":  "PERCENTAGE",
		"discount_value": 15,
	}, s.adminAuth.AccessToken)
	promoID := s.dataField(createResp)["id"].(string)

	// DRAFT -> ACTIVE
	resp := s.PATCH("/api/v1/admin/marketing/promotions/"+promoID+"/status", map[string]string{
		"status": "ACTIVE",
	}, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "activate promotion: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal("ACTIVE", data["status"])

	// ACTIVE -> ARCHIVED
	resp2 := s.PATCH("/api/v1/admin/marketing/promotions/"+promoID+"/status", map[string]string{
		"status": "ARCHIVED",
	}, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp2.StatusCode, "archive promotion: %v", resp2.Body)
	data2 := s.dataField(resp2)
	s.Equal("ARCHIVED", data2["status"])
}

func (s *MarketingTestSuite) TestUpdatePromotion() {
	createResp := s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
		"name":           "Original Promo",
		"code":           "ORIG25",
		"discount_type":  "PERCENTAGE",
		"discount_value": 25,
	}, s.adminAuth.AccessToken)
	promoID := s.dataField(createResp)["id"].(string)

	resp := s.PUT("/api/v1/admin/marketing/promotions/"+promoID, map[string]interface{}{
		"name":           "Updated Promo",
		"code":           "ORIG25",
		"discount_type":  "FIXED",
		"discount_value": 500,
		"description":     "Now $5 off",
	}, s.adminAuth.AccessToken)
	s.Require().Equal(200, resp.StatusCode, "update promotion: %v", resp.Body)
	data := s.dataField(resp)
	s.Equal("Updated Promo", data["name"])
	s.Equal("FIXED", data["discount_type"])
	s.Equal(500, int(data["discount_value"].(float64)))
}

func (s *MarketingTestSuite) TestMarketing_RequiresAdmin() {
	userAuth := s.RegisterUser("mkt-user@example.com", "Pass1!abc", "Regular", "User")

	resp := s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
		"name": "Forbidden", "code": "NOPE",
	}, userAuth.AccessToken)
	s.Equal(403, resp.StatusCode)
}
