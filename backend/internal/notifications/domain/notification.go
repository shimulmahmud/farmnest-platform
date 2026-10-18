package domain

import (
        "context"
        "time"

        "github.com/google/uuid"
)

// Channel identifies the notification delivery channel.
type Channel string

const (
        ChannelEmail Channel = "EMAIL"
        ChannelSMS   Channel = "SMS"
        ChannelPush  Channel = "PUSH"
)

// NotificationStatus represents the lifecycle state of a notification.
type NotificationStatus string

const (
        NotificationPending   NotificationStatus = "PENDING"
        NotificationSent      NotificationStatus = "SENT"
        NotificationFailed    NotificationStatus = "FAILED"
        NotificationSkipped   NotificationStatus = "SKIPPED" // e.g. duplicate, missing contact info
)

// NotificationRecord represents a notification that was dispatched (or attempted).
// Stored for audit trail and idempotency.
type NotificationRecord struct {
        ID             string             `bson:"_id" json:"id"`
        EventID        string             `bson:"event_id" json:"event_id"`
        CorrelationID  string             `bson:"correlation_id" json:"correlation_id"`
        EventType      string             `bson:"event_type" json:"event_type"`
        AggregateID    string             `bson:"aggregate_id" json:"aggregate_id"`
        AggregateType  string             `bson:"aggregate_type" json:"aggregate_type"`
        Channel        Channel            `bson:"channel" json:"channel"`
        Recipient      string             `bson:"recipient" json:"recipient"`
        Subject        string             `bson:"subject" json:"subject"`
        Body           string             `bson:"body" json:"body"`
        Status         NotificationStatus `bson:"status" json:"status"`
        ProviderRef    string             `bson:"provider_ref" json:"provider_ref,omitempty"`
        ProviderError  string             `bson:"provider_error" json:"provider_error,omitempty"`
        DeliveryCount  int                `bson:"delivery_count" json:"delivery_count"`
        FirstAttemptAt *time.Time         `bson:"first_attempt_at,omitempty" json:"first_attempt_at,omitempty"`
        LastAttemptAt  *time.Time         `bson:"last_attempt_at,omitempty" json:"last_attempt_at,omitempty"`
        SentAt         *time.Time         `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
        CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
        UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// NewNotificationRecord creates a notification record in PENDING status.
func NewNotificationRecord(eventID, correlationID, eventType, aggregateID, aggregateType string, channel Channel, recipient, subject, body string) *NotificationRecord {
        now := time.Now().UTC()
        return &NotificationRecord{
                ID:            uuid.New().String(),
                EventID:       eventID,
                CorrelationID: correlationID,
                EventType:     eventType,
                AggregateID:   aggregateID,
                AggregateType: aggregateType,
                Channel:       channel,
                Recipient:     recipient,
                Subject:       subject,
                Body:          body,
                Status:        NotificationPending,
                CreatedAt:     now,
                UpdatedAt:     now,
        }
}

// MarkSent records a successful delivery.
func (n *NotificationRecord) MarkSent(providerRef string) {
        now := time.Now().UTC()
        n.Status = NotificationSent
        n.ProviderRef = providerRef
        n.SentAt = &now
        n.DeliveryCount++
        n.LastAttemptAt = &now
        n.UpdatedAt = now
}

// MarkFailed records a failed delivery attempt.
func (n *NotificationRecord) MarkFailed(errMsg string) {
        now := time.Now().UTC()
        n.Status = NotificationFailed
        n.ProviderError = errMsg
        n.DeliveryCount++
        n.LastAttemptAt = &now
        if n.FirstAttemptAt == nil {
                n.FirstAttemptAt = &now
        }
        n.UpdatedAt = now
}

// MarkSkipped records that this notification was skipped (e.g. duplicate, missing contact).
func (n *NotificationRecord) MarkSkipped(reason string) {
        now := time.Now().UTC()
        n.Status = NotificationSkipped
        n.ProviderError = reason
        n.UpdatedAt = now
}

// NotificationRepository persists notification records.
type NotificationRepository interface {
        // Insert creates a new notification record.
        Insert(ctx context.Context, record *NotificationRecord) error

        // FindByEventIDAndChannel checks if a notification for this event+channel already exists.
        // Used for idempotency — if found, the event is a duplicate.
        FindByEventIDAndChannel(ctx context.Context, eventID string, channel Channel) (*NotificationRecord, error)

        // UpdateStatus persists the record after status change.
        UpdateStatus(ctx context.Context, record *NotificationRecord) error
}
