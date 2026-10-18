package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/backend/internal/vendors/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type repository struct{ coll *mongo.Collection }

// NewRepository creates a MongoDB-backed vendor repository.
func NewRepository(db *mongo.Database) domain.Repository {
	return &repository{coll: db.Collection("vendors")}
}

func (r *repository) Create(ctx context.Context, v *domain.Vendor) error {
	if _, err := r.coll.InsertOne(ctx, v); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Wrap("CONFLICT", "vendor slug or owner already exists", 409, err)
		}
		return errors.Wrap("INTERNAL_ERROR", "failed to create vendor", 500, err)
	}
	return nil
}

func (r *repository) find(ctx context.Context, filter bson.M) (*domain.Vendor, error) {
	var v domain.Vendor
	if err := r.coll.FindOne(ctx, filter).Decode(&v); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find vendor", 500, err)
	}
	return &v, nil
}

func (r *repository) FindByID(ctx context.Context, id string) (*domain.Vendor, error) {
	return r.find(ctx, bson.M{"_id": id})
}

func (r *repository) FindByOwner(ctx context.Context, userID string) (*domain.Vendor, error) {
	return r.find(ctx, bson.M{"owner_user_id": userID})
}

func (r *repository) Update(ctx context.Context, v *domain.Vendor) error {
	v.UpdatedAt = time.Now().UTC()
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": v.ID}, bson.M{"$set": v})
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to update vendor", 500, err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *repository) List(ctx context.Context, status string, p pagination.Params) ([]*domain.Vendor, int64, error) {
	q := bson.M{}
	if status != "" {
		q["status"] = status
	}
	total, err := r.coll.CountDocuments(ctx, q)
	if err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count vendors", 500, err)
	}
	cur, err := r.coll.Find(ctx, q, options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).SetSkip(int64(p.Skip)).SetLimit(int64(p.Limit)))
	if err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list vendors", 500, err)
	}
	defer cur.Close(ctx)
	var out []*domain.Vendor
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode vendors", 500, err)
	}
	return out, total, nil
}

// EnsureIndexes creates required indexes on the vendors collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("vendors").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_vendor_slug_unique")},
		{Keys: bson.D{{Key: "owner_user_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_vendor_owner_unique")},
	})
	if err != nil {
		return fmt.Errorf("failed to create vendor indexes: %w", err)
	}
	return nil
}
