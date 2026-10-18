package infrastructure

import (
        "context"
        "fmt"
        "sync"

        "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type webhookEventStore struct {
        coll *mongo.Collection
}

// NewWebhookEventStore creates a MongoDB-backed WebhookEventStore.
func NewWebhookEventStore(db *mongo.Database) domain.WebhookEventStore {
        return &webhookEventStore{coll: db.Collection("webhook_events")}
}

func (s *webhookEventStore) HasBeenProcessed(ctx context.Context, webhookID string) (bool, error) {
        filter := bson.M{"webhook_id": webhookID}
        count, err := s.coll.CountDocuments(ctx, filter)
        if err != nil {
                return false, errors.Wrap("INTERNAL_ERROR", "failed to check webhook idempotency", 500, err)
        }
        return count > 0, nil
}

func (s *webhookEventStore) MarkProcessed(ctx context.Context, pw *domain.ProcessedWebhook) error {
        _, err := s.coll.InsertOne(ctx, pw)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        // Already processed — not an error.
                        return nil
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to persist webhook event", 500, err)
        }
        return nil
}

// EnsureWebhookIndexes creates indexes for the webhook_events collection.
func EnsureWebhookIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("webhook_events")
        indexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "webhook_id", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_webhook_id_unique"),
                },
                {
                        Keys:    bson.D{{Key: "provider", Value: 1}, {Key: "event_type", Value: 1}, {Key: "processed_at", Value: -1}},
                        Options: options.Index().SetName("idx_webhook_provider_type_time"),
                },
                {
                        // Auto-expire processed webhooks after 7 days.
                        Keys:    bson.D{{Key: "processed_at", Value: 1}},
                                        Options: options.Index().SetExpireAfterSeconds(int32(7 * 24 * 60 * 60)).SetName("idx_webhook_ttl"),
                },
        }
        if _, err := coll.Indexes().CreateMany(ctx, indexes); err != nil {
                return fmt.Errorf("failed to create webhook event indexes: %w", err)
        }
        return nil
}

// --- In-memory store for testing ---

// InMemoryWebhookEventStore is a thread-safe in-memory WebhookEventStore.
type InMemoryWebhookEventStore struct {
        mu       sync.RWMutex
        processed map[string]bool // webhook_id -> true
}

// NewInMemoryWebhookEventStore creates an in-memory webhook event store.
func NewInMemoryWebhookEventStore() *InMemoryWebhookEventStore {
        return &InMemoryWebhookEventStore{
                processed: make(map[string]bool),
        }
}

func (s *InMemoryWebhookEventStore) HasBeenProcessed(_ context.Context, webhookID string) (bool, error) {
        s.mu.RLock()
        defer s.mu.RUnlock()
        return s.processed[webhookID], nil
}

func (s *InMemoryWebhookEventStore) MarkProcessed(_ context.Context, pw *domain.ProcessedWebhook) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        s.processed[pw.WebhookID] = true
        return nil
}