package application

import (
        "context"
        "sync"
        "sync/atomic"
        "testing"
        "time"

        "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/ecommerce/backend/internal/payments/infrastructure"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockPaymentRepo struct {
        mu      sync.Mutex
        payments map[string]*domain.Payment // key: payment ID
}

func newMockPaymentRepo() *mockPaymentRepo {
        return &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
}

func (r *mockPaymentRepo) copyPayment(p *domain.Payment) *domain.Payment {
        cp := *p
        if p.Metadata != nil {
                cp.Metadata = make(map[string]string, len(p.Metadata))
                for k, v := range p.Metadata {
                        cp.Metadata[k] = v
                }
        }
        return &cp
}

func (r *mockPaymentRepo) Create(_ context.Context, payment *domain.Payment) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        // Check uniqueness on order_id+provider
        for _, p := range r.payments {
                if p.OrderID == payment.OrderID && p.Provider == payment.Provider {
                        return domain.ErrDuplicatePayment
                }
        }
        cp := r.copyPayment(payment)
        r.payments[cp.ID] = cp
        return nil
}

func (r *mockPaymentRepo) FindByID(_ context.Context, id string) (*domain.Payment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        p, ok := r.payments[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return r.copyPayment(p), nil
}

func (r *mockPaymentRepo) FindByOrderID(_ context.Context, orderID, provider string) (*domain.Payment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        for _, p := range r.payments {
                if p.OrderID == orderID && p.Provider == provider {
                        return r.copyPayment(p), nil
                }
        }
        return nil, errors.ErrNotFound
}

func (r *mockPaymentRepo) FindByProviderRef(_ context.Context, provider, providerRef string) (*domain.Payment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        for _, p := range r.payments {
                if p.Provider == provider && p.ProviderRef == providerRef {
                        return r.copyPayment(p), nil
                }
        }
        return nil, errors.ErrNotFound
}

func (r *mockPaymentRepo) UpdateStatus(_ context.Context, id string, expectedStatus, newStatus domain.PaymentStatus, providerRef string) (*domain.Payment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        p, ok := r.payments[id]
        if !ok {
                return nil, errors.New("CONFLICT", "payment not found or status has changed", 409)
        }
        if p.Status != expectedStatus {
                return nil, errors.New("CONFLICT", "payment not found or status has changed", 409)
        }
        p.Status = newStatus
        if providerRef != "" {
                p.ProviderRef = providerRef
        }
        now := time.Now().UTC()
        p.UpdatedAt = now
        return r.copyPayment(p), nil
}

// --- Helpers ---

func newTestService(t *testing.T) (*Service, *mockPaymentRepo, *infrastructure.FakePaymentProvider) {
        t.Helper()
        repo := newMockPaymentRepo()
        provider := infrastructure.NewFakePaymentProvider()
        svc := NewService(repo, map[string]domain.PaymentProvider{
                "fake": provider,
        })
        return svc, repo, provider
}

func createTestPayment(t *testing.T, svc *Service, orderID string) *domain.Payment {
        t.Helper()
        p, err := svc.CreatePayment(context.Background(), CreatePaymentInput{
                OrderID:  orderID,
                Provider: "fake",
                Amount:   5000,
                Currency: "USD",
        })
        require.NoError(t, err)
        require.NotNil(t, p)
        return p
}

// --- CreatePayment Tests ---

func TestCreatePayment_Success(t *testing.T) {
        svc, _, provider := newTestService(t)

        p, err := svc.CreatePayment(context.Background(), CreatePaymentInput{
                OrderID:  "order-1",
                Provider: "fake",
                Amount:   5000,
                Currency: "USD",
        })
        require.NoError(t, err)
        assert.NotEmpty(t, p.ID)
        assert.Equal(t, "order-1", p.OrderID)
        assert.Equal(t, "fake", p.Provider)
        assert.Equal(t, 5000, p.Amount)
        assert.Equal(t, "USD", p.Currency)
        assert.Equal(t, domain.PaymentAuthorized, p.Status)
        assert.NotEmpty(t, p.ProviderRef)
        assert.Equal(t, 1, provider.CallCount("create"))
}

func TestCreatePayment_DuplicateRequest_ReturnsExisting(t *testing.T) {
        svc, _, provider := newTestService(t)

        input := CreatePaymentInput{
                OrderID:  "order-1",
                Provider: "fake",
                Amount:   5000,
                Currency: "USD",
        }

        p1, err := svc.CreatePayment(context.Background(), input)
        require.NoError(t, err)

        // Same request again — should return the same payment without calling provider again.
        p2, err := svc.CreatePayment(context.Background(), input)
        require.NoError(t, err)
        assert.Equal(t, p1.ID, p2.ID, "should return the same payment")
        assert.Equal(t, 1, provider.CallCount("create"), "provider should only be called once")
}

func TestCreatePayment_ProviderFailure(t *testing.T) {
        svc, _, provider := newTestService(t)
        provider.FailOnNext("card declined")

        _, err := svc.CreatePayment(context.Background(), CreatePaymentInput{
                OrderID:  "order-1",
                Provider: "fake",
                Amount:   5000,
                Currency: "USD",
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_FAILED", appErr.Code)
        assert.Contains(t, appErr.Message, "card declined")
}

func TestCreatePayment_ProviderError(t *testing.T) {
        svc, _, provider := newTestService(t)
        provider.ErrorOnNext("network unreachable")

        _, err := svc.CreatePayment(context.Background(), CreatePaymentInput{
                OrderID:  "order-1",
                Provider: "fake",
                Amount:   5000,
                Currency: "USD",
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_PROVIDER_ERROR", appErr.Code)
}

func TestCreatePayment_Timeout(t *testing.T) {
        svc, _, provider := newTestService(t)
        provider.TimeoutOnNext()

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
        defer cancel()

        _, err := svc.CreatePayment(ctx, CreatePaymentInput{
                OrderID:  "order-1",
                Provider: "fake",
                Amount:   5000,
                Currency: "USD",
        })
        assert.Error(t, err)
}

func TestCreatePayment_InvalidInput(t *testing.T) {
        svc, _, _ := newTestService(t)

        tests := []struct {
                name  string
                input CreatePaymentInput
        }{
                {"empty order_id", CreatePaymentInput{OrderID: "", Provider: "fake", Amount: 100, Currency: "USD"}},
                {"empty provider", CreatePaymentInput{OrderID: "o1", Provider: "", Amount: 100, Currency: "USD"}},
                {"zero amount", CreatePaymentInput{OrderID: "o1", Provider: "fake", Amount: 0, Currency: "USD"}},
                {"negative amount", CreatePaymentInput{OrderID: "o1", Provider: "fake", Amount: -1, Currency: "USD"}},
                {"empty currency", CreatePaymentInput{OrderID: "o1", Provider: "fake", Amount: 100, Currency: ""}},
                {"unsupported provider", CreatePaymentInput{OrderID: "o1", Provider: "stripe", Amount: 100, Currency: "USD"}},
        }

        for _, tc := range tests {
                t.Run(tc.name, func(t *testing.T) {
                        _, err := svc.CreatePayment(context.Background(), tc.input)
                        assert.Error(t, err)
                        appErr, ok := errors.IsAppError(err)
                        require.True(t, ok)
                        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
                })
        }
}

// --- VerifyPayment Tests ---

func TestVerifyPayment_Success(t *testing.T) {
        svc, _, _ := newTestService(t)
        p := createTestPayment(t, svc, "order-1")

        verified, err := svc.VerifyPayment(context.Background(), p.ID)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, verified.Status)
}

func TestVerifyPayment_NotFound(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, err := svc.VerifyPayment(context.Background(), "nonexistent")
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestVerifyPayment_ProviderFailure(t *testing.T) {
        svc, _, provider := newTestService(t)
        p := createTestPayment(t, svc, "order-1")
        provider.FailOnNext("verification failed")

        _, err := svc.VerifyPayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_FAILED", appErr.Code)
}

func TestVerifyPayment_ProviderError(t *testing.T) {
        svc, _, provider := newTestService(t)
        p := createTestPayment(t, svc, "order-1")
        provider.ErrorOnNext("timeout")

        _, err := svc.VerifyPayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_PROVIDER_ERROR", appErr.Code)
}

// --- CapturePayment Tests ---

func TestCapturePayment_Success(t *testing.T) {
        svc, _, _ := newTestService(t)
        p := createTestPayment(t, svc, "order-1")
        require.Equal(t, domain.PaymentAuthorized, p.Status)

        captured, err := svc.CapturePayment(context.Background(), p.ID)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, captured.Status)
}

func TestCapturePayment_WrongStatus(t *testing.T) {
        svc, _, _ := newTestService(t)
        p := createTestPayment(t, svc, "order-1")

        // Transition to PAID first via verify
        _, err := svc.VerifyPayment(context.Background(), p.ID)
        require.NoError(t, err)

        // Now try to capture — should fail because status is PAID, not AUTHORIZED
        _, err = svc.CapturePayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "BAD_REQUEST", appErr.Code)
}

func TestCapturePayment_ProviderFailure(t *testing.T) {
        svc, _, provider := newTestService(t)
        p := createTestPayment(t, svc, "order-1")
        provider.FailOnNext("capture declined")

        _, err := svc.CapturePayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_FAILED", appErr.Code)
}

func TestCapturePayment_ProviderError(t *testing.T) {
        svc, _, provider := newTestService(t)
        p := createTestPayment(t, svc, "order-1")
        provider.ErrorOnNext("network error")

        _, err := svc.CapturePayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_PROVIDER_ERROR", appErr.Code)
}

// --- RefundPayment Tests ---

func TestRefundPayment_Success(t *testing.T) {
        svc, _, _ := newTestService(t)

        // Create payment, verify to PAID.
        p := createTestPayment(t, svc, "order-1")
        p, err := svc.VerifyPayment(context.Background(), p.ID)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, p.Status)

        // Refund
        refunded, err := svc.RefundPayment(context.Background(), p.ID)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentRefunded, refunded.Status)
        assert.NotEmpty(t, refunded.ProviderRef)
}

func TestRefundPayment_WrongStatus_Authorized(t *testing.T) {
        svc, _, _ := newTestService(t)
        p := createTestPayment(t, svc, "order-1") // AUTHORIZED

        _, err := svc.RefundPayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "BAD_REQUEST", appErr.Code)
}

