package telemetry

import (
        "context"
        "time"

        "go.mongodb.org/mongo-driver/v2/mongo"
)

// InstrumentedCollection wraps a *mongo.Collection with OpenTelemetry tracing and metrics.
// It delegates all common operations to the underlying collection while recording
// spans (operation name + collection) and metrics (duration + errors).
//
// Usage: replace db.Collection("orders") with NewInstrumentedCollection(db.Collection("orders"), tracer, metrics).
// No secrets are exposed: collection and operation names are safe attributes.
type InstrumentedCollection struct {
        coll    *mongo.Collection
        tracer  *MongoDBTracer
        metrics *MongoDBMetrics
}

// NewInstrumentedCollection wraps a MongoDB collection with tracing and metrics.
// If tracer or metrics is nil, the corresponding instrumentation is skipped.
func NewInstrumentedCollection(coll *mongo.Collection, tracer *MongoDBTracer, metrics *MongoDBMetrics) *InstrumentedCollection {
        return &InstrumentedCollection{coll: coll, tracer: tracer, metrics: metrics}
}

// Collection returns the underlying *mongo.Collection for advanced operations
// (e.g. Indexes, Watch, Aggregate) that are not wrapped.
func (ic *InstrumentedCollection) Collection() *mongo.Collection {
        return ic.coll
}

// InsertOne wraps mongo.Collection.InsertOne with tracing and metrics.
func (ic *InstrumentedCollection) InsertOne(ctx context.Context, doc interface{}, opts ...any) (*mongo.InsertOneResult, error) {
        var result *mongo.InsertOneResult
        err := ic.trace(ctx, "insertOne", func(ctx context.Context) error {
                var innerErr error
                result, innerErr = ic.coll.InsertOne(ctx, doc)
                return innerErr
        })
        return result, err
}

// FindOne wraps mongo.Collection.FindOne with tracing and metrics.
// The FindOne call itself sends the query to the server, so tracing is applied here.
func (ic *InstrumentedCollection) FindOne(ctx context.Context, filter interface{}) *mongo.SingleResult {
        var sr *mongo.SingleResult
        _ = ic.trace(ctx, "findOne", func(ctx context.Context) error {
                sr = ic.coll.FindOne(ctx, filter)
                return nil
        })
        return sr
}

// FindOneAndUpdate wraps mongo.Collection.FindOneAndUpdate with tracing and metrics.
func (ic *InstrumentedCollection) FindOneAndUpdate(ctx context.Context, filter, update interface{}) *mongo.SingleResult {
        var sr *mongo.SingleResult
        _ = ic.trace(ctx, "findOneAndUpdate", func(ctx context.Context) error {
                sr = ic.coll.FindOneAndUpdate(ctx, filter, update)
                return nil
        })
        return sr
}

// UpdateOne wraps mongo.Collection.UpdateOne with tracing and metrics.
func (ic *InstrumentedCollection) UpdateOne(ctx context.Context, filter, update interface{}, opts ...any) (*mongo.UpdateResult, error) {
        var result *mongo.UpdateResult
        err := ic.trace(ctx, "updateOne", func(ctx context.Context) error {
                var innerErr error
                result, innerErr = ic.coll.UpdateOne(ctx, filter, update)
                return innerErr
        })
        return result, err
}

// UpdateMany wraps mongo.Collection.UpdateMany with tracing and metrics.
func (ic *InstrumentedCollection) UpdateMany(ctx context.Context, filter, update interface{}, opts ...any) (*mongo.UpdateResult, error) {
        var result *mongo.UpdateResult
        err := ic.trace(ctx, "updateMany", func(ctx context.Context) error {
                var innerErr error
                result, innerErr = ic.coll.UpdateMany(ctx, filter, update)
                return innerErr
        })
        return result, err
}

// DeleteOne wraps mongo.Collection.DeleteOne with tracing and metrics.
func (ic *InstrumentedCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...any) (*mongo.DeleteResult, error) {
        var result *mongo.DeleteResult
        err := ic.trace(ctx, "deleteOne", func(ctx context.Context) error {
                var innerErr error
                result, innerErr = ic.coll.DeleteOne(ctx, filter)
                return innerErr
        })
        return result, err
}

