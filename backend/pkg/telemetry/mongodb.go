package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MongoDBTracer wraps a tracer name for creating MongoDB operation spans.
type MongoDBTracer struct {
	tracerName string
}

// NewMongoDBTracer creates a MongoDB tracer with the given instrumented name.
func NewMongoDBTracer(tracerName string) *MongoDBTracer {
	return &MongoDBTracer{tracerName: tracerName}
}

// TraceOperation wraps a MongoDB operation with a span and records metrics.
// It measures duration, records errors, and creates a child span.
// Collection and operation names are safe to emit (no secrets).
func (t *MongoDBTracer) TraceOperation(
	ctx context.Context,
	metrics *MongoDBMetrics,
	collection, operation string,
	fn func(ctx context.Context) error,
) error {
	startTime := time.Now()
	var opErr error

	tracer := otel.Tracer(t.tracerName)
	spanName := "mongodb." + operation
	ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(
		attribute.String("mongodb.collection", collection),
		attribute.String("mongodb.operation", operation),
	))

	defer func() {
		if opErr != nil {
			RecordSpanError(span, opErr)
		}
		span.End()
		duration := time.Since(startTime).Seconds()
		if metrics != nil {
			metrics.RecordOperation(collection, operation, duration, opErr)
		}
	}()

	opErr = fn(ctx)
	return opErr
}

// TraceFind wraps a MongoDB find operation.
func (t *MongoDBTracer) TraceFind(ctx context.Context, metrics *MongoDBMetrics, collection string, fn func(ctx context.Context) error) error {
	return t.TraceOperation(ctx, metrics, collection, "find", fn)
}

// TraceInsertOne wraps a MongoDB insertOne operation.
func (t *MongoDBTracer) TraceInsertOne(ctx context.Context, metrics *MongoDBMetrics, collection string, fn func(ctx context.Context) error) error {
	return t.TraceOperation(ctx, metrics, collection, "insertOne", fn)
}

// TraceUpdateOne wraps a MongoDB updateOne operation.
func (t *MongoDBTracer) TraceUpdateOne(ctx context.Context, metrics *MongoDBMetrics, collection string, fn func(ctx context.Context) error) error {
	return t.TraceOperation(ctx, metrics, collection, "updateOne", fn)
}

// TraceDeleteOne wraps a MongoDB deleteOne operation.
func (t *MongoDBTracer) TraceDeleteOne(ctx context.Context, metrics *MongoDBMetrics, collection string, fn func(ctx context.Context) error) error {
	return t.TraceOperation(ctx, metrics, collection, "deleteOne", fn)
}

// TraceCountDocuments wraps a MongoDB countDocuments operation.
func (t *MongoDBTracer) TraceCountDocuments(ctx context.Context, metrics *MongoDBMetrics, collection string, fn func(ctx context.Context) error) error {
	return t.TraceOperation(ctx, metrics, collection, "countDocuments", fn)
}
