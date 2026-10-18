package domain

import (
	"context"
)

// EmailMessage represents an email to be sent.
type EmailMessage struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"` // plain-text body; HTML can be added later
	From    string `json:"from,omitempty"`
}

// EmailResult is the outcome of an email send attempt.
type EmailResult struct {
	Success   bool
	MessageID string // provider-specific reference
	Error     string
}

// EmailProvider sends transactional emails.
// Implementations must be safe for concurrent use.
type EmailProvider interface {
	// SendEmail delivers an email. Returns a result with provider message ID on success.
	SendEmail(ctx context.Context, msg *EmailMessage) (*EmailResult, error)
}

// SMSMessage represents an SMS to be sent.
type SMSMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
	From string `json:"from,omitempty"`
}

// SMSResult is the outcome of an SMS send attempt.
type SMSResult struct {
	Success   bool
	MessageID string
	Error     string
}

// SMSProvider sends SMS messages.
// Implementations must be safe for concurrent use.
type SMSProvider interface {
	// SendSMS delivers an SMS message. Returns a result with provider message ID on success.
	SendSMS(ctx context.Context, msg *SMSMessage) (*SMSResult, error)
}

// PushMessage represents a push notification to be sent.
type PushMessage struct {
	DeviceToken string `json:"device_token"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Data        map[string]string `json:"data,omitempty"` // key-value payload
}

// PushResult is the outcome of a push notification send attempt.
type PushResult struct {
	Success   bool
	MessageID string
	Error     string
}

// PushProvider sends push notifications.
// Implementations must be safe for concurrent use.
type PushProvider interface {
	// SendPush delivers a push notification. Returns a result on success.
	SendPush(ctx context.Context, msg *PushMessage) (*PushResult, error)
}
