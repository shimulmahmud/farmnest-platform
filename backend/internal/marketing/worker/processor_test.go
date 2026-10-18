package worker_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
	mktWorker "github.com/ecommerce/backend/internal/marketing/worker"
	"github.com/ecommerce/backend/pkg/eventstore"
	"github.com/ecommerce/backend/pkg/logger"
)

// --- Test doubles ---

type mockAnalyticsEventRepo struct {
	sync.Mutex
	events      map[string]*mktDomain.AnalyticsEvent // keyed by event_id
	inserted    []*mktDomain.AnalyticsEvent
	existsErr   error
	insertErr   error
}

func newMockAnalyticsEventRepo() *mockAnalyticsEventRepo {
	return &mockAnalyticsEventRepo{
		events:   make(map[string]*mktDomain.AnalyticsEvent),
		inserted: make([]*mktDomain.AnalyticsEvent, 0),
	}
}

func (r *mockAnalyticsEventRepo) Insert(_ context.Context, event *mktDomain.AnalyticsEvent) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	r.Lock()
	defer r.Unlock()
	r.events[event.EventID] = event
	r.inserted = append(r.inserted, event)
	return nil
}

func (r *mockAnalyticsEventRepo) ExistsByEventID(_ context.Context, eventID string) (bool, error) {
	if r.existsErr != nil {
		return false, r.existsErr
	}
	r.Lock()
	defer r.Unlock()
	_, ok := r.events[eventID]
	return ok, nil
}

func (r *mockAnalyticsEventRepo) CountByCustomer(_ context.Context, _ string, _ mktDomain.AnalyticsEventType, _ time.Time) (int64, error) {
	return 0, nil
}

func (r *mockAnalyticsEventRepo) ListByCustomer(_ context.Context, _ string, _ int) ([]*mktDomain.AnalyticsEvent, error) {
	return nil, nil
}

type mockCampaignRepo struct {
	sync.Mutex
	campaigns     map[string]*mktDomain.Campaign
	metricsIncr   map[string]map[string]int64
}

func newMockCampaignRepo() *mockCampaignRepo {
	return &mockCampaignRepo{
		campaigns:   make(map[string]*mktDomain.Campaign),
		metricsIncr: make(map[string]map[string]int64),
	}
}

func (r *mockCampaignRepo) Create(_ context.Context, campaign *mktDomain.Campaign) error {
	r.Lock()
	defer r.Unlock()
	r.campaigns[campaign.ID] = campaign
	return nil
}

func (r *mockCampaignRepo) FindByID(_ context.Context, id string) (*mktDomain.Campaign, error) {
	r.Lock()
	defer r.Unlock()
	c, ok := r.campaigns[id]
	if !ok {
		return nil, mktDomain.ErrCampaignNotFound
	}
	return c, nil
}

func (r *mockCampaignRepo) Update(_ context.Context, campaign *mktDomain.Campaign) error {
	r.Lock()
	defer r.Unlock()
	r.campaigns[campaign.ID] = campaign
	return nil
}

func (r *mockCampaignRepo) FindActiveByEventType(_ context.Context, eventType string) ([]*mktDomain.Campaign, error) {
	r.Lock()
	defer r.Unlock()
	var result []*mktDomain.Campaign
	for _, c := range r.campaigns {
		if c.IsActive() && c.MatchesEventType(eventType) {
			result = append(result, c)
		}
	}
	return result, nil
}

func (r *mockCampaignRepo) List(_ context.Context, _ string, _, _ int) ([]*mktDomain.Campaign, int64, error) {
	return nil, 0, nil
}

func (r *mockCampaignRepo) IncrementMetrics(_ context.Context, id string, field string, value int64) error {
	r.Lock()
	defer r.Unlock()
	if r.metricsIncr[id] == nil {
		r.metricsIncr[id] = make(map[string]int64)
	}
	r.metricsIncr[id][field] += value
	return nil
}

