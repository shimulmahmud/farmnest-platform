package domain

import (
        "context"

        "github.com/ecommerce/backend/pkg/pagination"
)

// StockRepository defines atomic inventory persistence operations.
type StockRepository interface {
        // CreateStock inserts a new stock item (fails on duplicate SKU+warehouse).
        CreateStock(ctx context.Context, stock *StockItem) error

        // GetStock retrieves the stock item for a SKU at a warehouse.
        GetStock(ctx context.Context, sku, warehouseID string) (*StockItem, error)

        // AdjustStock atomically adds quantity to total_stock. quantity can be negative.
        // Returns the updated stock item. Must prevent negative total_stock.
        AdjustStock(ctx context.Context, sku, warehouseID string, quantity int, reason string) (*StockItem, error)

        // ReserveStock atomically increments reserved_stock if available >= quantity.
        // Returns (updated stock, nil) on success or (nil, error) if insufficient stock.
        ReserveStock(ctx context.Context, sku, warehouseID string, quantity int) (*StockItem, error)

        // ReleaseStock atomically decrements reserved_stock.
        // Returns the updated stock item.
        ReleaseStock(ctx context.Context, sku, warehouseID string, quantity int) (*StockItem, error)

        // ConsumeStock atomically decrements both total_stock and reserved_stock.
        // Returns the updated stock item.
        ConsumeStock(ctx context.Context, sku, warehouseID string, quantity int) (*StockItem, error)

        // ListLowStock returns stock items at or below their low-stock threshold.
        ListLowStock(ctx context.Context) ([]*StockItem, error)
}

// ReservationRepository defines reservation persistence operations.
type ReservationRepository interface {
        // CreateReservation inserts a new reservation.
        CreateReservation(ctx context.Context, reservation *Reservation) error

        // GetReservation retrieves a reservation by ID.
        GetReservation(ctx context.Context, id string) (*Reservation, error)

        // GetActiveReservation retrieves a non-expired, active reservation by reference.
        // Used to prevent duplicate reservations for the same reference.
        GetActiveReservation(ctx context.Context, sku, reference string) (*Reservation, error)

        // FindActiveReservationsByReference returns all active reservations for a given reference.
        // Used to find and release all reservations tied to an order.
        FindActiveReservationsByReference(ctx context.Context, reference string) ([]*Reservation, error)

        // ReleaseReservation marks a reservation as released and updates timestamps.
        ReleaseReservation(ctx context.Context, id string) error

        // ConsumeReservation marks a reservation as consumed and updates timestamps.
        ConsumeReservation(ctx context.Context, id string) error

        // ExpireReservations marks all expired active reservations for a SKU as expired.
        // Returns the list of expired reservations that were updated.
        ExpireReservations(ctx context.Context, sku string) ([]*Reservation, error)

        // ListReservations returns reservations for a SKU with optional status filter.
        ListReservations(ctx context.Context, sku, status string, p pagination.Params) ([]*Reservation, int64, error)
}

// MovementRepository defines movement history persistence.
type MovementRepository interface {
        // CreateMovement appends a movement record.
        CreateMovement(ctx context.Context, movement *Movement) error

        // ListMovements returns movement history for a SKU.
        ListMovements(ctx context.Context, sku string, p pagination.Params) ([]*Movement, int64, error)
}

// WarehouseRepository defines warehouse persistence.
type WarehouseRepository interface {
        Create(ctx context.Context, warehouse *Warehouse) error
        FindByID(ctx context.Context, id string) (*Warehouse, error)
        FindByCode(ctx context.Context, code string) (*Warehouse, error)
        Update(ctx context.Context, warehouse *Warehouse) error
        List(ctx context.Context, p pagination.Params) ([]*Warehouse, int64, error)
}

// EventPublisher publishes inventory events to NATS.
type EventPublisher interface {
        Publish(ctx context.Context, subject string, event *InventoryEvent) error
}
