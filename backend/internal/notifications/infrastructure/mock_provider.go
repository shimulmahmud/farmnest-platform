package infrastructure

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ecommerce/backend/internal/notifications/domain"
)

// --- Log Email Provider ---

// LogEmailProvider is a development provider that logs emails to stdout
// via the application logger. It never fails.
type LogEmailProvider struct {
	Sent atomic.Int64 // total emails "sent"
}

// NewLogEmailProvider creates a new development email provider.
func NewLogEmailProvider() *LogEmailProvider {
	return &LogEmailProvider{}
}

// SendEmail logs the email and returns success.
func (p *LogEmailProvider) SendEmail(_ context.Context, msg *domain.EmailMessage) (*domain.EmailResult, error) {
	p.Sent.Add(1)
	msgID := fmt.Sprintf("log-email-%d", p.Sent.Load())
	fmt.Printf("[EMAIL] to=%s subject=%q body=%q msg_id=%s\n", msg.To, msg.Subject, msg.Body, msgID)
	return &domain.EmailResult{
		Success:   true,
		MessageID: msgID,
	}, nil
}

// Stats returns the count of sent emails.
func (p *LogEmailProvider) Stats() int64 {
	return p.Sent.Load()
}

// --- Log SMS Provider ---

// LogSMSProvider is a development provider that logs SMS to stdout.
type LogSMSProvider struct {
	Sent atomic.Int64
}

// NewLogSMSProvider creates a new development SMS provider.
func NewLogSMSProvider() *LogSMSProvider {
	return &LogSMSProvider{}
}

// SendSMS logs the SMS and returns success.
func (p *LogSMSProvider) SendSMS(_ context.Context, msg *domain.SMSMessage) (*domain.SMSResult, error) {
	p.Sent.Add(1)
	msgID := fmt.Sprintf("log-sms-%d", p.Sent.Load())
	fmt.Printf("[SMS] to=%s body=%q msg_id=%s\n", msg.To, msg.Body, msgID)
	return &domain.SMSResult{
		Success:   true,
		MessageID: msgID,
	}, nil
}

// Stats returns the count of sent SMS.
func (p *LogSMSProvider) Stats() int64 {
	return p.Sent.Load()
}

// --- Log Push Provider ---

// LogPushProvider is a development provider that logs push notifications to stdout.
type LogPushProvider struct {
	Sent atomic.Int64
}

// NewLogPushProvider creates a new development push provider.
func NewLogPushProvider() *LogPushProvider {
	return &LogPushProvider{}
}

// SendPush logs the push notification and returns success.
func (p *LogPushProvider) SendPush(_ context.Context, msg *domain.PushMessage) (*domain.PushResult, error) {
	p.Sent.Add(1)
	msgID := fmt.Sprintf("log-push-%d", p.Sent.Load())
	fmt.Printf("[PUSH] token=%s title=%q body=%q msg_id=%s\n", msg.DeviceToken, msg.Title, msg.Body, msgID)
	return &domain.PushResult{
		Success:   true,
		MessageID: msgID,
	}, nil
}

// Stats returns the count of sent push notifications.
func (p *LogPushProvider) Stats() int64 {
	return p.Sent.Load()
}

// --- In-Memory Test Provider (captures all sends for assertion) ---

// InMemoryProvider captures all sent notifications for testing.
// It implements all three provider interfaces via separate slices.
type InMemoryProvider struct {
	mu      sync.Mutex
	Emails  []*domain.EmailMessage
	SMSs    []*domain.SMSMessage
	Pushes  []*domain.PushMessage
	Err     error // set to force all sends to fail
}

// NewInMemoryProvider creates a test provider that captures sends.
func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{}
}

// SetError makes all subsequent sends return this error.
func (p *InMemoryProvider) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Err = err
}

// SendEmail records the email.
func (p *InMemoryProvider) SendEmail(_ context.Context, msg *domain.EmailMessage) (*domain.EmailResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Err != nil {
		return &domain.EmailResult{Success: false, Error: p.Err.Error()}, p.Err
	}
	p.Emails = append(p.Emails, msg)
	return &domain.EmailResult{Success: true, MessageID: fmt.Sprintf("test-email-%d", len(p.Emails))}, nil
}

// SendSMS records the SMS.
func (p *InMemoryProvider) SendSMS(_ context.Context, msg *domain.SMSMessage) (*domain.SMSResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Err != nil {
		return &domain.SMSResult{Success: false, Error: p.Err.Error()}, p.Err
	}
	p.SMSs = append(p.SMSs, msg)
	return &domain.SMSResult{Success: true, MessageID: fmt.Sprintf("test-sms-%d", len(p.SMSs))}, nil
}

// SendPush records the push notification.
func (p *InMemoryProvider) SendPush(_ context.Context, msg *domain.PushMessage) (*domain.PushResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Err != nil {
		return &domain.PushResult{Success: false, Error: p.Err.Error()}, p.Err
	}
	p.Pushes = append(p.Pushes, msg)
	return &domain.PushResult{Success: true, MessageID: fmt.Sprintf("test-push-%d", len(p.Pushes))}, nil
}

// EmailCount returns the number of sent emails.
func (p *InMemoryProvider) EmailCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.Emails)
}

// SMSCount returns the number of sent SMS.
func (p *InMemoryProvider) SMSCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.SMSs)
}

// PushCount returns the number of sent push notifications.
func (p *InMemoryProvider) PushCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.Pushes)
}
