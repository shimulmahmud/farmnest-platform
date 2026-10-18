package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps NATS connection, JetStream, and KV management.
type Client struct {
	conn     *nats.Conn
	js       jetstream.JetStream
	kvStores map[string]jetstream.KeyValue
}

// New connects to NATS and initializes JetStream.
func New(ctx context.Context, url string, clusterName string, reconnectWait time.Duration, maxReconnects int, connectTimeout time.Duration) (*Client, error) {
	opts := []nats.Option{
		nats.ReconnectWait(reconnectWait),
		nats.MaxReconnects(maxReconnects),
		nats.Timeout(connectTimeout),
		nats.Name(clusterName + "-client"),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create jetstream: %w", err)
	}

	return &Client{
		conn:     conn,
		js:       js,
		kvStores: make(map[string]jetstream.KeyValue),
	}, nil
}

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *nats.Conn {
	return c.conn
}

// JS returns the JetStream interface.
func (c *Client) JS() jetstream.JetStream {
	return c.js
}

// InitKVBucket creates (or retrieves) a KV bucket with the given config.
func (c *Client) InitKVBucket(ctx context.Context, bucketName string, history uint8) (jetstream.KeyValue, error) {
	kv, err := c.js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:  bucketName,
		History: history,
	})
	if err != nil {
		// Bucket may already exist — try to load it
		kv, err = c.js.KeyValue(ctx, bucketName)
		if err != nil {
			return nil, fmt.Errorf("failed to create/load KV bucket %q: %w", bucketName, err)
		}
	}

	c.kvStores[bucketName] = kv
	return kv, nil
}

// KV returns a previously initialized KV bucket by name.
func (c *Client) KV(bucketName string) (jetstream.KeyValue, error) {
	kv, ok := c.kvStores[bucketName]
	if !ok {
		return nil, fmt.Errorf("KV bucket %q not initialized", bucketName)
	}
	return kv, nil
}

// HealthCheck verifies NATS connectivity.
func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.conn.IsConnected() {
		return fmt.Errorf("nats: not connected")
	}
	return nil
}

// JetStreamHealthCheck verifies JetStream is operational.
func (c *Client) JetStreamHealthCheck(ctx context.Context) error {
	if !c.conn.IsConnected() {
		return fmt.Errorf("nats: not connected")
	}
	// Verify JetStream by listing streams (lightweight check)
	lister := c.js.StreamNames(ctx)
	// Drain the name channel to trigger any fetch errors
	for range lister.Name() {
	}
	if err := lister.Err(); err != nil {
		return fmt.Errorf("jetstream health check failed: %w", err)
	}
	return nil
}

// KVHealthCheck verifies that all required KV buckets are available.
func (c *Client) KVHealthCheck(ctx context.Context, requiredBuckets []string) error {
	for _, name := range requiredBuckets {
		kv, ok := c.kvStores[name]
		if !ok {
			return fmt.Errorf("KV bucket %q not initialized", name)
		}
		status, err := kv.Status(ctx)
		if err != nil {
			return fmt.Errorf("KV bucket %q status check failed: %w", name, err)
		}
		if status == nil {
			return fmt.Errorf("KV bucket %q returned nil status", name)
		}
	}
	return nil
}

// Close closes the NATS connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
