package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/backend/internal/notifications/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type notificationRepository struct {
	coll *mongo.Collection
}

// NewNotificationRepository creates a MongoDB-backed NotificationRepository.
func NewNotificationRepository(db *mongo.Database) domain.NotificationRepository {
	return &notificationRepository{coll: db.Collection("notifications")}
}

func (r *notificationRepository) Insert(ctx context.Context, record *domain.NotificationRecord) error {
	_, err := r.coll.InsertOne(ctx, record)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil // already exists — idempotent
		}
		return fmt.Errorf("failed to insert notification record: %w", err)
	}
	return nil
}

func (r *notificationRepository) FindByEventIDAndChannel(ctx context.Context, eventID string, channel domain.Channel) (*domain.NotificationRecord, error) {
	filter := bson.M{"event_id": eventID, "channel": channel}
	var record domain.NotificationRecord
	err := r.coll.FindOne(ctx, filter).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // not found = first attempt
		}
		return nil, fmt.Errorf("failed to find notification record: %w", err)
	}
	return &record, nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, record *domain.NotificationRecord) error {
	filter := bson.M{"_id": record.ID}
	update := bson.M{"$set": bson.M{
		"status":          record.Status,
		"provider_ref":    record.ProviderRef,
		"provider_error":  record.ProviderError,
		"delivery_count":  record.DeliveryCount,
		"first_attempt_at": record.FirstAttemptAt,
		"last_attempt_at":  record.LastAttemptAt,
		"sent_at":         record.SentAt,
		"updated_at":      record.UpdatedAt,
	}}
	_, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update notification record: %w", err)
	}
	return nil
}

// EnsureIndexes creates required indexes on the notifications collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("notifications")
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "channel", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetName("idx_notification_event_channel_unique"),
		},
		{
			Keys:    bson.D{{Key: "correlation_id", Value: 1}},
			Options: options.Index().SetName("idx_notification_correlation_id"),
		},
		{
			Keys:    bson.D{{Key: "aggregate_id", Value: 1}},
			Options: options.Index().SetName("idx_notification_aggregate_id"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_notification_status"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().
				SetExpireAfterSeconds(int32((24 * time.Hour).Seconds())).
				SetName("idx_notification_created_at_ttl"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create notification indexes: %w", err)
	}
	return nil
}