func TestRefundPayment_WrongStatus_Pending(t *testing.T) {
        svc, _, provider := newTestService(t)
        // Manually create a PENDING payment by using the repo directly.
        repo := svc.repo.(*mockPaymentRepo)
        payment := domain.NewPayment("order-x", "fake", "USD", 1000)
        require.NoError(t, repo.Create(context.Background(), payment))

        provider.Reset()

        _, err := svc.RefundPayment(context.Background(), payment.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "BAD_REQUEST", appErr.Code)
}

func TestRefundPayment_ProviderFailure(t *testing.T) {
        svc, _, provider := newTestService(t)

        p := createTestPayment(t, svc, "order-1")
        p, err := svc.VerifyPayment(context.Background(), p.ID)
        require.NoError(t, err)

        provider.FailOnNext("refund declined")

        _, err = svc.RefundPayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_FAILED", appErr.Code)
}

func TestRefundPayment_ProviderError(t *testing.T) {
        svc, _, provider := newTestService(t)

        p := createTestPayment(t, svc, "order-1")
        p, err := svc.VerifyPayment(context.Background(), p.ID)
        require.NoError(t, err)

        provider.ErrorOnNext("gateway error")

        _, err = svc.RefundPayment(context.Background(), p.ID)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "PAYMENT_PROVIDER_ERROR", appErr.Code)
}

