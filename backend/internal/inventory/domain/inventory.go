package domain

import (
	"time"

	"github.com/google/uuid"
)

// MovementType classifies inventory adjustments.
type MovementType string

const (
	MovementInbound    MovementType = "INBOUND"
	MovementOutbound   MovementType = "OUTBOUND"
	MovementAdjustment MovementType = "ADJUSTMENT"
	MovementReserved   MovementType = "RESERVED"
	MovementReleased   MovementType = "RELEASED"
	MovementConsumed   MovementType = "CONSUMED"
)

// ReservationStatus represents the state of a reservation.
type ReservationStatus string

const (
	ReservationStatusActive   ReservationStatus = "ACTIVE"
	ReservationStatusConsumed ReservationStatus = "CONSUMED"
	ReservationStatusReleased ReservationStatus = "RELEASED"
	ReservationStatusExpired  ReservationStatus = "EXPIRED"
)

// StockItem represents the authoritative inventory record for a variant SKU.
type StockItem struct {
	ID            string    `bson:"_id" json:"id"`
	SKU           string    `bson:"sku" json:"sku"`
	WarehouseID   string    `bson:"warehouse_id" json:"warehouse_id"`
	TotalStock    int       `bson:"total_stock" json:"total_stock"`
	ReservedStock int       `bson:"reserved_stock" json:"reserved_stock"`
	LowStockThreshold int   `bson:"low_stock_threshold" json:"low_stock_threshold"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
}

// AvailableStock returns total - reserved (never negative).
func (s *StockItem) AvailableStock() int {
	avail := s.TotalStock - s.ReservedStock
	if avail < 0 {
		return 0
	}
	return avail
}

// IsLowStock returns true if total stock is at or below the threshold.
func (s *StockItem) IsLowStock() bool {
	return s.LowStockThreshold > 0 && s.TotalStock <= s.LowStockThreshold
}

// NewStockItem creates a new StockItem with initial stock.
func NewStockItem(sku, warehouseID string, totalStock, lowStockThreshold int) *StockItem {
	now := time.Now().UTC()
	return &StockItem{
		ID:                uuid.New().String(),
		SKU:               sku,
		WarehouseID:       warehouseID,
		TotalStock:        totalStock,
		ReservedStock:     0,
		LowStockThreshold: lowStockThreshold,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// Movement is a historical record of inventory changes.
type Movement struct {
	ID          string       `bson:"_id" json:"id"`
	SKU         string       `bson:"sku" json:"sku"`
	WarehouseID string       `bson:"warehouse_id" json:"warehouse_id"`
	Type        MovementType `bson:"type" json:"type"`
	Quantity    int          `bson:"quantity" json:"quantity"`
	Reference   string       `bson:"reference" json:"reference"`
	Reason      string       `bson:"reason" json:"reason"`
	CreatedAt   time.Time    `bson:"created_at" json:"created_at"`
}

// NewMovement creates a new movement record.
func NewMovement(sku, warehouseID string, mtype MovementType, quantity int, reference, reason string) *Movement {
	return &Movement{
		ID:          uuid.New().String(),
		SKU:         sku,
		WarehouseID: warehouseID,
		Type:        mtype,
		Quantity:    quantity,
		Reference:   reference,
		Reason:      reason,
		CreatedAt:   time.Now().UTC(),
	}
}

// Reservation holds a temporary hold on stock.
type Reservation struct {
	ID          string            `bson:"_id" json:"id"`
	SKU         string            `bson:"sku" json:"sku"`
	WarehouseID string            `bson:"warehouse_id" json:"warehouse_id"`
	Quantity    int               `bson:"quantity" json:"quantity"`
	Status      ReservationStatus `bson:"status" json:"status"`
	Reference   string            `bson:"reference" json:"reference"`
	ExpiresAt   time.Time         `bson:"expires_at" json:"expires_at"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at" json:"updated_at"`
}

// IsExpired returns true if the reservation has passed its expiry time.
func (r *Reservation) IsExpired() bool {
	return time.Now().UTC().After(r.ExpiresAt)
}

// NewReservation creates a new reservation.
func NewReservation(sku, warehouseID string, quantity int, reference string, ttl time.Duration) *Reservation {
	now := time.Now().UTC()
	return &Reservation{
		ID:          uuid.New().String(),
		SKU:         sku,
		WarehouseID: warehouseID,
		Quantity:    quantity,
		Status:      ReservationStatusActive,
		Reference:   reference,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Warehouse represents a physical storage location.
type Warehouse struct {
	ID        string    `bson:"_id" json:"id"`
	Name      string    `bson:"name" json:"name"`
	Code      string    `bson:"code" json:"code"`
	Address   string    `bson:"address" json:"address"`
	IsActive  bool      `bson:"is_active" json:"is_active"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// NewWarehouse creates a new warehouse.
func NewWarehouse(name, code, address string) *Warehouse {
	now := time.Now().UTC()
	return &Warehouse{
		ID:        uuid.New().String(),
		Name:      name,
		Code:      code,
		Address:   address,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// InventoryEvent is published to NATS on state changes.
type InventoryEvent struct {
	EventType   string      `json:"event_type"`
	SKU         string      `json:"sku"`
	WarehouseID string      `json:"warehouse_id"`
	Quantity    int         `json:"quantity"`
	Reference   string      `json:"reference"`
	Stock       *StockSnap  `json:"stock,omitempty"`
}

// StockSnap is a snapshot of stock levels included in events.
type StockSnap struct {
	Total    int `json:"total"`
	Reserved int `json:"reserved"`
	Available int `json:"available"`
}