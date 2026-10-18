package application

import (
        "context"
        "fmt"
        "time"

        "github.com/ecommerce/backend/internal/inventory/domain"
        "github.com/ecommerce/backend/pkg/audit"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/ecommerce/backend/pkg/telemetry"
)

const (
        defaultReservationTTL = 30 * time.Minute
)

// Service implements inventory management use cases.
type Service struct {
        stock        domain.StockRepository
        reservations domain.ReservationRepository
        movements    domain.MovementRepository
        warehouses   domain.WarehouseRepository
        events       domain.EventPublisher
        auditor      audit.Auditor
        bizMetrics   *telemetry.BusinessMetrics
}

// NewService creates an inventory application service.
func NewService(
        stock domain.StockRepository,
        reservations domain.ReservationRepository,
        movements domain.MovementRepository,
        warehouses domain.WarehouseRepository,
        events domain.EventPublisher,
) *Service {
        return &Service{
                stock:        stock,
                reservations: reservations,
                movements:    movements,
                warehouses:   warehouses,
                events:       events,
                auditor:      audit.NopAuditor,
        }
}

// SetAuditor injects an async auditor for cross-cutting audit logging.
func (s *Service) SetAuditor(a audit.Auditor) {
        s.auditor = a
}

// SetBusinessMetrics injects business metrics for inventory events.
func (s *Service) SetBusinessMetrics(m *telemetry.BusinessMetrics) {
        s.bizMetrics = m
}

// --- Stock ---

// CreateStockInput holds fields for creating a stock item.
type CreateStockInput struct {
        SKU              string `json:"sku" validate:"required"`
        WarehouseID      string `json:"warehouse_id" validate:"required"`
        TotalStock       int    `json:"total_stock" validate:"required"`
        LowStockThreshold int  `json:"low_stock_threshold"`
}

// CreateStock creates a new stock item.
func (s *Service) CreateStock(ctx context.Context, input CreateStockInput) (*domain.StockItem, error) {
        if input.TotalStock < 0 {
                return nil, errors.New("VALIDATION_ERROR", "total_stock cannot be negative", 422)
        }

        stock := domain.NewStockItem(input.SKU, input.WarehouseID, input.TotalStock, input.LowStockThreshold)
        if err := s.stock.CreateStock(ctx, stock); err != nil {
                return nil, err
        }

        // Record inbound movement
        _ = s.movements.CreateMovement(ctx, domain.NewMovement(
                input.SKU, input.WarehouseID,
                domain.MovementInbound, input.TotalStock,
                "initial", "initial stock creation",
        ))

        return stock, nil
}

// GetStock returns the current stock for a SKU at a warehouse.
func (s *Service) GetStock(ctx context.Context, sku, warehouseID string) (*domain.StockItem, error) {
        return s.stock.GetStock(ctx, sku, warehouseID)
}

// AdjustStockInput holds fields for stock adjustment.
type AdjustStockInput struct {
        SKU         string `json:"sku" validate:"required"`
        WarehouseID string `json:"warehouse_id" validate:"required"`
        Quantity    int    `json:"quantity" validate:"required"`
        Reason      string `json:"reason"`
}

// AdjustStock adjusts total_stock by quantity (positive or negative).
func (s *Service) AdjustStock(ctx context.Context, input AdjustStockInput) (*domain.StockItem, error) {
        if input.Quantity == 0 {
                return nil, errors.New("VALIDATION_ERROR", "quantity must be non-zero", 422)
        }

        updated, err := s.stock.AdjustStock(ctx, input.SKU, input.WarehouseID, input.Quantity, input.Reason)
        if err != nil {
                return nil, err
        }

        // Record movement
        mtype := domain.MovementInbound
        if input.Quantity < 0 {
                mtype = domain.MovementOutbound
        }
        _ = s.movements.CreateMovement(ctx, domain.NewMovement(
                input.SKU, input.WarehouseID, mtype, input.Quantity,
                "adjustment", input.Reason,
        ))

        // Audit: record inventory adjustment
        oldVal := fmt.Sprintf("total=%d", updated.TotalStock-input.Quantity)
        newVal := fmt.Sprintf("total=%d", updated.TotalStock)
        s.auditor.Record("", audit.ActionInventoryAdjust, "stock", input.SKU+":"+input.WarehouseID, oldVal, newVal, input.Reason)

        // Check for low stock after adjustment
        if updated.IsLowStock() {
                s.publishEvent(ctx, "inventory.low_stock", &domain.InventoryEvent{
                        EventType:   "inventory.low_stock",
                        SKU:         input.SKU,
                        WarehouseID: input.WarehouseID,
                        Quantity:    updated.TotalStock,
                        Stock:       stockSnap(updated),
                })
        }

        return updated, nil
}

// --- Reservations ---