func (r *mockCampaignRepo) GetMetrics(id string) map[string]int64 {
	r.Lock()
	defer r.Unlock()
	if m, ok := r.metricsIncr[id]; ok {
		result := make(map[string]int64)
		for k, v := range m {
			result[k] = v
		}
		return result
	}
	return nil
}

type mockSegmentRepo struct {
	segments []*mktDomain.Segment
	err      error
}

func (r *mockSegmentRepo) Create(_ context.Context, _ *mktDomain.Segment) error { return r.err }
func (r *mockSegmentRepo) FindByID(_ context.Context, _ string) (*mktDomain.Segment, error) { return nil, nil }
func (r *mockSegmentRepo) Update(_ context.Context, _ *mktDomain.Segment) error { return r.err }
func (r *mockSegmentRepo) Delete(_ context.Context, _ string) error               { return r.err }
func (r *mockSegmentRepo) List(_ context.Context, _, _ int) ([]*mktDomain.Segment, int64, error) {
	return r.segments, int64(len(r.segments)), nil
}
func (r *mockSegmentRepo) FindMatchingSegments(_ context.Context, _, _ string) ([]*mktDomain.Segment, error) {
	return r.segments, r.err
}

type mockAnalyticsPublisher struct {
	sync.Mutex
	published []struct {
		eventType     string
		payload       interface{}
		aggregateID   string
		aggregateType string
	}
	err error
}

func (m *mockAnalyticsPublisher) PublishAnalyticsEvent(_ context.Context, eventType string, payload interface{}, aggregateID, aggregateType string) error {
	if m.err != nil {
		return m.err
	}
	m.Lock()
	defer m.Unlock()
	m.published = append(m.published, struct {
		eventType     string
		payload       interface{}
		aggregateID   string
		aggregateType string
	}{eventType, payload, aggregateID, aggregateType})
	return nil
}

// --- Helpers ---

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, _ := logger.New("debug", "json")
	return log
}

func newTestProcessor(t *testing.T, eventRepo *mockAnalyticsEventRepo, campaignRepo *mockCampaignRepo) *mktWorker.Processor {
	t.Helper()
	deps := mktWorker.ProcessorDependencies{
		EventRepo:    eventRepo,
		CampaignRepo: campaignRepo,
		SegmentRepo:  &mockSegmentRepo{},
		Log:          newTestLogger(t),
		Config:       mktWorker.DefaultProcessorConfig(),
	}
	return mktWorker.NewProcessor(deps)
}

func makeAnalyticsEnvelope(t *testing.T, eventType, aggregateID, aggregateType string, payload interface{}) *eventstore.Envelope {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return &eventstore.Envelope{
		EventID:       "evt-" + eventType,
		EventType:     eventType,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		CorrelationID: "corr-1",
		Payload:       data,
	}
}

// --- Tests ---

func TestHandleProductViewed_StoresEvent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.product_viewed", "product-1", "product",
		mktDomain.ProductViewedPayload{
			CustomerID: "cust-1",
			ProductID:  "product-1",
			CategoryID: "cat-1",
			Source:     "browse",
		})

	err := p.HandleProductViewed()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(eventRepo.inserted) != 1 {
		t.Fatalf("inserted = %d, want 1", len(eventRepo.inserted))
	}
	evt := eventRepo.inserted[0]
	if evt.EventType != mktDomain.AnalyticsProductViewed {
		t.Errorf("EventType = %q, want %q", evt.EventType, mktDomain.AnalyticsProductViewed)
	}
	if evt.CustomerID != "cust-1" {
		t.Errorf("CustomerID = %q, want cust-1", evt.CustomerID)
	}
	if evt.Payload["product_id"] != "product-1" {
		t.Errorf("payload product_id = %v", evt.Payload["product_id"])
	}

	metrics := p.Metrics().Snapshot()
	if metrics["processed"] != 1 {
		t.Errorf("processed = %d, want 1", metrics["processed"])
	}
	if metrics["events_stored"] != 1 {
		t.Errorf("events_stored = %d, want 1", metrics["events_stored"])
	}
}

