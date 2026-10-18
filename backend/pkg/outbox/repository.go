package outbox

import (
        "context"
        "fmt"
        "time"

        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type outboxRepository struct {
        coll *mongo.Collection
}

// NewOutboxRepository creates a MongoDB-backed OutboxRepository.
func NewOutboxRepository(db *mongo.Database) OutboxRepository {
        return &outboxRepository{coll: db.Collection("outbox_events")}
}

func (r *outboxRepository) Insert(ctx context.Context, event *OutboxEvent) error {
        _, err := r.coll.InsertOne(ctx, event)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        // Duplicate event_id — this is fine for idempotency.
                        return nil
                }
                return fmt.Errorf("failed to insert outbox event %s: %w", event.EventID, err)
        }
        return nil
}

// ClaimNext finds the oldest PENDING event, atomically transitions it to
// PUBLISHING, and increments attempts. Returns nil if no events are available.
func (r *outboxRepository) ClaimNext(ctx context.Context) (*OutboxEvent, error) {
        now := time.Now().UTC()

        filter := bson.M{"status": StatusPending}
        update := bson.M{
                "$set": bson.M{
                        "status": StatusPublishing,
                },
                "$inc": bson.M{"attempts": 1},
        }
        opts := options.FindOneAndUpdate().
                SetSort(bson.D{{Key: "created_at", Value: 1}}).
                SetReturnDocument(options.After)

        var event OutboxEvent
        err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&event)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, nil
                }
                return nil, fmt.Errorf("failed to claim next outbox event: %w", err)
        }

        _ = now // available for future use (e.g. claimed_at field)
        return &event, nil
}

// ResetToPending moves a PUBLISHING event back to PENDING for retry.
func (r *outboxRepository) ResetToPending(ctx context.Context, eventID string) error {
        filter := bson.M{"event_id": eventID, "status": StatusPublishing}
        update := bson.M{
                "$set": bson.M{
                        "status": StatusPending,
                },
        }
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return fmt.Errorf("failed to reset outbox event %s to pending: %w", eventID, err)
        }
        if result.MatchedCount == 0 {
                return fmt.Errorf("outbox event %s not found in PUBLISHING state", eventID)
        }
        return nil
}

// MarkPublished sets the event to PUBLISHED with the published timestamp.
func (r *outboxRepository) MarkPublished(ctx context.Context, eventID string) error {
        now := time.Now().UTC()
        filter := bson.M{"event_id": eventID, "status": StatusPublishing}
        update := bson.M{
                "$set": bson.M{
                        "status":       StatusPublished,
                        "published_at": now,
                },
        }
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return fmt.Errorf("failed to mark outbox event %s as published: %w", eventID, err)
        }
        if result.MatchedCount == 0 {
                return fmt.Errorf("outbox event %s not found in PUBLISHING state", eventID)
        }
        return nil
}

// MarkFailed sets the event to FAILED with the last error.
func (r *outboxRepository) MarkFailed(ctx context.Context, eventID string, errMsg string) error {
        filter := bson.M{"event_id": eventID, "status": StatusPublishing}
        update := bson.M{
                "$set": bson.M{
                        "status":     StatusFailed,
                        "last_error": errMsg,
                },
        }
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return fmt.Errorf("failed to mark outbox event %s as failed: %w", eventID, err)
        }
        if result.MatchedCount == 0 {
                return fmt.Errorf("outbox event %s not found in PUBLISHING state", eventID)
        }
        return nil
}

// ResetStale returns PUBLISHING events that have been in-flight longer
// than staleThreshold back to PENDING (crash recovery).
func (r *outboxRepository) ResetStale(ctx context.Context, staleThreshold time.Duration) (int64, error) {
        cutoff := time.Now().UTC().Add(-staleThreshold)
        filter := bson.M{
                "status":     StatusPublishing,
                "created_at": bson.M{"$lt": cutoff},
        }
        update := bson.M{
                "$set": bson.M{
                        "status": StatusPending,
                },
        }
        result, err := r.coll.UpdateMany(ctx, filter, update)
        if err != nil {
                return 0, fmt.Errorf("failed to reset stale outbox events: %w", err)
        }
        return result.ModifiedCount, nil
}

// DeletePublished removes successfully published events older than the given duration.
func (r *outboxRepository) DeletePublished(ctx context.Context, olderThan time.Duration) (int64, error) {
        cutoff := time.Now().UTC().Add(-olderThan)
        filter := bson.M{
                "status":       StatusPublished,
                "published_at": bson.M{"$lt": cutoff},
        }
        result, err := r.coll.DeleteMany(ctx, filter)
        if err != nil {
                return 0, fmt.Errorf("failed to delete published outbox events: %w", err)
        }
        return result.DeletedCount, nil
}

// CountByStatus returns the count of events grouped by status.
func (r *outboxRepository) CountByStatus(ctx context.Context) (map[OutboxStatus]int64, error) {
        pipeline := []bson.M{
                {
                        "$group": bson.M{
                                "_id":   "$status",
                                "count": bson.M{"$sum": 1},
                        },
                },
        }
        cursor, err := r.coll.Aggregate(ctx, pipeline)
        if err != nil {
                return nil, fmt.Errorf("failed to count outbox events by status: %w", err)
        }
        defer cursor.Close(ctx)

        counts := map[OutboxStatus]int64{}
        var result struct {
                ID    string `bson:"_id"`
                Count int64  `bson:"count"`
        }
        for cursor.Next(ctx) {
                if err := cursor.Decode(&result); err != nil {
                        return nil, fmt.Errorf("failed to decode outbox count: %w", err)
                }
                counts[OutboxStatus(result.ID)] = result.Count
        }
        return counts, nil
}

// EnsureIndexes creates all required indexes for the outbox_events collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("outbox_events")
        indexes := []mongo.IndexModel{
                {
                        // Unique event_id for idempotency.
                        Keys:    bson.D{{Key: "event_id", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_outbox_event_id_unique"),
                },
                {
                        // Polling index: claim oldest PENDING event.
                        Keys:    bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}},
                        Options: options.Index().SetName("idx_outbox_status_created"),
                },
                {
                        // Crash recovery: find stale PUBLISHING events.
                        Keys:    bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}},
                        Options: options.Index().SetPartialFilterExpression(bson.M{"status": StatusPublishing}).SetName("idx_outbox_publishing_created"),
                },
                {
                        // Cleanup: find old published events.
                        Keys:    bson.D{{Key: "status", Value: 1}, {Key: "published_at", Value: 1}},
                        Options: options.Index().SetPartialFilterExpression(bson.M{"status": StatusPublished}).SetName("idx_outbox_published_at"),
                },
                {
                        // Aggregate lookup: find events by aggregate.
                        Keys:    bson.D{{Key: "aggregate_type", Value: 1}, {Key: "aggregate_id", Value: 1}},
                        Options: options.Index().SetName("idx_outbox_aggregate"),
                },
        }
        if _, err := coll.Indexes().CreateMany(ctx, indexes); err != nil {
                return fmt.Errorf("failed to create outbox indexes: %w", err)
        }
        return nil
}
