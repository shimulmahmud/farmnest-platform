package application

import (
        "context"
        "sync"
        "sync/atomic"
        "testing"
        "time"

        "github.com/ecommerce/backend/internal/inventory/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Mocks ---

// copyStock returns a value copy of a stock item to avoid data races in concurrent tests.
func copyStock(s *domain.StockItem) *domain.StockItem {
        cp := *s
        return &cp
}

// mockStockRepo implements concurrency-safe in-memory stock operations
// using sync.Mutex to simulate MongoDB's atomic behavior.
type mockStockRepo struct {
        mu     sync.Mutex
        stocks map[string]*domain.StockItem // key: "sku:warehouseID"
}

func newMockStockRepo() *mockStockRepo {
        return &mockStockRepo{stocks: make(map[string]*domain.StockItem)}
}

func (r *mockStockRepo) stockKey(sku, wh string) string { return sku + ":" + wh }

func (r *mockStockRepo) CreateStock(_ context.Context, stock *domain.StockItem) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        key := r.stockKey(stock.SKU, stock.WarehouseID)
        if _, ok := r.stocks[key]; ok {
                return errors.New("CONFLICT", "stock item already exists", 409)
        }
        cp := *stock
        r.stocks[key] = &cp
        return nil
}

func (r *mockStockRepo) GetStock(_ context.Context, sku, wh string) (*domain.StockItem, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        s, ok := r.stocks[r.stockKey(sku, wh)]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return copyStock(s), nil
}

func (r *mockStockRepo) AdjustStock(_ context.Context, sku, wh string, quantity int, _ string) (*domain.StockItem, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        key := r.stockKey(sku, wh)
        s, ok := r.stocks[key]
        if !ok {
                return nil, errors.ErrNotFound
        }
        if s.TotalStock+quantity < 0 {
                return nil, errors.New("BAD_REQUEST", "adjustment would result in negative stock", 400)
        }
        s.TotalStock += quantity
        return copyStock(s), nil
}

func (r *mockStockRepo) ReserveStock(_ context.Context, sku, wh string, quantity int) (*domain.StockItem, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        key := r.stockKey(sku, wh)
        s, ok := r.stocks[key]
        if !ok {
                return nil, errors.ErrNotFound
        }
        available := s.TotalStock - s.ReservedStock
        if available < quantity {
                return nil, errors.New("INSUFFICIENT_STOCK", "not enough available stock", 409)
        }
        s.ReservedStock += quantity
        return copyStock(s), nil
}

func (r *mockStockRepo) ReleaseStock(_ context.Context, sku, wh string, quantity int) (*domain.StockItem, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        key := r.stockKey(sku, wh)
        s, ok := r.stocks[key]
        if !ok {
                return nil, errors.ErrNotFound
        }
        if s.ReservedStock < quantity {
                return nil, errors.New("BAD_REQUEST", "cannot release more than reserved", 400)
        }
        s.ReservedStock -= quantity
        return copyStock(s), nil
}

func (r *mockStockRepo) ConsumeStock(_ context.Context, sku, wh string, quantity int) (*domain.StockItem, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        key := r.stockKey(sku, wh)
        s, ok := r.stocks[key]
        if !ok {
                return nil, errors.ErrNotFound
        }
        if s.TotalStock < quantity || s.ReservedStock < quantity {
                return nil, errors.New("BAD_REQUEST", "insufficient stock to consume", 400)
        }
        s.TotalStock -= quantity
        s.ReservedStock -= quantity
        return copyStock(s), nil
}

func (r *mockStockRepo) ListLowStock(_ context.Context) ([]*domain.StockItem, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        var result []*domain.StockItem
        for _, s := range r.stocks {
                if s.IsLowStock() {
                        result = append(result, copyStock(s))
                }
        }
        return result, nil
}

// mockReservationRepo
type mockReservationRepo struct {
        mu      sync.Mutex
        resMap  map[string]*domain.Reservation
}

func newMockReservationRepo() *mockReservationRepo {
        return &mockReservationRepo{resMap: make(map[string]*domain.Reservation)}
}

func (r *mockReservationRepo) CreateReservation(_ context.Context, res *domain.Reservation) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        r.resMap[res.ID] = res
        return nil
}