func TestHandleProductViewed_Idempotent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.product_viewed", "product-1", "product",
		mktDomain.ProductViewedPayload{CustomerID: "cust-1", ProductID: "prod-1"})

	handler := p.HandleProductViewed()

	err := handler(context.Background(), env)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	err = handler(context.Background(), env)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if len(eventRepo.inserted) != 1 {
		t.Errorf("inserted = %d, want 1 (idempotent)", len(eventRepo.inserted))
	}
	metrics := p.Metrics().Snapshot()
	if metrics["idempotent_skipped"] != 1 {
		t.Errorf("idempotent_skipped = %d, want 1", metrics["idempotent_skipped"])
	}
}

func TestHandleProductViewed_AttributesToCampaign(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()

	// Create an active campaign that matches product_viewed
	campaign := mktDomain.NewCampaign("Browse Campaign", mktDomain.CampaignTypeProductBrowse, mktDomain.CampaignRules{
		EventTypes:  []string{"analytics.product_viewed"},
		ProductIDs:  []string{"product-1"},
	})
	campaign.Status = mktDomain.CampaignActive
	campaignRepo.Create(context.Background(), campaign)

	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.product_viewed", "product-1", "product",
		mktDomain.ProductViewedPayload{CustomerID: "cust-1", ProductID: "product-1"})

	err := p.HandleProductViewed()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metrics := p.Metrics().Snapshot()
	if metrics["campaigns_matched"] != 1 {
		t.Errorf("campaigns_matched = %d, want 1", metrics["campaigns_matched"])
	}
	if metrics["campaign_metrics_incr"] != 1 {
		t.Errorf("campaign_metrics_incr = %d, want 1", metrics["campaign_metrics_incr"])
	}

	cmetrics := campaignRepo.GetMetrics(campaign.ID)
	if cmetrics["events_received"] != 1 {
		t.Errorf("campaign events_received = %d, want 1", cmetrics["events_received"])
	}
}

func TestHandleProductViewed_CampaignProductMismatch_Skips(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()

	campaign := mktDomain.NewCampaign("Specific Product", mktDomain.CampaignTypeProductBrowse, mktDomain.CampaignRules{
		EventTypes:  []string{"analytics.product_viewed"},
		ProductIDs:  []string{"product-other"}, // targets different product
	})
	campaign.Status = mktDomain.CampaignActive
	campaignRepo.Create(context.Background(), campaign)

	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.product_viewed", "product-1", "product",
		mktDomain.ProductViewedPayload{CustomerID: "cust-1", ProductID: "product-1"})

	err := p.HandleProductViewed()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metrics := p.Metrics().Snapshot()
	if metrics["campaigns_matched"] != 0 {
		t.Errorf("campaigns_matched = %d, want 0 (product mismatch)", metrics["campaigns_matched"])
	}
}

func TestHandleCartCreated_StoresEvent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.cart_created", "cart-1", "cart",
		mktDomain.CartCreatedPayload{
			CustomerID: "cust-1",
			CartID:     "cart-1",
			ItemIDs:    []string{"item-1", "item-2"},
		})

	err := p.HandleCartCreated()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(eventRepo.inserted) != 1 {
		t.Fatalf("inserted = %d, want 1", len(eventRepo.inserted))
	}
	evt := eventRepo.inserted[0]
	if evt.EventType != mktDomain.AnalyticsCartCreated {
		t.Errorf("EventType = %q, want %q", evt.EventType, mktDomain.AnalyticsCartCreated)
	}
	if evt.Payload["cart_id"] != "cart-1" {
		t.Errorf("payload cart_id = %v", evt.Payload["cart_id"])
	}
}

