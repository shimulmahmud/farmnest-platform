package transport

import (
        "github.com/gofiber/fiber/v2"
)

// RegisterWebhookRoutes registers payment webhook routes.
// Webhook endpoints are public (no auth) since they are called by external providers.
func RegisterWebhookRoutes(router fiber.Router, handler *WebhookHandler) {
        webhooks := router.Group("/api/v1/webhooks")
        webhooks.Post("/payment/:provider", handler.HandleWebhook)
}