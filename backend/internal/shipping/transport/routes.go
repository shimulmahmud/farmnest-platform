package transport

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterShippingRoutes registers all shipping HTTP routes.
func RegisterShippingRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler) {
	// Customer-facing routes (authenticated)
	customer := router.Group("/api/v1/shipping", authMW)
	customer.Post("/rates", handler.CalculateRate)
	customer.Get("/:id", handler.GetShipment)
	customer.Get("/order/:order_id", handler.GetShipmentByOrder)
	customer.Get("/", handler.ListShipments)
	customer.Post("/:id/track", handler.TrackShipment)
	customer.Post("/:id/cancel", handler.CancelShipment)

	// Admin routes
	admin := router.Group("/api/v1/admin/shipments", authMW, adminMW)
	admin.Post("/", handler.CreateShipment)
	admin.Get("/", handler.AdminListShipments)
	admin.Patch("/:id/status", handler.UpdateShipmentStatus)
}
