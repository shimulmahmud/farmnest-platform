package transport

import "github.com/gofiber/fiber/v2"

// RegisterOrderRoutes mounts order endpoints on the given router.
func RegisterOrderRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler) {
	// Public routes
	router.Get("/api/v1/checkout/config", handler.CheckoutConfig)
	router.Get("/api/v1/track-order", handler.TrackOrder)

	// Customer-facing routes (require authentication)
	customer := router.Group("/api/v1/orders", authMW)

	router.Post("/api/v1/checkout/quote", authMW, handler.Quote)

	customer.Post("/", handler.CreateOrder)
	customer.Get("/", handler.ListOrders)
	customer.Get("/:id", handler.GetOrder)

	// Admin routes (require authentication + admin role)
	admin := router.Group("/api/v1/admin/orders", authMW, adminMW)

	admin.Get("/", handler.AdminListOrders)
	admin.Patch("/:id/status", handler.UpdateOrderStatus)
	admin.Post("/:id/ship", handler.ShipOrder)
	admin.Post("/:id/mark-paid", handler.MarkPaid)
}