func TestHandleCartAbandoned_StoresEvent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.cart_abandoned", "cart-1", "cart",
		mktDomain.CartAbandonedPayload{
			CustomerID: "cust-1",
			CartID:     "cart-1",
			ItemIDs:    []string{"item-1"},
			TotalCents: 5000,
			Currency:   "USD",
			ItemCount:  2,
		})

	err := p.HandleCartAbandoned()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(eventRepo.inserted) != 1 {
		t.Fatalf("inserted = %d, want 1", len(eventRepo.inserted))
	}
	evt := eventRepo.inserted[0]
	if evt.EventType != mktDomain.AnalyticsCartAbandoned {
		t.Errorf("EventType = %q, want %q", evt.EventType, mktDomain.AnalyticsCartAbandoned)
	}
	if evt.Payload["total_cents"] != float64(5000) {
		t.Errorf("payload total_cents = %v, want 5000", evt.Payload["total_cents"])
	}
}

func TestHandleCartAbandoned_AttributesToCampaign(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()

	campaign := mktDomain.NewCampaign("Abandoned Cart Recovery", mktDomain.CampaignTypeAbandonedCart, mktDomain.CampaignRules{
		EventTypes: []string{"analytics.cart_abandoned"},
	})
	campaign.Status = mktDomain.CampaignActive
	campaignRepo.Create(context.Background(), campaign)

	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.cart_abandoned", "cart-1", "cart",
		mktDomain.CartAbandonedPayload{CustomerID: "cust-1", CartID: "cart-1"})

	err := p.HandleCartAbandoned()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metrics := p.Metrics().Snapshot()
	if metrics["campaigns_matched"] != 1 {
		t.Errorf("campaigns_matched = %d, want 1", metrics["campaigns_matched"])
	}
}

func TestHandleCheckoutStarted_StoresEvent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.checkout_started", "cart-1", "cart",
		mktDomain.CheckoutStartedPayload{
			CustomerID: "cust-1",
			CartID:     "cart-1",
			TotalCents: 10000,
			Currency:   "USD",
			ItemCount:  3,
		})

	err := p.HandleCheckoutStarted()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(eventRepo.inserted) != 1 {
		t.Fatalf("inserted = %d, want 1", len(eventRepo.inserted))
	}
	evt := eventRepo.inserted[0]
	if evt.EventType != mktDomain.AnalyticsCheckoutStarted {
		t.Errorf("EventType = %q, want %q", evt.EventType, mktDomain.AnalyticsCheckoutStarted)
	}
}

func TestHandleOrderCompleted_StoresEvent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.order_completed", "order-1", "order",
		mktDomain.OrderCompletedPayload{
			CustomerID:  "cust-1",
			OrderID:     "order-1",
			OrderNumber: "ORD-123",
			TotalCents:  19999,
			Currency:    "USD",
			ItemIDs:     []string{"item-1"},
			CategoryIDs: []string{"cat-1"},
		})

	err := p.HandleOrderCompleted()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(eventRepo.inserted) != 1 {
		t.Fatalf("inserted = %d, want 1", len(eventRepo.inserted))
	}
	evt := eventRepo.inserted[0]
	if evt.EventType != mktDomain.AnalyticsOrderCompleted {
		t.Errorf("EventType = %q, want %q", evt.EventType, mktDomain.AnalyticsOrderCompleted)
	}
	if evt.Payload["order_id"] != "order-1" {
		t.Errorf("payload order_id = %v", evt.Payload["order_id"])
	}
	if evt.Payload["total_cents"] != float64(19999) {
		t.Errorf("payload total_cents = %v, want 19999", evt.Payload["total_cents"])
	}
}

