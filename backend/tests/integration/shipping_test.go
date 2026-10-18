package integration

import (
        "testing"

        "github.com/stretchr/testify/suite"
)

// ==================== Shipping Tests ====================

type ShippingTestSuite struct {
        IntegrationSuite
        customerAuth *AuthResult
        adminAuth    *AuthResult
}

func TestShippingSuite(t *testing.T) {
        suite.Run(t, new(ShippingTestSuite))
}

func (s *ShippingTestSuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.customerAuth = s.RegisterUser("ship-customer@example.com", "CustomerPass1!abc", "Jane", "Smith")
        s.adminAuth = s.RegisterUser("ship-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

func (s *ShippingTestSuite) TestCalculateRate() {
        resp := s.POST("/api/v1/shipping/rates", map[string]interface{}{
                "destination": map[string]string{
                        "line1": "456 Ship Ave", "city": "Portland", "state": "OR",
                        "postal_code": "97201", "country": "US",
                },
                "packages": []map[string]interface{}{{
                        "package_id":   "pkg-001",
                        "weight_grams": 500,
                        "length_cm":    20,
                        "width_cm":     15,
                        "height_cm":    10,
                }},
                "currency": "USD",
        }, s.customerAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "calculate rate: %v", resp.Body)
        // Response data is an array of rates, not a map
        s.assertSuccess(resp)
        rates := resp.Body["data"].([]interface{})
        s.GreaterOrEqual(len(rates), 1)
        firstRate := rates[0].(map[string]interface{})
        s.NotEmpty(firstRate["method"])
        s.NotEmpty(firstRate["carrier"])
}

func (s *ShippingTestSuite) TestAdminCreateShipment() {
        // Create an order first
        prodResp := s.POST("/api/v1/products", map[string]interface{}{
                "name":              "Ship Product",
                "slug":              "ship-product",
                "short_description": "For shipping tests",
                "description":       "Detailed description",
                "brand":             "TestBrand",
                "status":            "ACTIVE",
                "category_id":      "",
                "variants": []map[string]interface{}{{
                        "sku": "SP-001", "price": 3000, "stock": 50, "status": "ACTIVE",
                }},
                "tags": []string{"shipping-test"},
        }, s.adminAuth.AccessToken)
        prodData := s.dataField(prodResp)
        productID := prodData["id"].(string)
        variantID := prodData["variants"].([]interface{})[0].(map[string]interface{})["id"].(string)

        orderResp := s.POST("/api/v1/orders/", map[string]interface{}{
                "items": []map[string]interface{}{{
                        "product_id": productID, "variant_id": variantID,
                        "name": "Ship Product", "sku": "SP-001",
                        "quantity": 1, "unit_price": 3000,
                }},
                "shipping_address": map[string]string{"name": "Test User", "phone": "01712345678",
                        "line1": "789 Ship Blvd", "city": "Denver", "state": "CO",
                        "postal_code": "80201", "country": "US",
                },
                "billing_address": map[string]string{
                        "line1": "789 Ship Blvd", "city": "Denver", "state": "CO",
                        "postal_code": "80201", "country": "US",
                },
        }, s.customerAuth.AccessToken)
        orderData := s.dataField(orderResp)
        orderID := orderData["id"].(string)

        // Admin creates shipment
        resp := s.POST("/api/v1/admin/shipments/", map[string]interface{}{
                "order_id":  orderID,
                "customer_id": s.customerAuth.UserID,
                "method":    "STANDARD",
                "carrier":   "test-carrier",
                "destination": map[string]string{
                        "line1": "789 Ship Blvd", "city": "Denver", "state": "CO",
                        "postal_code": "80201", "country": "US",
                },
                "packages": []map[string]interface{}{{
                        "package_id":   "ship-pkg-001",
                        "weight_grams": 300,
                        "length_cm":    15,
                        "width_cm":     10,
                        "height_cm":    8,
                }},
                "currency": "USD",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "create shipment: %v", resp.Body)
        data := s.dataField(resp)
        s.Equal(orderID, data["order_id"])
        s.Equal("STANDARD", data["method"])
        s.Equal("test-carrier", data["carrier"])
        s.Equal("PENDING", data["status"])
        s.NotEmpty(data["id"])

        shipmentID := data["id"].(string)

        // Update shipment status
        statusResp := s.PATCH("/api/v1/admin/shipments/"+shipmentID+"/status", map[string]string{
                "status": "SHIPPED",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, statusResp.StatusCode, "update shipment status: %v", statusResp.Body)
        statusData := s.dataField(statusResp)
        s.Equal("SHIPPED", statusData["status"])

        // Customer can get shipment
        getResp := s.GET("/api/v1/shipments/"+shipmentID, s.customerAuth.AccessToken)
        s.Equal(200, getResp.StatusCode)
        getShipment := s.dataField(getResp)
        s.Equal("SHIPPED", getShipment["status"])
}

func (s *ShippingTestSuite) TestShipping_RequiresAuth() {
        resp := s.POST("/api/v1/shipping/rates", map[string]interface{}{
                "destination": map[string]string{
                        "line1": "a", "city": "b", "state": "c",
                        "postal_code": "d", "country": "e",
                },
                "packages": []map[string]interface{}{{
                        "package_id": "p", "weight_grams": 100,
                        "length_cm": 10, "width_cm": 10, "height_cm": 10,
                }},
        }, "")
        s.Equal(401, resp.StatusCode)
}
