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

// ==================== Order Lifecycle Tests ====================

type OrderLifecycleTestSuite struct {
        IntegrationSuite
        customerAuth *AuthResult
        adminAuth    *AuthResult
}

func TestOrderLifecycleSuite(t *testing.T) {
        suite.Run(t, new(OrderLifecycleTestSuite))
}

func (s *OrderLifecycleTestSuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.customerAuth = s.RegisterUser("ol-customer@example.com", "CustomerPass1!abc", "Jane", "Smith")
        s.adminAuth = s.RegisterUser("ol-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

func (s *OrderLifecycleTestSuite) createOrderWithProduct() (string, string) {
        // Create product
        prodResp := s.POST("/api/v1/products", map[string]interface{}{
                "name":              "Lifecycle Product",
                "slug":              "lifecycle-product",
                "short_description": "For lifecycle testing",
                "description":       "Detailed description",
                "brand":             "TestBrand",
                "status":            "ACTIVE",
                "category_id":      "",
                "variants": []map[string]interface{}{{
                        "sku": "LP-001", "price": 5000, "stock": 100, "status": "ACTIVE",
                }},
                "tags": []string{"lifecycle"},
        }, s.adminAuth.AccessToken)
        prodData := s.dataField(prodResp)
        productID := prodData["id"].(string)
        variants := prodData["variants"].([]interface{})
        variantID := variants[0].(map[string]interface{})["id"].(string)

        // Create order
        orderResp := s.POST("/api/v1/orders/", map[string]interface{}{
                "items": []map[string]interface{}{{
                        "product_id": productID,
                        "variant_id": variantID,
                        "name":       "Lifecycle Product",
                        "sku":        "LP-001",
                        "quantity":   3,
                        "unit_price": 5000,
                        "discount":   0,
                }},
                "shipping_address": map[string]string{"name": "Test User", "phone": "01712345678",
                        "line1": "100 Order St", "city": "Austin", "state": "TX",
                        "postal_code": "73301", "country": "US",
                },
                "billing_address": map[string]string{
                        "line1": "100 Order St", "city": "Austin", "state": "TX",
                        "postal_code": "73301", "country": "US",
                },
        }, s.customerAuth.AccessToken)
        s.Require().Equal(201, orderResp.StatusCode, "create order: %v", orderResp.Body)
        orderData := s.dataField(orderResp)
        return orderData["id"].(string), orderData["order_number"].(string)
}

func (s *OrderLifecycleTestSuite) TestOrderStatusTransitions() {
        orderID, _ := s.createOrderWithProduct()

        // Verify initial state
        getResp := s.GET("/api/v1/orders/"+orderID, s.customerAuth.AccessToken)
        orderData := s.dataField(getResp)
        s.Equal("PENDING", orderData["status"])

        // PENDING -> PAYMENT_PENDING
        resp := s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PAYMENT_PENDING", "reason": "Awaiting payment",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "PAYMENT_PENDING: %v", resp.Body)
        s.Equal("PAYMENT_PENDING", s.dataField(resp)["status"])

        // PAYMENT_PENDING -> PAID (via webhook in real flow, but admin can also trigger)
        resp = s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PAID", "reason": "Payment confirmed",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "PAID: %v", resp.Body)
        s.Equal("PAID", s.dataField(resp)["status"])

        // PAID -> PROCESSING
        resp = s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PROCESSING", "reason": "Processing order",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "PROCESSING: %v", resp.Body)
        s.Equal("PROCESSING", s.dataField(resp)["status"])

        // PROCESSING -> SHIPPED
        resp = s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status":            "SHIPPED",
                "reason":           "Shipped via carrier",
                "shipment_reference": "SHIP-001",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "SHIPPED: %v", resp.Body)
        shipData := s.dataField(resp)
        s.Equal("SHIPPED", shipData["status"])
        s.Equal("SHIP-001", shipData["shipment_reference"])

        // SHIPPED -> DELIVERED
        resp = s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "DELIVERED", "reason": "Delivered to customer",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "DELIVERED: %v", resp.Body)
        s.Equal("DELIVERED", s.dataField(resp)["status"])

        // Verify full status history
        finalResp := s.GET("/api/v1/orders/"+orderID, s.customerAuth.AccessToken)
        finalData := s.dataField(finalResp)
        history := finalData["status_history"].([]interface{})
        // PENDING -> PAYMENT_PENDING -> PAID -> PROCESSING -> SHIPPED -> DELIVERED = 6 entries
        s.Equal(6, len(history))
}

