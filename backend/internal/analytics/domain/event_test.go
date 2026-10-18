package domain

import (
	"testing"
	"time"
)

func TestTimeRange_Validate_Valid(t *testing.T) {
	tr := TimeRange{
		Start: time.Now().UTC().Add(-24 * time.Hour),
		End:   time.Now().UTC(),
	}
	if err := tr.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestTimeRange_Validate_Invalid(t *testing.T) {
	tr := TimeRange{
		Start: time.Now().UTC(),
		End:   time.Now().UTC().Add(-24 * time.Hour),
	}
	if err := tr.Validate(); err == nil {
		t.Error("expected error for inverted range, got nil")
	}
	if err != ErrInvalidDateRange {
		t.Errorf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestTimeRange_Validate_Equal(t *testing.T) {
	now := time.Now().UTC()
	tr := TimeRange{Start: now, End: now}
	if err := tr.Validate(); err != nil {
		t.Errorf("expected no error for equal times, got %v", err)
	}
}

func TestNewAnalyticsEvent(t *testing.T) {
	now := time.Now().UTC()
	payload := map[string]any{"product_id": "prod-123"}

	evt := NewAnalyticsEvent(
		"event-1",
		EventProductViewed,
		"customer-1",
		"prod-123",
		"product",
		"corr-1",
		now,
		payload,
	)

	if evt.ID == "" {
		t.Error("expected non-empty ID")
	}
	if evt.EventID != "event-1" {
		t.Errorf("expected EventID event-1, got %s", evt.EventID)
	}
	if evt.EventType != EventProductViewed {
		t.Errorf("expected EventType %s, got %s", EventProductViewed, evt.EventType)
	}
	if evt.CustomerID != "customer-1" {
		t.Errorf("expected CustomerID customer-1, got %s", evt.CustomerID)
	}
	if evt.AggregateID != "prod-123" {
		t.Errorf("expected AggregateID prod-123, got %s", evt.AggregateID)
	}
	if evt.AggregateType != "product" {
		t.Errorf("expected AggregateType product, got %s", evt.AggregateType)
	}
	if !evt.ProcessedAt.After(now.Add(-time.Second)) {
		t.Error("expected ProcessedAt to be recent")
	}
	if evt.OccurredAt != now {
		t.Errorf("expected OccurredAt %v, got %v", now, evt.OccurredAt)
	}
}

func TestAnalyticsError(t *testing.T) {
	err := ErrInvalidEventType
	if err.Error() != "INVALID_EVENT_TYPE: unrecognized analytics event type" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestEventType_Constants(t *testing.T) {
	types := []EventType{
		EventProductViewed,
		EventCartCreated,
		EventCheckoutStarted,
		EventOrderCompleted,
		EventPaymentCompleted,
	}
	for _, et := range types {
		if string(et) == "" {
			t.Error("event type should not be empty")
		}
	}
}
