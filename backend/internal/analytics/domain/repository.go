package domain

import (
	"context"
	"time"
)

// AnalyticsRepository persists analytics events and provides aggregation queries.
// The interface is storage-agnostic to allow future migration from MongoDB
// to a dedicated analytics engine (e.g. ClickHouse, BigQuery).
type AnalyticsRepository interface {
	// Insert persists a single analytics event. Duplicate event_ids are silently ignored.
	Insert(ctx context.Context, event *AnalyticsEvent) error

	// ExistsByEventID checks idempotency — whether an event has already been stored.
	ExistsByEventID(ctx context.Context, eventID string) (bool, error)

	// CountEvents counts events of a given type within a time range.
	// If eventType is empty, counts all event types.
	CountEvents(ctx context.Context, eventType EventType, start, end time.Time) (int64, error)

	// CountUniqueCustomers counts distinct customers who generated events
	// of a given type within a time range.
	CountUniqueCustomers(ctx context.Context, eventType EventType, start, end time.Time) (int64, error)

	// SumRevenue sums the total_cents from order.completed events within a time range.
	SumRevenue(ctx context.Context, start, end time.Time) (int64, error)

	// TopProducts returns the most-viewed products with their view counts
	// within a time range, limited to the given number of results.
	TopProducts(ctx context.Context, start, end time.Time, limit int) ([]*ProductViewStat, error)

	// ConversionRate computes the funnel conversion rate from checkout.started
	// to order.completed within a time range. Returns (checkouts, orders, rate).
	ConversionRate(ctx context.Context, start, end time.Time) (checkouts int64, orders int64, rate float64, err error)

	// DailyCounts returns per-day event counts for a given event type within
	// a time range, useful for time-series charts.
	DailyCounts(ctx context.Context, eventType EventType, start, end time.Time) ([]*DailyCount, error)
}

// ProductViewStat represents a product with its view count.
type ProductViewStat struct {
	ProductID string `bson:"product_id" json:"product_id"`
	Views     int64  `bson:"views" json:"views"`
}

// DailyCount represents a single day's event count.
type DailyCount struct {
	Date  string `bson:"date" json:"date"`   // "YYYY-MM-DD"
	Count int64  `bson:"count" json:"count"`
}
