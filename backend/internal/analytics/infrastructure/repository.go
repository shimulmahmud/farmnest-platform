package infrastructure

import (
        "context"
        "fmt"
        "time"

        "github.com/ecommerce/backend/internal/analytics/domain"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "analytics_data"

type analyticsRepository struct {
        coll   *mongo.Collection
        orders *mongo.Collection // real orders: order numbers and revenue come from here
}

// NewAnalyticsRepository creates a MongoDB-backed AnalyticsRepository.
func NewAnalyticsRepository(db *mongo.Database) domain.AnalyticsRepository {
        return &analyticsRepository{coll: db.Collection(collectionName), orders: db.Collection("orders")}
}

func (r *analyticsRepository) Insert(ctx context.Context, event *domain.AnalyticsEvent) error {
        _, err := r.coll.InsertOne(ctx, event)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        return nil // idempotent — already stored
                }
                return fmt.Errorf("failed to insert analytics event: %w", err)
        }
        return nil
}

func (r *analyticsRepository) ExistsByEventID(ctx context.Context, eventID string) (bool, error) {
        filter := bson.M{"event_id": eventID}
        count, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return false, fmt.Errorf("failed to check analytics event existence: %w", err)
        }
        return count > 0, nil
}

func (r *analyticsRepository) CountEvents(ctx context.Context, eventType domain.EventType, start, end time.Time) (int64, error) {
	if eventType == domain.EventOrderCompleted || eventType == domain.EventOrderCreated {
		return r.countOrders(ctx, eventType, start, end)
	}
        filter := bson.M{
                "occurred_at": bson.M{"$gte": start, "$lte": end},
        }
        if eventType != "" {
                filter["event_type"] = eventType
        }
        count, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return 0, fmt.Errorf("failed to count analytics events: %w", err)
        }
        return count, nil
}

func (r *analyticsRepository) CountUniqueCustomers(ctx context.Context, eventType domain.EventType, start, end time.Time) (int64, error) {
        filter := bson.M{
                "occurred_at": bson.M{"$gte": start, "$lte": end},
        }
        if eventType != "" {
                filter["event_type"] = eventType
        }
        pipeline := []bson.M{
                {"$match": filter},
                {"$group": bson.M{
                        "_id": "$customer_id",
                }},
                {"$count": "unique_customers"},
        }
        cursor, err := r.coll.Aggregate(ctx, pipeline)
        if err != nil {
                return 0, fmt.Errorf("failed to count unique customers: %w", err)
        }
        defer cursor.Close(ctx)

        var result struct {
                UniqueCustomers int64 `bson:"unique_customers"`
        }
        if cursor.Next(ctx) {
                if err := cursor.Decode(&result); err != nil {
                        return 0, fmt.Errorf("failed to decode unique customer count: %w", err)
                }
        }
        return result.UniqueCustomers, nil
}

func (r *analyticsRepository) SumRevenue(ctx context.Context, start, end time.Time) (int64, error) {
	// Revenue = total of orders delivered in the period (whole BDT taka).
	return r.sumDeliveredRevenue(ctx, start, end)
}

func (r *analyticsRepository) sumRevenueFromEvents(ctx context.Context, start, end time.Time) (int64, error) {
        filter := bson.M{
                "event_type":   domain.EventOrderCompleted,
                "occurred_at": bson.M{"$gte": start, "$lte": end},
        }
        pipeline := []bson.M{
                {"$match": filter},
                {"$group": bson.M{
                        "_id":   nil,
                        "total": bson.M{"$sum": "$payload.total_cents"},
                }},
        }
        cursor, err := r.coll.Aggregate(ctx, pipeline)
        if err != nil {
                return 0, fmt.Errorf("failed to sum revenue: %w", err)
        }
        defer cursor.Close(ctx)

        var result struct {
                Total int64 `bson:"total"`
        }
        if cursor.Next(ctx) {
                if err := cursor.Decode(&result); err != nil {
                        return 0, fmt.Errorf("failed to decode revenue sum: %w", err)
                }
        }
        return result.Total, nil
}

