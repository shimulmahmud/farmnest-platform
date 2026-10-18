package application

import (
        "context"

        "github.com/ecommerce/backend/internal/categories/domain"
        "github.com/ecommerce/backend/pkg/audit"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
)

// Service implements category catalog use cases.
type Service struct {
        repo    domain.CategoryRepository
        cache   domain.CategoryCache
        auditor audit.Auditor
}

// NewService creates a category application service.
func NewService(repo domain.CategoryRepository, cache domain.CategoryCache) *Service {
        return &Service{repo: repo, cache: cache, auditor: audit.NopAuditor}
}

// SetAuditor injects an async auditor for cross-cutting audit logging.
func (s *Service) SetAuditor(a audit.Auditor) {
        s.auditor = a
}

// CreateCategoryInput holds the fields for creating a category.
type CreateCategoryInput struct {
        Name        string                `json:"name"`
        Slug        string                `json:"slug"`
        Description string                `json:"description"`
        ParentID    *string               `json:"parent_id"`
        Image       string                `json:"image"`
        SortOrder   int                   `json:"sort_order"`
        SEO         domain.CategorySEO    `json:"seo"`
}

// UpdateCategoryInput holds the fields for updating a category.
type UpdateCategoryInput struct {
        Name        *string             `json:"name"`
        Slug        *string             `json:"slug"`
        Description *string             `json:"description"`
        ParentID    *string             `json:"parent_id"`
        Image       *string             `json:"image"`
        SortOrder   *int                `json:"sort_order"`
        Status      *string             `json:"status"`
        SEO         *domain.CategorySEO `json:"seo"`
}

// Create creates a new category with uniqueness checks.
func (s *Service) Create(ctx context.Context, input CreateCategoryInput) (*domain.Category, error) {
        exists, err := s.repo.ExistsBySlug(ctx, input.Slug)
        if err != nil {
                return nil, err
        }
        if exists {
                return nil, errors.New("CONFLICT", "category slug already exists", 409)
        }

        category := domain.NewCategory(
                input.Name,
                input.Slug,
                input.Description,
                input.ParentID,
                input.Image,
                input.SortOrder,
                input.SEO,
        )

        if err := s.repo.Create(ctx, category); err != nil {
                return nil, err
        }

        s.auditor.Record("", audit.ActionCategoryCreate, "category", category.ID, "", category.Name, "category created")

        return category, nil
}

// GetBySlug returns a category by slug, using cache when available.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
        // Try cache first
        if s.cache != nil {
                cached, err := s.cache.Get(ctx, slug)
                if err == nil && cached != nil {
                        return cached, nil
                }
        }

        category, err := s.repo.FindBySlug(ctx, slug)
        if err != nil {
                return nil, err
        }

        // Populate cache for active categories
        if s.cache != nil && category.IsActive() {
                _ = s.cache.Set(ctx, category)
        }

        return category, nil
}

// GetByID returns a category by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Category, error) {
        return s.repo.FindByID(ctx, id)
}

// Update updates an existing category and invalidates cache.
func (s *Service) Update(ctx context.Context, id string, input UpdateCategoryInput) (*domain.Category, error) {
        category, err := s.repo.FindByID(ctx, id)
        if err != nil {
                return nil, err
        }

        oldSlug := category.Slug

        if input.Slug != nil && *input.Slug != category.Slug {
                exists, err := s.repo.ExistsBySlug(ctx, *input.Slug)
                if err != nil {
                        return nil, err
                }
                if exists {
                        return nil, errors.New("CONFLICT", "category slug already exists", 409)
                }
                category.Slug = *input.Slug
        }

        if input.Name != nil {
                category.Name = *input.Name
        }
        if input.Description != nil {
                category.Description = *input.Description
        }
        if input.ParentID != nil {
                category.ParentID = input.ParentID
        }
        if input.Image != nil {
                category.Image = *input.Image
        }
        if input.SortOrder != nil {
                category.SortOrder = *input.SortOrder
        }
        if input.Status != nil {
                category.Status = domain.CategoryStatus(*input.Status)
        }
        if input.SEO != nil {
                category.SEO = *input.SEO
        }

        if err := s.repo.Update(ctx, category); err != nil {
                return nil, err
        }

        s.auditor.Record("", audit.ActionCategoryUpdate, "category", id, "", category.Name, "category updated")

        // Invalidate cache
        if s.cache != nil {
                _ = s.cache.Delete(ctx, oldSlug)
        }

        return category, nil
}

// Delete removes a category and invalidates cache.
func (s *Service) Delete(ctx context.Context, id string) error {
        category, err := s.repo.FindByID(ctx, id)
        if err != nil {
                return err
        }

        if err := s.repo.Delete(ctx, id); err != nil {
                return err
        }

        s.auditor.Record("", audit.ActionCategoryDelete, "category", id, category.Name, "", "category deleted")

        // Invalidate cache
        if s.cache != nil {
                _ = s.cache.Delete(ctx, category.Slug)
        }

        return nil
}

// List returns a paginated category listing.
func (s *Service) List(ctx context.Context, status string, p pagination.Params) (*domain.CategoryListResult, error) {
        return s.repo.List(ctx, status, p)
}
