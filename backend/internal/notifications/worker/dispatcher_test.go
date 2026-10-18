package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	custDomain "github.com/ecommerce/backend/internal/customers/domain"
	ordDomain "github.com/ecommerce/backend/internal/orders/domain"
	payDomain "github.com/ecommerce/backend/internal/payments/domain"
	notifDomain "github.com/ecommerce/backend/internal/notifications/domain"
	"github.com/ecommerce/backend/internal/notifications/infrastructure"
	notifWorker "github.com/ecommerce/backend/internal/notifications/worker"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/pkg/eventstore"
	"github.com/ecommerce/backend/pkg/logger"
)

// --- Test doubles ---

type mockContactResolver struct {
	contact *notifWorker.ContactInfo
	err     error
}

func (m *mockContactResolver) ResolveContact(_ context.Context, _ string) (*notifWorker.ContactInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.contact, nil
}

type mockNotificationRepo struct {
	sync.Mutex
	records      map[string]*notifDomain.NotificationRecord
	inserted     []*notifDomain.NotificationRecord
	updated      []*notifDomain.NotificationRecord
	dupOnInsert  bool
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{
		records:  make(map[string]*notifDomain.NotificationRecord),
		inserted: make([]*notifDomain.NotificationRecord, 0),
		updated:  make([]*notifDomain.NotificationRecord, 0),
	}
}

func repoKey(eventID string, ch notifDomain.Channel) string {
	return eventID + ":" + string(ch)
}

func (r *mockNotificationRepo) Insert(_ context.Context, record *notifDomain.NotificationRecord) error {
	r.Lock()
	defer r.Unlock()
	key := repoKey(record.EventID, record.Channel)
	if r.dupOnInsert {
		if _, exists := r.records[key]; exists {
			return nil
		}
	}
	r.records[key] = record
	r.inserted = append(r.inserted, record)
	return nil
}

func (r *mockNotificationRepo) FindByEventIDAndChannel(_ context.Context, eventID string, channel notifDomain.Channel) (*notifDomain.NotificationRecord, error) {
	r.Lock()
	defer r.Unlock()
	return r.records[repoKey(eventID, channel)], nil
}

func (r *mockNotificationRepo) UpdateStatus(_ context.Context, record *notifDomain.NotificationRecord) error {
	r.Lock()
	defer r.Unlock()
	r.updated = append(r.updated, record)
	return nil
}

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, _ := logger.New("debug", "json")
	return log
}

func newTestDispatcher(t *testing.T, repo *mockNotificationRepo, resolver *mockContactResolver, provider *infrastructure.InMemoryProvider) *notifWorker.Dispatcher {
	t.Helper()
	deps := notifWorker.DispatcherDependencies{
		EmailProvider:    provider,
		SMSProvider:      provider,
		PushProvider:     provider,
		NotificationRepo: repo,
		ContactResolver:  resolver,
		Log:              newTestLogger(t),
		Config: notifWorker.DispatcherConfig{
			EnableEmail: true,
			EnableSMS:   true,
			EnablePush:  true,
		},
	}
	return notifWorker.NewDispatcher(deps)
}

func makeOrderEnvelope(t *testing.T, eventType string, payload interface{}) *eventstore.Envelope {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return &eventstore.Envelope{
		EventID:       "evt-" + eventType,
		EventType:     eventType,
		AggregateID:   "order-123",
		AggregateType: "order",
		CorrelationID: "corr-1",
		Payload:       data,
	}
}

func makePaymentEnvelope(t *testing.T, eventType string, payload interface{}) *eventstore.Envelope {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return &eventstore.Envelope{
		EventID:       "evt-" + eventType,
		EventType:     eventType,
		AggregateID:   "payment-123",
		AggregateType: "payment",
		CorrelationID: "corr-2",
		Payload:       data,
	}
}

// --- Tests ---

