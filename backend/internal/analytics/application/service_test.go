package application

import (
	"context"
	"testing"
	"time"

	domain "github.com/ecommerce/backend/internal/analytics/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

// mockAnalyticsRepository implements domain.AnalyticsRepository for testing.
type mockAnalyticsRepository struct {
	events          map[string]*domain.AnalyticsEvent
	countByType     map[domain.EventType]int64
	uniqueCustomers map[domain.EventType]int64
	revenue         int64
	topProducts     []*domain.ProductViewStat
	conversion      struct {
		checkouts int64
		orders    int64
		rate      float64
	}
	dailyCounts map[domain.EventType][]*domain.DailyCount
	insertErr   error
	existsErr   error
}

func newMockRepo() *mockAnalyticsRepository {
	return &mockAnalyticsRepository{
		events:          make(map[string]*domain.AnalyticsEvent),
		countByType:     make(map[domain.EventType]int64),
		uniqueCustomers: make(map[domain.EventType]int64),
		dailyCounts:     make(map[domain.EventType][]*domain.DailyCount),
	}
}

func (m *mockAnalyticsRepository) Insert(_ context.Context, event *domain.AnalyticsEvent) error {
	if m.insertErr != nil {
		return m.insertErr
	}
	m.events[event.EventID] = event
	return nil
}

func (m *mockAnalyticsRepository) ExistsByEventID(_ context.Context, eventID string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	_, ok := m.events[eventID]
	return ok, nil
}

func (m *mockAnalyticsRepository) CountEvents(_ context.Context, eventType domain.EventType, _ time.Time, _ time.Time) (int64, error) {
	return m.countByType[eventType], nil
}

func (m *mockAnalyticsRepository) CountUniqueCustomers(_ context.Context, eventType domain.EventType, _ time.Time, _ time.Time) (int64, error) {
	return m.uniqueCustomers[eventType], nil
}

func (m *mockAnalyticsRepository) SumRevenue(_ context.Context, _ time.Time, _ time.Time) (int64, error) {
	return m.revenue, nil
}

func (m *mockAnalyticsRepository) TopProducts(_ context.Context, _ time.Time, _ time.Time, _ int) ([]*domain.ProductViewStat, error) {
	return m.topProducts, nil
}

func (m *mockAnalyticsRepository) ConversionRate(_ context.Context, _ time.Time, _ time.Time) (int64, int64, float64, error) {
	return m.conversion.checkouts, m.conversion.orders, m.conversion.rate, nil
}

func (m *mockAnalyticsRepository) DailyCounts(_ context.Context, eventType domain.EventType, _ time.Time, _ time.Time) ([]*domain.DailyCount, error) {
	return m.dailyCounts[eventType], nil
}

func TestService_GetOverview_Success(t *testing.T) {
	repo := newMockRepo()
	repo.countByType[domain.EventProductViewed] = 1000
	repo.uniqueCustomers[domain.EventProductViewed] = 350
	repo.countByType[domain.EventCartCreated] = 200
	repo.conversion.checkouts = 150
	repo.conversion.orders = 80
	repo.conversion.rate = 53.33
	repo.revenue = 5000000

	svc := NewService(repo)
	tr := domain.TimeRange{
		Start: time.Now().UTC().Add(-24 * time.Hour),
		End:   time.Now().UTC(),
	}

	result, err := svc.GetOverview(context.Background(), tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProductViews != 1000 {
		t.Errorf("expected 1000 product views, got %d", result.ProductViews)
	}
	if result.UniqueViewers != 350 {
		t.Errorf("expected 350 unique viewers, got %d", result.UniqueViewers)
	}
	if result.CartsCreated != 200 {
		t.Errorf("expected 200 carts, got %d", result.CartsCreated)
	}
	if result.CheckoutsStarted != 150 {
		t.Errorf("expected 150 checkouts, got %d", result.CheckoutsStarted)
	}
	if result.OrdersCompleted != 80 {
		t.Errorf("expected 80 orders, got %d", result.OrdersCompleted)
	}
	if result.RevenueCents != 5000000 {
		t.Errorf("expected 5000000 revenue, got %d", result.RevenueCents)
	}
}

func TestService_GetOverview_InvalidDateRange(t *testing.T) {
	svc := NewService(newMockRepo())
	tr := domain.TimeRange{
		Start: time.Now().UTC(),
		End:   time.Now().UTC().Add(-24 * time.Hour),
	}

	_, err := svc.GetOverview(context.Background(), tr)
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Errorf("expected 422 validation error, got %v", err)
	}
}

func TestService_GetTopProducts_Success(t *testing.T) {
	repo := newMockRepo()
	repo.topProducts = []*domain.ProductViewStat{
		{ProductID: "prod-1", Views: 500},
		{ProductID: "prod-2", Views: 300},
	}

	svc := NewService(repo)
	tr := domain.TimeRange{
		Start: time.Now().UTC().Add(-24 * time.Hour),
		End:   time.Now().UTC(),
	}

	stats, err := svc.GetTopProducts(context.Background(), tr, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 products, got %d", len(stats))
	}
	if stats[0].ProductID != "prod-1" {
		t.Errorf("expected prod-1, got %s", stats[0].ProductID)
	}
}

func TestService_GetTopProducts_DefaultLimit(t *testing.T) {
	svc := NewService(newMockRepo())
	tr := domain.TimeRange{
		Start: time.Now().UTC().Add(-24 * time.Hour),
		End:   time.Now().UTC(),
	}

	// Pass 0 — should default to 10
	_, err := svc.GetTopProducts(context.Background(), tr, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_GetDailyTrend_Success(t *testing.T) {
	repo := newMockRepo()
	repo.dailyCounts[domain.EventProductViewed] = []*domain.DailyCount{
		{Date: "2025-01-01", Count: 100},
		{Date: "2025-01-02", Count: 150},
	}

	svc := NewService(repo)
	tr := domain.TimeRange{
		Start: time.Now().UTC().Add(-7 * 24 * time.Hour),
		End:   time.Now().UTC(),
	}

	counts, err := svc.GetDailyTrend(context.Background(), domain.EventProductViewed, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("expected 2 daily counts, got %d", len(counts))
	}
}

func TestService_GetDailyTrend_InvalidEventType(t *testing.T) {
	svc := NewService(newMockRepo())
	tr := domain.TimeRange{
		Start: time.Now().UTC().Add(-24 * time.Hour),
		End:   time.Now().UTC(),
	}

	_, err := svc.GetDailyTrend(context.Background(), domain.EventType("unknown.event"), tr)
	if err == nil {
		t.Fatal("expected error for invalid event type")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok || appErr.Status != 422 {
		t.Errorf("expected 422 validation error, got %v", err)
	}
}

func TestDefaultTimeRange(t *testing.T) {
	tr := DefaultTimeRange()
	expectedEnd := time.Now().UTC()
	expectedStart := expectedEnd.AddDate(0, 0, -30)

	diffStart := expectedStart.Sub(tr.Start)
	if diffStart > time.Minute {
		t.Errorf("start time off by %v", diffStart)
	}
	diffEnd := expectedEnd.Sub(tr.End)
	if diffEnd > time.Minute {
		t.Errorf("end time off by %v", diffEnd)
	}
}
