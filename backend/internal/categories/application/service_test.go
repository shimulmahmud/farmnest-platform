package application

import (
	"context"
	"testing"

	"github.com/ecommerce/backend/internal/categories/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockCategoryRepo struct {
	categories map[string]*domain.Category
	bySlug     map[string]*domain.Category
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{
		categories: make(map[string]*domain.Category),
		bySlug:     make(map[string]*domain.Category),
	}
}

func (r *mockCategoryRepo) Create(_ context.Context, c *domain.Category) error {
	r.categories[c.ID] = c
	r.bySlug[c.Slug] = c
	return nil
}

func (r *mockCategoryRepo) FindByID(_ context.Context, id string) (*domain.Category, error) {
	c, ok := r.categories[id]
	if !ok {
		return nil, errors.ErrNotFound
	}
	return c, nil
}

func (r *mockCategoryRepo) FindBySlug(_ context.Context, slug string) (*domain.Category, error) {
	c, ok := r.bySlug[slug]
	if !ok {
		return nil, errors.ErrNotFound
	}
	return c, nil
}

func (r *mockCategoryRepo) Update(_ context.Context, c *domain.Category) error {
	delete(r.bySlug, r.categories[c.ID].Slug)
	r.categories[c.ID] = c
	r.bySlug[c.Slug] = c
	return nil
}

func (r *mockCategoryRepo) Delete(_ context.Context, id string) error {
	c, ok := r.categories[id]
	if !ok {
		return errors.ErrNotFound
	}
	delete(r.categories, id)
	delete(r.bySlug, c.Slug)
	return nil
}

func (r *mockCategoryRepo) List(_ context.Context, status string, p pagination.Params) (*domain.CategoryListResult, error) {
	var result []*domain.Category
	for _, c := range r.bySlug {
		if status != "" && string(c.Status) != status {
			continue
		}
		result = append(result, c)
	}
	total := int64(len(result))
	if p.Skip < len(result) {
		result = result[p.Skip:]
	} else {
		result = nil
	}
	if p.Limit < len(result) {
		result = result[:p.Limit]
	}
	return &domain.CategoryListResult{
		Categories: result,
		Total:      total,
		Page:       p.Page,
		PerPage:    p.PerPage,
	}, nil
}

func (r *mockCategoryRepo) ExistsBySlug(_ context.Context, slug string) (bool, error) {
	_, ok := r.bySlug[slug]
	return ok, nil
}

type mockCategoryCache struct {
	store   map[string]*domain.Category
	deleted map[string]bool
}

func newMockCategoryCache() *mockCategoryCache {
	return &mockCategoryCache{
		store:   make(map[string]*domain.Category),
		deleted: make(map[string]bool),
	}
}

func (c *mockCategoryCache) Get(_ context.Context, slug string) (*domain.Category, error) {
	cat, ok := c.store[slug]
	if !ok {
		return nil, errors.ErrNotFound
	}
	return cat, nil
}

func (c *mockCategoryCache) Set(_ context.Context, cat *domain.Category) error {
	c.store[cat.Slug] = cat
	return nil
}

func (c *mockCategoryCache) Delete(_ context.Context, slug string) error {
	c.deleted[slug] = true
	delete(c.store, slug)
	return nil
}

// --- Helpers ---

func newTestService(t *testing.T) (*Service, *mockCategoryRepo, *mockCategoryCache) {
	t.Helper()
	repo := newMockCategoryRepo()
	cache := newMockCategoryCache()
	svc := NewService(repo, cache)
	return svc, repo, cache
}

func sampleCreateInput() CreateCategoryInput {
	return CreateCategoryInput{
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic gadgets and devices",
		Image:       "https://example.com/electronics.jpg",
		SortOrder:   1,
		SEO: domain.CategorySEO{
			Title:       "Electronics Category",
			Description: "Browse electronics",
			Keywords:    "electronics, gadgets, tech",
		},
	}
}

// --- Tests ---

func TestCreateCategory_Success(t *testing.T) {
	svc, repo, _ := newTestService(t)
	input := sampleCreateInput()

	cat, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	assert.NotEmpty(t, cat.ID)
	assert.Equal(t, "Electronics", cat.Name)
	assert.Equal(t, "electronics", cat.Slug)
	assert.Equal(t, domain.CategoryStatusActive, cat.Status)
	assert.Equal(t, 1, cat.SortOrder)

	// Verify stored
	found, _ := repo.FindBySlug(context.Background(), "electronics")
	assert.NotNil(t, found)
}

func TestCreateCategory_DuplicateSlug(t *testing.T) {
	svc, _, _ := newTestService(t)
	input := sampleCreateInput()

	_, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), input)
	assert.Error(t, err)
	appErr, ok := errors.IsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 409, appErr.Status)
}

