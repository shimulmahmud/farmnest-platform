package infrastructure

import (
	"context"
	"math"

	"github.com/ecommerce/backend/internal/products/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ratingReader struct{ coll *mongo.Collection }

// NewRatingReader creates a domain.RatingReader over the "reviews" collection (APPROVED only).
func NewRatingReader(db *mongo.Database) domain.RatingReader {
	return &ratingReader{coll: db.Collection("reviews")}
}

func (r *ratingReader) RatingsByProduct(ctx context.Context, ids []string) (map[string]domain.Rating, error) {
	out := make(map[string]domain.Rating, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	cur, err := r.coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "product_id", Value: bson.D{{Key: "$in", Value: ids}}},
			{Key: "status", Value: "APPROVED"},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$product_id"},
			{Key: "avg", Value: bson.D{{Key: "$avg", Value: "$rating"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	})
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to read ratings", 500, err)
	}
	defer cur.Close(ctx)

	var rows []struct {
		ID    string  `bson:"_id"`
		Avg   float64 `bson:"avg"`
		Count int     `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode ratings", 500, err)
	}
	for _, x := range rows {
		out[x.ID] = domain.Rating{Average: math.Round(x.Avg*10) / 10, Count: x.Count}
	}
	return out, nil
}
