package orderworker

import (
        "context"
        "encoding/json"
        "fmt"
        "sync"
        "testing"
        "time"

        ordDomain "github.com/ecommerce/backend/internal/orders/domain"
        payDomain "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/ecommerce/backend/pkg/logger"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockOrderGetter struct {
        mu     sync.Mutex
        orders map[string]*ordDomain.Order
        err    error
}

func newMockOrderGetter() *mockOrderGetter {
        return &mockOrderGetter{orders: make(map[string]*ordDomain.Order)}
}

func (m *mockOrderGetter) GetOrderByID(_ context.Context, id string) (*ordDomain.Order, error) {
        m.mu.Lock()
        defer m.mu.Unlock()
        if m.err != nil {
                return nil, m.err
        }
        if o, ok := m.orders[id]; ok {
                return o, nil
        }
        return nil, fmt.Errorf("NOT_FOUND: order %s not found", id)
}

type mockOrderUpdater struct {
        mu      sync.Mutex
        updated []updatedCall
        err     error
}

type updatedCall struct {
        id       string
        status   string
        reason   string
        payRef   string
}

func newMockOrderUpdater() *mockOrderUpdater {
        return &mockOrderUpdater{}
}

func (m *mockOrderUpdater) UpdateOrderStatus(_ context.Context, orderID, status, reason, payRef string) (*ordDomain.Order, error) {
        m.mu.Lock()
        defer m.mu.Unlock()
        if m.err != nil {
                return nil, m.err
        }
        m.updated = append(m.updated, updatedCall{id: orderID, status: status, reason: reason, payRef: payRef})
        return &ordDomain.Order{ID: orderID, Status: ordDomain.OrderStatus(status)}, nil
}

type mockReservationReleaser struct {
        mu           sync.Mutex
        releasedRefs map[string]int
        errByRef     map[string]error
}

func newMockReservationReleaser() *mockReservationReleaser {
        return &mockReservationReleaser{
                releasedRefs: make(map[string]int),
                errByRef:     make(map[string]error),
        }
}

func (m *mockReservationReleaser) ReleaseByReference(_ context.Context, reference string) (int, error) {
        m.mu.Lock()
        defer m.mu.Unlock()
        if err, ok := m.errByRef[reference]; ok {
                return 0, err
        }
        m.releasedRefs[reference]++
        return 1, nil
}

func newTestLogger(t *testing.T) *logger.Logger {
        log, err := logger.New("info", "console")
        require.NoError(t, err)
        return log
}

func newPaymentEvent(eventType, orderID, paymentID, providerRef string) *payDomain.PaymentEvent {
        return &payDomain.PaymentEvent{
                EventType:   eventType,
                PaymentID:   paymentID,
                OrderID:     orderID,
                Provider:    "fake",
                Amount:      10000,
                Currency:    "USD",
                FromStatus:  "PENDING",
                ToStatus:    "PAID",
                ProviderRef: providerRef,
        }
}

func envelopeFromEvent(t *testing.T, evt *payDomain.PaymentEvent) *eventstore.Envelope {
        payload, err := json.Marshal(evt)
        require.NoError(t, err)
        return &eventstore.Envelope{
                EventID:      "test-event-id",
                EventType:    evt.EventType,
                AggregateID:  evt.PaymentID,
                AggregateType: "payment",
                Version:      1,
                OccurredAt:   time.Now().UTC(),
                CorrelationID: "test-corr-id",
                Payload:      payload,
        }
}

func makeOrder(id, status string) *ordDomain.Order {
        return &ordDomain.Order{
                ID:     id,
                Status: ordDomain.OrderStatus(status),
        }
}

// --- payment.completed tests ---

func TestPaymentCompleted_TransitionsToPaid(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAYMENT_PENDING")

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.completed", "ord-1", "pay-1", "ref-123")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err)
        require.Len(t, updater.updated, 1)
        assert.Equal(t, "ord-1", updater.updated[0].id)
        assert.Equal(t, "PAID", updater.updated[0].status)
        assert.Equal(t, "payment completed", updater.updated[0].reason)
        assert.Equal(t, "ref-123", updater.updated[0].payRef)
}

func TestPaymentCompleted_IdempotentSkip(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAID")

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.completed", "ord-1", "pay-1", "ref-123")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err)
        assert.Empty(t, updater.updated) // no transition attempted
}

func TestPaymentCompleted_DuplicateDelivery(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAYMENT_PENDING")

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.completed", "ord-1", "pay-1", "ref-123")

        // First delivery — should succeed.
        env1 := envelopeFromEvent(t, evt)
        err := handler(context.Background(), env1)
        require.NoError(t, err)
        require.Len(t, updater.updated, 1)

        // Second delivery — order is now PAID, should be idempotent skip.
        getter.mu.Lock()
        getter.orders["ord-1"].Status = ordDomain.OrderPaid
        getter.mu.Unlock()

        env2 := envelopeFromEvent(t, evt)
        err = handler(context.Background(), env2)
        require.NoError(t, err)
        assert.Len(t, updater.updated, 1) // still only one update
}

func TestPaymentCompleted_OrderNotFound(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.completed", "ord-missing", "pay-1", "ref-123")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.Error(t, err)
        assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestPaymentCompleted_MissingOrderID(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.completed", "", "pay-1", "ref-123")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err) // non-retryable: returns nil
        assert.Empty(t, updater.updated)
}

func TestPaymentCompleted_InvalidPayload(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        env, err := eventstore.NewEnvelope("payment.completed", "pay-1", "payment", 1, map[string]string{"key": "val"})
        require.NoError(t, err)
        env.Payload = json.RawMessage(`{invalid json}`)

        err = handler(context.Background(), env)
        require.NoError(t, err) // non-retryable
}

func TestPaymentCompleted_UpdateConflict(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAYMENT_PENDING")
        updater.err = fmt.Errorf("CONFLICT: status has changed")

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.completed", "ord-1", "pay-1", "ref-123")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.Error(t, err)
        assert.Contains(t, err.Error(), "CONFLICT")
}

// --- payment.failed tests ---

func TestPaymentFailed_CancelsOrderAndReleasesInventory(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAYMENT_PENDING")

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "ord-1", "pay-1", "ref-456")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err)
        require.Len(t, updater.updated, 1)
        assert.Equal(t, "ord-1", updater.updated[0].id)
        assert.Equal(t, "CANCELLED", updater.updated[0].status)
        assert.Contains(t, updater.updated[0].reason, "payment failed")
        assert.Equal(t, 1, releaser.releasedRefs["ord-1"])
}

