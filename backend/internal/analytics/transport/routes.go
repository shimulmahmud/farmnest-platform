package transport

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterAnalyticsRoutes registers analytics endpoints.
// All analytics routes require authentication and admin privileges.
func RegisterAnalyticsRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler) {
	analytics := router.Group("/api/v1/analytics", authMW, adminMW)

	analytics.Get("/overview", handler.GetOverview)
	analytics.Get("/top-products", handler.GetTopProducts)
	analytics.Get("/trends/:event_type", handler.GetDailyTrend)
}
