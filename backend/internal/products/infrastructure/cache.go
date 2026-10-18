package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/backend/internal/products/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/nats-io/nats.go/jetstream"
)

const productCacheBucket = "PRODUCT_CACHE"

type productCache struct {
	kv jetstream.KeyValue
}

// NewProductCache creates a NATS KV-backed product cache.
func NewProductCache(kv jetstream.KeyValue) domain.ProductCache {
	return &productCache{kv: kv}
}

func (c *productCache) Get(_ context.Context, slug string) (*domain.Product, error) {
	entry, err := c.kv.Get(context.Background(), slug)
	if err != nil {
		return nil, err // jetstream.KeyValueError — caller should treat as cache miss
	}

	var product domain.Product
	if err := json.Unmarshal(entry.Value(), &product); err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to deserialize product cache", 500, err)
	}
	return &product, nil
}

func (c *productCache) Set(_ context.Context, product *domain.Product) error {
	data, err := json.Marshal(product)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to serialize product cache", 500, err)
	}

	_, err = c.kv.Put(context.Background(), product.Slug, data)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to set product cache", 500, err)
	}
	return nil
}

func (c *productCache) Delete(_ context.Context, slug string) error {
	return c.kv.Delete(context.Background(), slug)
}