func TestHandleOrderCompleted_TracksConversion(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()

	// Create an abandoned cart campaign
	campaign := mktDomain.NewCampaign("Cart Recovery", mktDomain.CampaignTypeAbandonedCart, mktDomain.CampaignRules{
		EventTypes:          []string{"analytics.cart_abandoned"},
		MinAbandonedMinutes: 30,
	})
	campaign.Status = mktDomain.CampaignActive
	campaignRepo.Create(context.Background(), campaign)

	// Pre-seed an abandoned cart event for this customer
	abandonedEvt := mktDomain.NewAnalyticsEvent(
		"evt-prev-abandoned", string(mktDomain.AnalyticsCartAbandoned),
		"cust-1", "cart-1", "corr-1",
		map[string]any{"cart_id": "cart-1"},
	)
	eventRepo.Insert(context.Background(), abandonedEvt)

	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.order_completed", "order-1", "order",
		mktDomain.OrderCompletedPayload{
			CustomerID:  "cust-1",
			OrderID:     "order-1",
			OrderNumber: "ORD-123",
			TotalCents:  19999,
			Currency:    "USD",
		})

	err := p.HandleOrderCompleted()(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmetrics := campaignRepo.GetMetrics(campaign.ID)
	if cmetrics["conversions"] != 1 {
		t.Errorf("conversions = %d, want 1", cmetrics["conversions"])
	}
	if cmetrics["revenue_cents"] != 19999 {
		t.Errorf("revenue_cents = %d, want 19999", cmetrics["revenue_cents"])
	}
}

func TestHandleProductViewed_MalformedPayload_ReturnsNil(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := &eventstore.Envelope{
		EventID:   "evt-bad",
		EventType: "analytics.product_viewed",
		Payload:   []byte("not-json"),
	}

	err := p.HandleProductViewed()(context.Background(), env)
	if err != nil {
		t.Fatalf("expected nil for malformed payload, got: %v", err)
	}
	if len(eventRepo.inserted) != 0 {
		t.Error("no events should be stored for malformed payload")
	}
}

func TestHandleCartCreated_Idempotent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.cart_created", "cart-1", "cart",
		mktDomain.CartCreatedPayload{CustomerID: "cust-1", CartID: "cart-1"})

	handler := p.HandleCartCreated()
	_ = handler(context.Background(), env)
	_ = handler(context.Background(), env)

	if len(eventRepo.inserted) != 1 {
		t.Errorf("inserted = %d, want 1", len(eventRepo.inserted))
	}
	metrics := p.Metrics().Snapshot()
	if metrics["idempotent_skipped"] != 1 {
		t.Errorf("idempotent_skipped = %d, want 1", metrics["idempotent_skipped"])
	}
}

func TestHandleCheckoutStarted_Idempotent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.checkout_started", "cart-1", "cart",
		mktDomain.CheckoutStartedPayload{CustomerID: "cust-1", CartID: "cart-1"})

	handler := p.HandleCheckoutStarted()
	_ = handler(context.Background(), env)
	_ = handler(context.Background(), env)

	if len(eventRepo.inserted) != 1 {
		t.Errorf("inserted = %d, want 1", len(eventRepo.inserted))
	}
}

func TestHandleOrderCompleted_Idempotent(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.order_completed", "order-1", "order",
		mktDomain.OrderCompletedPayload{CustomerID: "cust-1", OrderID: "order-1", OrderNumber: "ORD-1", TotalCents: 100, Currency: "USD"})

	handler := p.HandleOrderCompleted()
	_ = handler(context.Background(), env)
	_ = handler(context.Background(), env)

	if len(eventRepo.inserted) != 1 {
		t.Errorf("inserted = %d, want 1", len(eventRepo.inserted))
	}
}

func TestMetricsSnapshot_ContainsAllKeys(t *testing.T) {
	eventRepo := newMockAnalyticsEventRepo()
	campaignRepo := newMockCampaignRepo()
	p := newTestProcessor(t, eventRepo, campaignRepo)

	env := makeAnalyticsEnvelope(t, "analytics.product_viewed", "p1", "product",
		mktDomain.ProductViewedPayload{CustomerID: "c1", ProductID: "p1"})

	_ = p.HandleProductViewed()(context.Background(), env)

	snapshot := p.Metrics().Snapshot()
	expectedKeys := []string{"processed", "idempotent_skipped", "events_stored", "campaigns_matched", "segments_matched", "campaign_metrics_incr", "errors"}
	for _, key := range expectedKeys {
		if _, ok := snapshot[key]; !ok {
			t.Errorf("missing metric key %q", key)
		}
	}
}

