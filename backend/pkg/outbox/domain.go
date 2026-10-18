package outbox

import (
        "context"
        "encoding/json"
        "time"

        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/google/uuid"
)

// OutboxStatus represents the processing state of an outbox event.
type OutboxStatus string

const (
        // StatusPending means the event is waiting to be published.
        StatusPending OutboxStatus = "PENDING"
        // StatusPublishing means the worker has claimed the event and is publishing it.
        StatusPublishing OutboxStatus = "PUBLISHING"
        // StatusPublished means the event was successfully published to NATS.
        StatusPublished OutboxStatus = "PUBLISHED"
        // StatusFailed means the event exhausted all retry attempts.
        StatusFailed OutboxStatus = "FAILED"
)

// MaxPublishAttempts is the maximum number of publish attempts before marking as FAILED.
const MaxPublishAttempts = 10

// OutboxEvent represents an event waiting to be published to NATS.
// It is stored in MongoDB as part of the same transaction as the business write,
// guaranteeing at-least-once delivery.
type OutboxEvent struct {
        EventID    string          `bson:"event_id" json:"event_id"`
        EventType  string          `bson:"event_type" json:"event_type"`
        AggregateID  string        `bson:"aggregate_id" json:"aggregate_id"`
        AggregateType string       `bson:"aggregate_type" json:"aggregate_type"`
        Payload    json.RawMessage `bson:"payload" json:"payload"`
        Status     OutboxStatus    `bson:"status" json:"status"`
        Attempts   int             `bson:"attempts" json:"attempts"`
        CreatedAt  time.Time       `bson:"created_at" json:"created_at"`
        PublishedAt *time.Time     `bson:"published_at" json:"published_at,omitempty"`
        LastError  string          `bson:"last_error" json:"last_error,omitempty"`
        // Metadata fields from the event envelope for the worker.
        CorrelationID string          `bson:"correlation_id" json:"correlation_id,omitempty"`
        CausationID  string          `bson:"causation_id" json:"causation_id,omitempty"`
        Version      int             `bson:"version" json:"version"`
        Subject      string          `bson:"subject" json:"subject,omitempty"`
}

// OutboxRepository defines persistence operations for the outbox pattern.
type OutboxRepository interface {
        // Insert saves a new outbox event as PENDING.
        Insert(ctx context.Context, event *OutboxEvent) error

        // ClaimNext atomically transitions a PENDING event to PUBLISHING,
        // incrementing attempts. Returns nil if no events are available.
        ClaimNext(ctx context.Context) (*OutboxEvent, error)

        // ResetToPending moves a PUBLISHING event back to PENDING for retry.
        // Used when a publish attempt fails but max attempts are not yet exhausted.
        ResetToPending(ctx context.Context, eventID string) error

        // MarkPublished sets the event to PUBLISHED with the published timestamp.
        MarkPublished(ctx context.Context, eventID string) error

        // MarkFailed sets the event to FAILED with the last error.
        MarkFailed(ctx context.Context, eventID string, errMsg string) error

        // ResetStale returns PUBLISHING events that have been in-flight longer
        // than the given duration back to PENDING so they can be retried (crash recovery).
        ResetStale(ctx context.Context, staleThreshold time.Duration) (int64, error)

        // DeletePublished removes successfully published events older than the given duration.
        DeletePublished(ctx context.Context, olderThan time.Duration) (int64, error)

        // CountByStatus returns the count of events grouped by status.
        CountByStatus(ctx context.Context) (map[OutboxStatus]int64, error)
}

// NewOutboxEvent creates an OutboxEvent from an eventstore.Envelope.
// The event is created in PENDING status with 0 attempts.
func NewOutboxEvent(env *eventstore.Envelope) *OutboxEvent {
        return &OutboxEvent{
                EventID:       env.EventID,
                EventType:     env.EventType,
                AggregateID:   env.AggregateID,
                AggregateType: env.AggregateType,
                Payload:       env.Payload,
                Status:        StatusPending,
                Attempts:      0,
                CreatedAt:     time.Now().UTC(),
                CorrelationID: env.CorrelationID,
                CausationID:   env.CausationID,
                Version:       env.Version,
                Subject:       env.Subject(),
        }
}

// NewOutboxEventDirect creates an OutboxEvent with explicit fields.
// This is used by the publisher adapters when building events from domain-specific structs.
func NewOutboxEventDirect(eventType, aggregateID, aggregateType, correlationID string, version int, payload interface{}) (*OutboxEvent, error) {
        payloadBytes, err := json.Marshal(payload)
        if err != nil {
                return nil, err
        }
        if correlationID == "" {
                correlationID = uuid.New().String()
        }

        env := &eventstore.Envelope{
                EventID:       uuid.New().String(),
                EventType:     eventType,
                Version:       version,
                AggregateID:   aggregateID,
                AggregateType: aggregateType,
                OccurredAt:    time.Now().UTC(),
                CorrelationID: correlationID,
                CausationID:   "",
                Payload:       payloadBytes,
        }

        return NewOutboxEvent(env), nil
}

// ToEnvelope converts an OutboxEvent back to an eventstore.Envelope for publishing.
func (e *OutboxEvent) ToEnvelope() *eventstore.Envelope {
        return &eventstore.Envelope{
                EventID:       e.EventID,
                EventType:     e.EventType,
                Version:       e.Version,
                AggregateID:   e.AggregateID,
                AggregateType: e.AggregateType,
                OccurredAt:    e.CreatedAt,
                CorrelationID: e.CorrelationID,
                CausationID:   e.CausationID,
                Payload:       e.Payload,
        }
}

// BackoffDuration returns the backoff wait time for a given attempt number (0-indexed).
// Uses exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 5m.
func BackoffDuration(attempt int) time.Duration {
        backoffs := []time.Duration{
                1 * time.Second,
                2 * time.Second,
                4 * time.Second,
                8 * time.Second,
                16 * time.Second,
                32 * time.Second,
                64 * time.Second,
                128 * time.Second,
                256 * time.Second,
                5 * time.Minute,
        }
        if attempt < 0 {
                attempt = 0
        }
        if attempt >= len(backoffs) {
                return backoffs[len(backoffs)-1]
        }
        return backoffs[attempt]
}
