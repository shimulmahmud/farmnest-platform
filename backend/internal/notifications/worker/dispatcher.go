package worker

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	ordDomain "github.com/ecommerce/backend/internal/orders/domain"
	payDomain "github.com/ecommerce/backend/internal/payments/domain"
	notifDomain "github.com/ecommerce/backend/internal/notifications/domain"
	"github.com/ecommerce/backend/pkg/eventstore"
	"github.com/ecommerce/backend/pkg/logger"
)

// CustomerContactResolver resolves an aggregate (e.g. order) to its
// customer contact information (email, phone) for notification delivery.
type CustomerContactResolver interface {
	ResolveContact(ctx context.Context, aggregateID string) (*ContactInfo, error)
}

// ContactInfo holds the resolved contact information for a customer.
type ContactInfo struct {
	Email     string
	Phone     string
	PushToken string // may be empty
}

// DispatcherConfig holds configuration for the notification dispatcher.
type DispatcherConfig struct {
	EnableEmail bool
	EnableSMS   bool
	EnablePush  bool
}

// DefaultDispatcherConfig returns config with email enabled.
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		EnableEmail: true,
	}
}

// DispatcherDependencies holds the injected dependencies.
type DispatcherDependencies struct {
	EmailProvider    notifDomain.EmailProvider
	SMSProvider      notifDomain.SMSProvider
	PushProvider     notifDomain.PushProvider
	NotificationRepo notifDomain.NotificationRepository
	ContactResolver  CustomerContactResolver
	Log              *logger.Logger
	Config           DispatcherConfig
}

// Metrics tracks notification dispatcher statistics.
type Metrics struct {
	Processed        atomic.Int64
	Sent             atomic.Int64
	Failed           atomic.Int64
	Skipped          atomic.Int64
	IdempotentSkipped atomic.Int64
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"processed":         m.Processed.Load(),
		"sent":              m.Sent.Load(),
		"failed":            m.Failed.Load(),
		"skipped":           m.Skipped.Load(),
		"idempotent_skipped": m.IdempotentSkipped.Load(),
	}
}

// Dispatcher routes domain events to notification providers.
// It never blocks HTTP requests — it runs exclusively as a JetStream consumer.
type Dispatcher struct {
	deps    DispatcherDependencies
	metrics Metrics
}

// NewDispatcher creates a notification dispatcher.
func NewDispatcher(deps DispatcherDependencies) *Dispatcher {
	return &Dispatcher{deps: deps}
}

// Metrics returns the dispatcher metrics.
func (d *Dispatcher) Metrics() *Metrics {
	return &d.metrics
}

// HandleOrderCreated processes order.created events.
func (d *Dispatcher) HandleOrderCreated() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var evt ordDomain.OrderEvent
		if err := env.DecodePayload(&evt); err != nil {
			d.deps.Log.Errorw("notification: failed to decode order.created payload",
				"event_id", env.EventID, "error", err)
			return nil // non-retryable
		}

		d.metrics.Processed.Add(1)

		contact, err := d.resolveOrSkip(ctx, env, evt.CustomerID)
		if err != nil {
			return err
		}
		if contact == nil {
			return nil
		}

		subject := fmt.Sprintf("Order Confirmed \u2014 %s", evt.OrderNumber)
		body := fmt.Sprintf(
			"Your order %s has been received and is being processed.\n\nOrder ID: %s\nStatus: %s",
			evt.OrderNumber, evt.OrderID, evt.ToStatus,
		)

		d.sendEmail(ctx, env, contact.Email, subject, body)
		d.sendSMS(ctx, env, contact.Phone,
			fmt.Sprintf("Order %s confirmed! We're processing it now.", evt.OrderNumber))
		d.sendPush(ctx, env, contact, "Order Confirmed",
			fmt.Sprintf("Your order %s has been received.", evt.OrderNumber),
			map[string]string{"order_id": evt.OrderID, "order_number": evt.OrderNumber})

		return nil
	}
}

