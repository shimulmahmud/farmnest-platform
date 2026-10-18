package eventstore

import (
        "context"
        "fmt"
        "time"

        "github.com/nats-io/nats.go/jetstream"
)

// StreamConfig defines a JetStream stream configuration.
type StreamConfig struct {
        Name     string
        Subjects []string // subject patterns this stream listens to
        // Retention: how long messages are kept.
        // Default: 7 days. Set to 0 for unlimited.
        Retention time.Duration
        // MaxMsgsPerSubject limits stored messages per subject.
        // 0 means unlimited.
        MaxMsgsPerSubject int64
        // Replicas is the number of stream replicas for HA.
        Replicas int
}

// DefaultStreamConfigs returns the 6 required event streams.
//
// Retention policy:
//   - ECOMMERCE_ORDERS:    30 days (order audit trail)
//   - ECOMMERCE_PAYMENTS: 30 days (payment audit trail)
//   - ECOMMERCE_INVENTORY: 7 days (high volume, short-lived)
//   - ECOMMERCE_PRODUCTS:  30 days (product change history)
//   - ECOMMERCE_NOTIFICATIONS: 3 days (transient, high volume)
//   - ECOMMERCE_ANALYTICS: 7 days (aggregated downstream)
func DefaultStreamConfigs() []StreamConfig {
        return []StreamConfig{
                {
                        Name: StreamOrders,
                        Subjects: []string{
                                "ECOMMERCE_ORDERS.order.created",
                                "ECOMMERCE_ORDERS.order.paid",
                                "ECOMMERCE_ORDERS.order.cancelled",
                                "ECOMMERCE_ORDERS.order.shipped",
                                "ECOMMERCE_ORDERS.order.delivered",
                        },
                        Retention:          30 * 24 * time.Hour,
                        MaxMsgsPerSubject: 0,
                        Replicas:          1,
                },
                {
                        Name: StreamPayments,
                        Subjects: []string{
                                "ECOMMERCE_PAYMENTS.payment.created",
                                "ECOMMERCE_PAYMENTS.payment.completed",
                                "ECOMMERCE_PAYMENTS.payment.failed",
                                "ECOMMERCE_PAYMENTS.payment.refunded",
                        },
                        Retention:          30 * 24 * time.Hour,
                        MaxMsgsPerSubject: 0,
                        Replicas:          1,
                },
                {
                        Name: StreamInventory,
                        Subjects: []string{
                                "ECOMMERCE_INVENTORY.inventory.reserved",
                                "ECOMMERCE_INVENTORY.inventory.released",
                                "ECOMMERCE_INVENTORY.inventory.low_stock",
                        },
                        Retention:          7 * 24 * time.Hour,
                        MaxMsgsPerSubject: 100000,
                        Replicas:          1,
                },
                {
                        Name: StreamProducts,
                        Subjects: []string{
                                "ECOMMERCE_PRODUCTS.product.created",
                                "ECOMMERCE_PRODUCTS.product.updated",
                                "ECOMMERCE_PRODUCTS.product.deleted",
                        },
                        Retention:          30 * 24 * time.Hour,
                        MaxMsgsPerSubject: 0,
                        Replicas:          1,
                },
                {
                        Name: StreamNotifications,
                        Subjects: []string{
                                "ECOMMERCE_NOTIFICATIONS.notification.email",
                                "ECOMMERCE_NOTIFICATIONS.notification.sms",
                        },
                        Retention:          3 * 24 * time.Hour,
                        MaxMsgsPerSubject: 50000,
                        Replicas:          1,
                },
                {
                        Name: StreamAnalytics,
                        Subjects: []string{
                                "ECOMMERCE_ANALYTICS.analytics.product_viewed",
                                "ECOMMERCE_ANALYTICS.analytics.cart_created",
                                "ECOMMERCE_ANALYTICS.analytics.cart_abandoned",
                                "ECOMMERCE_ANALYTICS.analytics.checkout_started",
                                "ECOMMERCE_ANALYTICS.analytics.order_completed",
                                "ECOMMERCE_ANALYTICS.product.viewed",
                                "ECOMMERCE_ANALYTICS.cart.created",
                                "ECOMMERCE_ANALYTICS.checkout.started",
                                "ECOMMERCE_ANALYTICS.order.completed",
                                "ECOMMERCE_ANALYTICS.payment.completed",
                        },
                        Retention:          7 * 24 * time.Hour,
                        MaxMsgsPerSubject: 500000,
                        Replicas:          1,
                },
                {
                        Name: StreamShipping,
                        Subjects: []string{
                                "ECOMMERCE_SHIPPING.shipment.created",
                                "ECOMMERCE_SHIPPING.shipment.shipped",
                                "ECOMMERCE_SHIPPING.shipment.delivered",
                        },
                        Retention:          14 * 24 * time.Hour,
                        MaxMsgsPerSubject: 100000,
                        Replicas:          1,
                },
        }
}