func (r *mockReservationRepo) GetReservation(_ context.Context, id string) (*domain.Reservation, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        res, ok := r.resMap[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return res, nil
}

func (r *mockReservationRepo) GetActiveReservation(_ context.Context, sku, reference string) (*domain.Reservation, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        now := time.Now().UTC()
        for _, res := range r.resMap {
                if res.SKU == sku && res.Reference == reference && res.Status == domain.ReservationStatusActive && res.ExpiresAt.After(now) {
                        return res, nil
                }
        }
        return nil, nil
}

func (r *mockReservationRepo) ReleaseReservation(_ context.Context, id string) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        res, ok := r.resMap[id]
        if !ok || res.Status != domain.ReservationStatusActive {
                return errors.New("BAD_REQUEST", "reservation not found or not active", 400)
        }
        res.Status = domain.ReservationStatusReleased
        return nil
}

func (r *mockReservationRepo) ConsumeReservation(_ context.Context, id string) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        res, ok := r.resMap[id]
        if !ok || res.Status != domain.ReservationStatusActive {
                return errors.New("BAD_REQUEST", "reservation not found or not active", 400)
        }
        res.Status = domain.ReservationStatusConsumed
        return nil
}

func (r *mockReservationRepo) ExpireReservations(_ context.Context, sku string) ([]*domain.Reservation, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        now := time.Now().UTC()
        var expired []*domain.Reservation
        for _, res := range r.resMap {
                if res.SKU == sku && res.Status == domain.ReservationStatusActive && now.After(res.ExpiresAt) {
                        res.Status = domain.ReservationStatusExpired
                        expired = append(expired, res)
                }
        }
        return expired, nil
}

func (r *mockReservationRepo) FindActiveReservationsByReference(_ context.Context, reference string) ([]*domain.Reservation, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        var result []*domain.Reservation
        for _, res := range r.resMap {
                        if res.Reference == reference && res.Status == domain.ReservationStatusActive && res.ExpiresAt.After(time.Now().UTC()) {
                        result = append(result, res)
                }
        }
        return result, nil
}

func (r *mockReservationRepo) ListReservations(_ context.Context, sku, status string, _ pagination.Params) ([]*domain.Reservation, int64, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        var result []*domain.Reservation
        for _, res := range r.resMap {
                if res.SKU == sku && (status == "" || string(res.Status) == status) {
                        result = append(result, res)
                }
        }
        return result, int64(len(result)), nil
}

// mockMovementRepo
type mockMovementRepo struct {
        mu        sync.Mutex
        movements []*domain.Movement
}

func newMockMovementRepo() *mockMovementRepo {
        return &mockMovementRepo{}
}

func (r *mockMovementRepo) CreateMovement(_ context.Context, m *domain.Movement) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        r.movements = append(r.movements, m)
        return nil
}

func (r *mockMovementRepo) ListMovements(_ context.Context, _ string, _ pagination.Params) ([]*domain.Movement, int64, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        return r.movements, int64(len(r.movements)), nil
}

// mockWarehouseRepo
type mockWarehouseRepo struct {
        warehouses map[string]*domain.Warehouse
}

func newMockWarehouseRepo() *mockWarehouseRepo {
        return &mockWarehouseRepo{warehouses: make(map[string]*domain.Warehouse)}
}

func (r *mockWarehouseRepo) Create(_ context.Context, w *domain.Warehouse) error {
        r.warehouses[w.ID] = w
        return nil
}