// ReserveInput holds fields for creating a reservation.
type ReserveInput struct {
        SKU         string        `json:"sku" validate:"required"`
        WarehouseID string        `json:"warehouse_id" validate:"required"`
        Quantity    int           `json:"quantity" validate:"required"`
        Reference   string        `json:"reference" validate:"required"`
        TTL         time.Duration `json:"ttl"`
}

// ReserveResult holds the result of a reservation.
type ReserveResult struct {
        Reservation *domain.Reservation
        Stock       *domain.StockItem
}

// Reserve creates a reservation and atomically increments reserved_stock.
func (s *Service) Reserve(ctx context.Context, input ReserveInput) (*ReserveResult, error) {
        if input.Quantity <= 0 {
                return nil, errors.New("VALIDATION_ERROR", "quantity must be positive", 422)
        }

        // Check for duplicate active reservation with same reference
        existing, err := s.reservations.GetActiveReservation(ctx, input.SKU, input.Reference)
        if err != nil {
                return nil, err
        }
        if existing != nil {
                return nil, errors.New("CONFLICT", "active reservation already exists for this reference", 409)
        }

        ttl := input.TTL
        if ttl <= 0 {
                ttl = defaultReservationTTL
        }

        reservation := domain.NewReservation(input.SKU, input.WarehouseID, input.Quantity, input.Reference, ttl)

        // Atomic: increment reserved_stock only if available >= quantity
        updated, err := s.stock.ReserveStock(ctx, input.SKU, input.WarehouseID, input.Quantity)
        if err != nil {
                // Business metric: inventory reservation failure
                if s.bizMetrics != nil {
                        s.bizMetrics.IncrementInventoryReservationFailure()
                }
                return nil, err
        }

        if err := s.reservations.CreateReservation(ctx, reservation); err != nil {
                // Rollback: release the stock we just reserved
                _, _ = s.stock.ReleaseStock(ctx, input.SKU, input.WarehouseID, input.Quantity)
                return nil, err
        }

        // Record movement
        _ = s.movements.CreateMovement(ctx, domain.NewMovement(
                input.SKU, input.WarehouseID,
                domain.MovementReserved, input.Quantity,
                input.Reference, "stock reservation",
        ))

        // Publish event
        s.publishEvent(ctx, "inventory.reserved", &domain.InventoryEvent{
                EventType:   "inventory.reserved",
                SKU:         input.SKU,
                WarehouseID: input.WarehouseID,
                Quantity:    input.Quantity,
                Reference:   input.Reference,
                Stock:       stockSnap(updated),
        })

        return &ReserveResult{Reservation: reservation, Stock: updated}, nil
}

// Release releases a reservation by ID, atomically decrementing reserved_stock.
func (s *Service) Release(ctx context.Context, reservationID string) (*domain.StockItem, error) {
        res, err := s.reservations.GetReservation(ctx, reservationID)
        if err != nil {
                return nil, err
        }
        if res.Status != domain.ReservationStatusActive {
                return nil, errors.New("BAD_REQUEST", "reservation is not active", 400)
        }

        if err := s.reservations.ReleaseReservation(ctx, reservationID); err != nil {
                return nil, err
        }

        updated, err := s.stock.ReleaseStock(ctx, res.SKU, res.WarehouseID, res.Quantity)
        if err != nil {
                return nil, err
        }

        // Record movement
        _ = s.movements.CreateMovement(ctx, domain.NewMovement(
                res.SKU, res.WarehouseID,
                domain.MovementReleased, res.Quantity,
                res.ID, "reservation released",
        ))

        // Publish event
        s.publishEvent(ctx, "inventory.released", &domain.InventoryEvent{
                EventType:   "inventory.released",
                SKU:         res.SKU,
                WarehouseID: res.WarehouseID,
                Quantity:    res.Quantity,
                Reference:   res.ID,
                Stock:       stockSnap(updated),
        })

        return updated, nil
}

// ReleaseByReference finds and releases all active reservations for a given reference
// (typically an order ID). Returns the count of reservations released.
// Individual release failures are silently skipped — best-effort release.
func (s *Service) ReleaseByReference(ctx context.Context, reference string) (int, error) {
        reservations, err := s.reservations.FindActiveReservationsByReference(ctx, reference)
        if err != nil {
                return 0, err
        }

        released := 0
        for _, res := range reservations {
                if relErr := s.reservations.ReleaseReservation(ctx, res.ID); relErr != nil {
                        continue
                }

                if _, stockErr := s.stock.ReleaseStock(ctx, res.SKU, res.WarehouseID, res.Quantity); stockErr != nil {
                        continue
                }

                _ = s.movements.CreateMovement(ctx, domain.NewMovement(
                        res.SKU, res.WarehouseID,
                        domain.MovementReleased, res.Quantity,
                        res.ID, "reservation released on payment failure",
                ))

                s.publishEvent(ctx, "inventory.released", &domain.InventoryEvent{
                        EventType:   "inventory.released",
                        SKU:         res.SKU,
                        WarehouseID: res.WarehouseID,
                        Quantity:    res.Quantity,
                        Reference:   res.ID,
                })

                released++
        }

        return released, nil
}

