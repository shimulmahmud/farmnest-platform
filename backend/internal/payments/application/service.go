package application

import (
        "context"

        "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/telemetry"
)

// Service implements payment use cases with idempotent creation.
type Service struct {
        repo       domain.PaymentRepository
        providers  map[string]domain.PaymentProvider
        bizMetrics *telemetry.BusinessMetrics
}

// NewService creates a payment application service.
// providers maps provider name (e.g. "fake", "stripe") to its implementation.
func NewService(repo domain.PaymentRepository, providers map[string]domain.PaymentProvider) *Service {
        return &Service{
                repo:      repo,
                providers: providers,
        }
}

// SetBusinessMetrics injects business metrics counters for payment events.
func (s *Service) SetBusinessMetrics(m *telemetry.BusinessMetrics) {
        s.bizMetrics = m
}

// CreatePaymentInput holds the data needed to create a payment.
type CreatePaymentInput struct {
        OrderID  string `json:"order_id" validate:"required"`
        Provider string `json:"provider" validate:"required"`
        Amount   int    `json:"amount" validate:"required"`
        Currency string `json:"currency" validate:"required"`
}

// CreatePayment creates a new payment idempotently.
// If a payment already exists for this order+provider, it is returned as-is.
// Otherwise, a new payment is created via the provider and persisted.
func (s *Service) CreatePayment(ctx context.Context, input CreatePaymentInput) (*domain.Payment, error) {
        if input.OrderID == "" {
                return nil, errors.New("VALIDATION_ERROR", "order_id is required", 422)
        }
        if input.Provider == "" {
                return nil, errors.New("VALIDATION_ERROR", "provider is required", 422)
        }
        if input.Amount <= 0 {
                return nil, errors.New("VALIDATION_ERROR", "amount must be positive", 422)
        }
        if input.Currency == "" {
                return nil, errors.New("VALIDATION_ERROR", "currency is required", 422)
        }

        provider, ok := s.providers[input.Provider]
        if !ok {
                return nil, errors.New("VALIDATION_ERROR", "unsupported payment provider: "+input.Provider, 422)
        }

        // Idempotency: check if payment already exists for this order+provider.
        existing, err := s.repo.FindByOrderID(ctx, input.OrderID, input.Provider)
        if err == nil && existing != nil {
                return existing, nil
        }
        // ErrNotFound is expected for new payments; any other error is real.
        if err != nil {
                appErr, ok := errors.IsAppError(err)
                if !ok || appErr.Code != "NOT_FOUND" {
                        return nil, err
                }
        }

        // Create the payment entity in PENDING status.
        payment := domain.NewPayment(input.OrderID, input.Provider, input.Currency, input.Amount)

        // Call the provider to initiate the payment.
        result, err := provider.CreatePayment(ctx, payment)
        if err != nil {
                // Provider error: mark payment as FAILED and persist.
                payment.Status = domain.PaymentFailed
                createErr := s.repo.Create(ctx, payment)
                if createErr != nil {
                        // If even the failure record is a duplicate, return the existing one.
                        if createErr == domain.ErrDuplicatePayment {
                                return s.repo.FindByOrderID(ctx, input.OrderID, input.Provider)
                        }
                        return nil, errors.Wrap("INTERNAL_ERROR", "failed to record failed payment", 500, createErr)
                }
                // Business metric: payment failure (provider error)
                if s.bizMetrics != nil {
                        s.bizMetrics.IncrementPaymentFailure()
                }
                return nil, errors.Wrap("PAYMENT_PROVIDER_ERROR", "payment provider error", 502, err)
        }

        if !result.Success {
                // Provider returned a failure result.
                payment.Status = domain.PaymentFailed
                if result.ProviderRef != "" {
                        payment.ProviderRef = result.ProviderRef
                }
                payment.Metadata["provider_error"] = result.ErrorMessage
                createErr := s.repo.Create(ctx, payment)
                if createErr != nil {
                        if createErr == domain.ErrDuplicatePayment {
                                return s.repo.FindByOrderID(ctx, input.OrderID, input.Provider)
                        }
                        return nil, errors.Wrap("INTERNAL_ERROR", "failed to record failed payment", 500, createErr)
                }
                // Business metric: payment failure (provider rejected)
                if s.bizMetrics != nil {
                        s.bizMetrics.IncrementPaymentFailure()
                }
                return nil, errors.New("PAYMENT_FAILED", result.ErrorMessage, 422)
        }

        // Success: update payment with provider reference and set to AUTHORIZED.
        payment.ProviderRef = result.ProviderRef
        payment.Status = domain.PaymentAuthorized

        if err := s.repo.Create(ctx, payment); err != nil {
                if err == domain.ErrDuplicatePayment {
                        // Another request created it first; return the existing one.
                        return s.repo.FindByOrderID(ctx, input.OrderID, input.Provider)
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to persist payment", 500, err)
        }

        // Business metric: payment success
        if s.bizMetrics != nil {
                s.bizMetrics.IncrementPaymentSuccess()
        }

        return payment, nil
}

// VerifyPayment checks the payment status with the provider and updates MongoDB.
func (s *Service) VerifyPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
        payment, err := s.repo.FindByID(ctx, paymentID)
        if err != nil {
                return nil, err
        }

        provider, ok := s.providers[payment.Provider]
        if !ok {
                return nil, errors.New("VALIDATION_ERROR", "unsupported payment provider: "+payment.Provider, 422)
        }

        result, err := provider.VerifyPayment(ctx, payment)
        if err != nil {
                return nil, errors.Wrap("PAYMENT_PROVIDER_ERROR", "payment verification failed", 502, err)
        }

        if !result.Success {
                // Payment is not valid at the provider; mark as FAILED.
                _, updateErr := s.repo.UpdateStatus(ctx, paymentID, payment.Status, domain.PaymentFailed, "")
                if updateErr != nil {
                        return nil, updateErr
                }
                return nil, errors.New("PAYMENT_FAILED", result.ErrorMessage, 422)
        }

        // Verification succeeded: transition to PAID.
        newStatus := domain.PaymentPaid
        // If currently PENDING, go through AUTHORIZED first.
        if payment.Status == domain.PaymentPending {
                updated, err := s.repo.UpdateStatus(ctx, paymentID, payment.Status, domain.PaymentAuthorized, result.ProviderRef)
                if err != nil {
                        return nil, err
                }
                payment = updated
                newStatus = domain.PaymentPaid
        }

        updated, err := s.repo.UpdateStatus(ctx, paymentID, payment.Status, newStatus, result.ProviderRef)
        if err != nil {
                return nil, err
        }

        return updated, nil
}

