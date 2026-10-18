package transport

import (
	"time"

	invApp "github.com/ecommerce/backend/internal/inventory/application"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/ecommerce/backend/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for inventory endpoints.
type Handler struct {
	service   *invApp.Service
	validator *validator.Validator
}

// NewHandler creates an inventory transport handler.
func NewHandler(service *invApp.Service) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

// CreateStock handles POST /api/v1/admin/inventory/stock.
func (h *Handler) CreateStock(c *fiber.Ctx) error {
	var req CreateStockRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	if err := h.validator.ValidateRequired(map[string]string{
		"sku":          req.SKU,
		"warehouse_id": req.WarehouseID,
	}); err != nil {
		return response.Error(c, err)
	}
	if req.TotalStock < 0 {
		return response.Error(c, errors.New("VALIDATION_ERROR", "total_stock cannot be negative", 422))
	}

	stock, err := h.service.CreateStock(c.Context(), invApp.CreateStockInput{
		SKU:               req.SKU,
		WarehouseID:       req.WarehouseID,
		TotalStock:        req.TotalStock,
		LowStockThreshold: req.LowStockThreshold,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, toStockResponse(stock))
}

// GetStock handles GET /api/v1/admin/inventory/stock/:sku/:warehouse_id.
func (h *Handler) GetStock(c *fiber.Ctx) error {
	sku := c.Params("sku")
	warehouseID := c.Params("warehouse_id")
	if sku == "" || warehouseID == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "sku and warehouse_id are required", 422))
	}

	stock, err := h.service.GetStock(c.Context(), sku, warehouseID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toStockResponse(stock))
}

// AdjustStock handles POST /api/v1/admin/inventory/stock/adjust.
func (h *Handler) AdjustStock(c *fiber.Ctx) error {
	var req AdjustStockRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	if err := h.validator.ValidateRequired(map[string]string{
		"sku":          req.SKU,
		"warehouse_id": req.WarehouseID,
	}); err != nil {
		return response.Error(c, err)
	}
	if req.Quantity == 0 {
		return response.Error(c, errors.New("VALIDATION_ERROR", "quantity must be non-zero", 422))
	}

	stock, err := h.service.AdjustStock(c.Context(), invApp.AdjustStockInput{
		SKU:         req.SKU,
		WarehouseID: req.WarehouseID,
		Quantity:    req.Quantity,
		Reason:      req.Reason,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toStockResponse(stock))
}

// Reserve handles POST /api/v1/admin/inventory/reserve.
func (h *Handler) Reserve(c *fiber.Ctx) error {
	var req ReserveRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	if err := h.validator.ValidateRequired(map[string]string{
		"sku":          req.SKU,
		"warehouse_id": req.WarehouseID,
		"reference":    req.Reference,
	}); err != nil {
		return response.Error(c, err)
	}
	if req.Quantity <= 0 {
		return response.Error(c, errors.New("VALIDATION_ERROR", "quantity must be positive", 422))
	}

	var ttl time.Duration
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}

	result, err := h.service.Reserve(c.Context(), invApp.ReserveInput{
		SKU:         req.SKU,
		WarehouseID: req.WarehouseID,
		Quantity:    req.Quantity,
		Reference:   req.Reference,
		TTL:         ttl,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, ReserveResponse{
		Reservation: ptrReservationResp(toReservationResponse(result.Reservation)),
		Stock:       ptrStockResp(toStockResponse(result.Stock)),
	})
}

// Release handles POST /api/v1/admin/inventory/release/:id.
func (h *Handler) Release(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "reservation id is required", 422))
	}

	stock, err := h.service.Release(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toStockResponse(stock))
}

// Consume handles POST /api/v1/admin/inventory/consume/:id.
func (h *Handler) Consume(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "reservation id is required", 422))
	}

	stock, err := h.service.Consume(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toStockResponse(stock))
}

// GetReservation handles GET /api/v1/admin/inventory/reservations/:id.
func (h *Handler) GetReservation(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "reservation id is required", 422))
	}

	res, err := h.service.GetReservation(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toReservationResponse(res))
}

// ListReservations handles GET /api/v1/admin/inventory/reservations.
func (h *Handler) ListReservations(c *fiber.Ctx) error {
	sku := c.Query("sku") // optional: empty = all SKUs, newest first

	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}

	reservations, total, err := h.service.ListReservations(c.Context(), sku, c.Query("status"), p)
	if err != nil {
		return response.Error(c, err)
	}

	resp := make([]ReservationResponse, len(reservations))
	for i, r := range reservations {
		resp[i] = toReservationResponse(r)
	}

	meta := response.PaginatedMeta(p.Page, p.PerPage, total)
	return response.OKWithMeta(c, resp, meta)
}

// ListMovements handles GET /api/v1/admin/inventory/movements.
func (h *Handler) ListMovements(c *fiber.Ctx) error {
	sku := c.Query("sku") // optional: empty = all SKUs, newest first

	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}

	movements, total, err := h.service.ListMovements(c.Context(), sku, p)
	if err != nil {
		return response.Error(c, err)
	}

	resp := make([]MovementResponse, len(movements))
	for i, m := range movements {
		resp[i] = toMovementResponse(m)
	}

	meta := response.PaginatedMeta(p.Page, p.PerPage, total)
	return response.OKWithMeta(c, resp, meta)
}

// ListLowStock handles GET /api/v1/admin/inventory/low-stock.
func (h *Handler) ListLowStock(c *fiber.Ctx) error {
	items, err := h.service.ListLowStock(c.Context())
	if err != nil {
		return response.Error(c, err)
	}

	resp := make([]StockResponse, len(items))
	for i, item := range items {
		resp[i] = toStockResponse(item)
	}

	return response.OK(c, resp)
}

// --- Warehouses ---

// CreateWarehouse handles POST /api/v1/admin/inventory/warehouses.
func (h *Handler) CreateWarehouse(c *fiber.Ctx) error {
	var req CreateWarehouseRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	if err := h.validator.ValidateRequired(map[string]string{
		"name":    req.Name,
		"code":    req.Code,
		"address": req.Address,
	}); err != nil {
		return response.Error(c, err)
	}

	wh, err := h.service.CreateWarehouse(c.Context(), invApp.CreateWarehouseInput{
		Name:    req.Name,
		Code:    req.Code,
		Address: req.Address,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, toWarehouseResponse(wh))
}

// ListWarehouses handles GET /api/v1/admin/inventory/warehouses.
func (h *Handler) ListWarehouses(c *fiber.Ctx) error {
	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}

	warehouses, total, err := h.service.ListWarehouses(c.Context(), p)
	if err != nil {
		return response.Error(c, err)
	}

	resp := make([]WarehouseResponse, len(warehouses))
	for i, w := range warehouses {
		resp[i] = toWarehouseResponse(w)
	}

	meta := response.PaginatedMeta(p.Page, p.PerPage, total)
	return response.OKWithMeta(c, resp, meta)
}

func ptrStockResp(s StockResponse) *StockResponse { return &s }
func ptrReservationResp(r ReservationResponse) *ReservationResponse { return &r }