// Consume consumes a reservation by ID, atomically decrementing both total and reserved.
func (s *Service) Consume(ctx context.Context, reservationID string) (*domain.StockItem, error) {
        res, err := s.reservations.GetReservation(ctx, reservationID)
        if err != nil {
                return nil, err
        }
        if res.Status != domain.ReservationStatusActive {
                return nil, errors.New("BAD_REQUEST", "reservation is not active", 400)
        }

        // Mark reservation as consumed
        if err := s.reservations.ConsumeReservation(ctx, reservationID); err != nil {
                return nil, err
        }

        // Atomically decrement both total and reserved
        updated, err := s.stock.ConsumeStock(ctx, res.SKU, res.WarehouseID, res.Quantity)
        if err != nil {
                return nil, err
        }

        // Record movement
        _ = s.movements.CreateMovement(ctx, domain.NewMovement(
                res.SKU, res.WarehouseID,
                domain.MovementConsumed, res.Quantity,
                res.ID, "reservation consumed",
        ))

        return updated, nil
}

// ExpireReservations finds and expires all past-due active reservations for a SKU,
// releasing their reserved stock.
func (s *Service) ExpireReservations(ctx context.Context, sku string) (int, error) {
        expired, err := s.reservations.ExpireReservations(ctx, sku)
        if err != nil {
                return 0, err
        }

        for _, res := range expired {
                _, _ = s.stock.ReleaseStock(ctx, res.SKU, res.WarehouseID, res.Quantity)

                _ = s.movements.CreateMovement(ctx, domain.NewMovement(
                        res.SKU, res.WarehouseID,
                        domain.MovementReleased, res.Quantity,
                        res.ID, "reservation expired",
                ))

                s.publishEvent(ctx, "inventory.released", &domain.InventoryEvent{
                        EventType:   "inventory.released",
                        SKU:         res.SKU,
                        WarehouseID: res.WarehouseID,
                        Quantity:    res.Quantity,
                        Reference:   res.ID,
                })
        }

        return len(expired), nil
}

// GetReservation returns a single reservation by ID.
func (s *Service) GetReservation(ctx context.Context, id string) (*domain.Reservation, error) {
        return s.reservations.GetReservation(ctx, id)
}

// ListReservations returns paginated reservations for a SKU.
func (s *Service) ListReservations(ctx context.Context, sku, status string, p pagination.Params) ([]*domain.Reservation, int64, error) {
        return s.reservations.ListReservations(ctx, sku, status, p)
}

// --- Movements ---

// ListMovements returns paginated movement history for a SKU.
func (s *Service) ListMovements(ctx context.Context, sku string, p pagination.Params) ([]*domain.Movement, int64, error) {
        return s.movements.ListMovements(ctx, sku, p)
}

// --- Low Stock ---

// ListLowStock returns all stock items at or below their low-stock threshold.
func (s *Service) ListLowStock(ctx context.Context) ([]*domain.StockItem, error) {
        return s.stock.ListLowStock(ctx)
}

// --- Warehouses ---

// CreateWarehouseInput holds fields for creating a warehouse.
type CreateWarehouseInput struct {
        Name    string `json:"name" validate:"required"`
        Code    string `json:"code" validate:"required"`
        Address string `json:"address" validate:"required"`
}

// CreateWarehouse creates a new warehouse.
func (s *Service) CreateWarehouse(ctx context.Context, input CreateWarehouseInput) (*domain.Warehouse, error) {
        wh := domain.NewWarehouse(input.Name, input.Code, input.Address)
        if err := s.warehouses.Create(ctx, wh); err != nil {
                return nil, err
        }
        return wh, nil
}

// GetWarehouse returns a warehouse by ID.
func (s *Service) GetWarehouse(ctx context.Context, id string) (*domain.Warehouse, error) {
        return s.warehouses.FindByID(ctx, id)
}

// ListWarehouses returns a paginated warehouse listing.
func (s *Service) ListWarehouses(ctx context.Context, p pagination.Params) ([]*domain.Warehouse, int64, error) {
        return s.warehouses.List(ctx, p)
}

// --- Helpers ---

func (s *Service) publishEvent(ctx context.Context, subject string, event *domain.InventoryEvent) {
        if s.events != nil {
                _ = s.events.Publish(ctx, subject, event)
        }
}

func stockSnap(stock *domain.StockItem) *domain.StockSnap {
        return &domain.StockSnap{
                Total:     stock.TotalStock,
                Reserved:  stock.ReservedStock,
                Available: stock.AvailableStock(),
        }
}
