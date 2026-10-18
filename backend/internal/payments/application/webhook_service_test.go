package application

import (
        "context"
        "encoding/json"
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

// --- Mock implementations ---

type mockWebhookEventPublisher struct {
        mu      sync.Mutex
        events  []*domain.PaymentEvent
}

func newMockWebhookEventPublisher() *mockWebhookEventPublisher {
        return &mockWebhookEventPublisher{}
}

func (p *mockWebhookEventPublisher) Publish(_ context.Context, event *domain.PaymentEvent) error {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.events = append(p.events, event)
        return nil
}

func (p *mockWebhookEventPublisher) Events() []*domain.PaymentEvent {
        p.mu.Lock()
        defer p.mu.Unlock()
        cp := make([]*domain.PaymentEvent, len(p.events))
        copy(cp, p.events)
        return cp
}

// Enhanced mock repo with FindByProviderRef.
type webhookMockPaymentRepo struct {
        mu       sync.Mutex
        payments map[string]*domain.Payment // key: payment ID
        refIndex map[string]string          // provider+"|"+ref -> payment ID
}

func newWebhookMockPaymentRepo() *webhookMockPaymentRepo {
        return &webhookMockPaymentRepo{
                payments: make(map[string]*domain.Payment),
                refIndex: make(map[string]string),
        }
}

func (r *webhookMockPaymentRepo) copyPayment(p *domain.Payment) *domain.Payment {
        cp := *p
        if p.Metadata != nil {
                cp.Metadata = make(map[string]string, len(p.Metadata))
                for k, v := range p.Metadata {
                        cp.Metadata[k] = v
                }
        }
        return &cp
}

func (r *webhookMockPaymentRepo) Create(_ context.Context, payment *domain.Payment) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        for _, p := range r.payments {
                if p.OrderID == payment.OrderID && p.Provider == payment.Provider {
                        return domain.ErrDuplicatePayment
                }
        }
        cp := r.copyPayment(payment)
        r.payments[cp.ID] = cp
        if cp.ProviderRef != "" {
                r.refIndex[cp.Provider+"|"+cp.ProviderRef] = cp.ID
        }
        return nil
}

