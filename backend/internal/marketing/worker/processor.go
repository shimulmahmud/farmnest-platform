package worker

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
	"github.com/ecommerce/backend/pkg/eventstore"
	"github.com/ecommerce/backend/pkg/logger"
)

// AnalyticsEventPublisher publishes analytics events to the outbox.
// This interface allows the processor to emit analytics events without
// depending on NATS or the outbox directly.
type AnalyticsEventPublisher interface {
	PublishAnalyticsEvent(ctx context.Context, eventType string, payload interface{}, aggregateID, aggregateType string) error
}

// ProcessorConfig holds configuration for the analytics processor.
type ProcessorConfig struct {
	// AbandonedCartThreshold is the minimum time since last cart activity
	// before emitting a cart_abandoned event (minutes).
	AbandonedCartThresholdMinutes int
	// MaxEventsPerCustomerPerDay limits how many analytics events are stored
	// per customer per day to prevent unbounded growth.
	MaxEventsPerCustomerPerDay int
}

// DefaultProcessorConfig returns sensible defaults.
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		AbandonedCartThresholdMinutes: 60,
		MaxEventsPerCustomerPerDay:    100,
	}
}

// ProcessorDependencies holds the injected dependencies for the event processor.
type ProcessorDependencies struct {
	EventRepo      mktDomain.AnalyticsEventRepository
	CampaignRepo   mktDomain.CampaignRepository
	SegmentRepo    mktDomain.SegmentRepository
	EventPublisher AnalyticsEventPublisher
	Log            *logger.Logger
	Config         ProcessorConfig
}

// Metrics tracks processor statistics.
type Metrics struct {
	Processed            atomic.Int64
	IdempotentSkipped    atomic.Int64
	EventsStored         atomic.Int64
	CampaignsMatched     atomic.Int64
	SegmentsMatched      atomic.Int64
	CampaignMetricsIncr  atomic.Int64
	Errors               atomic.Int64
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"processed":             m.Processed.Load(),
		"idempotent_skipped":     m.IdempotentSkipped.Load(),
		"events_stored":          m.EventsStored.Load(),
		"campaigns_matched":      m.CampaignsMatched.Load(),
		"segments_matched":       m.SegmentsMatched.Load(),
		"campaign_metrics_incr":  m.CampaignMetricsIncr.Load(),
		"errors":                 m.Errors.Load(),
	}
}

// Processor consumes analytics events from JetStream, persists them,
// evaluates customer segments, and attributes events to active campaigns.
// It never blocks HTTP requests — it runs exclusively as JetStream consumers.
type Processor struct {
	deps    ProcessorDependencies
	metrics Metrics
}

// NewProcessor creates a marketing analytics processor.
func NewProcessor(deps ProcessorDependencies) *Processor {
	return &Processor{deps: deps}
}

// Metrics returns the processor metrics.
func (p *Processor) Metrics() *Metrics {
	return &p.metrics
}

// HandleProductViewed processes analytics.product_viewed events.
func (p *Processor) HandleProductViewed() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var payload mktDomain.ProductViewedPayload
		if err := env.DecodePayload(&payload); err != nil {
			p.deps.Log.Errorw("marketing: failed to decode product_viewed payload",
				"event_id", env.EventID, "error", err)
			return nil // non-retryable
		}

		p.metrics.Processed.Add(1)

		// Idempotency check
		exists, err := p.deps.EventRepo.ExistsByEventID(ctx, env.EventID)
		if err != nil {
			p.deps.Log.Warnw("marketing: failed idempotency check for product_viewed",
				"event_id", env.EventID, "error", err)
		}
		if exists {
			p.metrics.IdempotentSkipped.Add(1)
			return nil
		}

		// Persist the analytics event
		analyticsEvt := mktDomain.NewAnalyticsEvent(
			env.EventID, string(mktDomain.AnalyticsProductViewed),
			payload.CustomerID, env.AggregateID, env.CorrelationID,
			map[string]any{
				"product_id":  payload.ProductID,
				"variant_id":  payload.VariantID,
				"category_id": payload.CategoryID,
				"source":      payload.Source,
			},
		)
		if storeErr := p.deps.EventRepo.Insert(ctx, analyticsEvt); storeErr != nil {
			p.deps.Log.Errorw("marketing: failed to store product_viewed event",
				"event_id", env.EventID, "error", storeErr)
			p.metrics.Errors.Add(1)
			return nil // non-retryable, event is in JetStream for replay
		}
		p.metrics.EventsStored.Add(1)

		// Attribute to matching campaigns
		p.attributeToCampaigns(ctx, env.EventType, payload.CustomerID, payload.ProductID, payload.CategoryID)

		return nil
	}
}

