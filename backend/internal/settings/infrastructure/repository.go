package infrastructure

import (
	"context"

	"github.com/ecommerce/backend/internal/settings/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	collectionName = "settings"
	deliveryKey    = "delivery"
)

type repository struct{ coll *mongo.Collection }

// NewRepository creates a MongoDB-backed settings repository (one document per setting).
func NewRepository(db *mongo.Database) domain.Repository {
	return &repository{coll: db.Collection(collectionName)}
}

type deliveryDoc struct {
	Key             string `bson:"_id"`
	domain.Delivery `bson:",inline"`
}

func (r *repository) GetDelivery(ctx context.Context) (*domain.Delivery, error) {
	var doc deliveryDoc
	if err := r.coll.FindOne(ctx, bson.M{"_id": deliveryKey}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to load delivery settings", 500, err)
	}
	return &doc.Delivery, nil
}

func (r *repository) SaveDelivery(ctx context.Context, d *domain.Delivery) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": deliveryKey}, deliveryDoc{Key: deliveryKey, Delivery: *d},
		options.Replace().SetUpsert(true))
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to save delivery settings", 500, err)
	}
	return nil
}