func TestHandleOrderCreated_SendsEmailAndSMS(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com", Phone: "+111", PushToken: "token-1"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.created", ordDomain.OrderEvent{
		EventType:   "order.created",
		OrderID:     "order-123",
		OrderNumber: "ORD-20260819-abc12345",
		CustomerID:  "user-1",
		FromStatus:  "",
		ToStatus:    "PAYMENT_PENDING",
	})

	err := d.HandleOrderCreated()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.EmailCount() != 1 {
		t.Errorf("EmailCount = %d, want 1", provider.EmailCount())
	}
	if provider.SMSCount() != 1 {
		t.Errorf("SMSCount = %d, want 1", provider.SMSCount())
	}
	if provider.PushCount() != 1 {
		t.Errorf("PushCount = %d, want 1", provider.PushCount())
	}

	metrics := d.Metrics().Snapshot()
	if metrics["processed"] != 1 {
		t.Errorf("processed = %d, want 1", metrics["processed"])
	}
	if metrics["sent"] != 3 {
		t.Errorf("sent = %d, want 3 (email+sms+push)", metrics["sent"])
	}

	if provider.Emails[0].To != "user@test.com" {
		t.Errorf("email To = %q", provider.Emails[0].To)
	}
	if len(repo.inserted) != 3 {
		t.Errorf("inserted records = %d, want 3", len(repo.inserted))
	}
}

func TestHandleOrderPaid_SendsEmail(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.paid", ordDomain.OrderEvent{
		EventType:   "order.paid",
		OrderID:     "order-123",
		OrderNumber: "ORD-20260819-xyz",
		CustomerID:  "user-1",
		FromStatus:  "PAYMENT_PENDING",
		ToStatus:    "PAID",
	})

	err := d.HandleOrderPaid()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.EmailCount() != 1 {
		t.Errorf("EmailCount = %d, want 1", provider.EmailCount())
	}
	if provider.SMSCount() != 0 {
		t.Errorf("SMSCount = %d, want 0 (no phone)", provider.SMSCount())
	}
	if provider.PushCount() != 0 {
		t.Errorf("PushCount = %d, want 0 (no push token)", provider.PushCount())
	}
}

func TestHandleOrderShipped_IncludesTrackingRef(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com", Phone: "+111"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.shipped", ordDomain.OrderEvent{
		EventType:         "order.shipped",
		OrderID:           "order-123",
		OrderNumber:       "ORD-SHIP",
		CustomerID:        "user-1",
		ShipmentReference: "TRACK-ABC-123",
		FromStatus:        "PROCESSING",
		ToStatus:          "SHIPPED",
	})

	err := d.HandleOrderShipped()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Emails[0].Body == "" {
		t.Fatal("email body is empty")
	}
}

func TestHandleOrderDelivered(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.delivered", ordDomain.OrderEvent{
		EventType:   "order.delivered",
		OrderID:     "order-123",
		OrderNumber: "ORD-DEL",
		CustomerID:  "user-1",
		FromStatus:  "SHIPPED",
		ToStatus:    "DELIVERED",
	})

	err := d.HandleOrderDelivered()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.EmailCount() != 1 {
		t.Errorf("EmailCount = %d, want 1", provider.EmailCount())
	}
	metrics := d.Metrics().Snapshot()
	if metrics["processed"] != 1 {
		t.Errorf("processed = %d", metrics["processed"])
	}
}

func TestHandlePaymentFailed_SendsNotification(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makePaymentEnvelope(t, "payment.failed", payDomain.PaymentEvent{
		EventType:  "payment.failed",
		PaymentID:  "pay-123",
		OrderID:    "order-456",
		Provider:   "stripe",
		Amount:     9999,
		Currency:   "USD",
		FromStatus: "PENDING",
		ToStatus:   "FAILED",
	})

	err := d.HandlePaymentFailed()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.EmailCount() != 1 {
		t.Errorf("EmailCount = %d, want 1", provider.EmailCount())
	}
}

func TestHandlePaymentFailed_MissingOrderID_ReturnsNil(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makePaymentEnvelope(t, "payment.failed", payDomain.PaymentEvent{
		EventType: "payment.failed",
		PaymentID: "pay-1",
		OrderID:   "",
	})

	err := d.HandlePaymentFailed()(context.Background(), env)
	if err != nil {
		t.Fatalf("expected nil error for missing order_id, got: %v", err)
	}
	if provider.EmailCount() != 0 {
		t.Errorf("no email should be sent, got %d", provider.EmailCount())
	}
}

