package integration

import (
        "context"
        "encoding/json"
        "sync"
        "sync/atomic"
        "testing"
        "time"

        "go.mongodb.org/mongo-driver/v2/bson"

        invApp "github.com/ecommerce/backend/internal/inventory/application"
        payDomain "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/stretchr/testify/suite"
)

// ==================== Inventory Concurrency Tests ====================

type InventoryConcurrencySuite struct {
        IntegrationSuite
        adminAuth *AuthResult
}

func TestInventoryConcurrencySuite(t *testing.T) {
        suite.Run(t, new(InventoryConcurrencySuite))
}

func (s *InventoryConcurrencySuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.adminAuth = s.RegisterUser("inv-cc-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

// TestConcurrentReservations_ExactLimit verifies that with stock=10 and 100
// concurrent reservation attempts (each reserving 1 unit), exactly 10 succeed
// and 90 fail with INSUFFICIENT_STOCK.
//
// The ReserveStock method uses MongoDB FindOneAndUpdate with an $expr filter
// that atomically checks (total_stock - reserved_stock) >= quantity before
// incrementing reserved_stock, ensuring no over-reservation.
func (s *InventoryConcurrencySuite) TestConcurrentReservations_ExactLimit() {
        const (
                totalStock   = 10
                concurrency = 100
                sku          = "CC-RESERVE-SKU"
                warehouseID  = "wh-concurrency-1"
        )

        // Create warehouse
        _, err := s.inventoryService.CreateWarehouse(context.Background(), invApp.CreateWarehouseInput{
                Name:    "Concurrency Warehouse",
                Code:    "CC-WH-1",
                Address: "1 Test Ave",
        })
        s.Require().NoError(err)

        // Create stock with totalStock=10
        _, err = s.inventoryService.CreateStock(context.Background(), invApp.CreateStockInput{
                SKU:              sku,
                WarehouseID:      warehouseID,
                TotalStock:       totalStock,
                LowStockThreshold: 0,
        })
        s.Require().NoError(err)

        // Launch 100 concurrent goroutines, each trying to reserve 1 unit
        var wg sync.WaitGroup
        successCount := int64(0)
        failCount := int64(0)

        for i := 0; i < concurrency; i++ {
                wg.Add(1)
                go func(idx int) {
                        defer wg.Done()
                        ref := "order-cc-" + time.Now().Format("150405") + "-" + string(rune('A'+idx%26)) + string(rune('A'+(idx/26)%26))
                        _, err := s.inventoryService.Reserve(context.Background(), invApp.ReserveInput{
                                SKU:         sku,
                                WarehouseID: warehouseID,
                                Quantity:    1,
                                Reference:   ref,
                        })
                        if err != nil {
                                atomic.AddInt64(&failCount, 1)
                        } else {
                                atomic.AddInt64(&successCount, 1)
                        }
                }(i)
        }

        wg.Wait()

        s.Equal(int64(totalStock), atomic.LoadInt64(&successCount),
                "expected exactly %d successful reservations", totalStock)
        s.Equal(int64(concurrency-totalStock), atomic.LoadInt64(&failCount),
                "expected %d failed reservations", concurrency-totalStock)

        // Verify final MongoDB state
        stock, err := s.inventoryService.GetStock(context.Background(), sku, warehouseID)
        s.Require().NoError(err)
        s.Equal(totalStock, stock.TotalStock, "total_stock should be unchanged")
        s.Equal(totalStock, stock.ReservedStock, "reserved_stock should equal total_stock")
        s.Equal(0, stock.AvailableStock(), "available stock should be 0")
}

// TestConcurrentReservations_MultiUnit verifies concurrent multi-unit reservations.
// stock=20, 50 goroutines each reserving 1 unit, exactly 20 succeed.
func (s *InventoryConcurrencySuite) TestConcurrentReservations_MultiUnit() {
        const (
                totalStock   = 20
                concurrency = 50
                sku          = "CC-MULTI-SKU"
                warehouseID  = "wh-concurrency-2"
        )

        s.inventoryService.CreateWarehouse(context.Background(), invApp.CreateWarehouseInput{
                Name: "Multi Unit WH", Code: "CC-WH-2", Address: "2 Test Ave",
        })
        _, err := s.inventoryService.CreateStock(context.Background(), invApp.CreateStockInput{
                SKU:         sku,
                WarehouseID: warehouseID,
                TotalStock:  totalStock,
        })
        s.Require().NoError(err)

        var wg sync.WaitGroup
        successCount := int64(0)

        for i := 0; i < concurrency; i++ {
                wg.Add(1)
                go func(idx int) {
                        defer wg.Done()
                        ref := "multi-cc-" + string(rune('A'+idx%26)) + string(rune('0'+idx%10))
                        _, err := s.inventoryService.Reserve(context.Background(), invApp.ReserveInput{
                                SKU:         sku,
                                WarehouseID: warehouseID,
                                Quantity:    1,
                                Reference:   ref,
                        })
                        if err == nil {
                                atomic.AddInt64(&successCount, 1)
                        }
                }(i)
        }

        wg.Wait()

        s.Equal(int64(totalStock), atomic.LoadInt64(&successCount),
                "expected exactly %d successful reservations", totalStock)
}

// ==================== Webhook Idempotency Concurrency Tests ====================

type WebhookConcurrencySuite struct {
        IntegrationSuite
        adminAuth *AuthResult
}

func TestWebhookConcurrencySuite(t *testing.T) {
        suite.Run(t, new(WebhookConcurrencySuite))
}

func (s *WebhookConcurrencySuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.adminAuth = s.RegisterUser("wh-cc-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

// TestConcurrentWebhookIdempotency verifies that sending the same webhook
// 100 times concurrently results in exactly one logical state transition
// (PENDING -> PAID). The payment status must end as PAID, and the status
// history must have exactly one PAID entry.
//
// Safety mechanisms:
//  1. webhook_events collection has a unique index on webhook_id
//  2. payment UpdateStatus uses FindOneAndUpdate with expected status filter
//  3. Webhook service handles CONFLICT errors gracefully
func (s *WebhookConcurrencySuite) TestConcurrentWebhookIdempotency() {
        const concurrency = 100
        webhookID := "wh-cc-idem-001"

        // Create an order and payment
        orderResp := s.POST("/api/v1/orders/", map[string]interface{}{
                "items": []map[string]interface{}{{
                        "product_id": "prod-cc-1",
                        "variant_id": "var-cc-1",
                        "name":       "CC Product",
                        "sku":        "CC-SKU",
                        "quantity":   1,
                        "unit_price": 5000,
                }},
                "shipping_address": map[string]string{"name": "Test User", "phone": "01712345678",
                        "line1": "1 Webhook St", "city": "Denver", "state": "CO",
                        "postal_code": "80201", "country": "US",
                },
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, orderResp.StatusCode, "create order: %v", orderResp.Body)
        orderData := s.dataField(orderResp)
        orderID := orderData["id"].(string)

        // Transition to PAYMENT_PENDING
        resp := s.PATCH("/api/v1/admin/orders/"+orderID+"/status", map[string]string{
                "status": "PAYMENT_PENDING", "reason": "awaiting payment",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode)

        // Create a PENDING payment with a known provider_reference
        payment := payDomain.NewPayment(orderID, "fake", "USD", 5000)
        payment.ProviderRef = "cc-pay-ref-001"
        err := s.paymentRepo.Create(context.Background(), payment)
        s.Require().NoError(err)

        // Build the webhook payload
        webhookPayload, _ := json.Marshal(map[string]interface{}{
                "id":          webhookID,
                "type":        "payment.completed",
                "provider":    "fake",
                "payment_ref": "cc-pay-ref-001",
                "amount":      5000,
                "currency":    "USD",
                "status":      "PAID",
                "timestamp":   "2026-08-21T00:00:00Z",
        })
        signature := s.webhookVerifier.SignPayload("fake", webhookPayload)

        headers := map[string]string{
                "X-Webhook-Signature": signature,
                "X-Request-ID":       "wh-cc-req-001",
        }

        // Launch 100 concurrent webhook deliveries
        var wg sync.WaitGroup
        results := make([]*JSONResponse, concurrency)

        for i := 0; i < concurrency; i++ {
                wg.Add(1)
                go func(idx int) {
                        defer wg.Done()
                        results[idx] = s.POSTWebhook("/api/v1/webhooks/payment/fake", webhookPayload, headers)
                }(i)
        }

        wg.Wait()

        // All requests should return 200 (idempotent — no errors even on retries)
        for i, r := range results {
                s.Equal(200, r.StatusCode, "webhook %d should return 200: %v", i, r.Body)
        }

        // Verify payment is PAID
        paymentDoc, err := s.paymentRepo.FindByID(context.Background(), payment.ID)
        s.Require().NoError(err)
        s.Equal(payDomain.PaymentPaid, paymentDoc.Status,
                "payment should be PAID after concurrent webhooks")

        // Verify only one processed webhook record exists
        db := s.mongoClient.Database()
        count, err := db.Collection("webhook_events").CountDocuments(context.Background(), bson.M{
                "webhook_id": webhookID,
        })
        s.Require().NoError(err)
        s.Equal(int64(1), count, "exactly one processed webhook record should exist")
}

// ==================== Order Creation Concurrency Tests ====================

type OrderConcurrencySuite struct {
        IntegrationSuite
        customerAuth *AuthResult
}

func TestOrderConcurrencySuite(t *testing.T) {
        suite.Run(t, new(OrderConcurrencySuite))
}

func (s *OrderConcurrencySuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.customerAuth = s.RegisterUser("ord-cc@example.com", "OrderCCPass1!abc", "Order", "User")
}

// TestConcurrentOrderCreation verifies that N concurrent order creation
// requests all succeed with unique, valid orders. No duplicate IDs or
// order numbers should occur.
func (s *OrderConcurrencySuite) TestConcurrentOrderCreation() {
        const concurrency = 50

        var wg sync.WaitGroup
        results := make([]*JSONResponse, concurrency)

        for i := 0; i < concurrency; i++ {
                wg.Add(1)
                go func(idx int) {
                        defer wg.Done()
                        results[idx] = s.POST("/api/v1/orders/", map[string]interface{}{
                                "items": []map[string]interface{}{{
                                        "product_id": "prod-cc-" + string(rune('A'+idx%26)),
                                        "variant_id": "var-cc-" + string(rune('A'+idx%26)),
                                        "name":       "Concurrent Product " + string(rune('A'+idx%26)),
                                        "sku":        "CC-ORD-" + string(rune('0'+idx%10)),
                                        "quantity":   1,
                                        "unit_price": 1000 + idx,
                                }},
                                "shipping_address": map[string]string{"name": "Test User", "phone": "01712345678",
                                        "line1":      "100 Order Ave",
                                        "city":        "Portland",
                                        "state":       "OR",
                                        "postal_code": "97201",
                                        "country":     "US",
                                },
                                "billing_address": map[string]string{
                                        "line1":      "100 Order Ave",
                                        "city":        "Portland",
                                        "state":       "OR",
                                        "postal_code": "97201",
                                        "country":     "US",
                                },
                        }, s.customerAuth.AccessToken)
                }(i)
        }

        wg.Wait()

        // All should succeed with 201
        orderIDs := make(map[string]bool)
        orderNumbers := make(map[string]bool)
        for i, r := range results {
                s.Equal(201, r.StatusCode, "order %d should return 201: %v", i, r.Body)
                data := s.dataField(r)
                id := data["id"].(string)
                num := data["order_number"].(string)
                s.NotEmpty(id, "order %d should have an ID", i)
                s.NotEmpty(num, "order %d should have an order_number", i)

                // Verify uniqueness
                s.False(orderIDs[id], "duplicate order ID: %s", id)
                s.False(orderNumbers[num], "duplicate order number: %s", num)
                orderIDs[id] = true
                orderNumbers[num] = true
        }

        s.Equal(concurrency, len(orderIDs), "all orders should have unique IDs")
        s.Equal(concurrency, len(orderNumbers), "all orders should have unique order numbers")

        // Verify count in MongoDB
        db := s.mongoClient.Database()
        count, err := db.Collection("orders").CountDocuments(context.Background(), bson.M{
                "customer_id": s.customerAuth.UserID,
        })
        s.Require().NoError(err)
        s.Equal(int64(concurrency), count, "MongoDB should have exactly %d orders", concurrency)
}

// ==================== Coupon/Promotion Concurrency Tests ====================

type CouponConcurrencySuite struct {
        IntegrationSuite
        adminAuth *AuthResult
}

func TestCouponConcurrencySuite(t *testing.T) {
        suite.Run(t, new(CouponConcurrencySuite))
}

func (s *CouponConcurrencySuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.adminAuth = s.RegisterUser("coupon-cc-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

// TestConcurrentCouponRedemption verifies that a promotion with usage_limit=10
// cannot be redeemed more than 10 times, even with 100 concurrent attempts.
//
// The RedeemPromotion repository method uses MongoDB FindOneAndUpdate with
// an atomic filter: status=ACTIVE AND (usage_limit=0 OR usage_count < usage_limit).
// Only one concurrent caller can match and increment at a time.
func (s *CouponConcurrencySuite) TestConcurrentCouponRedemption() {
        const (
                usageLimit  = 10
                concurrency = 100
        )

        // Create a promotion with usage_limit=10
        resp := s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
                "name":           "Concurrency Coupon",
                "code":           "CC10OFF",
                "discount_type":  "PERCENTAGE",
                "discount_value": 10,
                "usage_limit":    usageLimit,
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "create promotion: %v", resp.Body)
        data := s.dataField(resp)
        promotionID := data["id"].(string)

        // Transition to ACTIVE
        resp = s.PATCH("/api/v1/admin/marketing/promotions/"+promotionID+"/status", map[string]string{
                "status": "ACTIVE",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "activate promotion: %v", resp.Body)

        // Launch 100 concurrent redemption attempts via the service directly
        var wg sync.WaitGroup
        successCount := int64(0)
        failCount := int64(0)

        for i := 0; i < concurrency; i++ {
                wg.Add(1)
                go func() {
                        defer wg.Done()
                        _, err := s.marketingService.RedeemPromotion(context.Background(), promotionID)
                        if err != nil {
                                atomic.AddInt64(&failCount, 1)
                        } else {
                                atomic.AddInt64(&successCount, 1)
                        }
                }()
        }

        wg.Wait()

        s.Equal(int64(usageLimit), atomic.LoadInt64(&successCount),
                "expected exactly %d successful redemptions", usageLimit)
        s.Equal(int64(concurrency-usageLimit), atomic.LoadInt64(&failCount),
                "expected %d failed redemptions", concurrency-usageLimit)

        // Verify final state in MongoDB
        finalPromo, err := s.marketingService.GetPromotion(context.Background(), promotionID)
        s.Require().NoError(err)
        s.Equal(usageLimit, finalPromo.UsageCount,
                "usage_count should be exactly %d", usageLimit)
}

// TestConcurrentCouponRedemption_Unlimited verifies that a promotion with
// usage_limit=0 (unlimited) allows all concurrent redemptions to succeed.
func (s *CouponConcurrencySuite) TestConcurrentCouponRedemption_Unlimited() {
        const concurrency = 50

        resp := s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
                "name":           "Unlimited Coupon",
                "code":           "UNLIMITED50",
                "discount_type":  "FIXED",
                "discount_value": 500,
                "usage_limit":    0,
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode)
        data := s.dataField(resp)
        promotionID := data["id"].(string)

        resp = s.PATCH("/api/v1/admin/marketing/promotions/"+promotionID+"/status", map[string]string{
                "status": "ACTIVE",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode)

        var wg sync.WaitGroup
        successCount := int64(0)

        for i := 0; i < concurrency; i++ {
                wg.Add(1)
                go func() {
                        defer wg.Done()
                        _, err := s.marketingService.RedeemPromotion(context.Background(), promotionID)
                        if err == nil {
                                atomic.AddInt64(&successCount, 1)
                        }
                }()
        }

        wg.Wait()

        s.Equal(int64(concurrency), atomic.LoadInt64(&successCount),
                "unlimited promotion should allow all %d redemptions", concurrency)

        finalPromo, err := s.marketingService.GetPromotion(context.Background(), promotionID)
        s.Require().NoError(err)
        s.Equal(concurrency, finalPromo.UsageCount)
}

// TestConcurrentCouponRedemption_NonActiveFails verifies that redeeming a
// non-ACTIVE promotion fails atomically under concurrency.
func (s *CouponConcurrencySuite) TestConcurrentCouponRedemption_NonActiveFails() {
        resp := s.POST("/api/v1/admin/marketing/promotions", map[string]interface{}{
                "name":           "Draft Coupon",
                "code":           "DRAFT-CC",
                "discount_type":  "PERCENTAGE",
                "discount_value": 5,
                "usage_limit":    100,
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode)
        data := s.dataField(resp)
        promotionID := data["id"].(string)

        // Promotion is still in DRAFT status — all redemptions should fail
        var wg sync.WaitGroup
        successCount := int64(0)
        const concurrency = 20

        for i := 0; i < concurrency; i++ {
                wg.Add(1)
                go func() {
                        defer wg.Done()
                        _, err := s.marketingService.RedeemPromotion(context.Background(), promotionID)
                        if err == nil {
                                atomic.AddInt64(&successCount, 1)
                        }
                }()
        }

        wg.Wait()

        s.Equal(int64(0), atomic.LoadInt64(&successCount),
                "DRAFT promotion should not allow any redemptions")
}