func TestDefaultProcessorConfig(t *testing.T) {
	cfg := mktWorker.DefaultProcessorConfig()
	if cfg.AbandonedCartThresholdMinutes != 60 {
		t.Errorf("AbandonedCartThresholdMinutes = %d, want 60", cfg.AbandonedCartThresholdMinutes)
	}
	if cfg.MaxEventsPerCustomerPerDay != 100 {
		t.Errorf("MaxEventsPerCustomerPerDay = %d, want 100", cfg.MaxEventsPerCustomerPerDay)
	}
}

// --- Domain tests ---

func TestCampaignLifecycle(t *testing.T) {
	c := mktDomain.NewCampaign("Test", mktDomain.CampaignTypeAbandonedCart, mktDomain.CampaignRules{
		EventTypes: []string{"analytics.cart_abandoned"},
	})

	if c.Status != mktDomain.CampaignDraft {
		t.Errorf("initial status = %q, want DRAFT", c.Status)
	}

	// DRAFT -> ACTIVE
	if err := c.TransitionTo(mktDomain.CampaignActive); err != nil {
		t.Fatalf("DRAFT->ACTIVE: %v", err)
	}
	if !c.IsActive() {
		t.Error("campaign should be active")
	}

	// ACTIVE -> PAUSED
	if err := c.TransitionTo(mktDomain.CampaignPaused); err != nil {
		t.Fatalf("ACTIVE->PAUSED: %v", err)
	}

	// PAUSED -> ACTIVE
	if err := c.TransitionTo(mktDomain.CampaignActive); err != nil {
		t.Fatalf("PAUSED->ACTIVE: %v", err)
	}

	// ACTIVE -> COMPLETED
	if err := c.TransitionTo(mktDomain.CampaignCompleted); err != nil {
		t.Fatalf("ACTIVE->COMPLETED: %v", err)
	}

	// COMPLETED -> ARCHIVED
	if err := c.TransitionTo(mktDomain.CampaignArchived); err != nil {
		t.Fatalf("COMPLETED->ARCHIVED: %v", err)
	}

	// ARCHIVED -> ACTIVE should fail
	if err := c.TransitionTo(mktDomain.CampaignActive); err == nil {
		t.Error("ARCHIVED->ACTIVE should fail")
	}
}

func TestCampaignMatchesEventType(t *testing.T) {
	c := mktDomain.NewCampaign("Test", mktDomain.CampaignTypeAbandonedCart, mktDomain.CampaignRules{
		EventTypes: []string{"analytics.cart_abandoned", "analytics.checkout_started"},
	})

	if !c.MatchesEventType("analytics.cart_abandoned") {
		t.Error("should match cart_abandoned")
	}
	if !c.MatchesEventType("analytics.checkout_started") {
		t.Error("should match checkout_started")
	}
	if c.MatchesEventType("analytics.order_completed") {
		t.Error("should not match order_completed")
	}
}

func TestCampaignMatchesProduct(t *testing.T) {
	c := mktDomain.NewCampaign("Test", mktDomain.CampaignTypeProductBrowse, mktDomain.CampaignRules{
		ProductIDs: []string{"p1", "p2"},
	})

	if !c.MatchesProduct("p1") {
		t.Error("should match p1")
	}
	if c.MatchesProduct("p3") {
		t.Error("should not match p3")
	}

	// Empty product list = matches all
	cAll := mktDomain.NewCampaign("All", mktDomain.CampaignTypePromotional, mktDomain.CampaignRules{})
	if !cAll.MatchesProduct("anything") {
		t.Error("empty product list should match all")
	}
}

