package transport

import (
	"time"

	"github.com/ecommerce/backend/internal/inventory/domain"
)

// --- Request DTOs ---

// CreateStockRequest is the JSON body for POST /api/v1/admin/inventory/stock.
type CreateStockRequest struct {
	SKU               string `json:"sku" validate:"required"`
	WarehouseID       string `json:"warehouse_id" validate:"required"`
	TotalStock        int    `json:"total_stock" validate:"required"`
	LowStockThreshold int    `json:"low_stock_threshold"`
}

// AdjustStockRequest is the JSON body for POST /api/v1/admin/inventory/stock/adjust.
type AdjustStockRequest struct {
	SKU         string `json:"sku" validate:"required"`
	WarehouseID string `json:"warehouse_id" validate:"required"`
	Quantity    int    `json:"quantity" validate:"required"`
	Reason      string `json:"reason"`
}

// ReserveRequest is the JSON body for POST /api/v1/admin/inventory/reserve.
type ReserveRequest struct {
	SKU         string `json:"sku" validate:"required"`
	WarehouseID string `json:"warehouse_id" validate:"required"`
	Quantity    int    `json:"quantity" validate:"required"`
	Reference   string `json:"reference" validate:"required"`
	TTLSeconds  int64  `json:"ttl_seconds"`
}

// CreateWarehouseRequest is the JSON body for POST /api/v1/admin/inventory/warehouses.
type CreateWarehouseRequest struct {
	Name    string `json:"name" validate:"required"`
	Code    string `json:"code" validate:"required"`
	Address string `json:"address" validate:"required"`
}

// --- Response DTOs ---

// StockResponse is the stock data returned in API responses.
type StockResponse struct {
	ID                string `json:"id"`
	SKU               string `json:"sku"`
	WarehouseID       string `json:"warehouse_id"`
	TotalStock        int    `json:"total_stock"`
	ReservedStock     int    `json:"reserved_stock"`
	AvailableStock    int    `json:"available_stock"`
	LowStockThreshold int    `json:"low_stock_threshold"`
	IsLowStock        bool   `json:"is_low_stock"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// ReservationResponse is the reservation data returned in API responses.
type ReservationResponse struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	WarehouseID string `json:"warehouse_id"`
	Quantity    int    `json:"quantity"`
	Status      string `json:"status"`
	Reference   string `json:"reference"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ReserveResponse combines the reservation with updated stock.
type ReserveResponse struct {
	Reservation *ReservationResponse `json:"reservation"`
	Stock       *StockResponse       `json:"stock"`
}

// MovementResponse is the movement data returned in API responses.
type MovementResponse struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	WarehouseID string `json:"warehouse_id"`
	Type        string `json:"type"`
	Quantity    int    `json:"quantity"`
	Reference   string `json:"reference"`
	Reason      string `json:"reason"`
	CreatedAt   string `json:"created_at"`
}

// WarehouseResponse is the warehouse data returned in API responses.
type WarehouseResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Address   string `json:"address"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// --- Mappers ---

func toStockResponse(s *domain.StockItem) StockResponse {
	return StockResponse{
		ID:                s.ID,
		SKU:               s.SKU,
		WarehouseID:       s.WarehouseID,
		TotalStock:        s.TotalStock,
		ReservedStock:     s.ReservedStock,
		AvailableStock:    s.AvailableStock(),
		LowStockThreshold: s.LowStockThreshold,
		IsLowStock:        s.IsLowStock(),
		CreatedAt:         s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toReservationResponse(r *domain.Reservation) ReservationResponse {
	return ReservationResponse{
		ID:          r.ID,
		SKU:         r.SKU,
		WarehouseID: r.WarehouseID,
		Quantity:    r.Quantity,
		Status:      string(r.Status),
		Reference:   r.Reference,
		ExpiresAt:   r.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toMovementResponse(m *domain.Movement) MovementResponse {
	return MovementResponse{
		ID:          m.ID,
		SKU:         m.SKU,
		WarehouseID: m.WarehouseID,
		Type:        string(m.Type),
		Quantity:    m.Quantity,
		Reference:   m.Reference,
		Reason:      m.Reason,
		CreatedAt:   m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toWarehouseResponse(w *domain.Warehouse) WarehouseResponse {
	return WarehouseResponse{
		ID:        w.ID,
		Name:      w.Name,
		Code:      w.Code,
		Address:   w.Address,
		IsActive:  w.IsActive,
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: w.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
