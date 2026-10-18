package application

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/backend/internal/payments/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

// WebhookService processes incoming payment webhooks.
// Flow: verify signature -> parse event -> idempotency check ->
//	      find payment -> persist webhook event -> update payment -> publish event.
type WebhookService struct {
	repo           domain.PaymentRepository
	verifier       domain.SignatureVerifier
	eventStore     domain.WebhookEventStore
	eventPublisher domain.PaymentEventPublisher
}

// NewWebhookService creates a webhook processing service.
func NewWebhookService(
	repo domain.PaymentRepository,
	verifier domain.SignatureVerifier,
	eventStore domain.WebhookEventStore,
	eventPublisher domain.PaymentEventPublisher,
) *WebhookService {
	return &WebhookService{
		repo:           repo,
		verifier:       verifier,
		eventStore:     eventStore,
		eventPublisher: eventPublisher,
	}
}

// ProcessWebhook handles an incoming payment webhook end-to-end.
// It returns the updated payment if the webhook was successfully processed,
// or an error if something went wrong. Idempotent by webhook ID.
func (s *WebhookService) ProcessWebhook(ctx context.Context, provider, signature string, rawPayload []byte, webhook *domain.WebhookEvent) (*domain.Payment, error) {
	// 1. Verify signature.
	if err := s.verifier.Verify(ctx, provider, signature, rawPayload); err != nil {
		return nil, err
	}

	// 2. Validate webhook event fields.
	if err := validateWebhookEvent(webhook); err != nil {
		return nil, err
	}

	// 3. Ensure webhook provider matches URL param.
	if webhook.Provider != provider {
		return nil, errors.New("BAD_REQUEST", "webhook provider does not match URL parameter", 400)
	}

	// 4. Idempotency check.
	processed, err := s.eventStore.HasBeenProcessed(ctx, webhook.ID)
	if err != nil {
		return nil, err
	}
	if processed {
		// Already processed — return 200 OK (not an error). Find the current payment state.
		return s.repo.FindByProviderRef(ctx, provider, webhook.PaymentRef)
	}

	// 5. Find the payment by provider reference.
	payment, err := s.repo.FindByProviderRef(ctx, provider, webhook.PaymentRef)
	if err != nil {
		return nil, err
	}

	// 6. Determine target status and validate the transition.
	targetStatus := domain.MapWebhookTypeToPaymentStatus(webhook.Type)
	if targetStatus == "" {
		return nil, errors.New("BAD_REQUEST", "unknown webhook event type: "+string(webhook.Type), 400)
	}

	fromStatus := payment.Status

	// 7. Check if the payment is already in the target state (no-op).
	if payment.Status == targetStatus {
		err = s.recordProcessed(ctx, webhook, payment.ID)
		if err != nil {
			return nil, err
		}
		return payment, nil
	}

	// 8. Validate transition is valid for webhooks.
	if !isValidWebhookTransition(payment.Status, targetStatus) {
		// Out-of-order or no-op: record the webhook but don't change status.
		err = s.recordProcessed(ctx, webhook, payment.ID)
		if err != nil {
			return nil, err
		}
		return payment, nil
	}

	// 9. Update payment status.
	// For refund webhook, go through REFUND_PENDING first if currently PAID.
	if targetStatus == domain.PaymentRefunded && payment.Status == domain.PaymentPaid {
		_, updateErr := s.repo.UpdateStatus(ctx, payment.ID, domain.PaymentPaid, domain.PaymentRefundPending, webhook.PaymentRef)
		if updateErr != nil {
			appErr, ok := errors.IsAppError(updateErr)
			if ok && appErr.Code == "CONFLICT" {
				_ = s.recordProcessed(ctx, webhook, payment.ID)
				return s.repo.FindByID(ctx, payment.ID)
			}
			return nil, updateErr
		}
		updated, err := s.repo.UpdateStatus(ctx, payment.ID, domain.PaymentRefundPending, domain.PaymentRefunded, webhook.PaymentRef)
		if err != nil {
			_ = s.recordProcessed(ctx, webhook, payment.ID)
			return s.repo.FindByID(ctx, payment.ID)
		}
		payment = updated
	} else {
		updated, err := s.repo.UpdateStatus(ctx, payment.ID, payment.Status, targetStatus, webhook.PaymentRef)
		if err != nil {
			appErr, ok := errors.IsAppError(err)
			if ok && appErr.Code == "CONFLICT" {
				_ = s.recordProcessed(ctx, webhook, payment.ID)
				return s.repo.FindByID(ctx, payment.ID)
			}
			return nil, err
		}
		payment = updated
	}

	// 10. Record the webhook as processed.
	if err := s.recordProcessed(ctx, webhook, payment.ID); err != nil {
		return nil, err
	}

	// 11. Publish payment event (best-effort).
	if s.eventPublisher != nil {
		evt := &domain.PaymentEvent{
			EventType:   string(webhook.Type),
			PaymentID:   payment.ID,
			OrderID:     payment.OrderID,
			Provider:    provider,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
			FromStatus:  string(fromStatus),
			ToStatus:    string(targetStatus),
			ProviderRef: webhook.PaymentRef,
		}
		_ = s.eventPublisher.Publish(ctx, evt)
	}

	return payment, nil
}

// recordProcessed marks a webhook as processed.
func (s *WebhookService) recordProcessed(ctx context.Context, webhook *domain.WebhookEvent, paymentID string) error {
	pw := &domain.ProcessedWebhook{
		ID:          fmt.Sprintf("we-%s", webhook.ID),
		WebhookID:   webhook.ID,
		Provider:    webhook.Provider,
		EventType:   string(webhook.Type),
		PaymentID:   paymentID,
		ProcessedAt: time.Now().UTC(),
	}
	return s.eventStore.MarkProcessed(ctx, pw)
}

// validateWebhookEvent checks that required webhook fields are present.
func validateWebhookEvent(w *domain.WebhookEvent) error {
	if w.ID == "" {
		return errors.New("VALIDATION_ERROR", "webhook id is required", 422)
	}
	if w.Type == "" {
		return errors.New("VALIDATION_ERROR", "webhook event type is required", 422)
	}
	if w.Provider == "" {
		return errors.New("VALIDATION_ERROR", "webhook provider is required", 422)
	}
	if w.PaymentRef == "" {
		return errors.New("VALIDATION_ERROR", "webhook payment_ref is required", 422)
	}
	return nil
}

// isValidWebhookTransition checks whether a webhook-driven status transition is valid.
//   - payment.completed: PENDING|AUTHORIZED -> PAID
//   - payment.failed:    PENDING|AUTHORIZED -> FAILED
//   - payment.refunded:  PAID|REFUND_PENDING -> REFUNDED
func isValidWebhookTransition(current, target domain.PaymentStatus) bool {
	switch target {
	case domain.PaymentPaid:
		return current == domain.PaymentPending || current == domain.PaymentAuthorized
	case domain.PaymentFailed:
		return current == domain.PaymentPending || current == domain.PaymentAuthorized
	case domain.PaymentRefunded:
		return current == domain.PaymentPaid || current == domain.PaymentRefundPending
	default:
		return false
	}
}
