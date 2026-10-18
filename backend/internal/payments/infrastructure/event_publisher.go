package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/backend/internal/payments/domain"
	"github.com/nats-io/nats.go"
)

type paymentEventPublisher struct {
	conn *nats.Conn
}

// NewPaymentEventPublisher creates a NATS-backed PaymentEventPublisher.
func NewPaymentEventPublisher(conn *nats.Conn) domain.PaymentEventPublisher {
	return &paymentEventPublisher{conn: conn}
}

func (p *paymentEventPublisher) Publish(_ context.Context, event *domain.PaymentEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.conn.Publish("payment."+event.EventType, data)
}
