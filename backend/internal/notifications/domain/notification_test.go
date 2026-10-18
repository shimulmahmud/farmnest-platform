package domain_test

import (
	"testing"
	"time"

	"github.com/ecommerce/backend/internal/notifications/domain"
)

func TestChannelConstants(t *testing.T) {
	if domain.ChannelEmail != "EMAIL" {
		t.Errorf("ChannelEmail = %q, want EMAIL", domain.ChannelEmail)
	}
	if domain.ChannelSMS != "SMS" {
		t.Errorf("ChannelSMS = %q, want SMS", domain.ChannelSMS)
	}
	if domain.ChannelPush != "PUSH" {
		t.Errorf("ChannelPush = %q, want PUSH", domain.ChannelPush)
	}
}

func TestNotificationStatusConstants(t *testing.T) {
	statuses := []struct{
		name string
		val  domain.NotificationStatus
		want string
	}{
		{"Pending", domain.NotificationPending, "PENDING"},
		{"Sent", domain.NotificationSent, "SENT"},
		{"Failed", domain.NotificationFailed, "FAILED"},
		{"Skipped", domain.NotificationSkipped, "SKIPPED"},
	}
	for _, tc := range statuses {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.val) != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.val, tc.want)
			}
		})
	}
}

func TestNewNotificationRecord(t *testing.T) {
	r := domain.NewNotificationRecord(
		"evt-123", "corr-456", "order.created",
		"order-789", "order",
		domain.ChannelEmail, "user@example.com", "Order Confirmed", "Your order...",
	)

	if r.ID == "" {
		t.Error("expected non-empty ID")
	}
	if r.EventID != "evt-123" {
		t.Errorf("EventID = %q, want evt-123", r.EventID)
	}
	if r.CorrelationID != "corr-456" {
		t.Errorf("CorrelationID = %q", r.CorrelationID)
	}
	if r.EventType != "order.created" {
		t.Errorf("EventType = %q", r.EventType)
	}
	if r.AggregateID != "order-789" {
		t.Errorf("AggregateID = %q", r.AggregateID)
	}
	if r.Channel != domain.ChannelEmail {
		t.Errorf("Channel = %q", r.Channel)
	}
	if r.Status != domain.NotificationPending {
		t.Errorf("Status = %q, want PENDING", r.Status)
	}
	if r.DeliveryCount != 0 {
		t.Errorf("DeliveryCount = %d, want 0", r.DeliveryCount)
	}
	if r.SentAt != nil {
		t.Error("SentAt should be nil initially")
	}
	if r.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestMarkSent(t *testing.T) {
	r := domain.NewNotificationRecord("evt-1", "corr-1", "order.created", "ord-1", "order",
		domain.ChannelEmail, "a@b.com", "Subj", "Body")

	r.MarkSent("provider-ref-123")

	if r.Status != domain.NotificationSent {
		t.Errorf("Status = %q, want SENT", r.Status)
	}
	if r.ProviderRef != "provider-ref-123" {
		t.Errorf("ProviderRef = %q", r.ProviderRef)
	}
	if r.SentAt == nil {
		t.Error("SentAt should not be nil after MarkSent")
	}
	if r.DeliveryCount != 1 {
		t.Errorf("DeliveryCount = %d, want 1", r.DeliveryCount)
	}
	if r.LastAttemptAt == nil {
		t.Error("LastAttemptAt should not be nil")
	}
}

func TestMarkFailed(t *testing.T) {
	r := domain.NewNotificationRecord("evt-1", "corr-1", "order.created", "ord-1", "order",
		domain.ChannelEmail, "a@b.com", "Subj", "Body")

	r.MarkFailed("provider timeout")

	if r.Status != domain.NotificationFailed {
		t.Errorf("Status = %q, want FAILED", r.Status)
	}
	if r.ProviderError != "provider timeout" {
		t.Errorf("ProviderError = %q", r.ProviderError)
	}
	if r.DeliveryCount != 1 {
		t.Errorf("DeliveryCount = %d, want 1", r.DeliveryCount)
	}
	if r.FirstAttemptAt == nil {
		t.Error("FirstAttemptAt should be set on first failure")
	}
}

func TestMarkFailedSetsFirstAttemptOnlyOnce(t *testing.T) {
	r := domain.NewNotificationRecord("evt-1", "corr-1", "order.created", "ord-1", "order",
		domain.ChannelEmail, "a@b.com", "Subj", "Body")

	r.MarkFailed("error 1")
	first := r.FirstAttemptAt

	time.Sleep(1 * time.Millisecond)
	r.MarkFailed("error 2")

	if !r.FirstAttemptAt.Equal(*first) {
		t.Error("FirstAttemptAt should not change after first failure")
	}
	if r.DeliveryCount != 2 {
		t.Errorf("DeliveryCount = %d, want 2", r.DeliveryCount)
	}
}

func TestMarkSkipped(t *testing.T) {
	r := domain.NewNotificationRecord("evt-1", "corr-1", "order.created", "ord-1", "order",
		domain.ChannelEmail, "a@b.com", "Subj", "Body")

	r.MarkSkipped("no contact info")

	if r.Status != domain.NotificationSkipped {
		t.Errorf("Status = %q, want SKIPPED", r.Status)
	}
	if r.ProviderError != "no contact info" {
		t.Errorf("ProviderError = %q", r.ProviderError)
	}
	if r.DeliveryCount != 0 {
		t.Errorf("DeliveryCount = %d, want 0 for skipped", r.DeliveryCount)
	}
}

func TestMarkSentMultipleTimes(t *testing.T) {
	r := domain.NewNotificationRecord("evt-1", "corr-1", "order.created", "ord-1", "order",
		domain.ChannelEmail, "a@b.com", "Subj", "Body")

	r.MarkSent("ref-1")
	r.MarkSent("ref-2")

	if r.DeliveryCount != 2 {
		t.Errorf("DeliveryCount = %d, want 2", r.DeliveryCount)
	}
	if r.ProviderRef != "ref-2" {
		t.Errorf("ProviderRef = %q, want ref-2 (latest)", r.ProviderRef)
	}
}