// HandleCartCreated processes analytics.cart_created events.
func (p *Processor) HandleCartCreated() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var payload mktDomain.CartCreatedPayload
		if err := env.DecodePayload(&payload); err != nil {
			p.deps.Log.Errorw("marketing: failed to decode cart_created payload",
				"event_id", env.EventID, "error", err)
			return nil
		}

		p.metrics.Processed.Add(1)

		exists, err := p.deps.EventRepo.ExistsByEventID(ctx, env.EventID)
		if err != nil {
			p.deps.Log.Warnw("marketing: failed idempotency check for cart_created",
				"event_id", env.EventID, "error", err)
		}
		if exists {
			p.metrics.IdempotentSkipped.Add(1)
			return nil
		}

		analyticsEvt := mktDomain.NewAnalyticsEvent(
			env.EventID, string(mktDomain.AnalyticsCartCreated),
			payload.CustomerID, env.AggregateID, env.CorrelationID,
			map[string]any{
				"cart_id":  payload.CartID,
				"item_ids": payload.ItemIDs,
			},
		)
		if storeErr := p.deps.EventRepo.Insert(ctx, analyticsEvt); storeErr != nil {
			p.deps.Log.Errorw("marketing: failed to store cart_created event",
				"event_id", env.EventID, "error", storeErr)
			p.metrics.Errors.Add(1)
			return nil
		}
		p.metrics.EventsStored.Add(1)

		p.attributeToCampaigns(ctx, env.EventType, payload.CustomerID, "", "")

		return nil
	}
}

// HandleCartAbandoned processes analytics.cart_abandoned events.
func (p *Processor) HandleCartAbandoned() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var payload mktDomain.CartAbandonedPayload
		if err := env.DecodePayload(&payload); err != nil {
			p.deps.Log.Errorw("marketing: failed to decode cart_abandoned payload",
				"event_id", env.EventID, "error", err)
			return nil
		}

		p.metrics.Processed.Add(1)

		exists, err := p.deps.EventRepo.ExistsByEventID(ctx, env.EventID)
		if err != nil {
			p.deps.Log.Warnw("marketing: failed idempotency check for cart_abandoned",
				"event_id", env.EventID, "error", err)
		}
		if exists {
			p.metrics.IdempotentSkipped.Add(1)
			return nil
		}

		analyticsEvt := mktDomain.NewAnalyticsEvent(
			env.EventID, string(mktDomain.AnalyticsCartAbandoned),
			payload.CustomerID, env.AggregateID, env.CorrelationID,
			map[string]any{
				"cart_id":      payload.CartID,
				"item_ids":     payload.ItemIDs,
				"total_cents":  payload.TotalCents,
				"currency":     payload.Currency,
				"item_count":   payload.ItemCount,
			},
		)
		if storeErr := p.deps.EventRepo.Insert(ctx, analyticsEvt); storeErr != nil {
			p.deps.Log.Errorw("marketing: failed to store cart_abandoned event",
				"event_id", env.EventID, "error", storeErr)
			p.metrics.Errors.Add(1)
			return nil
		}
		p.metrics.EventsStored.Add(1)

		// Cart abandonment is the primary campaign trigger
		p.attributeToCampaigns(ctx, env.EventType, payload.CustomerID, "", "")

		return nil
	}
}

