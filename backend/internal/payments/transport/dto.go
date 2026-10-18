package transport

import (
	"encoding/json"
	"time"

	"github.com/ecommerce/backend/internal/payments/domain"
)

// WebhookRequest is the JSON payload sent by a payment provider.
// Providers have different schemas; this is a normalized form.
type WebhookRequest struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Provider   string `json:"provider"`
	PaymentRef string `json:"payment_ref"`
	Amount     int    `json:"amount"`
	Currency   string `json:"currency"`
	Status     string `json:"status"`
	Timestamp  string `json:"timestamp"` // ISO 8601
}

// PaymentResponse is the API response for a payment.
type PaymentResponse struct {
	ID          string            `json:"id"`
	OrderID     string            `json:"order_id"`
	Provider    string            `json:"provider"`
	ProviderRef string            `json:"provider_reference"`
	Amount      int               `json:"amount"`
	Currency    string            `json:"currency"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// toDomainWebhook converts a WebhookRequest to a domain.WebhookEvent.
func toDomainWebhook(req *WebhookRequest, rawBody []byte) (*domain.WebhookEvent, error) {
	var ts time.Time
	if req.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			parsed, err = time.Parse("2006-01-02T15:04:05Z07:00", req.Timestamp)
			if err != nil {
				parsed = time.Now().UTC()
			}
		}
		ts = parsed
	} else {
		ts = time.Now().UTC()
	}

	return &domain.WebhookEvent{
		ID:         req.ID,
		Type:       domain.WebhookEventType(req.Type),
		Provider:   req.Provider,
		PaymentRef: req.PaymentRef,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Status:     req.Status,
		Timestamp:  ts,
		RawPayload: rawBody,
	}, nil
}

// toPaymentResponse converts a domain.Payment to a PaymentResponse.
func toPaymentResponse(p *domain.Payment) *PaymentResponse {
	return &PaymentResponse{
		ID:          p.ID,
		OrderID:     p.OrderID,
		Provider:    p.Provider,
		ProviderRef: p.ProviderRef,
		Amount:      p.Amount,
		Currency:    p.Currency,
		Status:      string(p.Status),
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// parseWebhookBody parses the raw body into a WebhookRequest.
func parseWebhookBody(rawBody []byte) (*WebhookRequest, error) {
	var req WebhookRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		return nil, err
	}
	return &req, nil
}
