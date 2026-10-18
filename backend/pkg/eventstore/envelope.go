package eventstore

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Envelope is the standard event message structure for all domain events.
// Every event published to and consumed from JetStream must use this envelope.
type Envelope struct {
	EventID      string          `json:"event_id"`       // unique ID for this event
	EventType    string          `json:"event_type"`     // e.g. "order.created"
	Version      int             `json:"version"`        // event schema version
	AggregateID  string          `json:"aggregate_id"`   // the entity ID (order ID, payment ID, etc.)
	AggregateType string         `json:"aggregate_type"` // "order", "payment", "inventory", "product"
	OccurredAt   time.Time       `json:"occurred_at"`     // when the event occurred
	CorrelationID string          `json:"correlation_id"` // tracks a flow across events (e.g. one order lifecycle)
	CausationID  string          `json:"causation_id"`   // the event that caused this event
	Payload      json.RawMessage `json:"payload"`        // domain-specific event data
}

// NewEnvelope creates a new event envelope with a generated event_id and timestamp.
func NewEnvelope(eventType, aggregateID, aggregateType string, version int, payload interface{}) (*Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		EventID:      uuid.New().String(),
		EventType:    eventType,
		Version:      version,
		AggregateID:  aggregateID,
		AggregateType: aggregateType,
		OccurredAt:   time.Now().UTC(),
		CorrelationID: uuid.New().String(),
		CausationID:  "",
		Payload:      payloadBytes,
	}, nil
}

// NewEnvelopeWithCorrelation creates an envelope with explicit correlation and causation IDs.
func NewEnvelopeWithCorrelation(eventType, aggregateID, aggregateType string, version int, payload interface{}, correlationID, causationID string) (*Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return &Envelope{
		EventID:      uuid.New().String(),
		EventType:    eventType,
		Version:      version,
		AggregateID:  aggregateID,
		AggregateType: aggregateType,
		OccurredAt:   time.Now().UTC(),
		CorrelationID: correlationID,
		CausationID:  causationID,
		Payload:      payloadBytes,
	}, nil
}

// NewChildEnvelope creates an envelope that is a causal child of a parent event.
// The child inherits the parent's correlation_id and uses the parent's event_id as causation_id.
func NewChildEnvelope(parent *Envelope, eventType, aggregateID, aggregateType string, version int, payload interface{}) (*Envelope, error) {
	return NewEnvelopeWithCorrelation(eventType, aggregateID, aggregateType, version, payload, parent.CorrelationID, parent.EventID)
}

// DecodePayload unmarshals the envelope payload into the target struct.
func (e *Envelope) DecodePayload(target interface{}) error {
	return json.Unmarshal(e.Payload, target)
}

// Subject returns the NATS subject for this event.
// Subject format: <stream_name>.<event_type>
// e.g. "ECOMMERCE_ORDERS.order.created"
func (e *Envelope) Subject() string {
	stream := AggregateToStream(e.AggregateType)
	return stream + "." + e.EventType
}

// Serialize marshals the envelope to JSON bytes for publishing.
func (e *Envelope) Serialize() ([]byte, error) {
	return json.Marshal(e)
}

// DeserializeEnvelope unmarshals JSON bytes into an Envelope.
func DeserializeEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}