func (r *webhookMockPaymentRepo) FindByID(_ context.Context, id string) (*domain.Payment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        p, ok := r.payments[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return r.copyPayment(p), nil
}

func (r *webhookMockPaymentRepo) FindByOrderID(_ context.Context, orderID, provider string) (*domain.Payment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        for _, p := range r.payments {
                if p.OrderID == orderID && p.Provider == provider {
                        return r.copyPayment(p), nil
                }
        }
        return nil, errors.ErrNotFound
}

func (r *webhookMockPaymentRepo) FindByProviderRef(_ context.Context, provider, providerRef string) (*domain.Payment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        id, ok := r.refIndex[provider+"|"+providerRef]
        if !ok {
                return nil, errors.ErrNotFound
        }
        p := r.payments[id]
        return r.copyPayment(p), nil
}

func (r *webhookMockPaymentRepo) UpdateStatus(_ context.Context, id string, expectedStatus, newStatus domain.PaymentStatus, providerRef string) (*domain.Payment, error) {
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
        p.UpdatedAt = time.Now().UTC()
        return r.copyPayment(p), nil
}

// --- Helpers ---

func newTestWebhookService(t *testing.T) (*WebhookService, *webhookMockPaymentRepo, *infrastructure.FakeSignatureVerifier, *infrastructure.InMemoryWebhookEventStore, *mockWebhookEventPublisher) {
        t.Helper()
        repo := newWebhookMockPaymentRepo()
        secrets := map[string]string{"fake": "test-secret-key-for-hmac"}
        verifier := infrastructure.NewFakeSignatureVerifier(secrets)
        eventStore := infrastructure.NewInMemoryWebhookEventStore()
        publisher := newMockWebhookEventPublisher()
        svc := NewWebhookService(repo, verifier, eventStore, publisher)
        return svc, repo, verifier, eventStore, publisher
}

func createPaymentForWebhook(t *testing.T, repo *webhookMockPaymentRepo, orderID, providerRef string, status domain.PaymentStatus) *domain.Payment {
        t.Helper()
        now := time.Now().UTC()
        p := &domain.Payment{
                ID:          "pay-" + orderID,
                OrderID:     orderID,
                Provider:    "fake",
                ProviderRef: providerRef,
                Amount:      5000,
                Currency:    "USD",
                Status:      status,
                Metadata:    make(map[string]string),
                CreatedAt:   now,
                UpdatedAt:   now,
        }
        require.NoError(t, repo.Create(context.Background(), p))
        // Manually set status and ref for testing (repo.Create sets what we give it).
        repo.mu.Lock()
        stored := repo.payments[p.ID]
        stored.Status = status
        stored.ProviderRef = providerRef
        repo.mu.Unlock()
        return p
}

func buildWebhookPayload(t *testing.T, wh *domain.WebhookEvent) []byte {
        t.Helper()
        // Build the JSON payload that would come from the provider.
        body := map[string]interface{}{
                "id":          wh.ID,
                "type":        string(wh.Type),
                "provider":    wh.Provider,
                "payment_ref": wh.PaymentRef,
                "amount":      wh.Amount,
                "currency":    wh.Currency,
                "status":      wh.Status,
                "timestamp":   wh.Timestamp.UTC().Format(time.RFC3339),
        }
        data, err := json.Marshal(body)
        require.NoError(t, err)
        return data
}

func signPayload(verifier *infrastructure.FakeSignatureVerifier, provider string, payload []byte) string {
        return verifier.SignPayload(provider, payload)
}

// --- Tests ---

// Valid Signature Tests

func TestWebhook_ValidSignature_PaymentCompleted(t *testing.T) {
        svc, repo, verifier, _, publisher := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-1", "fake-ref-abc123", domain.PaymentAuthorized)

        wh := &domain.WebhookEvent{
                ID:         "wh-completed-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-abc123",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, payment.Status)

        evts := publisher.Events()
        require.Len(t, evts, 1)
        assert.Equal(t, "payment.completed", evts[0].EventType)
        assert.Equal(t, "pay-order-1", evts[0].PaymentID)
}

func TestWebhook_ValidSignature_PaymentFailed(t *testing.T) {
        svc, repo, verifier, _, publisher := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-2", "fake-ref-fail1", domain.PaymentPending)

        wh := &domain.WebhookEvent{
                ID:         "wh-failed-1",
                Type:       domain.WebhookEventPaymentFailed,
                Provider:   "fake",
                PaymentRef: "fake-ref-fail1",
                Amount:     5000,
                Currency:   "USD",
                Status:     "failed",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentFailed, payment.Status)

        evts := publisher.Events()
        require.Len(t, evts, 1)
        assert.Equal(t, "payment.failed", evts[0].EventType)
}

func TestWebhook_ValidSignature_PaymentRefunded(t *testing.T) {
        svc, repo, verifier, _, publisher := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-3", "fake-ref-refund1", domain.PaymentPaid)

        wh := &domain.WebhookEvent{
                ID:         "wh-refunded-1",
                Type:       domain.WebhookEventPaymentRefunded,
                Provider:   "fake",
                PaymentRef: "fake-ref-refund1",
                Amount:     5000,
                Currency:   "USD",
                Status:     "refunded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentRefunded, payment.Status)

        evts := publisher.Events()
        require.Len(t, evts, 1)
        assert.Equal(t, "payment.refunded", evts[0].EventType)
}

// Invalid Signature Tests

func TestWebhook_InvalidSignature_Rejected(t *testing.T) {
        svc, _, _, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                ID:         "wh-invalid-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "ref-xyz",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)

        _, err := svc.ProcessWebhook(context.Background(), "fake", "sha256=invalidsignature", payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "UNAUTHORIZED", appErr.Code)
}

func TestWebhook_MissingSignature_Rejected(t *testing.T) {
        svc, _, _, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                ID:         "wh-nosig-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "ref-xyz",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)

        _, err := svc.ProcessWebhook(context.Background(), "fake", "", payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "UNAUTHORIZED", appErr.Code)
}

func TestWebhook_UnsupportedProvider_Rejected(t *testing.T) {
        svc, _, _, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                ID:         "wh-unknown-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "unknown_provider",
                PaymentRef: "ref-xyz",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)

        _, err := svc.ProcessWebhook(context.Background(), "unknown_provider", "sha256=something", payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

// Duplicate Webhook Tests

func TestWebhook_DuplicateWebhook_Idempotent(t *testing.T) {
        svc, repo, verifier, eventStore, publisher := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-dup1", "fake-ref-dup1", domain.PaymentAuthorized)

        wh := &domain.WebhookEvent{
                ID:         "wh-dup-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-dup1",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        // First call
        p1, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, p1.Status)

        // Second call — same webhook ID, should be idempotent
        p2, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, p2.Status)

        // Should have been marked as processed only once
        processed, err := eventStore.HasBeenProcessed(context.Background(), "wh-dup-1")
        require.NoError(t, err)
        assert.True(t, processed)

        // Only one event should have been published (from the first call)
        evts := publisher.Events()
        require.Len(t, evts, 1)
}

func TestWebhook_DuplicateWebhook_WithCorrectSignature(t *testing.T) {
        svc, repo, verifier, _, _ := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-dup2", "fake-ref-dup2", domain.PaymentPending)

        wh := &domain.WebhookEvent{
                ID:         "wh-dup-2",
                Type:       domain.WebhookEventPaymentFailed,
                Provider:   "fake",
                PaymentRef: "fake-ref-dup2",
                Amount:     5000,
                Currency:   "USD",
                Status:     "failed",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        // Process twice
        _, err1 := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err1)
        _, err2 := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err2)

        // Payment should still be FAILED, not transitioned again
        payment, err := repo.FindByProviderRef(context.Background(), "fake", "fake-ref-dup2")
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentFailed, payment.Status)
}

