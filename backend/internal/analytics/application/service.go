package application

import (
	"context"
	"time"

	"github.com/ecommerce/backend/internal/analytics/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

// DefaultTimeRange returns a time range covering the last 30 days.
func DefaultTimeRange() domain.TimeRange {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -30)
	return domain.TimeRange{Start: start, End: end}
}

// Service provides analytics read operations (aggregations, summaries).
// Write operations are handled exclusively by the worker via JetStream.
type Service struct {
	repo domain.AnalyticsRepository
}

// NewService creates a new analytics service.
func NewService(repo domain.AnalyticsRepository) *Service {
	return &Service{repo: repo}
}

// GetOverview returns a high-level summary of analytics for a time range.
// This includes product views, cart creations, checkout starts, orders, revenue, and conversion rate.
func (s *Service) GetOverview(ctx context.Context, tr domain.TimeRange) (*OverviewResult, error) {
	if err := tr.Validate(); err != nil {
		return nil, errors.New("VALIDATION_ERROR", err.Error(), 422)
	}

	var result OverviewResult
	result.TimeRange = tr

	// Product views
	views, err := s.repo.CountEvents(ctx, domain.EventProductViewed, tr.Start, tr.End)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to count product views", 500, err)
	}
	result.ProductViews = views

	// Unique viewers
	uniqueViewers, err := s.repo.CountUniqueCustomers(ctx, domain.EventProductViewed, tr.Start, tr.End)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to count unique viewers", 500, err)
	}
	result.UniqueViewers = uniqueViewers

	// Cart creations
	carts, err := s.repo.CountEvents(ctx, domain.EventCartCreated, tr.Start, tr.End)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to count cart creations", 500, err)
	}
	result.CartsCreated = carts

	// Checkout starts
	checkouts, orders, rate, err := s.repo.ConversionRate(ctx, tr.Start, tr.End)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to compute conversion rate", 500, err)
	}
	result.CheckoutsStarted = checkouts
	result.OrdersCompleted = orders
	result.ConversionRate = rate

	// Revenue
	revenue, err := s.repo.SumRevenue(ctx, tr.Start, tr.End)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to sum revenue", 500, err)
	}
	result.RevenueCents = revenue

	return &result, nil
}

// GetTopProducts returns the most-viewed products within a time range.
func (s *Service) GetTopProducts(ctx context.Context, tr domain.TimeRange, limit int) ([]*domain.ProductViewStat, error) {
	if err := tr.Validate(); err != nil {
		return nil, errors.New("VALIDATION_ERROR", err.Error(), 422)
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	stats, err := s.repo.TopProducts(ctx, tr.Start, tr.End, limit)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to get top products", 500, err)
	}
	return stats, nil
}

// GetDailyTrend returns per-day event counts for a specific event type within a time range.
func (s *Service) GetDailyTrend(ctx context.Context, eventType domain.EventType, tr domain.TimeRange) ([]*domain.DailyCount, error) {
	if err := tr.Validate(); err != nil {
		return nil, errors.New("VALIDATION_ERROR", err.Error(), 422)
	}
	if !isValidEventType(eventType) {
		return nil, errors.New("VALIDATION_ERROR", "invalid event type: "+string(eventType), 422)
	}

	counts, err := s.repo.DailyCounts(ctx, eventType, tr.Start, tr.End)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to get daily trend", 500, err)
	}
	return counts, nil
}

// OverviewResult is the response structure for the analytics overview.
type OverviewResult struct {
	TimeRange       domain.TimeRange `json:"-"`
	ProductViews    int64   `json:"product_views"`
	UniqueViewers   int64   `json:"unique_viewers"`
	CartsCreated    int64   `json:"carts_created"`
	CheckoutsStarted int64  `json:"checkouts_started"`
	OrdersCompleted int64   `json:"orders_completed"`
	ConversionRate  float64 `json:"conversion_rate"`
	RevenueCents    int64   `json:"revenue_cents"`
}

// isValidEventType checks whether a string is a recognized analytics EventType.
func isValidEventType(eventType domain.EventType) bool {
	switch eventType {
	case domain.EventProductViewed,
		domain.EventCartCreated,
		domain.EventCheckoutStarted,
		domain.EventOrderCompleted,
		domain.EventOrderCreated,
		domain.EventPaymentCompleted:
		return true
	}
	return false
}
