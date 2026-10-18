package infrastructure

import (
        "context"
        "sync"

        "github.com/ecommerce/backend/internal/payments/domain"
)

// FakePaymentProvider is a test double that simulates payment provider behavior.
// It supports configurable success/failure/error/timeout modes.
type FakePaymentProvider struct {
        mu         sync.Mutex
        shouldFail  bool
        failMsg    string
        shouldError bool
        errMsg     string
        shouldTimeout bool

        // Track call counts per operation.
        calls map[string]int
}

// NewFakePaymentProvider creates a FakePaymentProvider that succeeds by default.
func NewFakePaymentProvider() *FakePaymentProvider {
        return &FakePaymentProvider{
                calls: make(map[string]int),
        }
}

// FailOnNext makes the next provider call return a failed result.
func (p *FakePaymentProvider) FailOnNext(msg string) {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.shouldFail = true
        p.failMsg = msg
}

// ErrorOnNext makes the next provider call return an error.
func (p *FakePaymentProvider) ErrorOnNext(msg string) {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.shouldError = true
        p.errMsg = msg
}

// TimeoutOnNext makes the next provider call simulate a timeout.
func (p *FakePaymentProvider) TimeoutOnNext() {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.shouldTimeout = true
}

// CallCount returns how many times an operation was called.
func (p *FakePaymentProvider) CallCount(op string) int {
        p.mu.Lock()
        defer p.mu.Unlock()
        return p.calls[op]
}

// Reset clears all configured behaviors and call counts.
func (p *FakePaymentProvider) Reset() {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.shouldFail = false
        p.failMsg = ""
        p.shouldError = false
        p.errMsg = ""
        p.shouldTimeout = false
        p.calls = make(map[string]int)
}

func (p *FakePaymentProvider) checkMode(ctx context.Context, op string) (*domain.ProviderResult, error) {
        p.mu.Lock()
        defer p.mu.Unlock()

        p.calls[op]++

        if p.shouldTimeout {
                p.shouldTimeout = false
                // Block until the context is cancelled, simulating a provider timeout.
                <-ctx.Done()
                return nil, ctx.Err()
        }

        if p.shouldError {
                p.shouldError = false
                msg := p.errMsg
                if msg == "" {
                        msg = "provider error"
                }
                return nil, &ProviderError{Message: msg}
        }

        if p.shouldFail {
                p.shouldFail = false
                msg := p.failMsg
                if msg == "" {
                        msg = "payment failed"
                }
                return &domain.ProviderResult{
                        Success:      false,
                        ErrorMessage: msg,
                }, nil
        }

        return nil, nil
}

func (p *FakePaymentProvider) CreatePayment(ctx context.Context, payment *domain.Payment) (*domain.ProviderResult, error) {
        if result, err := p.checkMode(ctx, "create"); result != nil || err != nil {
                if result != nil {
                        return result, nil
                }
                return nil, err
        }

        ref := "fake-ref-" + payment.ID[:8]
        return &domain.ProviderResult{
                Success:     true,
                ProviderRef: ref,
        }, nil
}

func (p *FakePaymentProvider) VerifyPayment(ctx context.Context, payment *domain.Payment) (*domain.ProviderResult, error) {
        if result, err := p.checkMode(ctx, "verify"); result != nil || err != nil {
                if result != nil {
                        return result, nil
                }
                return nil, err
        }

        return &domain.ProviderResult{
                Success:     true,
                ProviderRef: payment.ProviderRef,
        }, nil
}

func (p *FakePaymentProvider) CapturePayment(ctx context.Context, payment *domain.Payment) (*domain.ProviderResult, error) {
        if result, err := p.checkMode(ctx, "capture"); result != nil || err != nil {
                if result != nil {
                        return result, nil
                }
                return nil, err
        }

        return &domain.ProviderResult{
                Success:     true,
                ProviderRef: payment.ProviderRef,
        }, nil
}

func (p *FakePaymentProvider) RefundPayment(ctx context.Context, payment *domain.Payment) (*domain.ProviderResult, error) {
        if result, err := p.checkMode(ctx, "refund"); result != nil || err != nil {
                if result != nil {
                        return result, nil
                }
                return nil, err
        }

        ref := "fake-refund-" + payment.ID[:8]
        return &domain.ProviderResult{
                Success:     true,
                ProviderRef: ref,
        }, nil
}

// ProviderError represents an error from the payment provider.
type ProviderError struct {
        Message string
}

func (e *ProviderError) Error() string {
        return e.Message
}