// CapturePayment captures an authorized payment.
func (s *Service) CapturePayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
        payment, err := s.repo.FindByID(ctx, paymentID)
        if err != nil {
                return nil, err
        }

        if payment.Status != domain.PaymentAuthorized {
                return nil, errors.New("BAD_REQUEST", "payment must be in AUTHORIZED status to capture", 400)
        }

        provider, ok := s.providers[payment.Provider]
        if !ok {
                return nil, errors.New("VALIDATION_ERROR", "unsupported payment provider: "+payment.Provider, 422)
        }

        result, err := provider.CapturePayment(ctx, payment)
        if err != nil {
                return nil, errors.Wrap("PAYMENT_PROVIDER_ERROR", "payment capture failed", 502, err)
        }

        if !result.Success {
                return nil, errors.New("PAYMENT_FAILED", result.ErrorMessage, 422)
        }

        updated, err := s.repo.UpdateStatus(ctx, paymentID, domain.PaymentAuthorized, domain.PaymentPaid, result.ProviderRef)
        if err != nil {
                return nil, err
        }

        return updated, nil
}

// RefundPayment initiates a refund for a paid payment.
func (s *Service) RefundPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
        payment, err := s.repo.FindByID(ctx, paymentID)
        if err != nil {
                return nil, err
        }

        if payment.Status != domain.PaymentPaid {
                return nil, errors.New("BAD_REQUEST", "only PAID payments can be refunded", 400)
        }

        provider, ok := s.providers[payment.Provider]
        if !ok {
                return nil, errors.New("VALIDATION_ERROR", "unsupported payment provider: "+payment.Provider, 422)
        }

        result, err := provider.RefundPayment(ctx, payment)
        if err != nil {
                return nil, errors.Wrap("PAYMENT_PROVIDER_ERROR", "payment refund failed", 502, err)
        }

        if !result.Success {
                return nil, errors.New("PAYMENT_FAILED", result.ErrorMessage, 422)
        }

        // Transition PAID -> REFUND_PENDING -> REFUNDED atomically via two steps.
        updated, err := s.repo.UpdateStatus(ctx, paymentID, domain.PaymentPaid, domain.PaymentRefundPending, result.ProviderRef)
        if err != nil {
                return nil, err
        }

        // Immediately complete the refund.
        final, err := s.repo.UpdateStatus(ctx, paymentID, domain.PaymentRefundPending, domain.PaymentRefunded, result.ProviderRef)
        if err != nil {
                // Stuck in REFUND_PENDING; still return the intermediate state.
                return updated, nil
        }

        return final, nil
}

// GetPayment retrieves a payment by ID.
func (s *Service) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
        return s.repo.FindByID(ctx, id)
}

// GetPaymentByOrder retrieves the payment for an order+provider.
func (s *Service) GetPaymentByOrder(ctx context.Context, orderID, provider string) (*domain.Payment, error) {
        return s.repo.FindByOrderID(ctx, orderID, provider)
}