func (r *analyticsRepository) TopProducts(ctx context.Context, start, end time.Time, limit int) ([]*domain.ProductViewStat, error) {
        filter := bson.M{
                "event_type":   domain.EventProductViewed,
                "occurred_at": bson.M{"$gte": start, "$lte": end},
        }
        pipeline := []bson.M{
                {"$match": filter},
                {"$group": bson.M{
                        "_id":   "$payload.product_id",
                        "views": bson.M{"$sum": 1},
                }},
                {"$sort": bson.M{"views": -1}},
                {"$limit": int64(limit)},
                {"$project": bson.M{
                        "_id":       0,
                        "product_id": "$_id",
                        "views":     1,
                }},
        }
        cursor, err := r.coll.Aggregate(ctx, pipeline)
        if err != nil {
                return nil, fmt.Errorf("failed to aggregate top products: %w", err)
        }
        defer cursor.Close(ctx)

        var stats []*domain.ProductViewStat
        if err := cursor.All(ctx, &stats); err != nil {
                return nil, fmt.Errorf("failed to decode top products: %w", err)
        }
        return stats, nil
}

func (r *analyticsRepository) ConversionRate(ctx context.Context, start, end time.Time) (int64, int64, float64, error) {
	// Orders placed vs orders delivered in the period (the shop has no checkout tracking events).
	placed, err := r.countOrders(ctx, domain.EventOrderCreated, start, end)
	if err != nil {
		return 0, 0, 0, err
	}
	delivered, err := r.countOrders(ctx, domain.EventOrderCompleted, start, end)
	if err != nil {
		return 0, 0, 0, err
	}
	var rate float64
	if placed > 0 {
		rate = float64(delivered) / float64(placed) * 100
	}
	return placed, delivered, rate, nil
}

func (r *analyticsRepository) conversionFromEvents(ctx context.Context, start, end time.Time) (int64, int64, float64, error) {
        timeFilter := bson.M{"$gte": start, "$lte": end}

        // Count checkouts
        checkoutCount, err := r.coll.CountDocuments(ctx, bson.M{
                "event_type":   domain.EventCheckoutStarted,
                "occurred_at": timeFilter,
        })
        if err != nil {
                return 0, 0, 0, fmt.Errorf("failed to count checkout started events: %w", err)
        }

        // Count orders
        orderCount, err := r.coll.CountDocuments(ctx, bson.M{
                "event_type":   domain.EventOrderCompleted,
                "occurred_at": timeFilter,
        })
        if err != nil {
                return 0, 0, 0, fmt.Errorf("failed to count order completed events: %w", err)
        }

        var rate float64
        if checkoutCount > 0 {
                rate = float64(orderCount) / float64(checkoutCount) * 100
        }

        return checkoutCount, orderCount, rate, nil
}

func (r *analyticsRepository) DailyCounts(ctx context.Context, eventType domain.EventType, start, end time.Time) ([]*domain.DailyCount, error) {
	if eventType == domain.EventOrderCompleted || eventType == domain.EventOrderCreated {
		return r.dailyOrderCounts(ctx, eventType, start, end)
	}
        filter := bson.M{
                "event_type":   eventType,
                "occurred_at": bson.M{"$gte": start, "$lte": end},
        }

        // Date truncation to YYYY-MM-DD using MongoDB $dateToString.
        // The timezone "UTC" ensures consistent day boundaries.
        pipeline := []bson.M{
                {"$match": filter},
                {"$group": bson.M{
                        "_id": bson.M{
                                "$dateToString": bson.M{
                                        "format":   "%Y-%m-%d",
                                        "date":     "$occurred_at",
                                        "timezone": "UTC",
                                },
                        },
                        "count": bson.M{"$sum": 1},
                }},
                {"$sort": bson.M{"_id": 1}},
                {"$project": bson.M{
                        "_id":   0,
                        "date":  "$_id",
                        "count": 1,
                }},
        }
        cursor, err := r.coll.Aggregate(ctx, pipeline)
        if err != nil {
                return nil, fmt.Errorf("failed to aggregate daily counts: %w", err)
        }
        defer cursor.Close(ctx)

        var counts []*domain.DailyCount
        if err := cursor.All(ctx, &counts); err != nil {
                return nil, fmt.Errorf("failed to decode daily counts: %w", err)
        }
        return counts, nil
}

