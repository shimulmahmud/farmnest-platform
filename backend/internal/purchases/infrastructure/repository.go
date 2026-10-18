package infrastructure

import (
	"context"

	"github.com/ecommerce/backend/internal/purchases/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type repository struct{ coll *mongo.Collection }

// NewRepository creates a MongoDB-backed purchase repository.
func NewRepository(db *mongo.Database) domain.Repository {
	return &repository{coll: db.Collection("purchases")}
}

func (r *repository) Create(ctx context.Context, p *domain.Purchase) error {
	if _, err := r.coll.InsertOne(ctx, p); err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to save purchase", 500, err)
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, id string) (*domain.Purchase, error) {
	var p domain.Purchase
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&p); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find purchase", 500, err)
	}
	return &p, nil
}

func (r *repository) List(ctx context.Context, supplier string, p pagination.Params) ([]*domain.Purchase, int64, error) {
	q := bson.M{}
	if supplier != "" {
		q["supplier_name"] = supplier
	}
	total, err := r.coll.CountDocuments(ctx, q)
	if err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count purchases", 500, err)
	}
	cur, err := r.coll.Find(ctx, q, options.Find().
		SetSort(bson.D{{Key: "received_at", Value: -1}}).SetSkip(int64(p.Skip)).SetLimit(int64(p.Limit)))
	if err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list purchases", 500, err)
	}
	defer cur.Close(ctx)
	var out []*domain.Purchase
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode purchases", 500, err)
	}
	return out, total, nil
}

// EnsureIndexes creates indexes on the purchases collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("purchases").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "received_at", Value: -1}},
		Options: options.Index().SetName("idx_purchase_received"),
	})
	return err
}
