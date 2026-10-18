package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/backend/internal/inventory/domain"
	"github.com/nats-io/nats.go"
)

type eventPublisher struct {
	conn *nats.Conn
}

// NewEventPublisher creates a NATS-backed EventPublisher.
func NewEventPublisher(conn *nats.Conn) domain.EventPublisher {
	return &eventPublisher{conn: conn}
}

func (p *eventPublisher) Publish(_ context.Context, subject string, event *domain.InventoryEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.conn.Publish(subject, data)
}