// DeleteMany wraps mongo.Collection.DeleteMany with tracing and metrics.
func (ic *InstrumentedCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...any) (*mongo.DeleteResult, error) {
        var result *mongo.DeleteResult
        err := ic.trace(ctx, "deleteMany", func(ctx context.Context) error {
                var innerErr error
                result, innerErr = ic.coll.DeleteMany(ctx, filter)
                return innerErr
        })
        return result, err
}

// CountDocuments wraps mongo.Collection.CountDocuments with tracing and metrics.
func (ic *InstrumentedCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...any) (int64, error) {
        var count int64
        err := ic.trace(ctx, "countDocuments", func(ctx context.Context) error {
                var innerErr error
                count, innerErr = ic.coll.CountDocuments(ctx, filter)
                return innerErr
        })
        return count, err
}

// Aggregate wraps mongo.Collection.Aggregate with tracing and metrics.
func (ic *InstrumentedCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...any) (*mongo.Cursor, error) {
        var cursor *mongo.Cursor
        err := ic.trace(ctx, "aggregate", func(ctx context.Context) error {
                var innerErr error
                cursor, innerErr = ic.coll.Aggregate(ctx, pipeline)
                return innerErr
        })
        return cursor, err
}

// Find wraps mongo.Collection.Find with tracing and metrics.
func (ic *InstrumentedCollection) Find(ctx context.Context, filter interface{}, opts ...any) (*mongo.Cursor, error) {
        var cursor *mongo.Cursor
        err := ic.trace(ctx, "find", func(ctx context.Context) error {
                var innerErr error
                cursor, innerErr = ic.coll.Find(ctx, filter)
                return innerErr
        })
        return cursor, err
}

// Name returns the collection name (for tracing attributes).
func (ic *InstrumentedCollection) Name() string {
        return ic.coll.Name()
}

// trace is a helper that wraps an operation with tracing and metrics.
func (ic *InstrumentedCollection) trace(ctx context.Context, operation string, fn func(ctx context.Context) error) error {
        if ic.tracer != nil {
                return ic.tracer.TraceOperation(ctx, ic.metrics, ic.coll.Name(), operation, fn)
        }
        // Metrics-only path (no tracer).
        if ic.metrics != nil {
                return TraceMongoOperationMetricsOnly(ctx, ic.metrics, ic.coll.Name(), operation, fn)
        }
        return fn(ctx)
}

// TraceMongoOperationMetricsOnly records metrics for a MongoDB operation without creating a span.
// Used when only metrics are configured (no tracer).
func TraceMongoOperationMetricsOnly(ctx context.Context, metrics *MongoDBMetrics, collection, operation string, fn func(ctx context.Context) error) error {
        if metrics == nil {
                return fn(ctx)
        }
        return metrics.TraceOperation(ctx, collection, operation, fn)
}

// MongoDBMetrics.TraceOperation records metrics (duration + errors) for a MongoDB operation.
// This is the metrics-only counterpart to MongoDBTracer.TraceOperation.
func (m *MongoDBMetrics) TraceOperation(ctx context.Context, collection, operation string, fn func(ctx context.Context) error) error {
        if m == nil {
                return fn(ctx)
        }
        return TraceOperation(ctx, m, collection, operation, fn)
}

// TraceOperation is a package-level helper that records metrics for a MongoDB operation.
// It measures duration, calls fn, and records errors.
func TraceOperation(ctx context.Context, metrics *MongoDBMetrics, collection, operation string, fn func(ctx context.Context) error) error {
        if metrics == nil {
                return fn(ctx)
        }
        startTime := timeNow()
        err := fn(ctx)
        duration := timeSince(startTime).Seconds()
        metrics.RecordOperation(collection, operation, duration, err)
        return err
}

// timeNow and timeSince are package-level variables for testability.
var timeNow = func() time.Time { return time.Now() }
var timeSince = func(t time.Time) time.Duration { return time.Since(t) }
