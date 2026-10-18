package infrastructure

import (
        "context"
        "fmt"
        "time"

        "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type paymentRepository struct {
        coll *mongo.Collection
}

// NewPaymentRepository creates a MongoDB-backed PaymentRepository.
func NewPaymentRepository(db *mongo.Database) domain.PaymentRepository {
        return &paymentRepository{coll: db.Collection("payments")}
}

func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
        _, err := r.coll.InsertOne(ctx, payment)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        return domain.ErrDuplicatePayment
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to create payment", 500, err)
        }
        return nil
}

func (r *paymentRepository) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
        var p domain.Payment
        filter := bson.M{"_id": id}
        if err := r.coll.FindOne(ctx, filter).Decode(&p); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find payment", 500, err)
        }
        return &p, nil
}

func (r *paymentRepository) FindByOrderID(ctx context.Context, orderID, provider string) (*domain.Payment, error) {
        var p domain.Payment
        filter := bson.M{"order_id": orderID, "provider": provider}
        if err := r.coll.FindOne(ctx, filter).Decode(&p); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find payment by order", 500, err)
        }
        return &p, nil
}

// UpdateStatus atomically transitions payment status via FindOneAndUpdate.
func (r *paymentRepository) UpdateStatus(ctx context.Context, id string, expectedStatus, newStatus domain.PaymentStatus, providerRef string) (*domain.Payment, error) {
        now := time.Now().UTC()

        filter := bson.M{
                "_id":    id,
                "status": expectedStatus,
        }

        update := bson.M{
                "$set": bson.M{
                        "status":     newStatus,
                        "updated_at": now,
                },
        }

        if providerRef != "" {
                update["$set"].(bson.M)["provider_reference"] = providerRef
        }

        opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

        var updated domain.Payment
        err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.New("CONFLICT", "payment not found or status has changed", 409)
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to update payment status", 500, err)
        }
        return &updated, nil
}

func (r *paymentRepository) FindByProviderRef(ctx context.Context, provider, providerRef string) (*domain.Payment, error) {
        var p domain.Payment
        filter := bson.M{"provider": provider, "provider_reference": providerRef}
        if err := r.coll.FindOne(ctx, filter).Decode(&p); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find payment by provider ref", 500, err)
        }
        return &p, nil
}

// EnsureIndexes creates all required indexes for the payments collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("payments")
        indexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "order_id", Value: 1}, {Key: "provider", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_payment_order_provider_unique"),
                },
                {
                        Keys:    bson.D{{Key: "order_id", Value: 1}},
                        Options: options.Index().SetName("idx_payment_order"),
                },
                {
                        Keys:    bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}},
                        Options: options.Index().SetName("idx_payment_status_created"),
                },
                {
                        Keys:    bson.D{{Key: "provider_reference", Value: 1}},
                        Options: options.Index().SetName("idx_payment_provider_ref"),
                },
        }
        if _, err := coll.Indexes().CreateMany(ctx, indexes); err != nil {
                return fmt.Errorf("failed to create payment indexes: %w", err)
        }
        return nil
}