func TestPaymentFailed_IdempotentSkip_Cancelled(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "CANCELLED")

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "ord-1", "pay-1", "ref-456")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err)
        assert.Empty(t, updater.updated)
        assert.Equal(t, 0, releaser.releasedRefs["ord-1"])
}

func TestPaymentFailed_IdempotentSkip_Refunded(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "REFUNDED")

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "ord-1", "pay-1", "ref-456")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err)
        assert.Empty(t, updater.updated)
}

func TestPaymentFailed_DuplicateDelivery(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAYMENT_PENDING")

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "ord-1", "pay-1", "ref-456")

        // First delivery.
        env1 := envelopeFromEvent(t, evt)
        err := handler(context.Background(), env1)
        require.NoError(t, err)
        require.Len(t, updater.updated, 1)

        // Second delivery — order is now CANCELLED, should skip.
        getter.mu.Lock()
        getter.orders["ord-1"].Status = ordDomain.OrderCancelled
        getter.mu.Unlock()

        env2 := envelopeFromEvent(t, evt)
        err = handler(context.Background(), env2)
        require.NoError(t, err)
        assert.Len(t, updater.updated, 1) // still only one
        assert.Equal(t, 1, releaser.releasedRefs["ord-1"]) // only one release
}

func TestPaymentFailed_MissingOrderID(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "", "pay-1", "ref-456")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err) // non-retryable
}

func TestPaymentFailed_OrderNotFound(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "ord-missing", "pay-1", "ref-456")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.Error(t, err)
        assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestPaymentFailed_InventoryReleaseError(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAYMENT_PENDING")
        releaser.errByRef["ord-1"] = fmt.Errorf("db error")

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "ord-1", "pay-1", "ref-456")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err) // order is cancelled, inventory error is logged but not returned
        require.Len(t, updater.updated, 1)
        assert.Equal(t, "CANCELLED", updater.updated[0].status)
}

func TestPaymentFailed_CancelConflict(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        getter.orders["ord-1"] = makeOrder("ord-1", "PAYMENT_PENDING")
        updater.err = fmt.Errorf("CONFLICT: status has changed")

        handler := PaymentFailedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.failed", "ord-1", "pay-1", "ref-456")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.Error(t, err)
        assert.Contains(t, err.Error(), "CONFLICT")
}

func TestPaymentCompleted_FromProcessingState(t *testing.T) {
        getter := newMockOrderGetter()
        updater := newMockOrderUpdater()
        releaser := newMockReservationReleaser()
        log := newTestLogger(t)

        // Order in PROCESSING (already past PAID) — should be idempotent skip.
        getter.orders["ord-1"] = makeOrder("ord-1", "PROCESSING")

        handler := PaymentCompletedHandler(HandlerConfig{
                Orders:    getter,
                Updaters:  updater,
                Inventory: releaser,
                Log:       log,
        })

        evt := newPaymentEvent("payment.completed", "ord-1", "pay-1", "ref-123")
        env := envelopeFromEvent(t, evt)

        err := handler(context.Background(), env)
        require.NoError(t, err)
        assert.Empty(t, updater.updated)
}