func TestGetBySlug_CacheMissThenHit(t *testing.T) {
	svc, _, cache := newTestService(t)
	input := sampleCreateInput()

	_, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	// First call — cache miss, then populated
	found, err := svc.GetBySlug(context.Background(), "electronics")
	require.NoError(t, err)
	assert.Equal(t, "electronics", found.Slug)

	// Verify cached
	_, ok := cache.store["electronics"]
	assert.True(t, ok, "category should be cached after get")

	// Second call — cache hit
	found2, err := svc.GetBySlug(context.Background(), "electronics")
	require.NoError(t, err)
	assert.Equal(t, found.ID, found2.ID)
}

func TestGetBySlug_NotFound(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.GetBySlug(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, 404, errors.FromError(err).Status)
}

func TestGetByID(t *testing.T) {
	svc, _, _ := newTestService(t)
	input := sampleCreateInput()

	cat, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	found, err := svc.GetByID(context.Background(), cat.ID)
	require.NoError(t, err)
	assert.Equal(t, cat.ID, found.ID)
}

func TestUpdateCategory(t *testing.T) {
	svc, _, _ := newTestService(t)
	input := sampleCreateInput()

	cat, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	newName := "Consumer Electronics"
	updated, err := svc.Update(context.Background(), cat.ID, UpdateCategoryInput{
		Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "Consumer Electronics", updated.Name)
}

func TestUpdateCategory_DuplicateSlug(t *testing.T) {
	svc, _, _ := newTestService(t)

	input1 := sampleCreateInput()
	input1.Slug = "cat-a"
	_, err := svc.Create(context.Background(), input1)
	require.NoError(t, err)

	input2 := sampleCreateInput()
	input2.Slug = "cat-b"
	cat2, err := svc.Create(context.Background(), input2)
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), cat2.ID, UpdateCategoryInput{
		Slug: strPtr("cat-a"),
	})
	assert.Error(t, err)
	appErr, ok := errors.IsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 409, appErr.Status)
}

func TestDeleteCategory(t *testing.T) {
	svc, repo, _ := newTestService(t)
	input := sampleCreateInput()

	cat, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	err = svc.Delete(context.Background(), cat.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(context.Background(), cat.ID)
	assert.Equal(t, 404, errors.FromError(err).Status)
}

func TestDeleteCategory_NotFound(t *testing.T) {
	svc, _, _ := newTestService(t)

	err := svc.Delete(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.Equal(t, 404, errors.FromError(err).Status)
}

func TestListCategories_Pagination(t *testing.T) {
	svc, _, _ := newTestService(t)

	for i := 0; i < 3; i++ {
		input := sampleCreateInput()
		input.Slug = "cat-" + itoa(i)
		input.Name = "Category " + itoa(i)
		_, err := svc.Create(context.Background(), input)
		require.NoError(t, err)
	}

	result, err := svc.List(context.Background(), "", pagination.Params{
		Page: 1, PerPage: 2, Skip: 0, Limit: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	assert.Len(t, result.Categories, 2)
}

func TestListCategories_FilterByStatus(t *testing.T) {
	svc, _, _ := newTestService(t)

	input := sampleCreateInput()
	cat, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	// Deactivate one
	inactive := "INACTIVE"
	_, err = svc.Update(context.Background(), cat.ID, UpdateCategoryInput{Status: &inactive})
	require.NoError(t, err)

	// List active only
	result, err := svc.List(context.Background(), "ACTIVE", pagination.Params{
		Page: 1, PerPage: 20, Skip: 0, Limit: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)

	// List all
	result, err = svc.List(context.Background(), "", pagination.Params{
		Page: 1, PerPage: 20, Skip: 0, Limit: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

func TestCacheInvalidation_OnUpdate(t *testing.T) {
	svc, _, cache := newTestService(t)
	input := sampleCreateInput()

	_, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	// Access to populate cache
	_, err = svc.GetBySlug(context.Background(), "electronics")
	require.NoError(t, err)
	assert.NotNil(t, cache.store["electronics"])

	// Update with new slug — should invalidate old cache
	newSlug := "updated-electronics"
	_, err = svc.Update(context.Background(), "any", UpdateCategoryInput{Slug: &newSlug})
	// This will fail (not found) but let's test invalidation path
	// Use valid ID
	cat, _ := svc.GetBySlug(context.Background(), "electronics")
	_, err = svc.Update(context.Background(), cat.ID, UpdateCategoryInput{Slug: &newSlug})
	require.NoError(t, err)

	_, deleted := cache.deleted["electronics"]
	assert.True(t, deleted, "old slug should be deleted from cache")
}

func TestCacheInvalidation_OnDelete(t *testing.T) {
	svc, _, cache := newTestService(t)
	input := sampleCreateInput()

	cat, err := svc.Create(context.Background(), input)
	require.NoError(t, err)

	// Access to populate cache
	_, err = svc.GetBySlug(context.Background(), "electronics")
	require.NoError(t, err)
	assert.NotNil(t, cache.store["electronics"])

	// Delete should invalidate cache
	err = svc.Delete(context.Background(), cat.ID)
	require.NoError(t, err)

	_, deleted := cache.deleted["electronics"]
	assert.True(t, deleted, "slug should be deleted from cache on category delete")
}

// --- Helpers ---

func strPtr(s string) *string {
	return &s
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