// HandleOrderPaid processes order.paid events.
func (d *Dispatcher) HandleOrderPaid() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var evt ordDomain.OrderEvent
		if err := env.DecodePayload(&evt); err != nil {
			d.deps.Log.Errorw("notification: failed to decode order.paid payload",
				"event_id", env.EventID, "error", err)
			return nil
		}

		d.metrics.Processed.Add(1)

		contact, err := d.resolveOrSkip(ctx, env, evt.CustomerID)
		if err != nil {
			return err
		}
		if contact == nil {
			return nil
		}

		subject := fmt.Sprintf("Payment Received \u2014 %s", evt.OrderNumber)
		body := fmt.Sprintf(
			"Payment for order %s has been confirmed. Your order is now being prepared for shipment.\n\nOrder ID: %s",
			evt.OrderNumber, evt.OrderID,
		)

		d.sendEmail(ctx, env, contact.Email, subject, body)
		d.sendSMS(ctx, env, contact.Phone,
			fmt.Sprintf("Payment confirmed for order %s. Preparing your order!", evt.OrderNumber))
		d.sendPush(ctx, env, contact, "Payment Confirmed",
			fmt.Sprintf("Payment for order %s received.", evt.OrderNumber),
			map[string]string{"order_id": evt.OrderID, "order_number": evt.OrderNumber})

		return nil
	}
}

// HandleOrderShipped processes order.shipped events.
func (d *Dispatcher) HandleOrderShipped() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var evt ordDomain.OrderEvent
		if err := env.DecodePayload(&evt); err != nil {
			d.deps.Log.Errorw("notification: failed to decode order.shipped payload",
				"event_id", env.EventID, "error", err)
			return nil
		}

		d.metrics.Processed.Add(1)

		contact, err := d.resolveOrSkip(ctx, env, evt.CustomerID)
		if err != nil {
			return err
		}
		if contact == nil {
			return nil
		}

		subject := fmt.Sprintf("Your Order Has Shipped \u2014 %s", evt.OrderNumber)
		body := fmt.Sprintf(
			"Great news! Your order %s has been shipped.\n\nTracking Reference: %s\nOrder ID: %s",
			evt.OrderNumber, evt.ShipmentReference, evt.OrderID,
		)

		d.sendEmail(ctx, env, contact.Email, subject, body)
		d.sendSMS(ctx, env, contact.Phone,
			fmt.Sprintf("Your order %s has shipped! Track: %s", evt.OrderNumber, evt.ShipmentReference))
		d.sendPush(ctx, env, contact, "Order Shipped",
			fmt.Sprintf("Order %s is on its way!", evt.OrderNumber),
			map[string]string{"order_id": evt.OrderID, "order_number": evt.OrderNumber, "shipment_reference": evt.ShipmentReference})

		return nil
	}
}

// HandleOrderDelivered processes order.delivered events.
func (d *Dispatcher) HandleOrderDelivered() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var evt ordDomain.OrderEvent
		if err := env.DecodePayload(&evt); err != nil {
			d.deps.Log.Errorw("notification: failed to decode order.delivered payload",
				"event_id", env.EventID, "error", err)
			return nil
		}

		d.metrics.Processed.Add(1)

		contact, err := d.resolveOrSkip(ctx, env, evt.CustomerID)
		if err != nil {
			return err
		}
		if contact == nil {
			return nil
		}

		subject := fmt.Sprintf("Order Delivered \u2014 %s", evt.OrderNumber)
		body := fmt.Sprintf(
			"Your order %s has been delivered. Thank you for shopping with us!\n\nOrder ID: %s",
			evt.OrderNumber, evt.OrderID,
		)

		d.sendEmail(ctx, env, contact.Email, subject, body)
		d.sendSMS(ctx, env, contact.Phone,
			fmt.Sprintf("Order %s has been delivered. Enjoy!", evt.OrderNumber))
		d.sendPush(ctx, env, contact, "Order Delivered",
			fmt.Sprintf("Your order %s has arrived!", evt.OrderNumber),
			map[string]string{"order_id": evt.OrderID, "order_number": evt.OrderNumber})

		return nil
	}
}

