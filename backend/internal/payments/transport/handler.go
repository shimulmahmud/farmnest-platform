package transport

import (
	"github.com/ecommerce/backend/internal/payments/application"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// WebhookHandler handles payment webhook HTTP requests.
type WebhookHandler struct {
	webhookService *application.WebhookService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(webhookService *application.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookService: webhookService}
}

// HandleWebhook processes POST /api/v1/webhooks/payment/:provider
// The handler reads the raw body, extracts the signature header, and
// delegates to the webhook service.
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	provider := c.Params("provider")
	rawBody := c.Body()

	// Extract signature from header. Common headers: X-Webhook-Signature, Stripe-Signature, etc.
	signature := c.Get("X-Webhook-Signature")
	if signature == "" {
		signature = c.Get("X-Signature")
	}

	// Parse the webhook body.
	req, err := parseWebhookBody(rawBody)
	if err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid webhook payload", 422, err))
	}

	// Convert to domain event.
	webhook, err := toDomainWebhook(req, rawBody)
	if err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid webhook data", 422, err))
	}

	// Process the webhook.
	payment, err := h.webhookService.ProcessWebhook(c.Context(), provider, signature, rawBody, webhook)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toPaymentResponse(payment))
}