func TestHandleOrderCreated_MalformedPayload_ReturnsNil(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := &eventstore.Envelope{
		EventID:   "evt-bad",
		EventType: "order.created",
		Payload:   []byte("not-json"),
	}

	err := d.HandleOrderCreated()(context.Background(), env)
	if err != nil {
		t.Fatalf("expected nil for malformed payload, got: %v", err)
	}
}

func TestHandleOrderCreated_NoContact_Skips(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{}, // empty
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.created", ordDomain.OrderEvent{
		EventType:  "order.created",
		OrderID:    "order-1",
		OrderNumber: "ORD-1",
		CustomerID: "user-1",
		ToStatus:   "PAYMENT_PENDING",
	})

	err := d.HandleOrderCreated()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.EmailCount() != 0 {
		t.Error("no notifications should be sent without contact info")
	}
	metrics := d.Metrics().Snapshot()
	if metrics["skipped"] != 1 {
		t.Errorf("skipped = %d, want 1", metrics["skipped"])
	}
}

func TestHandleOrderCreated_ResolverError_ReturnsError(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		err: errors.New("db down"),
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.created", ordDomain.OrderEvent{
		EventType:  "order.created",
		OrderID:    "order-1",
		OrderNumber: "ORD-1",
		CustomerID: "user-1",
		ToStatus:   "PAYMENT_PENDING",
	})

	err := d.HandleOrderCreated()(context.Background(), env)
	if err == nil {
		t.Fatal("expected error when resolver fails")
	}
}

func TestIdempotency_DuplicateEvent_Skips(t *testing.T) {
	repo := newMockNotificationRepo()
	repo.dupOnInsert = true
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.created", ordDomain.OrderEvent{
		EventType:  "order.created",
		OrderID:    "order-1",
		OrderNumber: "ORD-1",
		CustomerID: "user-1",
		ToStatus:   "PAYMENT_PENDING",
	})

	handler := d.HandleOrderCreated()

	err := handler(context.Background(), env)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if provider.EmailCount() != 1 {
		t.Errorf("first call: EmailCount = %d, want 1", provider.EmailCount())
	}

	err = handler(context.Background(), env)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if provider.EmailCount() != 1 {
		t.Errorf("second call: EmailCount = %d, want 1 (idempotent)", provider.EmailCount())
	}
	metrics := d.Metrics().Snapshot()
	if metrics["idempotent_skipped"] != 1 {
		t.Errorf("idempotent_skipped = %d, want 1", metrics["idempotent_skipped"])
	}
}

func TestProviderFailure_MarksFailed(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	provider.SetError(errors.New("SMTP unavailable"))
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.created", ordDomain.OrderEvent{
		EventType:  "order.created",
		OrderID:    "order-1",
		OrderNumber: "ORD-1",
		CustomerID: "user-1",
		ToStatus:   "PAYMENT_PENDING",
	})

	err := d.HandleOrderCreated()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.EmailCount() != 0 {
		t.Error("no successful sends expected")
	}

	failedCount := 0
	for _, r := range repo.updated {
		if r.Status == notifDomain.NotificationFailed {
			failedCount++
		}
	}
	if failedCount != 1 {
		t.Errorf("expected 1 failed record, got %d", failedCount)
	}

	metrics := d.Metrics().Snapshot()
	if metrics["failed"] != 1 {
		t.Errorf("failed = %d, want 1", metrics["failed"])
	}
}

func TestDisabledChannels_NoSends(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "a@b.com", Phone: "+1", PushToken: "tok"},
	}
	provider := infrastructure.NewInMemoryProvider()

	deps := notifWorker.DispatcherDependencies{
		EmailProvider:    provider,
		SMSProvider:      provider,
		PushProvider:     provider,
		NotificationRepo: repo,
		ContactResolver:  resolver,
		Log:              newTestLogger(t),
		Config: notifWorker.DispatcherConfig{
			EnableEmail: false,
			EnableSMS:   false,
			EnablePush:  false,
		},
	}
	d := notifWorker.NewDispatcher(deps)

	env := makeOrderEnvelope(t, "order.created", ordDomain.OrderEvent{
		EventType:  "order.created",
		OrderID:    "order-1",
		OrderNumber: "ORD-1",
		CustomerID: "user-1",
		ToStatus:   "PAYMENT_PENDING",
	})

	err := d.HandleOrderCreated()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.EmailCount() != 0 || provider.SMSCount() != 0 || provider.PushCount() != 0 {
		t.Error("no sends expected when all channels disabled")
	}
}

