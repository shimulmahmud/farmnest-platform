package integration

import (
        "context"
        "encoding/json"
        "testing"
        "time"

        "go.mongodb.org/mongo-driver/v2/bson"

        payDomain "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/stretchr/testify/suite"
)

func TestIntegrationSuite(t *testing.T) {
        suite.Run(t, new(IntegrationSuite))
}

// ==================== Auth Tests ====================

type AuthTestSuite struct {
        IntegrationSuite
}

func TestAuthSuite(t *testing.T) {
        suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) TestRegisterAndLogin() {
        resp := s.POST("/api/v1/auth/register", map[string]string{
                "email":      "john@example.com",
                "password":   "Password1!abc",
                "first_name": "John",
                "last_name":  "Doe",
        }, "")
        s.Require().Equal(201, resp.StatusCode, "registration failed: %v", resp.Body)
        data := s.dataField(resp)
        tokens := data["tokens"].(map[string]interface{})
        user := data["user"].(map[string]interface{})
        s.NotEmpty(user["id"])
        s.NotEmpty(tokens["access_token"])
        s.NotEmpty(tokens["refresh_token"])
        s.Equal("john@example.com", user["email"])
        s.Equal("CUSTOMER", user["role"])

        // Login with same credentials
        loginResp := s.POST("/api/v1/auth/login", map[string]string{
                "email":    "john@example.com",
                "password": "Password1!abc",
        }, "")
        s.Require().Equal(200, loginResp.StatusCode)
        loginData := s.dataField(loginResp)
        s.Equal(user["id"], loginData["user"].(map[string]interface{})["id"])
}

func (s *AuthTestSuite) TestRegister_DuplicateEmail() {
        s.RegisterUser("dup@example.com", "Password1!abc", "A", "B")

        resp := s.POST("/api/v1/auth/register", map[string]string{
                "email":      "dup@example.com",
                "password":   "Password1!xyz",
                "first_name": "C",
                "last_name":  "D",
        }, "")
        s.Equal(409, resp.StatusCode)
        s.assertErrorCode(resp, "CONFLICT")
}

func (s *AuthTestSuite) TestRegister_WeakPassword() {
        resp := s.POST("/api/v1/auth/register", map[string]string{
                "email":      "weak@example.com",
                "password":   "aaaaaaaa",
                "first_name": "A",
                "last_name":  "B",
        }, "")
        s.Equal(422, resp.StatusCode)
        s.assertErrorCode(resp, "VALIDATION_ERROR")
}

func (s *AuthTestSuite) TestLogin_WrongPassword() {
        s.RegisterUser("user@example.com", "Password1!abc", "A", "B")

        resp := s.POST("/api/v1/auth/login", map[string]string{
                "email":    "user@example.com",
                "password": "WrongPassword1!",
        }, "")
        s.Equal(401, resp.StatusCode)
}

func (s *AuthTestSuite) TestRefreshToken() {
        auth := s.RegisterUser("refresh@example.com", "Password1!abc", "A", "B")

        // Use refresh token to get a new pair
        resp := s.POST("/api/v1/auth/refresh", map[string]string{
                "refresh_token": auth.RefreshToken,
        }, "")
        s.Equal(200, resp.StatusCode, "refresh failed: %v", resp.Body)
        data := s.dataField(resp)
        newToken := data["access_token"].(string)
        newRefresh := data["refresh_token"].(string)
        s.NotEmpty(newToken)
        s.NotEmpty(newRefresh)
        s.NotEqual(auth.AccessToken, newToken)
        s.NotEqual(auth.RefreshToken, newRefresh)
}

func (s *AuthTestSuite) TestRefreshToken_Reuse() {
        auth := s.RegisterUser("reuse@example.com", "Password1!abc", "A", "B")

        // First refresh succeeds
        resp1 := s.POST("/api/v1/auth/refresh", map[string]string{
                "refresh_token": auth.RefreshToken,
        }, "")
        s.Equal(200, resp1.StatusCode)

        // Second use of same token fails (revoked by rotation)
        resp2 := s.POST("/api/v1/auth/refresh", map[string]string{
                "refresh_token": auth.RefreshToken,
        }, "")
        s.Equal(401, resp2.StatusCode)
}

func (s *AuthTestSuite) TestLogout() {
        auth := s.RegisterUser("logout@example.com", "Password1!abc", "A", "B")

        resp := s.POST("/api/v1/auth/logout", map[string]string{
                "refresh_token": auth.RefreshToken,
        }, "")
        s.Equal(200, resp.StatusCode)

        // Attempting to use the token after logout
        resp2 := s.POST("/api/v1/auth/refresh", map[string]string{
                "refresh_token": auth.RefreshToken,
        }, "")
        s.Equal(401, resp2.StatusCode)
}

// ==================== Product Tests ====================

type ProductTestSuite struct {
        IntegrationSuite
        adminAuth *AuthResult
}

func TestProductSuite(t *testing.T) {
        suite.Run(t, new(ProductTestSuite))
}

func (s *ProductTestSuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.adminAuth = s.RegisterUser("admin@example.com", "AdminPass1!abc", "Admin", "User")
}