func (r *mockWarehouseRepo) FindByID(_ context.Context, id string) (*domain.Warehouse, error) {
        w, ok := r.warehouses[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return w, nil
}

func (r *mockWarehouseRepo) FindByCode(_ context.Context, code string) (*domain.Warehouse, error) {
        for _, w := range r.warehouses {
                if w.Code == code {
                        return w, nil
                }
        }
        return nil, errors.ErrNotFound
}

func (r *mockWarehouseRepo) Update(_ context.Context, w *domain.Warehouse) error {
        r.warehouses[w.ID] = w
        return nil
}

func (r *mockWarehouseRepo) List(_ context.Context, _ pagination.Params) ([]*domain.Warehouse, int64, error) {
        var list []*domain.Warehouse
        for _, w := range r.warehouses {
                list = append(list, w)
        }
        return list, int64(len(list)), nil
}

// mockEventPublisher
type mockEventPublisher struct {
        mu     sync.Mutex
        events []string // subjects published
}

func newMockEventPublisher() *mockEventPublisher {
        return &mockEventPublisher{}
}

func (p *mockEventPublisher) Publish(_ context.Context, subject string, _ *domain.InventoryEvent) error {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.events = append(p.events, subject)
        return nil
}

func (p *mockEventPublisher) getEvents() []string {
        p.mu.Lock()
        defer p.mu.Unlock()
        return append([]string{}, p.events...)
}

// --- Helpers ---

func newTestService(t *testing.T) (*Service, *mockStockRepo, *mockReservationRepo, *mockMovementRepo, *mockEventPublisher) {
        t.Helper()
        stock := newMockStockRepo()
        res := newMockReservationRepo()
        mov := newMockMovementRepo()
        wh := newMockWarehouseRepo()
        evt := newMockEventPublisher()
        svc := NewService(stock, res, mov, wh, evt)
        return svc, stock, res, mov, evt
}

func setupStock(t *testing.T, svc *Service, sku string, totalStock, threshold int) {
        t.Helper()
        _, err := svc.CreateStock(context.Background(), CreateStockInput{
                SKU:               sku,
                WarehouseID:       "wh-1",
                TotalStock:        totalStock,
                LowStockThreshold: threshold,
        })
        require.NoError(t, err)
}

// --- Tests ---

func TestCreateStock_Success(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)

        stock, err := svc.CreateStock(context.Background(), CreateStockInput{
                SKU:               "SKU-A",
                WarehouseID:       "wh-1",
                TotalStock:        100,
                LowStockThreshold: 10,
        })
        require.NoError(t, err)

        assert.NotEmpty(t, stock.ID)
        assert.Equal(t, 100, stock.TotalStock)
        assert.Equal(t, 0, stock.ReservedStock)
        assert.Equal(t, 100, stock.AvailableStock())
        assert.False(t, stock.IsLowStock())
}

func TestCreateStock_NegativeStock(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)

        _, err := svc.CreateStock(context.Background(), CreateStockInput{
                SKU:        "SKU-A",
                WarehouseID: "wh-1",
                TotalStock:  -5,
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, 422, appErr.Status)
}

func TestAdjustStock_Increase(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 50, 5)

        updated, err := svc.AdjustStock(context.Background(), AdjustStockInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 30, Reason: "restock",
        })
        require.NoError(t, err)
        assert.Equal(t, 80, updated.TotalStock)
}

func TestAdjustStock_Decrease(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 50, 0)

        updated, err := svc.AdjustStock(context.Background(), AdjustStockInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: -20, Reason: "damaged",
        })
        require.NoError(t, err)
        assert.Equal(t, 30, updated.TotalStock)
}

func TestAdjustStock_NegativePrevention(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 10, 0)

        _, err := svc.AdjustStock(context.Background(), AdjustStockInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: -20, Reason: "too many",
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, 400, appErr.Status)

        // Verify stock unchanged
        stock, _ := svc.GetStock(context.Background(), "SKU-A", "wh-1")
        assert.Equal(t, 10, stock.TotalStock)
}

func TestAdjustStock_LowStockEvent(t *testing.T) {
        svc, _, _, _, evt := newTestService(t)
        setupStock(t, svc, "SKU-A", 20, 15)

        _, err := svc.AdjustStock(context.Background(), AdjustStockInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: -10, Reason: "sale",
        })
        require.NoError(t, err)

        events := evt.getEvents()
        found := false
        for _, e := range events {
                if e == "inventory.low_stock" {
                        found = true
                }
        }
        assert.True(t, found, "expected inventory.low_stock event")
}

func TestReserve_Success(t *testing.T) {
        svc, _, _, _, evt := newTestService(t)
        setupStock(t, svc, "SKU-A", 100, 0)

        result, err := svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 10, Reference: "order-1",
        })
        require.NoError(t, err)

        assert.Equal(t, 10, result.Stock.ReservedStock)
        assert.Equal(t, 90, result.Stock.AvailableStock())
        assert.Equal(t, domain.ReservationStatusActive, result.Reservation.Status)

        events := evt.getEvents()
        assert.Contains(t, events, "inventory.reserved")
}

func TestReserve_InsufficientStock(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 5, 0)

        _, err := svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 10, Reference: "order-1",
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, 409, appErr.Status)
}

