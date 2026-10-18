package domain

import (
	"context"

	"github.com/ecommerce/backend/pkg/pagination"
)

// CategoryListResult holds a paginated category listing.
type CategoryListResult struct {
	Categories []*Category
	Total      int64
	Page       int
	PerPage    int
}

// CategoryRepository defines the persistence operations for categories.
type CategoryRepository interface {
	Create(ctx context.Context, category *Category) error
	FindByID(ctx context.Context, id string) (*Category, error)
	FindBySlug(ctx context.Context, slug string) (*Category, error)
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, status string, p pagination.Params) (*CategoryListResult, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

// CategoryCache defines the caching operations for categories.
type CategoryCache interface {
	Get(ctx context.Context, slug string) (*Category, error)
	Set(ctx context.Context, category *Category) error
	Delete(ctx context.Context, slug string) error
}
