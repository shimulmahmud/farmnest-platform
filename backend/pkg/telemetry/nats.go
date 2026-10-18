package telemetry

import (
        "context"
        "time"

        "go.opentelemetry.io/otel"
        "go.opentelemetry.io/otel/attribute"
        "go.opentelemetry.io/otel/metric"
        "go.opentelemetry.io/otel/trace"
)

// NATSTracer wraps a tracer name for creating NATS publish spans.
type NATSTracer struct {
        tracerName string
}

// NewNATSTracer creates a NATS tracer with the given instrumented name.
func NewNATSTracer(tracerName string) *NATSTracer {
        return &NATSTracer{tracerName: tracerName}
}

// TracePublish wraps a NATS publish operation with a span and records metrics.
// Subject is safe to emit (no secrets in NATS subjects).
// The span links to the parent trace via context, and the correlation_id
// and causation_id are set as span attributes for event correlation.
func (t *NATSTracer) TracePublish(
        ctx context.Context,
        metrics *NATSMetrics,
        subject string,
        fn func(ctx context.Context) error,
) error {
        startTime := time.Now()
        var pubErr error

        tracer := otel.Tracer(t.tracerName)
        attrs := []attribute.KeyValue{
                attribute.String("nats.subject", subject),
        }
        // Propagate correlation and causation IDs from the current trace context.
        if traceID := CorrelationIDFromContext(ctx); traceID != "" {
                attrs = append(attrs, attribute.String("correlation_id", traceID))
        }
        if spanID := CausationIDFromContext(ctx); spanID != "" {
                attrs = append(attrs, attribute.String("causation_id", spanID))
        }

        ctx, span := tracer.Start(ctx, "nats.publish", trace.WithAttributes(attrs...))

        defer func() {
                if pubErr != nil {
                        RecordSpanError(span, pubErr)
                }
                span.End()
                duration := time.Since(startTime).Seconds()
                if metrics != nil {
                        metrics.RecordPublish(subject, duration, pubErr)
                }
        }()

        pubErr = fn(ctx)
        return pubErr
}

// RecordConsumerLag records the processing lag for a consumed message.
func (m *NATSMetrics) RecordConsumerLag(stream, consumer string, lagSeconds float64) {
        attrs := metric.WithAttributes(
                attribute.String("nats.stream", stream),
                attribute.String("nats.consumer", consumer),
        )
        m.ConsumerLag.Record(context.Background(), lagSeconds, attrs)
}

// RecordRedelivery increments the redelivery counter for a message.
func (m *NATSMetrics) RecordRedelivery(stream, consumer string) {
        attrs := metric.WithAttributes(
                attribute.String("nats.stream", stream),
                attribute.String("nats.consumer", consumer),
        )
        m.Redeliveries.Add(context.Background(), 1, attrs)
}