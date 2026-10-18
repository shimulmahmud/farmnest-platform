package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceOperation_NilMetrics(t *testing.T) {
	err := TraceOperation(context.Background(), nil, "test", "insertOne", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestMongoDBMetrics_TraceOperation_NilReceiver(t *testing.T) {
	var m *MongoDBMetrics
	err := m.TraceOperation(context.Background(), nil, "test", "find", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestTraceMongoOperationMetricsOnly_NilMetrics(t *testing.T) {
	err := TraceMongoOperationMetricsOnly(context.Background(), nil, "test", "find", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestNewInstrumentedCollection_NilTracerAndMetrics(t *testing.T) {
	// Should not panic — returns a usable wrapper that delegates directly.
	ic := NewInstrumentedCollection(nil, nil, nil)
	assert.Nil(t, ic.Collection())
	assert.Equal(t, "", ic.Name())
}

// --- Secret sanitization tests ---

func TestSanitizeErrorMessage_BearerToken(t *testing.T) {
	msg := sanitizeErrorMessage("connection refused: Bearer eyJhbGciOiJIUzI1NiIs")
	assert.Equal(t, "[REDACTED: potential secret in error message]", msg)
}

func TestSanitizeErrorMessage_Password(t *testing.T) {
	msg := sanitizeErrorMessage("auth failed: password=mypassword123")
	assert.Equal(t, "[REDACTED: potential secret in error message]", msg)
}

func TestSanitizeErrorMessage_SafeMessage(t *testing.T) {
	msg := sanitizeErrorMessage("connection refused")
	assert.Equal(t, "connection refused", msg)
}

func TestSanitizeErrorMessage_Secret(t *testing.T) {
	msg := sanitizeErrorMessage("config error: secret=abc123")
	assert.Equal(t, "[REDACTED: potential secret in error message]", msg)
}

func TestSanitizeErrorMessage_Token(t *testing.T) {
	msg := sanitizeErrorMessage("invalid token=abc")
	assert.Equal(t, "[REDACTED: potential secret in error message]", msg)
}

func TestSanitizeErrorMessage_ApiKey(t *testing.T) {
	msg := sanitizeErrorMessage("unauthorized: api_key=sk-12345")
	assert.Equal(t, "[REDACTED: potential secret in error message]", msg)
}

func TestSanitizeErrorMessage_ApiKeyLower(t *testing.T) {
	msg := sanitizeErrorMessage("unauthorized: apikey=sk-12345")
	assert.Equal(t, "[REDACTED: potential secret in error message]", msg)
}

func TestSanitizeErrorMessage_MultipleSecrets(t *testing.T) {
	msg := sanitizeErrorMessage("Bearer token")
	assert.Equal(t, "[REDACTED: potential secret in error message]", msg)
}

// --- FilterAttributes tests ---

func TestFilterAttributes_OnlySafeKeys(t *testing.T) {
	attrs := map[string]string{
		"http.method":       "GET",
		"http.route":        "/api/v1/orders",
		"mongodb.collection": "orders",
		"nats.subject":      "ECOMMERCE_ORDERS.order.created",
		"request_id":        "abc-123",
		"authorization":     "Bearer secret", // should be dropped
		"password":          "my-password",  // should be dropped
	}

	result := FilterAttributes(attrs)
	assert.Len(t, result, 5)
}

func TestFilterAttributes_Empty(t *testing.T) {
	result := FilterAttributes(map[string]string{})
	assert.Empty(t, result)
}

func TestFilterAttributes_NoSafeKeys(t *testing.T) {
	result := FilterAttributes(map[string]string{
		"authorization": "Bearer x",
		"cookie":        "session=abc",
	})
	assert.Empty(t, result)
}

// --- Context helper tests ---

func TestCorrelationIDFromContext_Empty(t *testing.T) {
	id := CorrelationIDFromContext(context.Background())
	assert.Empty(t, id)
}

func TestCausationIDFromContext_Empty(t *testing.T) {
	id := CausationIDFromContext(context.Background())
	assert.Empty(t, id)
}

func TestRecordSpanError_NilSpan(t *testing.T) {
	// Should not panic.
	RecordSpanError(nil, nil)
}

func TestStartSpan_NilTracer(t *testing.T) {
	ctx, span := StartSpan(context.Background(), nil, "test")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestContains(t *testing.T) {
	assert.True(t, contains("hello world", "world"))
	assert.True(t, contains("hello world", "hello"))
	assert.False(t, contains("hello world", "xyz"))
	assert.False(t, contains("", "a"))
	assert.True(t, contains("", ""))
}
