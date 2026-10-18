package nats

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestClient_KV_NotInitialized(t *testing.T) {
	c := &Client{kvStores: make(map[string]jetstream.KeyValue)}
	_, err := c.KV("NONEXISTENT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestClient_KVHealthCheck_EmptyRequired(t *testing.T) {
	c := &Client{kvStores: make(map[string]jetstream.KeyValue)}
	err := c.KVHealthCheck(nil, []string{})
	assert.NoError(t, err)
}

func TestClient_KVHealthCheck_MissingBucket(t *testing.T) {
	c := &Client{kvStores: make(map[string]jetstream.KeyValue)}
	err := c.KVHealthCheck(nil, []string{"MISSING_BUCKET"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New(context.Background(), "nats://invalid-host:99999", "test-cluster", 1, 1, 1)
	assert.Error(t, err)
}
