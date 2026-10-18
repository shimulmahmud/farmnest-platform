package eventstore

import (
	"context"
	"fmt"

	"github.com/ecommerce/backend/pkg/logger"
	"github.com/ecommerce/backend/pkg/telemetry"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes domain events as Envelopes to JetStream.
// It is safe for concurrent use.
type Publisher struct {
	js          jetstream.JetStream
	log         *logger.Logger
	natsMetrics *telemetry.NATSMetrics
	natsTracer  *telemetry.NATSTracer
}

// NewPublisher creates a new event publisher.
func NewPublisher(js jetstream.JetStream, log *logger.Logger) *Publisher {
	return &Publisher{js: js, log: log}
}

// SetNATSMetrics injects NATS publish metrics.
func (p *Publisher) SetNATSMetrics(m *telemetry.NATSMetrics) {
	p.natsMetrics = m
}

// SetNATSTracer injects NATS publish tracing.
func (p *Publisher) SetNATSTracer(t *telemetry.NATSTracer) {
	p.natsTracer = t
}

// Publish creates an envelope, serializes it, and publishes to the correct stream subject.
func (p *Publisher) Publish(ctx context.Context, eventType, aggregateID, aggregateType string, version int, payload interface{}) error {
	env, err := NewEnvelope(eventType, aggregateID, aggregateType, version, payload)
	if err != nil {
		return fmt.Errorf("failed to create envelope: %w", err)
	}
	return p.publishEnvelope(ctx, env)
}

// PublishWithCorrelation publishes with explicit correlation and causation IDs.
func (p *Publisher) PublishWithCorrelation(ctx context.Context, eventType, aggregateID, aggregateType string, version int, payload interface{}, correlationID, causationID string) error {
	env, err := NewEnvelopeWithCorrelation(eventType, aggregateID, aggregateType, version, payload, correlationID, causationID)
	if err != nil {
		return fmt.Errorf("failed to create envelope: %w", err)
	}
	return p.publishEnvelope(ctx, env)
}

// PublishChild publishes a causal child event of a parent envelope.
func (p *Publisher) PublishChild(ctx context.Context, parent *Envelope, eventType, aggregateID, aggregateType string, version int, payload interface{}) error {
	env, err := NewChildEnvelope(parent, eventType, aggregateID, aggregateType, version, payload)
	if err != nil {
		return fmt.Errorf("failed to create child envelope: %w", err)
	}
	return p.publishEnvelope(ctx, env)
}

// PublishRaw publishes a pre-built envelope.
func (p *Publisher) PublishRaw(ctx context.Context, env *Envelope) error {
	return p.publishEnvelope(ctx, env)
}

func (p *Publisher) publishEnvelope(ctx context.Context, env *Envelope) error {
	subject := env.Subject()
	data, err := env.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize envelope: %w", err)
	}

	// Trace the NATS publish with an OTel span. The NATSTracer auto-propagates
	// correlation_id and causation_id from the current trace context as span attributes.
	var pubErr error
	if p.natsTracer != nil {
		pubErr = p.natsTracer.TracePublish(ctx, p.natsMetrics, subject, func(ctx context.Context) error {
			_, innerErr := p.js.Publish(ctx, subject, data)
			return innerErr
		})
	} else {
		_, pubErr = p.js.Publish(ctx, subject, data)
		// Record NATS publish metrics (duration + errors) when no tracer.
		if p.natsMetrics != nil {
			p.natsMetrics.RecordPublish(subject, 0, pubErr)
		}
	}

	if pubErr != nil {
		p.log.Errorw("failed to publish event",
			"event_id", env.EventID,
			"event_type", env.EventType,
			"aggregate_id", env.AggregateID,
			"aggregate_type", env.AggregateType,
			"subject", subject,
			"error", pubErr,
		)
		return fmt.Errorf("failed to publish event %s to %s: %w", env.EventType, subject, pubErr)
	}

	p.log.Debugw("event published",
		"event_id", env.EventID,
		"event_type", env.EventType,
		"aggregate_id", env.AggregateID,
		"aggregate_type", env.AggregateType,
		"subject", subject,
	)

	return nil
}