// HandleCheckoutStarted processes analytics.checkout_started events.
func (p *Processor) HandleCheckoutStarted() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var payload mktDomain.CheckoutStartedPayload
		if err := env.DecodePayload(&payload); err != nil {
			p.deps.Log.Errorw("marketing: failed to decode checkout_started payload",
				"event_id", env.EventID, "error", err)
			return nil
		}

		p.metrics.Processed.Add(1)

		exists, err := p.deps.EventRepo.ExistsByEventID(ctx, env.EventID)
		if err != nil {
			p.deps.Log.Warnw("marketing: failed idempotency check for checkout_started",
				"event_id", env.EventID, "error", err)
		}
		if exists {
			p.metrics.IdempotentSkipped.Add(1)
			return nil
		}

		analyticsEvt := mktDomain.NewAnalyticsEvent(
			env.EventID, string(mktDomain.AnalyticsCheckoutStarted),
			payload.CustomerID, env.AggregateID, env.CorrelationID,
			map[string]any{
				"cart_id":     payload.CartID,
				"item_count":  payload.ItemCount,
				"total_cents": payload.TotalCents,
				"currency":    payload.Currency,
			},
		)
		if storeErr := p.deps.EventRepo.Insert(ctx, analyticsEvt); storeErr != nil {
			p.deps.Log.Errorw("marketing: failed to store checkout_started event",
				"event_id", env.EventID, "error", storeErr)
			p.metrics.Errors.Add(1)
			return nil
		}
		p.metrics.EventsStored.Add(1)

		p.attributeToCampaigns(ctx, env.EventType, payload.CustomerID, "", "")

		return nil
	}
}

// HandleOrderCompleted processes analytics.order_completed events.
func (p *Processor) HandleOrderCompleted() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var payload mktDomain.OrderCompletedPayload
		if err := env.DecodePayload(&payload); err != nil {
			p.deps.Log.Errorw("marketing: failed to decode order_completed payload",
			"event_id", env.EventID, "error", err)
			return nil
		}

		p.metrics.Processed.Add(1)

		exists, err := p.deps.EventRepo.ExistsByEventID(ctx, env.EventID)
		if err != nil {
			p.deps.Log.Warnw("marketing: failed idempotency check for order_completed",
				"event_id", env.EventID, "error", err)
		}
		if exists {
			p.metrics.IdempotentSkipped.Add(1)
			return nil
		}

		analyticsEvt := mktDomain.NewAnalyticsEvent(
			env.EventID, string(mktDomain.AnalyticsOrderCompleted),
			payload.CustomerID, env.AggregateID, env.CorrelationID,
			map[string]any{
				"order_id":      payload.OrderID,
				"order_number":  payload.OrderNumber,
				"total_cents":   payload.TotalCents,
				"currency":      payload.Currency,
				"item_ids":      payload.ItemIDs,
				"category_ids":  payload.CategoryIDs,
			},
		)
		if storeErr := p.deps.EventRepo.Insert(ctx, analyticsEvt); storeErr != nil {
			p.deps.Log.Errorw("marketing: failed to store order_completed event",
				"event_id", env.EventID, "error", storeErr)
			p.metrics.Errors.Add(1)
			return nil
		}
		p.metrics.EventsStored.Add(1)

		// Order completed can attribute to post-purchase campaigns and track conversions
		p.attributeToCampaigns(ctx, env.EventType, payload.CustomerID, "", "")

		// Track conversion metrics for campaigns that had previous cart_abandoned events
		p.trackConversion(ctx, payload.CustomerID, payload.TotalCents)

		return nil
	}
}

// --- Internal helpers ---