// EnsureIndexes creates required indexes for the analytics_data collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection(collectionName)

        indexes := []mongo.IndexModel{
                {
                        Keys: bson.D{{Key: "event_id", Value: 1}},
                        Options: options.Index().
                                SetUnique(true).
                                SetName("idx_analytics_data_event_id_unique"),
                },
                {
                        Keys: bson.D{
                                {Key: "event_type", Value: 1},
                                {Key: "occurred_at", Value: -1},
                        },
                        Options: options.Index().
                                SetName("idx_analytics_data_type_occurred"),
                },
                {
                        Keys: bson.D{
                                {Key: "customer_id", Value: 1},
                                {Key: "event_type", Value: 1},
                                {Key: "occurred_at", Value: -1},
                        },
                        Options: options.Index().
                                SetName("idx_analytics_data_customer_type_occurred"),
                },
                {
                        Keys: bson.D{
                                {Key: "payload.product_id", Value: 1},
                                {Key: "event_type", Value: 1},
                                {Key: "occurred_at", Value: -1},
                        },
                        Options: options.Index().
                                SetName("idx_analytics_data_product_type_occurred"),
                },
                {
                        Keys:    bson.D{{Key: "created_at", Value: 1}},
                        Options: options.Index().
                                SetExpireAfterSeconds(int32((30 * 24 * time.Hour).Seconds())).
                                SetName("idx_analytics_data_created_at_ttl"),
                },
        }

        _, err := coll.Indexes().CreateMany(ctx, indexes)
        if err != nil {
                return fmt.Errorf("failed to create analytics data indexes: %w", err)
        }
        return nil
}
// ---------------------------------------------------------------------------
// Order-based figures (the orders collection is the source of truth)
// ---------------------------------------------------------------------------

// dayTZ cuts days at Bangladesh midnight.
const dayTZ = "+06:00"

// deliveredStage adds delivered_at: the time the order reached DELIVERED.
func deliveredStage() bson.D {
	return bson.D{{Key: "$addFields", Value: bson.D{{Key: "delivered_at", Value: bson.D{{Key: "$max", Value: bson.D{
		{Key: "$map", Value: bson.D{
			{Key: "input", Value: bson.D{{Key: "$filter", Value: bson.D{
				{Key: "input", Value: "$status_history"}, {Key: "as", Value: "e"},
				{Key: "cond", Value: bson.D{{Key: "$eq", Value: bson.A{"$$e.status", "DELIVERED"}}}},
			}}}},
			{Key: "as", Value: "h"}, {Key: "in", Value: "$$h.changed_at"},
		}},
	}}}}}}}
}

// orderPipeline selects orders placed (created_at) or delivered (delivered_at) in [start, end].
func orderPipeline(eventType domain.EventType, start, end time.Time) mongo.Pipeline {
	rng := bson.D{{Key: "$gte", Value: start}, {Key: "$lte", Value: end}}
	if eventType == domain.EventOrderCreated {
		return mongo.Pipeline{{{Key: "$match", Value: bson.D{{Key: "created_at", Value: rng}}}}}
	}
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "status", Value: "DELIVERED"}}}},
		deliveredStage(),
		{{Key: "$match", Value: bson.D{{Key: "delivered_at", Value: rng}}}},
	}
}

func (r *analyticsRepository) countOrders(ctx context.Context, eventType domain.EventType, start, end time.Time) (int64, error) {
	pipeline := append(orderPipeline(eventType, start, end), bson.D{{Key: "$count", Value: "n"}})
	cur, err := r.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}
	defer cur.Close(ctx)
	var row struct {
		N int64 `bson:"n"`
	}
	if cur.Next(ctx) {
		if err := cur.Decode(&row); err != nil {
			return 0, fmt.Errorf("failed to decode order count: %w", err)
		}
	}
	return row.N, nil
}

func (r *analyticsRepository) sumDeliveredRevenue(ctx context.Context, start, end time.Time) (int64, error) {
	pipeline := append(orderPipeline(domain.EventOrderCompleted, start, end),
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "total", Value: bson.D{{Key: "$sum", Value: "$pricing.total"}}}}}})
	cur, err := r.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to sum order revenue: %w", err)
	}
	defer cur.Close(ctx)
	var row struct {
		Total int64 `bson:"total"`
	}
	if cur.Next(ctx) {
		if err := cur.Decode(&row); err != nil {
			return 0, fmt.Errorf("failed to decode order revenue: %w", err)
		}
	}
	return row.Total, nil
}

func (r *analyticsRepository) dailyOrderCounts(ctx context.Context, eventType domain.EventType, start, end time.Time) ([]*domain.DailyCount, error) {
	field := "$delivered_at"
	if eventType == domain.EventOrderCreated {
		field = "$created_at"
	}
	pipeline := append(orderPipeline(eventType, start, end),
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{
				{Key: "format", Value: "%Y-%m-%d"}, {Key: "date", Value: field}, {Key: "timezone", Value: dayTZ},
			}}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	)
	cur, err := r.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate daily orders: %w", err)
	}
	defer cur.Close(ctx)
	var rows []struct {
		Date  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("failed to decode daily orders: %w", err)
	}
	out := make([]*domain.DailyCount, len(rows))
	for i, x := range rows {
		out[i] = &domain.DailyCount{Date: x.Date, Count: x.Count}
	}
	return out, nil
}