func (s *ProductTestSuite) TestCreateAndGetProduct() {
        resp := s.POST("/api/v1/products", map[string]interface{}{
                "name":             "Test Widget",
                "slug":             "test-widget",
                "short_description": "A test product",
                "description":      "A detailed test product description",
                "brand":            "TestBrand",
                "status":           "ACTIVE",
                "category_id":      "",
                "variants": []map[string]interface{}{
                        {"sku": "TW-001", "price": 1999, "stock": 50, "status": "ACTIVE"},
                },
                "tags": []string{"test", "widget"},
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "create product: %v", resp.Body)
        data := s.dataField(resp)
        productID := data["id"].(string)
        s.NotEmpty(productID)

        // Get product
        resp2 := s.GET("/api/v1/products/"+productID, "")
        s.Equal(200, resp2.StatusCode)
        product := s.dataField(resp2)
        s.Equal("Test Widget", product["name"])
        s.Equal("test-widget", product["slug"])
        s.Equal("ACTIVE", product["status"])
}

func (s *ProductTestSuite) TestSearchProducts() {
        for _, name := range []string{"Alpha Gadget", "Beta Widget"} {
                s.POST("/api/v1/products", map[string]interface{}{
                        "name":             name,
                        "slug":             "slug-" + name,
                        "short_description": "desc",
                        "description":      "detailed desc",
                        "brand":            "TestBrand",
                        "status":           "ACTIVE",
                        "category_id":      "",
                        "variants": []map[string]interface{}{{
                                "sku": "SKU-" + name, "price": 1000, "stock": 10, "status": "ACTIVE",
                        }},
                        "tags": []string{"search-test"},
                }, s.adminAuth.AccessToken)
        }

        resp := s.GET("/api/v1/products/search?query=Alpha", "")
        s.Equal(200, resp.StatusCode)
        data := s.dataField(resp)
        products := data["products"].([]interface{})
        s.Equal(1, len(products))
}

// ==================== Full E2E Flow Test ====================

type FlowTestSuite struct {
        IntegrationSuite
        customerAuth *AuthResult
        adminAuth    *AuthResult
}

func TestFlowSuite(t *testing.T) {
        suite.Run(t, new(FlowTestSuite))
}

func (s *FlowTestSuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.customerAuth = s.RegisterUser("customer@example.com", "CustomerPass1!abc", "Jane", "Smith")
        s.adminAuth = s.RegisterUser("admin@example.com", "AdminPass1!abc", "Admin", "User")
}

func (s *FlowTestSuite) TestFullE2EFlow() {
        // --- Step 1: Admin creates a product ---
        productResp := s.POST("/api/v1/products", map[string]interface{}{
                "name":             "Premium Headphones",
                "slug":             "premium-headphones",
                "short_description": "Noise-cancelling headphones",
                "description":      "Premium noise-cancelling over-ear headphones with 30h battery",
                "brand":            "AudioMax",
                "status":           "ACTIVE",
                "category_id":      "",
                "variants": []map[string]interface{}{{
                        "sku": "AM-PH-001", "price": 24999, "stock": 100, "status": "ACTIVE",
                }},
                "tags": []string{"audio", "headphones", "premium"},
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, productResp.StatusCode, "create product: %v", productResp.Body)
        productData := s.dataField(productResp)
        productID := productData["id"].(string)
        variants := productData["variants"].([]interface{})
        variantID := variants[0].(map[string]interface{})["id"].(string)

        // --- Step 2: Customer creates an order ---
        orderResp := s.POST("/api/v1/orders/", map[string]interface{}{
                "items": []map[string]interface{}{{
                        "product_id": productID,
                        "variant_id": variantID,
                        "name":       "Premium Headphones",
                        "sku":        "AM-PH-001",
                        "quantity":   2,
                        "unit_price": 24999,
                        "discount":   0,
                }},
                "shipping_address": map[string]string{"name": "Test User", "phone": "01712345678",
                        "line1": "123 Main St", "city": "New York", "state": "NY",
                        "postal_code": "10001", "country": "US",
                },
                "billing_address": map[string]string{
                        "line1": "123 Main St", "city": "New York", "state": "NY",
                        "postal_code": "10001", "country": "US",
                },
        }, s.customerAuth.AccessToken)
        s.Require().Equal(201, orderResp.StatusCode, "create order: %v", orderResp.Body)
        orderData := s.dataField(orderResp)
        orderID := orderData["id"].(string)
        s.Equal("PENDING", orderData["status"])
        pricing := orderData["pricing"].(map[string]interface{})
        s.Equal(float64(49998), pricing["total"])

        // --- Step 3: Admin transitions order to PAYMENT_PENDING ---
        resp := s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PAYMENT_PENDING",
                "reason": "Awaiting payment",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "PAYMENT_PENDING: %v", resp.Body)
        s.Equal("PAYMENT_PENDING", s.dataField(resp)["status"])

        // --- Step 4: Create a payment record (directly via repository) ---
        payment := payDomain.NewPayment(orderID, "fake", "USD", 49998)
        err := s.paymentRepo.Create(context.Background(), payment)
        s.Require().NoError(err)
        s.Equal(payDomain.PaymentPending, payment.Status)

        // Verify payment in MongoDB
        db := s.mongoClient.Database()
        var paymentDoc bson.M
        err = db.Collection("payments").FindOne(context.Background(), bson.M{"_id": payment.ID}).Decode(&paymentDoc)
        s.Require().NoError(err)
        s.Equal(49998, int(paymentDoc["amount"].(int32)))

        // --- Step 5: Simulate payment webhook ---
        webhookPayload, _ := json.Marshal(map[string]interface{}{
                "id":          "wh-evt-001",
                "type":        "payment.succeeded",
                "provider":    "fake",
                "payment_ref": "fake-pay-001",
                "amount":      49998,
                "currency":    "USD",
                "status":      "PAID",
                "timestamp":   "2026-01-01T00:00:00Z",
        })

        signature := s.webhookVerifier.SignPayload("fake", webhookPayload)

        webhookResp := s.POSTWebhook("/api/v1/webhooks/payment/fake", webhookPayload, map[string]string{
                "X-Webhook-Signature": signature,
                "X-Request-ID":       "test-webhook-001",
        })

        s.Require().Equal(200, webhookResp.StatusCode, "webhook failed: %v", webhookResp.Body)

        // --- Step 6: Verify order transitioned to PAID ---
        s.WaitEventually(5*time.Second, func() bool {
                resp := s.GET("/api/v1/orders/"+orderID, s.customerAuth.AccessToken)
                if resp.StatusCode != 200 {
                        return false
                }
                data := s.dataField(resp)
                return data["status"] == "PAID"
        }, "order should be PAID after webhook")

        // --- Step 7: Verify MongoDB state ---
        var orderDoc bson.M
        err = db.Collection("orders").FindOne(context.Background(), bson.M{"_id": orderID}).Decode(&orderDoc)
        s.Require().NoError(err)
        s.Equal("PAID", orderDoc["status"])

        // Status history
        history := orderDoc["status_history"].(bson.A)
        s.Require().GreaterOrEqual(len(history), 3, "expected at least 3 status transitions")

        // Payment status in MongoDB
        var paidDoc bson.M
        err = db.Collection("payments").FindOne(context.Background(), bson.M{"_id": payment.ID}).Decode(&paidDoc)
        s.Require().NoError(err)
        s.Equal("PAID", paidDoc["status"])
        s.Equal("fake-pay-001", paidDoc["provider_reference"])

        // --- Step 8: Verify outbox events were published ---
        outboxCount, _ := db.Collection("outbox").CountDocuments(context.Background(), bson.M{})
        s.Greater(outboxCount, int64(0), "outbox should have events")

        // --- Step 9: Verify audit log was written ---
        auditCount, _ := db.Collection("audit_logs").CountDocuments(context.Background(), bson.M{})
        s.Greater(auditCount, int64(0), "audit logs should have entries")

        // --- Step 10: Verify webhook idempotency ---
        webhookResp2 := s.POSTWebhook("/api/v1/webhooks/payment/fake", webhookPayload, map[string]string{
                "X-Webhook-Signature": signature,
                "X-Request-ID":       "test-webhook-001",
        })
        s.Equal(200, webhookResp2.StatusCode, "idempotent webhook should still be 200")

        // --- Step 11: Verify list orders ---
        listResp := s.GET("/api/v1/orders/?page=1&per_page=10", s.customerAuth.AccessToken)
        s.Equal(200, listResp.StatusCode)
        listData := s.dataField(listResp)
        orders := listData["data"].([]interface{})
        s.Equal(1, len(orders))
}

func (s *FlowTestSuite) TestOrderCreation_Unauthenticated() {
        resp := s.POST("/api/v1/orders/", map[string]interface{}{
                "items":            []interface{}{},
                "shipping_address": map[string]string{"name": "Test User", "phone": "01712345678","line1": "a", "city": "b", "state": "c", "postal_code": "d", "country": "e"},
        }, "")
        s.Equal(401, resp.StatusCode)
}

func (s *FlowTestSuite) TestWebhook_InvalidSignature() {
        resp := s.POSTWebhook("/api/v1/webhooks/payment/fake", []byte(`{"id":"x","type":"payment.succeeded","provider":"fake","payment_ref":"ref","amount":100,"currency":"USD","status":"PAID"}`), map[string]string{
                "X-Webhook-Signature": "sha256=invalidsignature",
                "X-Request-ID":       "test-req",
        })
        s.Equal(401, resp.StatusCode)
}

func (s *FlowTestSuite) TestWebhook_InvalidProvider() {
        resp := s.POSTWebhook("/api/v1/webhooks/payment/unknown", []byte(`{"id":"x","type":"payment.succeeded","provider":"unknown","payment_ref":"ref","amount":100,"currency":"USD","status":"PAID"}`), map[string]string{
                "X-Webhook-Signature": "sha256=anything",
                "X-Request-ID":       "test-req",
        })
        s.Equal(422, resp.StatusCode)
        s.assertErrorCode(resp, "VALIDATION_ERROR")
}