func (s *OrderLifecycleTestSuite) TestOrderCancelAndRefund() {
        orderID, _ := s.createOrderWithProduct()

        // PENDING -> CANCELLED
        resp := s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "CANCELLED", "reason": "Customer requested cancellation",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "cancel: %v", resp.Body)
        s.Equal("CANCELLED", s.dataField(resp)["status"])

        // CANCELLED is a terminal state — verify no further transitions
        resp = s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "REFUNDED", "reason": "Refund processed",
        }, s.adminAuth.AccessToken)
        s.Equal(400, resp.StatusCode)
        s.assertErrorCode(resp, "INVALID_TRANSITION")
}

func (s *OrderLifecycleTestSuite) TestOrderRefundFlow() {
        orderID, _ := s.createOrderWithProduct()

        // PENDING -> PAYMENT_PENDING -> PAID -> REFUND_PENDING -> REFUNDED
        s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PAYMENT_PENDING", "reason": "Awaiting payment",
        }, s.adminAuth.AccessToken)
        s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PAID", "reason": "Payment confirmed",
        }, s.adminAuth.AccessToken)

        // PAID -> REFUND_PENDING
        resp := s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "REFUND_PENDING", "reason": "Customer requested refund",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "REFUND_PENDING: %v", resp.Body)
        s.Equal("REFUND_PENDING", s.dataField(resp)["status"])

        // REFUND_PENDING -> REFUNDED
        resp = s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "REFUNDED", "reason": "Refund processed",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "REFUNDED: %v", resp.Body)
        s.Equal("REFUNDED", s.dataField(resp)["status"])
}

func (s *OrderLifecycleTestSuite) TestAdminListOrders() {
        s.createOrderWithProduct()
        s.createOrderWithProduct()

        resp := s.GET("/api/v1/admin/orders/?page=1&per_page=10", s.adminAuth.AccessToken)
        s.Equal(200, resp.StatusCode)
        data := s.dataField(resp)
        orders := data["data"].([]interface{})
        s.Equal(2, len(orders))
}

func (s *OrderLifecycleTestSuite) TestCustomerCanOnlySeeOwnOrders() {
        otherCustomer := s.RegisterUser("ol-other@example.com", "Pass1!abc", "Other", "User")
        s.createOrderWithProduct()

        // Other customer should see 0 orders
        resp := s.GET("/api/v1/orders/?page=1&per_page=10", otherCustomer.AccessToken)
        data := s.dataField(resp)
        orders := data["data"].([]interface{})
        s.Equal(0, len(orders))
}

func (s *OrderLifecycleTestSuite) TestOrderPricingCalculation() {
        orderID, orderNumber := s.createOrderWithProduct()

        getResp := s.GET("/api/v1/orders/"+orderID, s.customerAuth.AccessToken)
        data := s.dataField(getResp)
        s.Equal(orderNumber, data["order_number"])
        pricing := data["pricing"].(map[string]interface{})
        // 3 * 5000 = 15000
        s.Equal(15000, int(pricing["subtotal"].(float64)))
        s.Equal(0, int(pricing["discount"].(float64)))
        s.Equal(15000, int(pricing["total"].(float64)))

        items := data["items"].([]interface{})
        s.Equal(1, len(items))
        item := items[0].(map[string]interface{})
        itemPricing := item["pricing"].(map[string]interface{})
        s.Equal(5000, int(itemPricing["unit_price"].(float64)))
        s.Equal(15000, int(itemPricing["total"].(float64)))
}