// Stream name constants.
const (
        StreamOrders        = "ECOMMERCE_ORDERS"
        StreamPayments      = "ECOMMERCE_PAYMENTS"
        StreamInventory     = "ECOMMERCE_INVENTORY"
        StreamProducts      = "ECOMMERCE_PRODUCTS"
        StreamNotifications = "ECOMMERCE_NOTIFICATIONS"
        StreamAnalytics     = "ECOMMERCE_ANALYTICS"
        StreamShipping      = "ECOMMERCE_SHIPPING"
)

// AggregateToStream maps an aggregate type to its stream name.
func AggregateToStream(aggregateType string) string {
        switch aggregateType {
        case "order":
                return StreamOrders
        case "payment":
                return StreamPayments
        case "inventory":
                return StreamInventory
        case "product":
                return StreamProducts
        case "notification":
                return StreamNotifications
        case "analytics":
                return StreamAnalytics
        case "shipment":
                return StreamShipping
        default:
                return "ECOMMERCE_UNKNOWN"
        }
}

// StreamManager creates and manages JetStream streams.
type StreamManager struct {
        js jetstream.JetStream
}

// NewStreamManager creates a StreamManager.
func NewStreamManager(js jetstream.JetStream) *StreamManager {
        return &StreamManager{js: js}
}

// EnsureStreams creates all configured streams if they don't exist.
// If a stream already exists, it is left unchanged.
func (m *StreamManager) EnsureStreams(ctx context.Context, configs []StreamConfig) error {
        for _, cfg := range configs {
                if err := m.ensureStream(ctx, cfg); err != nil {
                        return fmt.Errorf("failed to ensure stream %q: %w", cfg.Name, err)
                }
        }
        return nil
}

func (m *StreamManager) ensureStream(ctx context.Context, cfg StreamConfig) error {
        streamCfg := jetstream.StreamConfig{
                Name:      cfg.Name,
                Subjects:  cfg.Subjects,
                Retention: jetstream.LimitsPolicy,
        }

        if cfg.Retention > 0 {
                streamCfg.MaxAge = cfg.Retention
        }
        if cfg.MaxMsgsPerSubject > 0 {
                streamCfg.MaxMsgsPerSubject = cfg.MaxMsgsPerSubject
        }
        if cfg.Replicas > 0 {
                streamCfg.Replicas = cfg.Replicas
        }

        _, err := m.js.CreateOrUpdateStream(ctx, streamCfg)
        return err
}

// ConsumerConfig defines a durable consumer configuration.
type ConsumerConfig struct {
        Stream      string // stream to consume from
        Name        string // durable consumer name
        Description string // human-readable description
        // FilterSubject limits the consumer to a specific subject within the stream.
        // Empty means all subjects in the stream.
        FilterSubject string
        // AckPolicy controls when messages are acknowledged.
        // Defaults to explicit ack.
        AckExplicit bool
        // MaxDeliveries is the max number of redeliveries before the message is NAKed permanently.
        // 0 means unlimited. Default: 10.
        MaxDeliveries int
        // Backoff is the retry backoff durations. Defaults to exponential backoff.
        Backoff []time.Duration
}

// EnsureConsumer creates a durable consumer if it doesn't exist.
func (m *StreamManager) EnsureConsumer(ctx context.Context, cfg ConsumerConfig) (jetstream.Consumer, error) {
        consCfg := jetstream.ConsumerConfig{
                Name:              cfg.Name,
                Durable:           cfg.Name,
                Description:       cfg.Description,
                AckPolicy:         jetstream.AckExplicitPolicy,
                MaxDeliver:        cfg.MaxDeliveries,
        }

        if cfg.FilterSubject != "" {
                consCfg.FilterSubject = cfg.FilterSubject
        }

        if cfg.MaxDeliveries == 0 {
                consCfg.MaxDeliver = 10
        }

        if len(cfg.Backoff) > 0 {
                consCfg.BackOff = cfg.Backoff
        } else {
                // Default exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 300s
                consCfg.BackOff = []time.Duration{
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
        }

        return m.js.CreateOrUpdateConsumer(ctx, cfg.Stream, consCfg)
}
