package telemetry

import (
        "context"

        "go.opentelemetry.io/otel/attribute"
        "go.opentelemetry.io/otel/metric"
)

// HTTPMetrics holds all HTTP-related Prometheus instruments.
type HTTPMetrics struct {
        RequestCount   metric.Int64Counter
        RequestErrors  metric.Int64Counter
        RequestLatency metric.Float64Histogram
}

// NewHTTPMetrics registers HTTP instruments on the global meter.
func NewHTTPMetrics(meter metric.Meter) (*HTTPMetrics, error) {
        requestCount, err := meter.Int64Counter(
                "http.server.request.count",
                metric.WithDescription("Total number of HTTP requests"),
                metric.WithUnit("{request}"),
        )
        if err != nil {
                return nil, err
        }

        requestErrors, err := meter.Int64Counter(
                "http.server.request.errors",
                metric.WithDescription("Total number of HTTP requests that resulted in errors (5xx)"),
                metric.WithUnit("{error}"),
        )
        if err != nil {
                return nil, err
        }

        requestLatency, err := meter.Float64Histogram(
                "http.server.request.duration",
                metric.WithDescription("HTTP request duration in seconds"),
                metric.WithUnit("s"),
        )
        if err != nil {
                return nil, err
        }

        return &HTTPMetrics{
                RequestCount:   requestCount,
                RequestErrors:  requestErrors,
                RequestLatency: requestLatency,
        }, nil
}

// RecordRequest records a single HTTP request's metrics.
// No secrets are exposed: method and path are safe to emit.
func (m *HTTPMetrics) RecordRequest(method, path, status string, durationSeconds float64) {
        commonAttrs := metric.WithAttributes(
                attribute.String("http.method", method),
                attribute.String("http.route", path),
                attribute.String("http.status_code", status),
        )
        m.RequestCount.Add(context.Background(), 1, commonAttrs)
        m.RequestLatency.Record(context.Background(), durationSeconds, commonAttrs)

        statusCode := status
        if len(statusCode) >= 1 && statusCode[0] == '5' {
                m.RequestErrors.Add(context.Background(), 1, commonAttrs)
        }
}

// MongoDBMetrics holds MongoDB operation instruments.
type MongoDBMetrics struct {
        OperationDuration metric.Float64Histogram
        OperationErrors   metric.Int64Counter
}

// NewMongoDBMetrics registers MongoDB instruments.
func NewMongoDBMetrics(meter metric.Meter) (*MongoDBMetrics, error) {
        opDuration, err := meter.Float64Histogram(
                "mongodb.operation.duration",
                metric.WithDescription("MongoDB operation duration in seconds"),
                metric.WithUnit("s"),
        )
        if err != nil {
                return nil, err
        }

        opErrors, err := meter.Int64Counter(
                "mongodb.operation.errors",
                metric.WithDescription("Total number of MongoDB operation errors"),
                metric.WithUnit("{error}"),
        )
        if err != nil {
                return nil, err
        }

        return &MongoDBMetrics{
                OperationDuration: opDuration,
                OperationErrors:   opErrors,
        }, nil
}

// RecordOperation records a MongoDB operation. Collection is safe (no secrets).
func (m *MongoDBMetrics) RecordOperation(collection, operation string, durationSeconds float64, err error) {
        attrs := metric.WithAttributes(
                attribute.String("mongodb.collection", collection),
                attribute.String("mongodb.operation", operation),
        )
        m.OperationDuration.Record(context.Background(), durationSeconds, attrs)
        if err != nil {
                m.OperationErrors.Add(context.Background(), 1, attrs)
        }
}

// NATSMetrics holds NATS publish instruments.
type NATSMetrics struct {
        PublishDuration  metric.Float64Histogram
        PublishErrors    metric.Int64Counter
        ConsumerLag      metric.Float64Histogram
        Redeliveries     metric.Int64Counter
}

// NewNATSMetrics registers NATS instruments.
func NewNATSMetrics(meter metric.Meter) (*NATSMetrics, error) {
        pubDuration, err := meter.Float64Histogram(
                "nats.publish.duration",
                metric.WithDescription("NATS publish duration in seconds"),
                metric.WithUnit("s"),
        )
        if err != nil {
                return nil, err
        }

        pubErrors, err := meter.Int64Counter(
                "nats.publish.errors",
                metric.WithDescription("Total number of NATS publish errors"),
                metric.WithUnit("{error}"),
        )
        if err != nil {
                return nil, err
        }

        consumerLag, err := meter.Float64Histogram(
                "nats.consumer.lag",
                metric.WithDescription("NATS consumer processing lag in seconds"),
                metric.WithUnit("s"),
        )
        if err != nil {
                return nil, err
        }

        redeliveries, err := meter.Int64Counter(
                "nats.consumer.redeliveries",
                metric.WithDescription("Total number of NATS message redeliveries"),
                metric.WithUnit("{redelivery}"),
        )
        if err != nil {
                return nil, err
        }

        return &NATSMetrics{
                PublishDuration: pubDuration,
                PublishErrors:   pubErrors,
                ConsumerLag:   consumerLag,
                Redeliveries:    redeliveries,
        }, nil
}

