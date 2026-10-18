package eventstore

import (
        "encoding/json"
        "testing"
        "time"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Envelope Creation ---

func TestNewEnvelope_GeneratesFields(t *testing.T) {
        payload := map[string]string{"order_id": "ord-123"}
        env, err := NewEnvelope("order.created", "ord-123", "order", 1, payload)
        require.NoError(t, err)

        assert.NotEmpty(t, env.EventID)
        assert.Equal(t, "order.created", env.EventType)
        assert.Equal(t, 1, env.Version)
        assert.Equal(t, "ord-123", env.AggregateID)
        assert.Equal(t, "order", env.AggregateType)
        assert.False(t, env.OccurredAt.IsZero())
        assert.NotEmpty(t, env.CorrelationID)
        assert.Empty(t, env.CausationID)
        assert.NotNil(t, env.Payload)
}

func TestNewEnvelopeWithCorrelation_SetsIDs(t *testing.T) {
        payload := map[string]string{"payment_id": "pay-1"}
        env, err := NewEnvelopeWithCorrelation(
                "payment.completed", "pay-1", "payment", 1, payload,
                "corr-abc", "cause-xyz",
        )
        require.NoError(t, err)

        assert.Equal(t, "corr-abc", env.CorrelationID)
        assert.Equal(t, "cause-xyz", env.CausationID)
}

func TestNewEnvelopeWithCorrelation_GeneratesCorrelationIfEmpty(t *testing.T) {
        payload := map[string]string{}
        env, err := NewEnvelopeWithCorrelation("order.paid", "ord-1", "order", 1, payload, "", "")
        require.NoError(t, err)
        assert.NotEmpty(t, env.CorrelationID)
}

func TestNewChildEnvelope_InheritsIDs(t *testing.T) {
        parent, err := NewEnvelope("order.created", "ord-1", "order", 1, map[string]string{})
        require.NoError(t, err)

        child, err := NewChildEnvelope(parent, "order.paid", "ord-1", "order", 1, map[string]string{})
        require.NoError(t, err)

        assert.Equal(t, parent.CorrelationID, child.CorrelationID)
        assert.Equal(t, parent.EventID, child.CausationID)
        assert.NotEqual(t, parent.EventID, child.EventID)
}

func TestNewEnvelope_BadPayload_ReturnsError(t *testing.T) {
        // Channels cannot be marshaled to JSON.
        _, err := NewEnvelope("test", "id", "type", 1, make(chan int))
        assert.Error(t, err)
}

// --- DecodePayload ---

func TestDecodePayload_Struct(t *testing.T) {
        type OrderData struct {
                OrderID  string `json:"order_id"`
                CustomerID string `json:"customer_id"`
        }

        original := OrderData{OrderID: "ord-1", CustomerID: "cust-1"}
        env, err := NewEnvelope("order.created", "ord-1", "order", 1, original)
        require.NoError(t, err)

        var decoded OrderData
        err = env.DecodePayload(&decoded)
        require.NoError(t, err)
        assert.Equal(t, original.OrderID, decoded.OrderID)
        assert.Equal(t, original.CustomerID, decoded.CustomerID)
}

// --- Subject ---

func TestSubject_Order(t *testing.T) {
        env, _ := NewEnvelope("order.created", "ord-1", "order", 1, map[string]string{})
        assert.Equal(t, "ECOMMERCE_ORDERS.order.created", env.Subject())
}

func TestSubject_Payment(t *testing.T) {
        env, _ := NewEnvelope("payment.completed", "pay-1", "payment", 1, map[string]string{})
        assert.Equal(t, "ECOMMERCE_PAYMENTS.payment.completed", env.Subject())
}

func TestSubject_Inventory(t *testing.T) {
        env, _ := NewEnvelope("inventory.reserved", "sku-1", "inventory", 1, map[string]string{})
        assert.Equal(t, "ECOMMERCE_INVENTORY.inventory.reserved", env.Subject())
}

func TestSubject_Product(t *testing.T) {
        env, _ := NewEnvelope("product.created", "prod-1", "product", 1, map[string]string{})
        assert.Equal(t, "ECOMMERCE_PRODUCTS.product.created", env.Subject())
}

func TestSubject_Notification(t *testing.T) {
        env, _ := NewEnvelope("notification.email", "n-1", "notification", 1, map[string]string{})
        assert.Equal(t, "ECOMMERCE_NOTIFICATIONS.notification.email", env.Subject())
}

func TestSubject_Analytics(t *testing.T) {
        env, _ := NewEnvelope("analytics.order_completed", "a-1", "analytics", 1, map[string]string{})
        assert.Equal(t, "ECOMMERCE_ANALYTICS.analytics.order_completed", env.Subject())
}

func TestSubject_Unknown(t *testing.T) {
        env, _ := NewEnvelope("test.event", "id", "unknown_type", 1, map[string]string{})
        assert.Equal(t, "ECOMMERCE_UNKNOWN.test.event", env.Subject())
}

// --- Serialize / Deserialize ---

func TestSerialize_Deserialize_RoundTrip(t *testing.T) {
        payload := map[string]interface{}{"order_id": "ord-1", "total": 5000}
        original, err := NewEnvelope("order.created", "ord-1", "order", 1, payload)
        require.NoError(t, err)

        data, err := original.Serialize()
        require.NoError(t, err)

        decoded, err := DeserializeEnvelope(data)
        require.NoError(t, err)

        assert.Equal(t, original.EventID, decoded.EventID)
        assert.Equal(t, original.EventType, decoded.EventType)
        assert.Equal(t, original.Version, decoded.Version)
        assert.Equal(t, original.AggregateID, decoded.AggregateID)
        assert.Equal(t, original.AggregateType, decoded.AggregateType)
        assert.Equal(t, original.CorrelationID, decoded.CorrelationID)
        assert.Equal(t, original.CausationID, decoded.CausationID)
        assert.Equal(t, original.Payload, decoded.Payload)
        // Timestamps should match within 1 second.
        assert.WithinDuration(t, original.OccurredAt, decoded.OccurredAt, time.Second)
}

func TestDeserialize_InvalidJSON_ReturnsError(t *testing.T) {
        _, err := DeserializeEnvelope([]byte("not json"))
        assert.Error(t, err)
}

// --- AggregateToStream ---

func TestAggregateToStream(t *testing.T) {
        tests := []struct{
                aggregate string
                expected  string
        }{
                {"order", StreamOrders},
                {"payment", StreamPayments},
                {"inventory", StreamInventory},
                {"product", StreamProducts},
                {"notification", StreamNotifications},
                {"analytics", StreamAnalytics},
                {"unknown", "ECOMMERCE_UNKNOWN"},
        }

        for _, tc := range tests {
                t.Run(tc.aggregate, func(t *testing.T) {
                        assert.Equal(t, tc.expected, AggregateToStream(tc.aggregate))
                })
        }
}

// --- DefaultStreamConfigs ---

func TestDefaultStreamConfigs_SevenStreams(t *testing.T) {
        configs := DefaultStreamConfigs()
        require.Len(t, configs, 7)

        names := make(map[string]bool)
        for _, cfg := range configs {
                names[cfg.Name] = true
                assert.NotEmpty(t, cfg.Subjects)
                assert.True(t, cfg.Retention > 0)
        }

        assert.True(t, names[StreamOrders])
        assert.True(t, names[StreamPayments])
        assert.True(t, names[StreamInventory])
        assert.True(t, names[StreamProducts])
        assert.True(t, names[StreamNotifications])
        assert.True(t, names[StreamAnalytics])
        assert.True(t, names[StreamShipping])
}

func TestDefaultStreamConfigs_OrdersSubjects(t *testing.T) {
        configs := DefaultStreamConfigs()
        ordersCfg := configs[0]
        assert.Equal(t, StreamOrders, ordersCfg.Name)
        assert.ElementsMatch(t, []string{
                "ECOMMERCE_ORDERS.order.created",
                "ECOMMERCE_ORDERS.order.paid",
                "ECOMMERCE_ORDERS.order.cancelled",
                "ECOMMERCE_ORDERS.order.shipped",
                "ECOMMERCE_ORDERS.order.delivered",
        }, ordersCfg.Subjects)
}

// --- DefaultBackoff ---

func TestDefaultBackoff_TenSteps(t *testing.T) {
        bo := DefaultBackoff()
        require.Len(t, bo, 10)
        assert.Equal(t, 1*time.Second, bo[0])
        assert.Equal(t, 5*time.Minute, bo[9])
}

// --- Envelope JSON Structure ---

func TestEnvelopeJSON_AllFieldsPresent(t *testing.T) {
        env, err := NewEnvelopeWithCorrelation(
                "order.paid", "ord-1", "order", 2,
                map[string]string{"customer_id": "c-1"},
                "corr-id", "cause-id",
        )
        require.NoError(t, err)

        data, err := json.Marshal(env)
        require.NoError(t, err)

        var raw map[string]interface{}
        require.NoError(t, json.Unmarshal(data, &raw))

        assert.Contains(t, raw, "event_id")
        assert.Contains(t, raw, "event_type")
        assert.Contains(t, raw, "version")
        assert.Contains(t, raw, "aggregate_id")
        assert.Contains(t, raw, "aggregate_type")
        assert.Contains(t, raw, "occurred_at")
        assert.Contains(t, raw, "correlation_id")
        assert.Contains(t, raw, "causation_id")
        assert.Contains(t, raw, "payload")
}
