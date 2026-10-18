package infrastructure_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ecommerce/backend/internal/notifications/domain"
	"github.com/ecommerce/backend/internal/notifications/infrastructure"
)

func TestInMemoryProvider_SendEmail(t *testing.T) {
	p := infrastructure.NewInMemoryProvider()

	result, err := p.SendEmail(context.Background(), &domain.EmailMessage{
		To:      "user@example.com",
		Subject: "Test",
		Body:    "Hello",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.MessageID != "test-email-1" {
		t.Errorf("MessageID = %q, want test-email-1", result.MessageID)
	}
	if p.EmailCount() != 1 {
		t.Errorf("EmailCount = %d, want 1", p.EmailCount())
	}
}

func TestInMemoryProvider_SendSMS(t *testing.T) {
	p := infrastructure.NewInMemoryProvider()

	result, err := p.SendSMS(context.Background(), &domain.SMSMessage{
		To:   "+1234567890",
		Body: "Hello SMS",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if p.SMSCount() != 1 {
		t.Errorf("SMSCount = %d, want 1", p.SMSCount())
	}
}

func TestInMemoryProvider_SendPush(t *testing.T) {
	p := infrastructure.NewInMemoryProvider()

	result, err := p.SendPush(context.Background(), &domain.PushMessage{
		DeviceToken: "token-abc",
		Title:       "Hi",
		Body:        "Push body",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if p.PushCount() != 1 {
		t.Errorf("PushCount = %d, want 1", p.PushCount())
	}
}

func TestInMemoryProvider_SetError(t *testing.T) {
	p := infrastructure.NewInMemoryProvider()
	p.SetError(errors.New("provider down"))

	_, err := p.SendEmail(context.Background(), &domain.EmailMessage{
		To: "a@b.com", Subject: "s", Body: "b",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "provider down" {
		t.Errorf("error = %q, want provider down", err.Error())
	}

	result, _ := p.SendSMS(context.Background(), &domain.SMSMessage{
		To: "+1", Body: "b",
	})
	if result.Success {
		t.Error("expected SMS to fail")
	}
	if result.Error != "provider down" {
		t.Errorf("result.Error = %q", result.Error)
	}

	// Error should not affect count.
	if p.EmailCount() != 0 {
		t.Error("failed sends should not increment count")
	}
}

func TestInMemoryProvider_MultipleSends(t *testing.T) {
	p := infrastructure.NewInMemoryProvider()

	for i := 0; i < 3; i++ {
		p.SendEmail(context.Background(), &domain.EmailMessage{
			To: "a@b.com", Subject: "s", Body: "b",
		})
	}
	if p.EmailCount() != 3 {
		t.Errorf("EmailCount = %d, want 3", p.EmailCount())
	}
}

func TestLogEmailProvider(t *testing.T) {
	p := infrastructure.NewLogEmailProvider()

	result, err := p.SendEmail(context.Background(), &domain.EmailMessage{
		To: "test@example.com", Subject: "Test Subject", Body: "Test Body",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.MessageID == "" {
		t.Error("expected non-empty MessageID")
	}
	if p.Stats() != 1 {
		t.Errorf("Stats = %d, want 1", p.Stats())
	}
}

func TestLogSMSProvider(t *testing.T) {
	p := infrastructure.NewLogSMSProvider()

	result, err := p.SendSMS(context.Background(), &domain.SMSMessage{
		To: "+1234567890", Body: "Test SMS",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if p.Stats() != 1 {
		t.Errorf("Stats = %d, want 1", p.Stats())
	}
}

func TestLogPushProvider(t *testing.T) {
	p := infrastructure.NewLogPushProvider()

	result, err := p.SendPush(context.Background(), &domain.PushMessage{
		DeviceToken: "token-123", Title: "Title", Body: "Body",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if p.Stats() != 1 {
		t.Errorf("Stats = %d, want 1", p.Stats())
	}
}
