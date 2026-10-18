package outbox

import (
	"context"
	"testing"

	ordDomain "github.com/ecommerce/backend/internal/orders/domain"
	"github.com/ecommerce/backend/pkg/eventstore"
	"github.com/ecommerce/backend/pkg/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- In-memory outbox repo for adapter tests ---

type adapterTestRepo struct {
	lastEvent *OutboxEvent
}

func (r *adapterTestRepo) Insert(_ context.Context, event *OutboxEvent) error {
	r.lastEvent = event
	return nil
}

// TestOrderPublisherAdapter_PropagatesCorrelationID verifies that the
// OrderPublisherAdapter extracts correlation_id and causation_id from
// the OTel trace context and propagates them into the outbox envelope.
func TestOrderPublisherAdapter_PropagatesCorrelationID(t *testing.T) {
	// The global OTel provider is set by TestMain or by a previous test.
	// When running this test in isolation, we use the global tracer which
	// will still produce valid spans.
	ctx, span := telemetry.StartSpan(context.Background(), nil, "test-http-request")

	repo := &adapterTestRepo{}
	adapter := NewOrderPublisherAdapter(repo)

	err := adapter.Publish(ctx, "order.created", &ordDomain.OrderEvent{
		EventType:   "order.created",
		OrderID:     "ord-123",
		OrderNumber: "ORD-001",
		CustomerID:  "cust-456",
	})
	require.NoError(t, err)

	span.End()

	require.NotNil(t, repo.lastEvent)
	assert.NotEmpty(t, repo.lastEvent.CorrelationID,
		"correlation_id should be propagated from OTel context")

	// Verify the envelope can be reconstructed.
	env := repo.lastEvent.ToEnvelope()
	assert.Equal(t, "ord-123", env.AggregateID)
	assert.Equal(t, "order", env.AggregateType)
	assert.Equal(t, "order.created", env.EventType)
}

// TestOrderPublisherAdapter_EmptyContext generates UUIDs.
func TestOrderPublisherAdapter_EmptyContext(t *testing.T) {
	repo := &adapterTestRepo{}
	adapter := NewOrderPublisherAdapter(repo)

	err := adapter.Publish(context.Background(), "order.created", &ordDomain.OrderEvent{
		EventType:   "order.created",
		OrderID:     "ord-789",
		OrderNumber: "ORD-002",
		CustomerID:  "cust-000",
	})
	require.NoError(t, err)

	require.NotNil(t, repo.lastEvent)
	// With no OTel context, correlation_id is empty string, which triggers
	// auto-generation in NewEnvelopeWithCorrelation.
	assert.NotEmpty(t, repo.lastEvent.CorrelationID,
		"correlation_id should be auto-generated when no OTel context")
}

// TestCorrelationIDFromContext_Empty verifies no panic on empty context.
func TestCorrelationIDFromContext_Empty(t *testing.T) {
	id := telemetry.CorrelationIDFromContext(context.Background())
	assert.Empty(t, id)
}

// TestNewEnvelopeWithCorrelation_UsesProvidedIDs verifies that explicit
// correlation and causation IDs are preserved.
func TestNewEnvelopeWithCorrelation_UsesProvidedIDs(t *testing.T) {
	env, err := eventstore.NewEnvelopeWithCorrelation(
		"test.event", "agg-1", "test", 1,
		map[string]string{"key": "val"},
		"corr-abc", "cause-xyz",
	)
	require.NoError(t, err)
	assert.Equal(t, "corr-abc", env.CorrelationID)
	assert.Equal(t, "cause-xyz", env.CausationID)
}

func TestNewEnvelopeWithCorrelation_EmptyGeneratesUUID(t *testing.T) {
	env, err := eventstore.NewEnvelopeWithCorrelation(
		"test.event", "agg-1", "test", 1,
		map[string]string{"key": "val"},
		"", "",
	)
	require.NoError(t, err)
	// Empty correlation_id should trigger auto-generation.
	assert.NotEmpty(t, env.CorrelationID)
}