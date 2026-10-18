package infrastructure

import (
	"context"

	"github.com/ecommerce/backend/internal/products/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// stockChecker reads availability straight from the inventory collection
// (written only by the inventory module), summing across warehouses.
type stockChecker struct {
	coll *mongo.Collection
}

// NewStockChecker creates a domain.StockChecker backed by the "inventory" collection.
func NewStockChecker(db *mongo.Database) domain.StockChecker {
	return &stockChecker{coll: db.Collection("inventory")}
}

type availableRow struct {
	SKU       string `bson:"_id"`
	Available int    `bson:"available"`
}

func (c *stockChecker) aggregate(ctx context.Context, match bson.D, having bool) ([]availableRow, error) {
	pipeline := mongo.Pipeline{}
	if len(match) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: match}})
	}
	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: "$sku"},
		{Key: "available", Value: bson.D{{Key: "$sum", Value: bson.D{
			{Key: "$subtract", Value: bson.A{"$total_stock", "$reserved_stock"}},
		}}}},
	}}})
	if having {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
			{Key: "available", Value: bson.D{{Key: "$gt", Value: 0}}},
		}}})
	}

	cursor, err := c.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to read stock availability", 500, err)
	}
	defer cursor.Close(ctx)

	var rows []availableRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode stock availability", 500, err)
	}
	return rows, nil
}

func (c *stockChecker) AvailableBySKU(ctx context.Context, skus []string) (map[string]int, error) {
	out := make(map[string]int, len(skus))
	if len(skus) == 0 {
		return out, nil
	}
	rows, err := c.aggregate(ctx, bson.D{{Key: "sku", Value: bson.D{{Key: "$in", Value: skus}}}}, false)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Available < 0 {
			r.Available = 0
		}
		out[r.SKU] = r.Available
	}
	return out, nil
}

func (c *stockChecker) InStockSKUs(ctx context.Context) ([]string, error) {
	rows, err := c.aggregate(ctx, nil, true)
	if err != nil {
		return nil, err
	}
	skus := make([]string, len(rows))
	for i, r := range rows {
		skus[i] = r.SKU
	}
	return skus, nil
}