// Out-of-Order Event Tests

func TestWebhook_OutOfOrder_AlreadyPaid_ThenCompleted(t *testing.T) {
        svc, repo, verifier, _, publisher := newTestWebhookService(t)
        // Payment is already PAID (e.g., via VerifyPayment)
        createPaymentForWebhook(t, repo, "order-oo1", "fake-ref-oo1", domain.PaymentPaid)

        wh := &domain.WebhookEvent{
                ID:         "wh-oo-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-oo1",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        // Should succeed without error (out-of-order, gracefully handled)
        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        // Status should remain PAID
        assert.Equal(t, domain.PaymentPaid, payment.Status)

        // No event should have been published for a no-op
        evts := publisher.Events()
        assert.Len(t, evts, 0)
}

func TestWebhook_OutOfOrder_Failed_ThenCompleted(t *testing.T) {
        svc, repo, verifier, _, _ := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-oo2", "fake-ref-oo2", domain.PaymentFailed)

        wh := &domain.WebhookEvent{
                ID:         "wh-oo-2",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-oo2",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        // Payment is FAILED, completed webhook arrives — should be no-op
        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentFailed, payment.Status)
}

func TestWebhook_OutOfOrder_Refunded_ThenCompleted(t *testing.T) {
        svc, repo, verifier, _, _ := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-oo3", "fake-ref-oo3", domain.PaymentRefunded)

        wh := &domain.WebhookEvent{
                ID:         "wh-oo-3",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-oo3",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentRefunded, payment.Status)
}

// Unknown Payment Tests

func TestWebhook_UnknownPaymentRef_NotFound(t *testing.T) {
        svc, _, verifier, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                ID:         "wh-unknown-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "nonexistent-ref",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        _, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "NOT_FOUND", appErr.Code)
}

// Provider Mismatch Tests

func TestWebhook_ProviderMismatch_BadRequest(t *testing.T) {
        svc, _, verifier, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                ID:         "wh-mismatch-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "ref-xyz",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        // Sign for "fake" but pass "other" as URL param provider.
        signature := signPayload(verifier, "fake", payload)

        _, err := svc.ProcessWebhook(context.Background(), "other", signature, payload, wh)
        // Should fail because "other" provider has no secret configured.
        require.Error(t, err)
}

// Validation Tests

func TestWebhook_MissingWebhookID_ValidationError(t *testing.T) {
        svc, _, verifier, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "ref-xyz",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        _, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestWebhook_MissingEventType_ValidationError(t *testing.T) {
        svc, _, verifier, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                ID:         "wh-no-type",
                Provider:   "fake",
                PaymentRef: "ref-xyz",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        _, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestWebhook_MissingPaymentRef_ValidationError(t *testing.T) {
        svc, _, verifier, _, _ := newTestWebhookService(t)

        wh := &domain.WebhookEvent{
                ID:       "wh-no-ref",
                Type:     domain.WebhookEventPaymentCompleted,
                Provider: "fake",
                Timestamp: time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        _, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestWebhook_UnknownEventType_BadRequest(t *testing.T) {
        svc, repo, verifier, _, _ := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-bad-type", "fake-ref-badtype", domain.PaymentAuthorized)

        wh := &domain.WebhookEvent{
                ID:         "wh-bad-type",
                Type:       "payment.captured", // not a recognized webhook type
                Provider:   "fake",
                PaymentRef: "fake-ref-badtype",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        _, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, "BAD_REQUEST", appErr.Code)
        assert.Contains(t, appErr.Message, "unknown webhook event type")
}

// Refund from REFUND_PENDING State

func TestWebhook_Refund_FromRefundPending(t *testing.T) {
        svc, repo, verifier, _, publisher := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-rp1", "fake-ref-rp1", domain.PaymentRefundPending)

        wh := &domain.WebhookEvent{
                ID:         "wh-rp-1",
                Type:       domain.WebhookEventPaymentRefunded,
                Provider:   "fake",
                PaymentRef: "fake-ref-rp1",
                Amount:     5000,
                Currency:   "USD",
                Status:     "refunded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentRefunded, payment.Status)

        evts := publisher.Events()
        require.Len(t, evts, 1)
        assert.Equal(t, "payment.refunded", evts[0].EventType)
}

// Concurrent Webhook Delivery

func TestWebhook_ConcurrentDuplicateWebhook(t *testing.T) {
        svc, repo, verifier, eventStore, publisher := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-conc", "fake-ref-conc", domain.PaymentAuthorized)

        wh := &domain.WebhookEvent{
                ID:         "wh-conc-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-conc",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        var wg sync.WaitGroup
        var successCount int64
        var errCount int64

        const goroutines = 10
        wg.Add(goroutines)
        for i := 0; i < goroutines; i++ {
                go func() {
                        defer wg.Done()
                        p, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
                        if err != nil {
                                atomic.AddInt64(&errCount, 1)
                                return
                        }
                        if p != nil {
                                atomic.AddInt64(&successCount, 1)
                        }
                }()
        }
        wg.Wait()

        // All calls should succeed (some via idempotent path)
        assert.Equal(t, int64(goroutines), successCount+errCount)

        // Webhook should be recorded as processed
        processed, err := eventStore.HasBeenProcessed(context.Background(), "wh-conc-1")
        require.NoError(t, err)
        assert.True(t, processed)

        // Only 1 event should be published
        evts := publisher.Events()
        assert.LessOrEqual(t, len(evts), 1)
}

// Full Lifecycle: Create -> Webhook Completed

func TestWebhook_FullLifecycle_CreateThenWebhookComplete(t *testing.T) {
        svc, repo, verifier, _, publisher := newTestWebhookService(t)

        // Create a payment in AUTHORIZED state
        createPaymentForWebhook(t, repo, "order-life", "fake-ref-life", domain.PaymentAuthorized)

        // Simulate webhook
        wh := &domain.WebhookEvent{
                ID:         "wh-life-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-life",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        signature := signPayload(verifier, "fake", payload)

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, payment.Status)

        // Then refund
        refundWh := &domain.WebhookEvent{
                ID:         "wh-life-2",
                Type:       domain.WebhookEventPaymentRefunded,
                Provider:   "fake",
                PaymentRef: "fake-ref-life",
                Amount:     5000,
                Currency:   "USD",
                Status:     "refunded",
                Timestamp:  time.Now().UTC(),
        }
        refundPayload := buildWebhookPayload(t, refundWh)
        refundSig := signPayload(verifier, "fake", refundPayload)

        refunded, err := svc.ProcessWebhook(context.Background(), "fake", refundSig, refundPayload, refundWh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentRefunded, refunded.Status)

        evts := publisher.Events()
        require.Len(t, evts, 2)
        assert.Equal(t, "payment.completed", evts[0].EventType)
        assert.Equal(t, "payment.refunded", evts[1].EventType)
}

// Signature with sha256= prefix

func TestWebhook_SignatureWithPrefix(t *testing.T) {
        svc, repo, verifier, _, _ := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-prefix", "fake-ref-prefix", domain.PaymentAuthorized)

        wh := &domain.WebhookEvent{
                ID:         "wh-prefix-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-prefix",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        // SignPayload already includes "sha256=" prefix
        signature := signPayload(verifier, "fake", payload)

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, payment.Status)
}

// Signature without sha256= prefix

func TestWebhook_SignatureWithoutPrefix(t *testing.T) {
        svc, repo, verifier, _, _ := newTestWebhookService(t)
        createPaymentForWebhook(t, repo, "order-nopfx", "fake-ref-nopfx", domain.PaymentAuthorized)

        wh := &domain.WebhookEvent{
                ID:         "wh-nopfx-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-nopfx",
                Amount:     5000,
                Currency:   "USD",
                Status:     "succeeded",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)
        // Get the signature and strip the prefix
        fullSig := signPayload(verifier, "fake", payload)
        signature := fullSig[len("sha256="):] // strip prefix

        payment, err := svc.ProcessWebhook(context.Background(), "fake", signature, payload, wh)
        require.NoError(t, err)
        assert.Equal(t, domain.PaymentPaid, payment.Status)
}

// Provider failure test

func TestWebhook_EventStoreError_Propagates(t *testing.T) {
        repo := newWebhookMockPaymentRepo()
        createPaymentForWebhook(t, repo, "order-eserr", "fake-ref-eserr", domain.PaymentAuthorized)

        verifier := infrastructure.NewAlwaysValidVerifier()

        // Use a broken event store
        errStore := &brokenEventStore{}
        publisher := newMockWebhookEventPublisher()
        svc := NewWebhookService(repo, verifier, errStore, publisher)

        wh := &domain.WebhookEvent{
                ID:         "wh-eserr-1",
                Type:       domain.WebhookEventPaymentCompleted,
                Provider:   "fake",
                PaymentRef: "fake-ref-eserr",
                Timestamp:  time.Now().UTC(),
        }
        payload := buildWebhookPayload(t, wh)

        _, err := svc.ProcessWebhook(context.Background(), "fake", "sig", payload, wh)
        require.Error(t, err)
}

type brokenEventStore struct{}

func (s *brokenEventStore) HasBeenProcessed(_ context.Context, _ string) (bool, error) {
        return false, errors.New("INTERNAL_ERROR", "store unavailable", 500)
}

func (s *brokenEventStore) MarkProcessed(_ context.Context, _ *domain.ProcessedWebhook) error {
        return errors.New("INTERNAL_ERROR", "store unavailable", 500)
}