func TestPromotionLifecycle(t *testing.T) {
	p := mktDomain.NewPromotion("Sale", "SAVE20", mktDomain.DiscountPercentage, 20)

	if p.Status != mktDomain.PromotionDraft {
		t.Errorf("initial status = %q, want DRAFT", p.Status)
	}

	// DRAFT -> ACTIVE
	if err := p.TransitionTo(mktDomain.PromotionActive); err != nil {
		t.Fatalf("DRAFT->ACTIVE: %v", err)
	}

	// ACTIVE -> PAUSED
	if err := p.TransitionTo(mktDomain.PromotionPaused); err != nil {
		t.Fatalf("ACTIVE->PAUSED: %v", err)
	}

	// PAUSED -> ACTIVE
	if err := p.TransitionTo(mktDomain.PromotionActive); err != nil {
		t.Fatalf("PAUSED->ACTIVE: %v", err)
	}

	// ACTIVE -> EXPIRED
	if err := p.TransitionTo(mktDomain.PromotionExpired); err != nil {
		t.Fatalf("ACTIVE->EXPIRED: %v", err)
	}

	// EXPIRED -> ARCHIVED
	if err := p.TransitionTo(mktDomain.PromotionArchived); err != nil {
		t.Fatalf("EXPIRED->ARCHIVED: %v", err)
	}

	// ARCHIVED -> ACTIVE should fail
	if err := p.TransitionTo(mktDomain.PromotionActive); err == nil {
		t.Error("ARCHIVED->ACTIVE should fail")
	}
}

func TestPromotionIsUsable(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	p := mktDomain.NewPromotion("Test", "CODE1", mktDomain.DiscountFixed, 1000)
	p.Status = mktDomain.PromotionActive
	p.StartAt = &future

	if p.IsUsable() {
		t.Error("should not be usable before start")
	}

	p.StartAt = &past
	p.EndAt = &past
	if p.IsUsable() {
		t.Error("should not be usable after end")
	}

	p.EndAt = nil
	if !p.IsUsable() {
		t.Error("should be usable with no end date")
	}

	p.UsageLimit = 5
	p.UsageCount = 5
	if p.IsUsable() {
		t.Error("should not be usable when usage limit reached")
	}
}

func TestPromotionAppliesToProduct(t *testing.T) {
	p := mktDomain.NewPromotion("Test", "CODE", mktDomain.DiscountPercentage, 10)
	p.AppliesToProductIDs = []string{"p1", "p2"}

	if !p.AppliesToProduct("p1") {
		t.Error("should apply to p1")
	}
	if p.AppliesToProduct("p3") {
		t.Error("should not apply to p3")
	}

	pAll := mktDomain.NewPromotion("All", "ALL", mktDomain.DiscountPercentage, 10)
	if !pAll.AppliesToProduct("anything") {
		t.Error("empty list should apply to all")
	}
}

func TestSegmentCreation(t *testing.T) {
	s := mktDomain.NewSegment("VIP", mktDomain.SegmentTypeDynamic, []mktDomain.SegmentRule{
		{Field: "event_type", Operator: "eq", Value: "analytics.order_completed"},
	})

	if s.Type != mktDomain.SegmentTypeDynamic {
		t.Errorf("type = %q, want DYNAMIC", s.Type)
	}
	if len(s.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(s.Rules))
	}
	if s.Rules[0].Field != "event_type" {
		t.Errorf("rule field = %q, want event_type", s.Rules[0].Field)
	}
	if s.CustomerCount != 0 {
		t.Errorf("customer_count = %d, want 0", s.CustomerCount)
	}
}

func TestAnalyticsEventTypeConstants(t *testing.T) {
	types := []struct {
		constVal mktDomain.AnalyticsEventType
		expected string
	}{
		{mktDomain.AnalyticsProductViewed, "analytics.product_viewed"},
		{mktDomain.AnalyticsCartCreated, "analytics.cart_created"},
		{mktDomain.AnalyticsCartAbandoned, "analytics.cart_abandoned"},
		{mktDomain.AnalyticsCheckoutStarted, "analytics.checkout_started"},
		{mktDomain.AnalyticsOrderCompleted, "analytics.order_completed"},
	}

	for _, tt := range types {
		if string(tt.constVal) != tt.expected {
			t.Errorf("%s = %q, want %q", tt.expected, tt.constVal, tt.expected)
		}
	}
}