// attributeToCampaigns finds active campaigns matching the event type and increments their metrics.
func (p *Processor) attributeToCampaigns(ctx context.Context, eventType, customerID, productID, categoryID string) {
	campaigns, err := p.deps.CampaignRepo.FindActiveByEventType(ctx, eventType)
	if err != nil {
		p.deps.Log.Warnw("marketing: failed to find campaigns for event",
			"event_type", eventType, "error", err)
		return
	}

	for _, campaign := range campaigns {
		// Check product/category targeting
		if !campaign.MatchesProduct(productID) {
			continue
		}
		if !campaign.MatchesCategory(categoryID) {
			continue
		}

		p.metrics.CampaignsMatched.Add(1)

		// Increment events_received
		if err := p.deps.CampaignRepo.IncrementMetrics(ctx, campaign.ID, "events_received", 1); err != nil {
			p.deps.Log.Warnw("marketing: failed to increment campaign events_received",
				"campaign_id", campaign.ID, "error", err)
			continue
		}
		p.metrics.CampaignMetricsIncr.Add(1)

		// Increment customers_reached (non-atomic: approximate, acceptable for analytics)
		if err := p.deps.CampaignRepo.IncrementMetrics(ctx, campaign.ID, "customers_reached", 1); err != nil {
			p.deps.Log.Warnw("marketing: failed to increment campaign customers_reached",
				"campaign_id", campaign.ID, "error", err)
		}

		p.deps.Log.Debugw("marketing: event attributed to campaign",
			"event_type", eventType, "campaign_id", campaign.ID, "campaign_name", campaign.Name)
	}
}

// trackConversion looks for campaigns that previously received cart_abandoned events
// for this customer and increments their conversion metrics.
func (p *Processor) trackConversion(ctx context.Context, customerID string, totalCents int) {
	// Find campaigns that have been tracking this customer via cart abandonment
	campaigns, err := p.deps.CampaignRepo.FindActiveByEventType(ctx, string(mktDomain.AnalyticsCartAbandoned))
	if err != nil {
		p.deps.Log.Warnw("marketing: failed to find campaigns for conversion tracking",
			"customer_id", customerID, "error", err)
		return
	}

	for _, campaign := range campaigns {
		// Check if this customer had recent cart_abandoned events (within campaign window)
		if campaign.Rules.MinAbandonedMinutes > 0 {
			since := time.Now().UTC().Add(-time.Duration(campaign.Rules.MinAbandonedMinutes*2) * time.Minute)
			count, err := p.deps.EventRepo.CountByCustomer(ctx, customerID, mktDomain.AnalyticsCartAbandoned, since)
			if err != nil {
				p.deps.Log.Warnw("marketing: failed to count abandoned cart events",
					"campaign_id", campaign.ID, "customer_id", customerID, "error", err)
				continue
			}
			if count == 0 {
				continue // no abandoned carts for this customer, skip conversion attribution
			}
		}

		// Increment conversion metrics
		if err := p.deps.CampaignRepo.IncrementMetrics(ctx, campaign.ID, "conversions", 1); err != nil {
			p.deps.Log.Warnw("marketing: failed to increment campaign conversions",
				"campaign_id", campaign.ID, "error", err)
			continue
		}
		if err := p.deps.CampaignRepo.IncrementMetrics(ctx, campaign.ID, "revenue_cents", int64(totalCents)); err != nil {
			p.deps.Log.Warnw("marketing: failed to increment campaign revenue",
				"campaign_id", campaign.ID, "error", err)
		}
		if err := p.deps.CampaignRepo.IncrementMetrics(ctx, campaign.ID, "messages_sent", 1); err != nil {
			p.deps.Log.Warnw("marketing: failed to increment campaign messages_sent",
				"campaign_id", campaign.ID, "error", err)
		}
	}
}

// PublishAnalyticsEvent publishes an analytics event through the outbox.
// This is a convenience method for external callers (e.g., cart service) to emit analytics events.
func PublishAnalyticsEvent(publisher AnalyticsEventPublisher, eventType string, payload interface{}, aggregateID, aggregateType string) error {
	return publisher.PublishAnalyticsEvent(context.Background(), eventType, payload, aggregateID, aggregateType)
}

// toMap converts a payload struct to a map[string]any for storage.
func toMap(v interface{}) map[string]any {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}