// HandlePaymentFailed processes payment.failed events.
func (d *Dispatcher) HandlePaymentFailed() eventstore.EventHandler {
	return func(ctx context.Context, env *eventstore.Envelope) error {
		var evt payDomain.PaymentEvent
		if err := env.DecodePayload(&evt); err != nil {
			d.deps.Log.Errorw("notification: failed to decode payment.failed payload",
				"event_id", env.EventID, "error", err)
			return nil
		}

		if evt.OrderID == "" {
			d.deps.Log.Errorw("notification: payment.failed missing order_id",
				"event_id", env.EventID, "payment_id", evt.PaymentID)
			return nil
		}

		d.metrics.Processed.Add(1)

		contact, err := d.resolveOrSkip(ctx, env, evt.OrderID)
		if err != nil {
			return err
		}
		if contact == nil {
			return nil
		}

		amountStr := formatAmount(evt.Amount, evt.Currency)
		subject := "Payment Failed \u2014 Action Required"
		body := fmt.Sprintf(
			"We were unable to process your payment of %s for order %s. Please try again or update your payment method.\n\nPayment ID: %s",
			amountStr, evt.OrderID, evt.PaymentID,
		)

		d.sendEmail(ctx, env, contact.Email, subject, body)
		d.sendSMS(ctx, env, contact.Phone,
			fmt.Sprintf("Payment of %s for your order failed. Please retry.", amountStr))
		d.sendPush(ctx, env, contact, "Payment Failed",
			"Your payment could not be processed. Please try again.",
			map[string]string{"order_id": evt.OrderID, "payment_id": evt.PaymentID})

		return nil
	}
}

// --- Internal helpers ---

// resolveOrSkip resolves customer contact info, handling missing contact.
// Returns nil contact if the event should be skipped.
// Returns an error only for retryable failures.
func (d *Dispatcher) resolveOrSkip(ctx context.Context, env *eventstore.Envelope, aggregateID string) (*ContactInfo, error) {
	contact, err := d.deps.ContactResolver.ResolveContact(ctx, aggregateID)
	if err != nil {
		d.deps.Log.Errorw("notification: failed to resolve customer contact",
			"event_id", env.EventID, "aggregate_id", aggregateID, "error", err)
		return nil, err // retryable
	}

	if contact == nil || (contact.Email == "" && contact.Phone == "") {
		d.deps.Log.Infow("notification: skipping, no contact info available",
			"event_id", env.EventID, "aggregate_id", aggregateID)
		d.metrics.Skipped.Add(1)
		return nil, nil
	}

	return contact, nil
}

// sendEmail sends an email notification with idempotency tracking.
func (d *Dispatcher) sendEmail(ctx context.Context, env *eventstore.Envelope, to, subject, body string) {
	if !d.deps.Config.EnableEmail || to == "" {
		return
	}

	existing, err := d.deps.NotificationRepo.FindByEventIDAndChannel(ctx, env.EventID, notifDomain.ChannelEmail)
	if err != nil {
		d.deps.Log.Warnw("notification: failed to check email idempotency",
			"event_id", env.EventID, "error", err)
	}
	if existing != nil {
		d.metrics.IdempotentSkipped.Add(1)
		d.deps.Log.Debugw("notification: email already sent, skipping (idempotent)",
			"event_id", env.EventID, "notification_id", existing.ID)
		return
	}

	record := notifDomain.NewNotificationRecord(
		env.EventID, env.CorrelationID, env.EventType,
		env.AggregateID, env.AggregateType,
		notifDomain.ChannelEmail, to, subject, body,
	)

	if insertErr := d.deps.NotificationRepo.Insert(ctx, record); insertErr != nil {
		d.deps.Log.Errorw("notification: failed to insert email record",
			"event_id", env.EventID, "error", insertErr)
		return
	}

	result, sendErr := d.deps.EmailProvider.SendEmail(ctx, &notifDomain.EmailMessage{
		To:      to,
		Subject: subject,
		Body:    body,
	})

	if sendErr != nil {
		record.MarkFailed(sendErr.Error())
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Failed.Add(1)
		d.deps.Log.Errorw("notification: email send failed",
			"event_id", env.EventID, "recipient", to, "error", sendErr)
		return
	}

	if result.Success {
		record.MarkSent(result.MessageID)
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Sent.Add(1)
		d.deps.Log.Infow("notification: email sent",
			"event_id", env.EventID, "recipient", to, "provider_ref", result.MessageID)
	} else {
		record.MarkFailed(result.Error)
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Failed.Add(1)
		d.deps.Log.Warnw("notification: email provider returned failure",
			"event_id", env.EventID, "recipient", to, "provider_error", result.Error)
	}
}