// TestFullFlowWithNATSWorker tests the complete flow including:
// Order creation -> webhook -> outbox -> NATS -> order worker -> PAID
func (s *OrderLifecycleTestSuite) TestFullFlowWithNATSWorker() {
        // Create product
        prodResp := s.POST("/api/v1/products", map[string]interface{}{
                "name":              "NATS Flow Product",
                "slug":              "nats-flow-product",
                "short_description": "For NATS flow testing",
                "description":       "Detailed description",
                "brand":             "TestBrand",
                "status":            "ACTIVE",
                "category_id":      "",
                "variants": []map[string]interface{}{{
                        "sku": "NF-001", "price": 9999, "stock": 50, "status": "ACTIVE",
                }},
                "tags": []string{"nats-flow"},
        }, s.adminAuth.AccessToken)
        prodData := s.dataField(prodResp)
        productID := prodData["id"].(string)
        variantID := prodData["variants"].([]interface{})[0].(map[string]interface{})["id"].(string)

        // Create order
        orderResp := s.POST("/api/v1/orders/", map[string]interface{}{
                "items": []map[string]interface{}{{
                        "product_id": productID,
                        "variant_id": variantID,
                        "name":       "NATS Flow Product",
                        "sku":        "NF-001",
                        "quantity":   1,
                        "unit_price": 9999,
                }},
                "shipping_address": map[string]string{"name": "Test User", "phone": "01712345678",
                        "line1": "200 NATS St", "city": "Seattle", "state": "WA",
                        "postal_code": "98101", "country": "US",
                },
                "billing_address": map[string]string{
                        "line1": "200 NATS St", "city": "Seattle", "state": "WA",
                        "postal_code": "98101", "country": "US",
                },
        }, s.customerAuth.AccessToken)
        s.Require().Equal(201, orderResp.StatusCode)
        orderData := s.dataField(orderResp)
        orderID := orderData["id"].(string)

        // Transition to PAYMENT_PENDING
        resp := s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PAYMENT_PENDING", "reason": "Awaiting payment",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode)

        // Create payment record
        payment := payDomain.NewPayment(orderID, "fake", "USD", 9999)
        err := s.paymentRepo.Create(context.Background(), payment)
        s.Require().NoError(err)

        // Simulate payment webhook
        webhookPayload, _ := json.Marshal(map[string]interface{}{
                "id":          "wh-nats-flow-001",
                "type":        "payment.succeeded",
                "provider":    "fake",
                "payment_ref": "nats-pay-001",
                "amount":      9999,
                "currency":    "USD",
                "status":      "PAID",
                "timestamp":   "2026-01-01T00:00:00Z",
        })
        signature := s.webhookVerifier.SignPayload("fake", webhookPayload)

        webhookResp := s.POSTWebhook("/api/v1/webhooks/payment/fake", webhookPayload, map[string]string{
                "X-Webhook-Signature": signature,
                "X-Request-ID":       "nats-flow-req-001",
        })

        s.Require().Equal(200, webhookResp.StatusCode, "webhook failed: %v", webhookResp.Body)

        // The outbox worker should pick up the payment.completed event and publish to NATS.
        // The order worker (if running) would process it. But in the integration test,
        // the order worker is NOT running as a subscriber - the webhook service directly
        // transitions the order via the payment event publisher adapter + order worker
        // that's wired in cmd/worker/main.go.
        //
        // In this test suite, we verify the webhook processed correctly by checking
        // that the payment record is updated and outbox events exist.
        // The actual order PAID transition happens via the outbox -> NATS -> order worker pipeline.

        // Wait for outbox to be processed (the outbox worker is running in SetupSuite)
        s.WaitEventually(5*time.Second, func() bool {
                // Check if the order transitioned to PAID via the webhook handler's
                // direct call chain (webhook service -> payment publisher adapter)
                resp := s.GET("/api/v1/orders/"+orderID, s.customerAuth.AccessToken)
                if resp.StatusCode != 200 {
                        return false
                }
                data := s.dataField(resp)
                return data["status"] == "PAID"
        }, "order should be PAID after webhook + outbox")

        // Verify MongoDB state
        db := s.mongoClient.Database()
        var orderDoc bson.M
        err = db.Collection("orders").FindOne(context.Background(), bson.M{"_id": orderID}).Decode(&orderDoc)
        s.Require().NoError(err)
        s.Equal("PAID", orderDoc["status"])
        s.Equal("nats-pay-001", orderDoc["payment_reference"])

        // Verify outbox events
        outboxCount, _ := db.Collection("outbox").CountDocuments(context.Background(), bson.M{})
        s.Greater(outboxCount, int64(0), "outbox should have events")
}
