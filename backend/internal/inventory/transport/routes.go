package transport

import "github.com/gofiber/fiber/v2"

// RegisterInventoryRoutes mounts inventory endpoints on the given router.
func RegisterInventoryRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler) {
	inv := router.Group("/api/v1/admin/inventory", authMW, adminMW)

	// Stock
	inv.Post("/stock", handler.CreateStock)
	inv.Get("/stock/:sku/:warehouse_id", handler.GetStock)
	inv.Post("/stock/adjust", handler.AdjustStock)

	// Reservations
	inv.Post("/reserve", handler.Reserve)
	inv.Post("/release/:id", handler.Release)
	inv.Post("/consume/:id", handler.Consume)
	inv.Get("/reservations/:id", handler.GetReservation)
	inv.Get("/reservations", handler.ListReservations)

	// Movements
	inv.Get("/movements", handler.ListMovements)

	// Low stock
	inv.Get("/low-stock", handler.ListLowStock)

	// Warehouses
	inv.Post("/warehouses", handler.CreateWarehouse)
	inv.Get("/warehouses", handler.ListWarehouses)
}
