package integration

import (
        "testing"

        "github.com/stretchr/testify/suite"
)

// ==================== Inventory Tests ====================

type InventoryTestSuite struct {
        IntegrationSuite
        adminAuth *AuthResult
}

func TestInventorySuite(t *testing.T) {
        suite.Run(t, new(InventoryTestSuite))
}

func (s *InventoryTestSuite) SetupTest() {
        s.IntegrationSuite.SetupTest()
        s.adminAuth = s.RegisterUser("inv-admin@example.com", "AdminPass1!abc", "Admin", "User")
}

// --- Warehouse Tests ---

func (s *InventoryTestSuite) TestCreateWarehouse() {
        resp := s.POST("/api/v1/admin/inventory/warehouses", map[string]string{
                "name":    "Main Warehouse",
                "code":    "WH-MAIN",
                "address": "123 Warehouse Blvd, Chicago, IL",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "create warehouse: %v", resp.Body)
        data := s.dataField(resp)
        s.Equal("Main Warehouse", data["name"])
        s.Equal("WH-MAIN", data["code"])
        s.Equal("123 Warehouse Blvd, Chicago, IL", data["address"])
        s.True(data["is_active"].(bool))
        s.NotEmpty(data["id"])
}

func (s *InventoryTestSuite) TestListWarehouses() {
        for i := 0; i < 3; i++ {
                s.POST("/api/v1/admin/inventory/warehouses", map[string]string{
                        "name":    "Warehouse " + string(rune('A'+i)),
                        "code":    "WH-" + string(rune('A'+i)),
                        "address": "Address " + string(rune('A'+i)),
                }, s.adminAuth.AccessToken)
        }

        resp := s.GET("/api/v1/admin/inventory/warehouses?page=1&per_page=10", s.adminAuth.AccessToken)
        s.Equal(200, resp.StatusCode)
        data := s.dataField(resp)
        items := data["data"].([]interface{})
        s.Equal(3, len(items))
}

func (s *InventoryTestSuite) TestCreateWarehouse_RequiresAuth() {
        resp := s.POST("/api/v1/admin/inventory/warehouses", map[string]string{
                "name": "X", "code": "Y", "address": "Z",
        }, "")
        s.Equal(401, resp.StatusCode)
}

// --- Stock Tests ---

func (s *InventoryTestSuite) TestCreateAndGetStock() {
        // Create warehouse first
        whResp := s.POST("/api/v1/admin/inventory/warehouses", map[string]string{
                "name": "Stock WH", "code": "STOCK-WH", "address": "10 Stock St",
        }, s.adminAuth.AccessToken)
        whData := s.dataField(whResp)
        whID := whData["id"].(string)

        // Create stock
        resp := s.POST("/api/v1/admin/inventory/stock", map[string]interface{}{
                "sku":               "SKU-001",
                "warehouse_id":       whID,
                "total_stock":        100,
                "low_stock_threshold": 10,
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "create stock: %v", resp.Body)
        data := s.dataField(resp)
        s.Equal("SKU-001", data["sku"])
        s.Equal(100, int(data["total_stock"].(float64)))
        s.Equal(0, int(data["reserved_stock"].(float64)))
        s.Equal(100, int(data["available_stock"].(float64)))
        s.Equal(false, data["is_low_stock"])

        // Get stock
        getResp := s.GET("/api/v1/admin/inventory/stock/SKU-001/"+whID, s.adminAuth.AccessToken)
        s.Equal(200, getResp.StatusCode)
        getStock := s.dataField(getResp)
        s.Equal("SKU-001", getStock["sku"])
}

func (s *InventoryTestSuite) TestAdjustStock_Increase() {
        whID := s.createTestWarehouse("AdjWH", "ADJ-WH")
        s.createTestStock("ADJ-SKU", whID, 50, 5)

        resp := s.POST("/api/v1/admin/inventory/stock/adjust", map[string]interface{}{
                "sku":         "ADJ-SKU",
                "warehouse_id": whID,
                "quantity":    25,
                "reason":      "restock",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "adjust stock: %v", resp.Body)
        data := s.dataField(resp)
        s.Equal(75, int(data["total_stock"].(float64)))
        s.Equal(75, int(data["available_stock"].(float64)))
}

func (s *InventoryTestSuite) TestAdjustStock_Decrease() {
        whID := s.createTestWarehouse("DecWH", "DEC-WH")
        s.createTestStock("DEC-SKU", whID, 50, 5)

        resp := s.POST("/api/v1/admin/inventory/stock/adjust", map[string]interface{}{
                "sku":         "DEC-SKU",
                "warehouse_id": whID,
                "quantity":    -10,
                "reason":      "damaged",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(200, resp.StatusCode, "adjust stock: %v", resp.Body)
        data := s.dataField(resp)
        s.Equal(40, int(data["total_stock"].(float64)))
        s.Equal(40, int(data["available_stock"].(float64)))
}

func (s *InventoryTestSuite) TestAdjustStock_ZeroQuantity() {
        whID := s.createTestWarehouse("ZeroWH", "ZERO-WH")
        s.createTestStock("ZERO-SKU", whID, 50, 5)

        resp := s.POST("/api/v1/admin/inventory/stock/adjust", map[string]interface{}{
                "sku":         "ZERO-SKU",
                "warehouse_id": whID,
                "quantity":    0,
        }, s.adminAuth.AccessToken)
        s.Equal(422, resp.StatusCode)
        s.assertErrorCode(resp, "VALIDATION_ERROR")
}

// --- Reservation Tests ---

func (s *InventoryTestSuite) TestReserveAndRelease() {
        whID := s.createTestWarehouse("ResWH", "RES-WH")
        s.createTestStock("RES-SKU", whID, 100, 5)

        // Reserve 30 units
        resp := s.POST("/api/v1/admin/inventory/reserve", map[string]interface{}{
                "sku":         "RES-SKU",
                "warehouse_id": whID,
                "quantity":    30,
                "reference":   "order-123",
                "ttl_seconds":  3600,
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "reserve: %v", resp.Body)
        data := s.dataField(resp)

        // Check reservation
        reservation := data["reservation"].(map[string]interface{})
        s.Equal("RES-SKU", reservation["sku"])
        s.Equal(30, int(reservation["quantity"].(float64)))
        s.Equal("ACTIVE", reservation["status"])

        // Check stock updated
        stock := data["stock"].(map[string]interface{})
        s.Equal(100, int(stock["total_stock"].(float64)))
        s.Equal(30, int(stock["reserved_stock"].(float64)))
        s.Equal(70, int(stock["available_stock"].(float64)))

        // Release reservation
        resID := reservation["id"].(string)
        relResp := s.POST("/api/v1/admin/inventory/release/"+resID, nil, s.adminAuth.AccessToken)
        s.Require().Equal(200, relResp.StatusCode, "release: %v", relResp.Body)
        relStock := s.dataField(relResp)
        s.Equal(0, int(relStock["reserved_stock"].(float64)))
        s.Equal(100, int(relStock["available_stock"].(float64)))
}

func (s *InventoryTestSuite) TestReserveAndConsume() {
        whID := s.createTestWarehouse("ConWH", "CON-WH")
        s.createTestStock("CON-SKU", whID, 100, 5)

        // Reserve
        resp := s.POST("/api/v1/admin/inventory/reserve", map[string]interface{}{
                "sku":         "CON-SKU",
                "warehouse_id": whID,
                "quantity":    20,
                "reference":   "order-consume-1",
        }, s.adminAuth.AccessToken)
        data := s.dataField(resp)
        resID := data["reservation"].(map[string]interface{})["id"].(string)

        // Consume
        conResp := s.POST("/api/v1/admin/inventory/consume/"+resID, nil, s.adminAuth.AccessToken)
        s.Require().Equal(200, conResp.StatusCode, "consume: %v", conResp.Body)
        conStock := s.dataField(conResp)
        s.Equal(80, int(conStock["total_stock"].(float64)))
        s.Equal(0, int(conStock["reserved_stock"].(float64)))
        s.Equal(80, int(conStock["available_stock"].(float64)))

        // Get reservation should show CONSUMED
        getResp := s.GET("/api/v1/admin/inventory/reservations/"+resID, s.adminAuth.AccessToken)
        getRes := s.dataField(getResp)
        s.Equal("CONSUMED", getRes["status"])
}

func (s *InventoryTestSuite) TestReserve_ExceedsAvailable() {
        whID := s.createTestWarehouse("OverWH", "OVER-WH")
        s.createTestStock("OVER-SKU", whID, 10, 5)

        resp := s.POST("/api/v1/admin/inventory/reserve", map[string]interface{}{
                "sku":         "OVER-SKU",
                "warehouse_id": whID,
                "quantity":    20,
                "reference":   "order-over",
        }, s.adminAuth.AccessToken)
        s.Equal(400, resp.StatusCode)
}

// --- Movements ---

func (s *InventoryTestSuite) TestListMovements() {
        whID := s.createTestWarehouse("MovWH", "MOV-WH")
        s.createTestStock("MOV-SKU", whID, 50, 5)

        // Adjust creates a movement
        s.POST("/api/v1/admin/inventory/stock/adjust", map[string]interface{}{
                "sku":         "MOV-SKU",
                "warehouse_id": whID,
                "quantity":    10,
                "reason":      "test adjustment",
        }, s.adminAuth.AccessToken)

        resp := s.GET("/api/v1/admin/inventory/movements?sku=MOV-SKU&page=1&per_page=10", s.adminAuth.AccessToken)
        s.Equal(200, resp.StatusCode)
        data := s.dataField(resp)
        movements := data["data"].([]interface{})
        s.GreaterOrEqual(len(movements), 1)
        // First movement should be the initial stock creation
        firstMove := movements[0].(map[string]interface{})
        s.Equal("MOV-SKU", firstMove["sku"])
}

// --- Low Stock ---

func (s *InventoryTestSuite) TestListLowStock() {
        whID := s.createTestWarehouse("LowWH", "LOW-WH")
        // Create stock with low threshold higher than total
        s.createTestStock("LOW-SKU", whID, 3, 5)

        resp := s.GET("/api/v1/admin/inventory/low-stock", s.adminAuth.AccessToken)
        s.Equal(200, resp.StatusCode)
        data := s.dataField(resp)
        items := data.([]interface{})
        s.GreaterOrEqual(len(items), 1)

        // Find our low-stock item
        found := false
        for _, item := range items {
                m := item.(map[string]interface{})
                if m["sku"] == "LOW-SKU" {
                        s.True(m["is_low_stock"].(bool))
                        found = true
                }
        }
        s.True(found, "LOW-SKU should be in low-stock list")
}

// --- Helpers ---

func (s *InventoryTestSuite) createTestWarehouse(name, code string) string {
        resp := s.POST("/api/v1/admin/inventory/warehouses", map[string]string{
                "name": name, "code": code, "address": "Test Address",
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode)
        return s.dataField(resp)["id"].(string)
}

func (s *InventoryTestSuite) createTestStock(sku, warehouseID string, total, threshold int) {
        resp := s.POST("/api/v1/admin/inventory/stock", map[string]interface{}{
                "sku":                 sku,
                "warehouse_id":       warehouseID,
                "total_stock":        total,
                "low_stock_threshold": threshold,
        }, s.adminAuth.AccessToken)
        s.Require().Equal(201, resp.StatusCode, "create stock %s: %v", sku, resp.Body)
}
