package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/backend/internal/categories/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/nats-io/nats.go/jetstream"
)

const categoryCacheBucket = "CATEGORY_CACHE"

type categoryCache struct {
	kv jetstream.KeyValue
}

// NewCategoryCache creates a NATS KV-backed category cache.
func NewCategoryCache(kv jetstream.KeyValue) domain.CategoryCache {
	return &categoryCache{kv: kv}
}

func (c *categoryCache) Get(_ context.Context, slug string) (*domain.Category, error) {
	entry, err := c.kv.Get(context.Background(), slug)
	if err != nil {
		return nil, err
	}

	var category domain.Category
	if err := json.Unmarshal(entry.Value(), &category); err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to deserialize category cache", 500, err)
	}
	return &category, nil
}

func (c *categoryCache) Set(_ context.Context, category *domain.Category) error {
	data, err := json.Marshal(category)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to serialize category cache", 500, err)
	}

	_, err = c.kv.Put(context.Background(), category.Slug, data)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to set category cache", 500, err)
	}
	return nil
}

func (c *categoryCache) Delete(_ context.Context, slug string) error {
	return c.kv.Delete(context.Background(), slug)
}