func TestDefaultDispatcherConfig(t *testing.T) {
	cfg := notifWorker.DefaultDispatcherConfig()
	if !cfg.EnableEmail {
		t.Error("email should be enabled by default")
	}
	if cfg.EnableSMS {
		t.Error("SMS should be disabled by default")
	}
	if cfg.EnablePush {
		t.Error("push should be disabled by default")
	}
}

func TestMetricsSnapshot_ContainsAllKeys(t *testing.T) {
	repo := newMockNotificationRepo()
	resolver := &mockContactResolver{
		contact: &notifWorker.ContactInfo{Email: "user@test.com"},
	}
	provider := infrastructure.NewInMemoryProvider()
	d := newTestDispatcher(t, repo, resolver, provider)

	env := makeOrderEnvelope(t, "order.created", ordDomain.OrderEvent{
		EventType:  "order.created",
		OrderID:    "order-1",
		OrderNumber: "ORD-1",
		CustomerID: "user-1",
		ToStatus:   "PAYMENT_PENDING",
	})

	_ = d.HandleOrderCreated()(context.Background(), env)

	snapshot := d.Metrics().Snapshot()
	expectedKeys := []string{"processed", "sent", "failed", "skipped", "idempotent_skipped"}
	for _, key := range expectedKeys {
		if _, ok := snapshot[key]; !ok {
			t.Errorf("missing metric key %q", key)
		}
	}
}

func TestContactResolver(t *testing.T) {
	t.Run("nil user returns error", func(t *testing.T) {
		finder := &mockUserFinder{err: errors.New("not found")}
		r := notifWorker.NewOrderCustomerResolver(finder, &mockCustomerFinder{})
		_, err := r.ResolveContact(context.Background(), "user-1")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns email from user", func(t *testing.T) {
		finder := &mockUserFinder{user: &userDomain.User{
			ID:    "u1",
			Email: "test@example.com",
		}}
		r := notifWorker.NewOrderCustomerResolver(finder, &mockCustomerFinder{})
		contact, err := r.ResolveContact(context.Background(), "u1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if contact.Email != "test@example.com" {
			t.Errorf("Email = %q", contact.Email)
		}
	})

	t.Run("returns phone from customer", func(t *testing.T) {
		finder := &mockUserFinder{user: &userDomain.User{
			ID:    "u1",
			Email: "test@example.com",
		}}
		custFinder := &mockCustomerFinder{customer: &custDomain.Customer{
			UserID: "u1",
			Phone:  "+1234567890",
		}}
		r := notifWorker.NewOrderCustomerResolver(finder, custFinder)
		contact, err := r.ResolveContact(context.Background(), "u1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if contact.Phone != "+1234567890" {
			t.Errorf("Phone = %q, want +1234567890", contact.Phone)
		}
		if contact.Email != "test@example.com" {
			t.Errorf("Email = %q, want test@example.com", contact.Email)
		}
	})

	t.Run("missing customer is OK", func(t *testing.T) {
		finder := &mockUserFinder{user: &userDomain.User{
			ID:    "u1",
			Email: "test@example.com",
		}}
		r := notifWorker.NewOrderCustomerResolver(finder, &mockCustomerFinder{err: errors.New("not found")})
		contact, err := r.ResolveContact(context.Background(), "u1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if contact.Phone != "" {
			t.Errorf("Phone = %q, want empty", contact.Phone)
		}
	})
}

// --- Additional mock types for resolver tests ---

type mockUserFinder struct {
	user *userDomain.User
	err  error
}

func (m *mockUserFinder) FindByID(_ context.Context, _ string) (*userDomain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

type mockCustomerFinder struct {
	customer *custDomain.Customer
	err      error
}

func (m *mockCustomerFinder) FindByUserID(_ context.Context, _ string) (*custDomain.Customer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.customer, nil
}
