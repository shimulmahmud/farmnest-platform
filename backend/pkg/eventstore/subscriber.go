package eventstore

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ecommerce/backend/pkg/logger"
	"github.com/ecommerce/backend/pkg/telemetry"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// EventHandler processes a single domain event.
// Return nil to ACK the message, or an error to NAK it (triggering retry/backoff).
type EventHandler func(ctx context.Context, env *Envelope) error

// SubscriberOptions configures a consumer subscription.
type SubscriberOptions struct {
	Stream        string
	ConsumerName  string
	FilterSubject string // optional: only receive specific subject
	Description   string
	MaxDeliveries int
	Backoff       []time.Duration
}

// Subscriber manages a durable JetStream consumer subscription.
// It automatically ACKs on success, NAKs on error (triggering retries),
// and provides structured logging and error handling.
type Subscriber struct {
	js          jetstream.JetStream
	log         *logger.Logger
	handler     EventHandler
	natsMetrics *telemetry.NATSMetrics
	stream      string
	consumer    string

	// Metrics (atomic counters for thread safety).
	processed    atomic.Int64
	errors       atomic.Int64
	handledOnce atomic.Bool
}

// NewSubscriber creates a new subscriber.
func NewSubscriber(js jetstream.JetStream, log *logger.Logger, handler EventHandler) *Subscriber {
	return &Subscriber{js: js, log: log, handler: handler}
}

// SetNATSMetrics injects NATS consumer metrics.
func (s *Subscriber) SetNATSMetrics(m *telemetry.NATSMetrics) {
	s.natsMetrics = m
}

// Subscribe creates a durable consumer and starts consuming messages.
// It blocks until ctx is cancelled or an unrecoverable error occurs.
func (s *Subscriber) Subscribe(ctx context.Context, opts SubscriberOptions) error {
	s.stream = opts.Stream
	s.consumer = opts.ConsumerName

	s.log.Infow("starting event subscriber",
		"stream", opts.Stream,
		"consumer", opts.ConsumerName,
		"filter_subject", opts.FilterSubject,
	)

	consCfg := jetstream.ConsumerConfig{
		Name:        opts.ConsumerName,
		Durable:     opts.ConsumerName,
		Description: opts.Description,
		AckPolicy:   jetstream.AckExplicitPolicy,
	}

	if opts.FilterSubject != "" {
		consCfg.FilterSubject = opts.FilterSubject
	}

	if opts.MaxDeliveries > 0 {
		consCfg.MaxDeliver = opts.MaxDeliveries
	} else {
		consCfg.MaxDeliver = 10
	}

	if len(opts.Backoff) > 0 {
		consCfg.BackOff = opts.Backoff
	} else {
		consCfg.BackOff = DefaultBackoff()
	}

	consumer, err := s.js.CreateOrUpdateConsumer(ctx, opts.Stream, consCfg)
	if err != nil {
		return fmt.Errorf("failed to create consumer %q on stream %q: %w", opts.ConsumerName, opts.Stream, err)
	}

	consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		s.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to start consumer %q: %w", opts.ConsumerName, err)
	}

	// Block until context is cancelled.
	<-ctx.Done()

	// Stop consuming.
	consumeCtx.Stop()

	// Wait for processing to complete.
	<-consumeCtx.Closed()

	s.log.Infow("event subscriber stopped",
		"consumer", opts.ConsumerName,
		"stream", opts.Stream,
	)

	return nil
}

// handleMessage processes a single message with full error handling and OTel tracing.
// Each message gets its own span as a child of the stream consumer.
func (s *Subscriber) handleMessage(parentCtx context.Context, msg jetstream.Msg) {
	// Deserialize the envelope.
	data := msg.Data()
	env, err := DeserializeEnvelope(data)
	if err != nil {
		s.log.Errorw("failed to deserialize event envelope",
			"error", err,
			"subject", msg.Subject(),
		)
		// Cannot process — NAK to move on (no point retrying a bad message format).
		_ = msg.Nak()
		return
	}

	// Get metadata for delivery count and timestamp for lag calculation.
	meta, metaErr := msg.Metadata()
	delivery := uint64(0)
	var msgTimestamp time.Time
	if metaErr == nil && meta != nil {
		delivery = meta.NumDelivered
		msgTimestamp = meta.Timestamp
	}

	// Record consumer lag (time since message was stored in the stream).
	if s.natsMetrics != nil && !msgTimestamp.IsZero() {
		lag := time.Since(msgTimestamp).Seconds()
		s.natsMetrics.RecordConsumerLag(s.stream, s.consumer, lag)
	}

	// Record redelivery when delivery count > 1.
	if s.natsMetrics != nil && delivery > 1 {
		s.natsMetrics.RecordRedelivery(s.stream, s.consumer)
	}

	// Create an OTel span for this message processing.
	// The span is a root span in the worker process (no incoming HTTP trace parent),
	// but carries correlation_id and causation_id from the envelope for distributed tracing.
	tracer := otel.Tracer("github.com/ecommerce/backend/pkg/eventstore")
	spanName := "consumer." + env.EventType
	spanAttrs := []attribute.KeyValue{
		attribute.String("event.type", env.EventType),
		attribute.String("aggregate.type", env.AggregateType),
		attribute.String("aggregate.id", env.AggregateID),
		attribute.String("nats.stream", s.stream),
		attribute.String("nats.consumer", s.consumer),
		attribute.String("nats.subject", msg.Subject()),
		attribute.Int64("nats.delivery", int64(delivery)),
	}
	// Propagate correlation and causation IDs from the envelope.
	if env.CorrelationID != "" {
		spanAttrs = append(spanAttrs, attribute.String("correlation_id", env.CorrelationID))
	}
	if env.CausationID != "" {
		spanAttrs = append(spanAttrs, attribute.String("causation_id", env.CausationID))
	}

	ctx, span := tracer.Start(parentCtx, spanName, trace.WithAttributes(spanAttrs...))
	defer span.End()

	start := time.Now()

	// Create a per-message context with timeout.
	msgCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Invoke the handler.
	err = s.handler(msgCtx, env)

	duration := time.Since(start)

	if err != nil {
		s.errors.Add(1)
		telemetry.RecordSpanError(span, err)

		s.log.Errorw("event handler failed",
			"event_id", env.EventID,
			"event_type", env.EventType,
			"aggregate_id", env.AggregateID,
			"aggregate_type", env.AggregateType,
			"correlation_id", env.CorrelationID,
			"delivery", delivery,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		// NAK the message to trigger redelivery with backoff.
		_ = msg.Nak()
		return
	}

	// Success — acknowledge the message.
	if err := msg.Ack(); err != nil {
		s.log.Warnw("failed to ack message",
			"event_id", env.EventID,
			"event_type", env.EventType,
			"error", err,
		)
		return
	}

	s.processed.Add(1)

	s.log.Debugw("event processed",
		"event_id", env.EventID,
		"event_type", env.EventType,
		"aggregate_id", env.AggregateID,
		"aggregate_type", env.AggregateType,
		"correlation_id", env.CorrelationID,
		"duration_ms", duration.Milliseconds(),
		"delivery", delivery,
	)
}

// Stats returns current subscriber metrics.
func (s *Subscriber) Stats() (processed, errors int64) {
	return s.processed.Load(), s.errors.Load()
}

// DefaultBackoff returns exponential backoff durations for 10 retries.
func DefaultBackoff() []time.Duration {
	return []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		32 * time.Second,
		64 * time.Second,
		128 * time.Second,
		256 * time.Second,
		5 * time.Minute,
	}
}