// RecordPublish records a NATS publish operation. Subject is safe (no secrets).
func (m *NATSMetrics) RecordPublish(subject string, durationSeconds float64, err error) {
        attrs := metric.WithAttributes(
                attribute.String("nats.subject", subject),
        )
        m.PublishDuration.Record(context.Background(), durationSeconds, attrs)
        if err != nil {
                m.PublishErrors.Add(context.Background(), 1, attrs)
        }
}

// BusinessMetrics holds domain-specific business instruments.
type BusinessMetrics struct {
        OrdersCreated           metric.Int64Counter
        CheckoutSuccess         metric.Int64Counter
        CheckoutFailure         metric.Int64Counter
        PaymentSuccess          metric.Int64Counter
        PaymentFailure          metric.Int64Counter
        InventoryReservationFail metric.Int64Counter
}

// NewBusinessMetrics registers business instruments.
func NewBusinessMetrics(meter metric.Meter) (*BusinessMetrics, error) {
        ordersCreated, err := meter.Int64Counter(
                "ecommerce.orders.created",
                metric.WithDescription("Total number of orders created"),
                metric.WithUnit("{order}"),
        )
        if err != nil {
                return nil, err
        }

        checkoutSuccess, err := meter.Int64Counter(
                "ecommerce.checkout.success",
                metric.WithDescription("Total number of successful checkouts"),
                metric.WithUnit("{checkout}"),
        )
        if err != nil {
                return nil, err
        }

        checkoutFailure, err := meter.Int64Counter(
                "ecommerce.checkout.failure",
                metric.WithDescription("Total number of failed checkouts"),
                metric.WithUnit("{checkout}"),
        )
        if err != nil {
                return nil, err
        }

        paymentSuccess, err := meter.Int64Counter(
                "ecommerce.payment.success",
                metric.WithDescription("Total number of successful payments"),
                metric.WithUnit("{payment}"),
        )
        if err != nil {
                return nil, err
        }

        paymentFailure, err := meter.Int64Counter(
                "ecommerce.payment.failure",
                metric.WithDescription("Total number of failed payments"),
                metric.WithUnit("{payment}"),
        )
        if err != nil {
                return nil, err
        }

        invResFail, err := meter.Int64Counter(
                "ecommerce.inventory.reservation_failures",
                metric.WithDescription("Total number of inventory reservation failures"),
                metric.WithUnit("{failure}"),
        )
        if err != nil {
                return nil, err
        }

        return &BusinessMetrics{
                OrdersCreated:           ordersCreated,
                CheckoutSuccess:         checkoutSuccess,
                CheckoutFailure:         checkoutFailure,
                PaymentSuccess:          paymentSuccess,
                PaymentFailure:          paymentFailure,
                InventoryReservationFail: invResFail,
        }, nil
}

// IncrementOrdersCreated increments the orders created counter.
func (m *BusinessMetrics) IncrementOrdersCreated() {
        m.OrdersCreated.Add(context.Background(), 1)
}

// IncrementCheckoutSuccess increments the checkout success counter.
func (m *BusinessMetrics) IncrementCheckoutSuccess() {
        m.CheckoutSuccess.Add(context.Background(), 1)
}

// IncrementCheckoutFailure increments the checkout failure counter.
func (m *BusinessMetrics) IncrementCheckoutFailure() {
        m.CheckoutFailure.Add(context.Background(), 1)
}

// IncrementPaymentSuccess increments the payment success counter.
func (m *BusinessMetrics) IncrementPaymentSuccess() {
        m.PaymentSuccess.Add(context.Background(), 1)
}

// IncrementPaymentFailure increments the payment failure counter.
func (m *BusinessMetrics) IncrementPaymentFailure() {
        m.PaymentFailure.Add(context.Background(), 1)
}

// IncrementInventoryReservationFailure increments the inventory reservation failure counter.
func (m *BusinessMetrics) IncrementInventoryReservationFailure() {
        m.InventoryReservationFail.Add(context.Background(), 1)
}

// AllMetrics is a convenience struct that holds all metric groups.
type AllMetrics struct {
        HTTP     *HTTPMetrics
        MongoDB  *MongoDBMetrics
        NATS     *NATSMetrics
        Business *BusinessMetrics
}

// NewAllMetrics creates all metric groups on the given meter.
func NewAllMetrics(meter metric.Meter) (*AllMetrics, error) {
        httpMetrics, err := NewHTTPMetrics(meter)
        if err != nil {
                return nil, err
        }

        mongoMetrics, err := NewMongoDBMetrics(meter)
        if err != nil {
                return nil, err
        }

        natsMetrics, err := NewNATSMetrics(meter)
        if err != nil {
                return nil, err
        }

        businessMetrics, err := NewBusinessMetrics(meter)
        if err != nil {
                return nil, err
        }

        return &AllMetrics{
                HTTP:     httpMetrics,
                MongoDB:  mongoMetrics,
                NATS:     natsMetrics,
                Business: businessMetrics,
        }, nil
}
