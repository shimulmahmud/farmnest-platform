package outbox

import (
        "context"

        ordDomain "github.com/ecommerce/backend/internal/orders/domain"
        "github.com/ecommerce/backend/internal/payments/domain"
        invDomain "github.com/ecommerce/backend/internal/inventory/domain"
        shipDomain "github.com/ecommerce/backend/internal/shipping/domain"
        prodDomain "github.com/ecommerce/backend/internal/products/domain"
        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/ecommerce/backend/pkg/telemetry"
)

// --- Order Event Publisher Adapter ---

// OrderPublisherAdapter implements orders.domain.EventPublisher
// by writing events to the outbox instead of publishing directly to NATS.
type OrderPublisherAdapter struct {
        repo OutboxRepository
}

// NewOrderPublisherAdapter creates an outbox-backed order event publisher.
func NewOrderPublisherAdapter(repo OutboxRepository) *OrderPublisherAdapter {
        return &OrderPublisherAdapter{repo: repo}
}

func (a *OrderPublisherAdapter) Publish(ctx context.Context, subject string, event *ordDomain.OrderEvent) error {
        aggregateID := event.OrderID

        // Propagate correlation_id and causation_id from the OTel trace context.
        correlationID := telemetry.CorrelationIDFromContext(ctx)
        causationID := telemetry.CausationIDFromContext(ctx)

        env, err := eventstore.NewEnvelopeWithCorrelation(subject, aggregateID, "order", 1, event, correlationID, causationID)
        if err != nil {
                return err
        }

        // Override the event type with the subject-based name.
        env.EventType = subject
        evt := NewOutboxEvent(env)
        return a.repo.Insert(ctx, evt)
}

// --- Payment Event Publisher Adapter ---

// PaymentPublisherAdapter implements payments.domain.PaymentEventPublisher
// by writing events to the outbox.
type PaymentPublisherAdapter struct {
        repo OutboxRepository
}

// NewPaymentPublisherAdapter creates an outbox-backed payment event publisher.
func NewPaymentPublisherAdapter(repo OutboxRepository) *PaymentPublisherAdapter {
        return &PaymentPublisherAdapter{repo: repo}
}

func (a *PaymentPublisherAdapter) Publish(ctx context.Context, event *domain.PaymentEvent) error {
        eventType := event.EventType
        aggregateID := event.PaymentID

        // Propagate correlation_id and causation_id from the OTel trace context.
        correlationID := telemetry.CorrelationIDFromContext(ctx)
        causationID := telemetry.CausationIDFromContext(ctx)

        env, err := eventstore.NewEnvelopeWithCorrelation(eventType, aggregateID, "payment", 1, event, correlationID, causationID)
        if err != nil {
                return err
        }

        evt := NewOutboxEvent(env)
        return a.repo.Insert(ctx, evt)
}

// --- Inventory Event Publisher Adapter ---

// InventoryPublisherAdapter implements inventory.domain.EventPublisher
// by writing events to the outbox.
type InventoryPublisherAdapter struct {
        repo OutboxRepository
}

// NewInventoryPublisherAdapter creates an outbox-backed inventory event publisher.
func NewInventoryPublisherAdapter(repo OutboxRepository) *InventoryPublisherAdapter {
        return &InventoryPublisherAdapter{repo: repo}
}

func (a *InventoryPublisherAdapter) Publish(ctx context.Context, subject string, event *invDomain.InventoryEvent) error {
        aggregateID := event.SKU + ":" + event.WarehouseID

        // Propagate correlation_id and causation_id from the OTel trace context.
        correlationID := telemetry.CorrelationIDFromContext(ctx)
        causationID := telemetry.CausationIDFromContext(ctx)

        env, err := eventstore.NewEnvelopeWithCorrelation(subject, aggregateID, "inventory", 1, event, correlationID, causationID)
        if err != nil {
                return err
        }

        // Override the event type with the subject-based name.
        env.EventType = subject

        evt := NewOutboxEvent(env)
        return a.repo.Insert(ctx, evt)
}

// --- Shipping Event Publisher Adapter ---

// ShippingPublisherAdapter implements shipping.domain.EventPublisher
// by writing events to the outbox.
type ShippingPublisherAdapter struct {
        repo OutboxRepository
}

// NewShippingPublisherAdapter creates an outbox-backed shipping event publisher.
func NewShippingPublisherAdapter(repo OutboxRepository) *ShippingPublisherAdapter {
        return &ShippingPublisherAdapter{repo: repo}
}

func (a *ShippingPublisherAdapter) Publish(ctx context.Context, subject string, event *shipDomain.ShippingEvent) error {
        aggregateID := event.ShipmentID

        // Propagate correlation_id and causation_id from the OTel trace context.
        correlationID := telemetry.CorrelationIDFromContext(ctx)
        causationID := telemetry.CausationIDFromContext(ctx)

        env, err := eventstore.NewEnvelopeWithCorrelation(subject, aggregateID, "shipment", 1, event, correlationID, causationID)
        if err != nil {
                return err
        }

        env.EventType = subject
        evt := NewOutboxEvent(env)
        return a.repo.Insert(ctx, evt)
}

// --- Product Event Publisher Adapter ---

// ProductPublisherAdapter implements products.domain.EventPublisher
// by writing events to the outbox.
type ProductPublisherAdapter struct {
        repo OutboxRepository
}

// NewProductPublisherAdapter creates an outbox-backed product event publisher.
func NewProductPublisherAdapter(repo OutboxRepository) *ProductPublisherAdapter {
        return &ProductPublisherAdapter{repo: repo}
}

func (a *ProductPublisherAdapter) Publish(ctx context.Context, subject string, event *prodDomain.ProductEvent) error {
        aggregateID := event.ProductID

        // Propagate correlation_id and causation_id from the OTel trace context.
        correlationID := telemetry.CorrelationIDFromContext(ctx)
        causationID := telemetry.CausationIDFromContext(ctx)

        env, err := eventstore.NewEnvelopeWithCorrelation(subject, aggregateID, "product", 1, event, correlationID, causationID)
        if err != nil {
                return err
        }

        env.EventType = subject
        evt := NewOutboxEvent(env)
        return a.repo.Insert(ctx, evt)
}

// --- Marketing Analytics Event Publisher Adapter ---

// MarketingPublisherAdapter publishes analytics events to the outbox.
// It implements worker.AnalyticsEventPublisher.
type MarketingPublisherAdapter struct {
        repo OutboxRepository
}

// NewMarketingPublisherAdapter creates an outbox-backed marketing analytics event publisher.
func NewMarketingPublisherAdapter(repo OutboxRepository) *MarketingPublisherAdapter {
        return &MarketingPublisherAdapter{repo: repo}
}

func (a *MarketingPublisherAdapter) PublishAnalyticsEvent(ctx context.Context, eventType string, payload interface{}, aggregateID, aggregateType string) error {
        aggType := "analytics"
        if aggregateType != "" {
                aggType = aggregateType
        }

        // Propagate correlation_id and causation_id from the OTel trace context.
        correlationID := telemetry.CorrelationIDFromContext(ctx)
        causationID := telemetry.CausationIDFromContext(ctx)

        env, err := eventstore.NewEnvelopeWithCorrelation(eventType, aggregateID, aggType, 1, payload, correlationID, causationID)
        if err != nil {
                return err
        }

        env.EventType = eventType
        evt := NewOutboxEvent(env)
        return a.repo.Insert(ctx, evt)
}