// sendSMS sends an SMS notification with idempotency tracking.
func (d *Dispatcher) sendSMS(ctx context.Context, env *eventstore.Envelope, to, body string) {
	if !d.deps.Config.EnableSMS || to == "" {
		return
	}

	existing, err := d.deps.NotificationRepo.FindByEventIDAndChannel(ctx, env.EventID, notifDomain.ChannelSMS)
	if err != nil {
		d.deps.Log.Warnw("notification: failed to check SMS idempotency",
			"event_id", env.EventID, "error", err)
	}
	if existing != nil {
		d.metrics.IdempotentSkipped.Add(1)
		d.deps.Log.Debugw("notification: SMS already sent, skipping (idempotent)",
			"event_id", env.EventID, "notification_id", existing.ID)
		return
	}

	record := notifDomain.NewNotificationRecord(
		env.EventID, env.CorrelationID, env.EventType,
		env.AggregateID, env.AggregateType,
		notifDomain.ChannelSMS, to, "", body,
	)

	if insertErr := d.deps.NotificationRepo.Insert(ctx, record); insertErr != nil {
		d.deps.Log.Errorw("notification: failed to insert SMS record",
			"event_id", env.EventID, "error", insertErr)
		return
	}

	result, sendErr := d.deps.SMSProvider.SendSMS(ctx, &notifDomain.SMSMessage{
		To:   to,
		Body: body,
	})

	if sendErr != nil {
		record.MarkFailed(sendErr.Error())
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Failed.Add(1)
		d.deps.Log.Errorw("notification: SMS send failed",
			"event_id", env.EventID, "recipient", to, "error", sendErr)
		return
	}

	if result.Success {
		record.MarkSent(result.MessageID)
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Sent.Add(1)
		d.deps.Log.Infow("notification: SMS sent",
			"event_id", env.EventID, "recipient", to, "provider_ref", result.MessageID)
	} else {
		record.MarkFailed(result.Error)
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Failed.Add(1)
	}
}

// sendPush sends a push notification with idempotency tracking.
func (d *Dispatcher) sendPush(ctx context.Context, env *eventstore.Envelope, contact *ContactInfo, title, body string, data map[string]string) {
	if !d.deps.Config.EnablePush || contact.PushToken == "" {
		return
	}

	existing, err := d.deps.NotificationRepo.FindByEventIDAndChannel(ctx, env.EventID, notifDomain.ChannelPush)
	if err != nil {
		d.deps.Log.Warnw("notification: failed to check push idempotency",
			"event_id", env.EventID, "error", err)
	}
	if existing != nil {
		d.metrics.IdempotentSkipped.Add(1)
		d.deps.Log.Debugw("notification: push already sent, skipping (idempotent)",
			"event_id", env.EventID, "notification_id", existing.ID)
		return
	}

	record := notifDomain.NewNotificationRecord(
		env.EventID, env.CorrelationID, env.EventType,
		env.AggregateID, env.AggregateType,
		notifDomain.ChannelPush, contact.PushToken, title, body,
	)

	if insertErr := d.deps.NotificationRepo.Insert(ctx, record); insertErr != nil {
		d.deps.Log.Errorw("notification: failed to insert push record",
			"event_id", env.EventID, "error", insertErr)
		return
	}

	result, sendErr := d.deps.PushProvider.SendPush(ctx, &notifDomain.PushMessage{
		DeviceToken: contact.PushToken,
		Title:       title,
		Body:        body,
		Data:        data,
	})

	if sendErr != nil {
		record.MarkFailed(sendErr.Error())
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Failed.Add(1)
		d.deps.Log.Errorw("notification: push send failed",
			"event_id", env.EventID, "error", sendErr)
		return
	}

	if result.Success {
		record.MarkSent(result.MessageID)
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Sent.Add(1)
		d.deps.Log.Infow("notification: push sent",
			"event_id", env.EventID, "provider_ref", result.MessageID)
	} else {
		record.MarkFailed(result.Error)
		_ = d.deps.NotificationRepo.UpdateStatus(ctx, record)
		d.metrics.Failed.Add(1)
	}
}

// formatAmount formats an amount + currency into a human-readable string.
// BDT is stored in whole taka (no minor unit); other currencies are in cents.
func formatAmount(cents int, currency string) string {
	switch currency {
	case "BDT":
		return fmt.Sprintf("BDT %d", cents)
	case "USD", "EUR", "GBP", "AUD", "CAD":
		return fmt.Sprintf("$%d.%02d %s", cents/100, cents%100, currency)
	default:
		return fmt.Sprintf("%d.%02d %s", cents/100, cents%100, currency)
	}
}

// now is a variable for testing (can be overridden).
var now = time.Now
