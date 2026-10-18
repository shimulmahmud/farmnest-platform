package mongodb

import (
        "context"
        "fmt"
        "time"

        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client wraps a mongo.Client with health check capability.
type Client struct {
        client   *mongo.Client
        database string
}

// New creates a new MongoDB client and connects to the server.
func New(ctx context.Context, uri, database string, maxPoolSize, minPoolSize uint64, connectTimeout time.Duration) (*Client, error) {
        opts := options.Client().
                ApplyURI(uri).
                SetMaxPoolSize(maxPoolSize).
                SetMinPoolSize(minPoolSize).
                SetConnectTimeout(connectTimeout)

        client, err := mongo.Connect(opts)
        if err != nil {
                return nil, fmt.Errorf("failed to create mongo client: %w", err)
        }

        // Verify connection
        pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
        defer cancel()
        if err := client.Ping(pingCtx, nil); err != nil {
                return nil, fmt.Errorf("failed to ping mongodb: %w", err)
        }

        return &Client{client: client, database: database}, nil
}

// Database returns the mongo.Database instance.
func (c *Client) Database() *mongo.Database {
        return c.client.Database(c.database)
}

// Collection returns a mongo.Collection for the given name.
func (c *Client) Collection(name string) *mongo.Collection {
        return c.Database().Collection(name)
}

// HealthCheck verifies the MongoDB connection is alive.
func (c *Client) HealthCheck(ctx context.Context) error {
        pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        return c.client.Ping(pingCtx, nil)
}

// Client returns the underlying *mongo.Client for advanced operations
// such as sessions and transactions.
func (c *Client) Client() *mongo.Client {
        return c.client
}

// Close disconnects the MongoDB client.
func (c *Client) Close(ctx context.Context) error {
        return c.client.Disconnect(ctx)
}

// Ping sends a ping command to verify connectivity.
func Ping(ctx context.Context, c *Client) error {
        result := c.Database().RunCommand(ctx, bson.D{{Key: "ping", Value: 1}})
        if result.Err() != nil {
                return fmt.Errorf("mongodb ping failed: %w", result.Err())
        }
        return nil
}