func TestReserve_DuplicateReference(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 100, 0)

        _, err := svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 5, Reference: "order-1",
        })
        require.NoError(t, err)

        _, err = svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 5, Reference: "order-1",
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, 409, appErr.Status)
}

func TestReserve_InvalidQuantity(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 100, 0)

        _, err := svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 0, Reference: "order-1",
        })
        assert.Error(t, err)
        assert.Equal(t, 422, errors.FromError(err).Status)

        _, err = svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: -5, Reference: "order-2",
        })
        assert.Error(t, err)
        assert.Equal(t, 422, errors.FromError(err).Status)
}

func TestRelease_Success(t *testing.T) {
        svc, _, _, _, evt := newTestService(t)
        setupStock(t, svc, "SKU-A", 100, 0)

        result, err := svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 30, Reference: "order-1",
        })
        require.NoError(t, err)

        updated, err := svc.Release(context.Background(), result.Reservation.ID)
        require.NoError(t, err)
        assert.Equal(t, 0, updated.ReservedStock)
        assert.Equal(t, 100, updated.AvailableStock())

        events := evt.getEvents()
        assert.Contains(t, events, "inventory.released")
}

func TestRelease_NotActive(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 100, 0)

        result, err := svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 30, Reference: "order-1",
        })
        require.NoError(t, err)

        // Release it once
        _, err = svc.Release(context.Background(), result.Reservation.ID)
        require.NoError(t, err)

        // Try again — should fail
        _, err = svc.Release(context.Background(), result.Reservation.ID)
        assert.Error(t, err)
        assert.Equal(t, 400, errors.FromError(err).Status)
}

func TestConsume_Success(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 100, 0)

        result, err := svc.Reserve(context.Background(), ReserveInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 25, Reference: "order-1",
        })
        require.NoError(t, err)

        updated, err := svc.Consume(context.Background(), result.Reservation.ID)
        require.NoError(t, err)
        assert.Equal(t, 75, updated.TotalStock)
        assert.Equal(t, 0, updated.ReservedStock)
}

func TestExpireReservations(t *testing.T) {
        svc, _, resRepo, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 100, 0)

        // Manually insert an already-expired reservation
        expiredRes := domain.NewReservation("SKU-A", "wh-1", 20, "order-exp", -1*time.Second)
        // Bypass Reserve to avoid duplicate/ref checks; insert directly into mock
        resRepo.mu.Lock()
        resRepo.resMap[expiredRes.ID] = expiredRes
        resRepo.mu.Unlock()

        // We need to also manually reserve the stock (since we bypassed Reserve)
        svc.stock.(*mockStockRepo).mu.Lock()
        svc.stock.(*mockStockRepo).stocks["SKU-A:wh-1"].ReservedStock = 20
        svc.stock.(*mockStockRepo).mu.Unlock()

        // Verify stock is reserved by checking via repo directly
        stock, _ := svc.GetStock(context.Background(), "SKU-A", "wh-1")
        assert.Equal(t, 20, stock.ReservedStock)

        // Expire
        count, err := svc.ExpireReservations(context.Background(), "SKU-A")
        require.NoError(t, err)
        assert.Equal(t, 1, count)

        // Verify stock released
        stock, _ = svc.GetStock(context.Background(), "SKU-A", "wh-1")
        assert.Equal(t, 0, stock.ReservedStock)
}

// --- Concurrency Tests ---

func TestConcurrentReservations_Stock1(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-RACE", 1, 0)

        var wg sync.WaitGroup
        successCount := int64(0)
        failCount := int64(0)

        for i := 0; i < 10; i++ {
                wg.Add(1)
                go func(idx int) {
                        defer wg.Done()
                        _, err := svc.Reserve(context.Background(), ReserveInput{
                                SKU: "SKU-RACE", WarehouseID: "wh-1",
                                Quantity: 1, Reference: "ref-" + itoa(idx),
                        })
                        if err != nil {
                                atomic.AddInt64(&failCount, 1)
                        } else {
                                atomic.AddInt64(&successCount, 1)
                        }
                }(i)
        }
        wg.Wait()

        assert.Equal(t, int64(1), successCount, "exactly 1 reservation should succeed")
        assert.Equal(t, int64(9), failCount, "9 reservations should fail")
}