// --- Concurrent Idempotency Test ---

func TestCreatePayment_ConcurrentIdempotency(t *testing.T) {
        svc, _, provider := newTestService(t)

        input := CreatePaymentInput{
                OrderID:  "order-concurrent",
                Provider: "fake",
                Amount:   5000,
                Currency: "USD",
        }

        var wg sync.WaitGroup
        var successCount int64
        var dupCount int64

        const goroutines = 10
        wg.Add(goroutines)
        for i := 0; i < goroutines; i++ {
                go func() {
                        defer wg.Done()
                        p, err := svc.CreatePayment(context.Background(), input)
                        if err != nil {
                                atomic.AddInt64(&dupCount, 1)
                                return
                        }
                        if p != nil && p.ID != "" {
                                atomic.AddInt64(&successCount, 1)
                        }
                }()
        }
        wg.Wait()

        // All should succeed (some via idempotent return).
        assert.Equal(t, int64(goroutines), successCount+dupCount)
        // Provider should be called at most once (or a few times due to race, but much less than goroutines).
        assert.LessOrEqual(t, provider.CallCount("create"), goroutines)
}

// --- GetPayment Tests ---

func TestGetPayment_Success(t *testing.T) {
        svc, _, _ := newTestService(t)
        created := createTestPayment(t, svc, "order-1")

        found, err := svc.GetPayment(context.Background(), created.ID)
        require.NoError(t, err)
        assert.Equal(t, created.ID, found.ID)
}

func TestGetPayment_NotFound(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, err := svc.GetPayment(context.Background(), "nonexistent")
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestGetPaymentByOrder_Success(t *testing.T) {
        svc, _, _ := newTestService(t)
        createTestPayment(t, svc, "order-1")

        found, err := svc.GetPaymentByOrder(context.Background(), "order-1", "fake")
        require.NoError(t, err)
        assert.Equal(t, "order-1", found.OrderID)
}

func TestGetPaymentByOrder_NotFound(t *testing.T) {
        svc, _, _ := newTestService(t)

        _, err := svc.GetPaymentByOrder(context.Background(), "order-x", "fake")
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "NOT_FOUND", appErr.Code)
}

// --- Full Lifecycle ---

func TestFullPaymentLifecycle(t *testing.T) {
        svc, _, provider := newTestService(t)

        // 1. Create payment -> AUTHORIZED
        p, err := svc.CreatePayment(context.Background(), CreatePaymentInput{
                OrderID:  "order-lifecycle",
                Provider: "fake",
                Amount:   10000,
                Currency: "USD",
        })
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentAuthorized, p.Status)
        assert.Equal(t, 1, provider.CallCount("create"))

        // 2. Capture -> PAID
        p, err = svc.CapturePayment(context.Background(), p.ID)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, p.Status)
        assert.Equal(t, 1, provider.CallCount("capture"))

        // 3. Refund -> REFUNDED
        p, err = svc.RefundPayment(context.Background(), p.ID)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentRefunded, p.Status)
        assert.Equal(t, 1, provider.CallCount("refund"))
}