func TestConcurrentReservations_Stock10_100Goroutines(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-RACE2", 10, 0)

        var wg sync.WaitGroup
        successCount := int64(0)
        failCount := int64(0)

        for i := 0; i < 100; i++ {
                wg.Add(1)
                go func(idx int) {
                        defer wg.Done()
                        _, err := svc.Reserve(context.Background(), ReserveInput{
                                SKU: "SKU-RACE2", WarehouseID: "wh-1",
                                Quantity: 1, Reference: "ref-" + itoa(idx),
                        })
                        if err != nil {
                                atomic.AddInt64(&failCount, 1)
                        } else {
                                atomic.AddInt64(&successCount, 1)
                        }
                }(i)
        }
        wg.Wait()

        assert.Equal(t, int64(10), successCount, "exactly 10 reservations should succeed")
        assert.Equal(t, int64(90), failCount, "90 reservations should fail")

        // Verify stock integrity
        stock, _ := svc.GetStock(context.Background(), "SKU-RACE2", "wh-1")
        assert.Equal(t, 10, stock.TotalStock)
        assert.Equal(t, 10, stock.ReservedStock)
        assert.Equal(t, 0, stock.AvailableStock())
}

func TestConcurrentMixedReserveRelease(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-MIX", 10, 0)

        // Reserve all 10 first
        var resIDs []string
        for i := 0; i < 10; i++ {
                result, err := svc.Reserve(context.Background(), ReserveInput{
                        SKU: "SKU-MIX", WarehouseID: "wh-1",
                        Quantity: 1, Reference: "order-" + itoa(i),
                })
                require.NoError(t, err)
                resIDs = append(resIDs, result.Reservation.ID)
        }

        // Concurrently release all 10
        var wg sync.WaitGroup
        successCount := int64(0)
        for _, id := range resIDs {
                wg.Add(1)
                go func(resID string) {
                        defer wg.Done()
                        _, err := svc.Release(context.Background(), resID)
                        if err == nil {
                                atomic.AddInt64(&successCount, 1)
                        }
                }(id)
        }
        wg.Wait()

        assert.Equal(t, int64(10), successCount)

        stock, _ := svc.GetStock(context.Background(), "SKU-MIX", "wh-1")
        assert.Equal(t, 10, stock.TotalStock)
        assert.Equal(t, 0, stock.ReservedStock)
}

func TestListLowStock(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-LOW", 5, 10)
        setupStock(t, svc, "SKU-OK", 50, 10)

        items, err := svc.ListLowStock(context.Background())
        require.NoError(t, err)
        assert.Len(t, items, 1)
        assert.Equal(t, "SKU-LOW", items[0].SKU)
}

func TestListMovements(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)
        setupStock(t, svc, "SKU-A", 50, 0)

        _, err := svc.AdjustStock(context.Background(), AdjustStockInput{
                SKU: "SKU-A", WarehouseID: "wh-1", Quantity: 10, Reason: "restock",
        })
        require.NoError(t, err)

        movements, total, err := svc.ListMovements(context.Background(), "SKU-A", pagination.Params{Page: 1, PerPage: 20, Skip: 0, Limit: 20})
        require.NoError(t, err)
        // Initial creation + adjustment = 2 movements
        assert.Equal(t, int64(2), total)
        assert.Len(t, movements, 2)
}

func TestWarehouse_CRUD(t *testing.T) {
        svc, _, _, _, _ := newTestService(t)

        wh, err := svc.CreateWarehouse(context.Background(), CreateWarehouseInput{
                Name: "Main Warehouse", Code: "MW-01", Address: "123 Commerce St",
        })
        require.NoError(t, err)
        assert.NotEmpty(t, wh.ID)
        assert.True(t, wh.IsActive)

        found, err := svc.GetWarehouse(context.Background(), wh.ID)
        require.NoError(t, err)
        assert.Equal(t, "MW-01", found.Code)

        warehouses, total, err := svc.ListWarehouses(context.Background(), pagination.Params{Page: 1, PerPage: 20, Skip: 0, Limit: 20})
        require.NoError(t, err)
        assert.Equal(t, int64(1), total)
        assert.Len(t, warehouses, 1)
}

// --- Helpers ---

func itoa(i int) string {
        if i == 0 {
                return "0"
        }
        s := ""
        neg := i < 0
        if neg {
                i = -i
        }
        for i > 0 {
                s = string(rune('0'+i%10)) + s
                i /= 10
        }
        if neg {
                s = "-" + s
        }
        return s
}
