package application

import (
        "context"
        "strings"
        "testing"

        "github.com/ecommerce/backend/internal/products/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockProductRepo struct {
        products map[string]*domain.Product
        bySlug   map[string]*domain.Product
}

func newMockProductRepo() *mockProductRepo {
        return &mockProductRepo{
                products: make(map[string]*domain.Product),
                bySlug:   make(map[string]*domain.Product),
        }
}

func (r *mockProductRepo) Create(_ context.Context, p *domain.Product) error {
        r.products[p.ID] = p
        r.bySlug[p.Slug] = p
        return nil
}

func (r *mockProductRepo) FindByID(_ context.Context, id string) (*domain.Product, error) {
        p, ok := r.products[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return p, nil
}

func (r *mockProductRepo) FindBySlug(_ context.Context, slug string) (*domain.Product, error) {
        p, ok := r.bySlug[slug]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return p, nil
}

func (r *mockProductRepo) Update(_ context.Context, p *domain.Product) error {
        delete(r.bySlug, r.products[p.ID].Slug)
        r.products[p.ID] = p
        r.bySlug[p.Slug] = p
        return nil
}

func (r *mockProductRepo) Delete(_ context.Context, id string) error {
        p, ok := r.products[id]
        if !ok {
                return errors.ErrNotFound
        }
        delete(r.products, id)
        delete(r.bySlug, p.Slug)
        return nil
}

func (r *mockProductRepo) List(_ context.Context, filter domain.ProductFilter, p pagination.Params) (*domain.ProductListResult, error) {
        var result []*domain.Product
        for _, prod := range r.bySlug {
                if filter.Status != "" && string(prod.Status) != filter.Status {
                        continue
                }
                if filter.CategoryID != "" && prod.CategoryID != filter.CategoryID {
                        continue
                }
                if filter.Brand != "" && prod.Brand != filter.Brand {
                        continue
                }
                result = append(result, prod)
        }
        total := int64(len(result))
        // Apply skip/limit
        if p.Skip < len(result) {
                result = result[p.Skip:]
        } else {
                result = nil
        }
        if p.Limit < len(result) {
                result = result[:p.Limit]
        }
        return &domain.ProductListResult{
                Products: result,
                Total:    total,
                Page:     p.Page,
                PerPage:  p.PerPage,
        }, nil
}

func (r *mockProductRepo) ExistsBySlug(_ context.Context, slug string) (bool, error) {
        _, ok := r.bySlug[slug]
        return ok, nil
}

func (r *mockProductRepo) FindBySKU(_ context.Context, sku string) (*domain.Product, error) {
        for _, p := range r.products {
                for _, v := range p.Variants {
                        if v.SKU == sku {
                                return p, nil
                        }
                }
        }
        return nil, errors.ErrNotFound
}

func (r *mockProductRepo) SetVariantCost(_ context.Context, sku string, cost int) error {
        for _, p := range r.products {
                for i := range p.Variants {
                        if p.Variants[i].SKU == sku {
                                p.Variants[i].CostPrice = cost
                                return nil
                        }
                }
        }
        return errors.ErrNotFound
}

func (r *mockProductRepo) ExistsBySKU(_ context.Context, sku string) (bool, error) {
        for _, p := range r.products {
                for _, v := range p.Variants {
                        if v.SKU == sku {
                                return true, nil
                        }
                }
        }
        return false, nil
}

func (r *mockProductRepo) FindAllActive(_ context.Context) ([]*domain.Product, error) {
        var result []*domain.Product
        for _, p := range r.products {
                if p.IsActive() {
                        result = append(result, p)
                }
        }
        return result, nil
}

func (r *mockProductRepo) Search(_ context.Context, input domain.SearchInput) (*domain.ProductListResult, error) {
        var result []*domain.Product
        for _, prod := range r.products {
                // Status filter
                if input.Status != "" {
                        if string(prod.Status) != input.Status {
                                continue
                }
                } else if !prod.IsActive() {
                        continue
                }
                // Category filter
                if input.CategoryID != "" && prod.CategoryID != input.CategoryID {
                        continue
                }
                // Brand filter
                if input.Brand != "" && prod.Brand != input.Brand {
                        continue
                }
                // Tags filter (all must match, case-insensitive)
                if len(input.Tags) > 0 {
                        matched := true
                        for _, tag := range input.Tags {
                                found := false
                                for _, pt := range prod.Tags {
                                        if strings.EqualFold(pt, tag) {
                                                found = true
                                                break
                                        }
                                }
                                if !found {
                                        matched = false
                                        break
                                }
                        }
                        if !matched {
                                continue
                        }
                }
                // Simple text search on name (mimics $text behavior)
                if input.Query != "" {
                        terms := domain.NormalizeSearchTerms(input.Query)
                        found := false
                        for _, term := range terms {
                                if strings.Contains(strings.ToLower(prod.Name), term) {
                                        found = true
                                        break
                                }
                        }
                        // Also check SKU
                        if !found {
                                for _, v := range prod.Variants {
                                        if strings.Contains(strings.ToLower(v.SKU), strings.ToLower(input.Query)) {
                                                found = true
                                                break
                                        }
                                }
                        }
                        if !found {
                                continue
                        }
                }
                // In-stock filter
                if input.InStock {
                        has := false
                        for _, v := range prod.Variants {
                                for _, sku := range input.InStockSKUs {
                                        if v.Status == domain.VariantStatusActive && v.SKU == sku {
                                                has = true
                                        }
                                }
                        }
                        if !has {
                                continue
                        }
                }
                result = append(result, prod)
        }
        total := int64(len(result))
        skip := (input.Page - 1) * input.PerPage
        if skip < 0 {
                skip = 0
        }
        if skip < len(result) {
                result = result[skip:]
        } else {
                result = nil
        }
        if input.PerPage < len(result) {
                result = result[:input.PerPage]
        }
        return &domain.ProductListResult{
                Products: result,
                Total:    total,
                Page:     input.Page,
                PerPage:  input.PerPage,
        }, nil
}

type mockProductCache struct {
        store    map[string]*domain.Product
        deleted  map[string]bool
}

func newMockProductCache() *mockProductCache {
        return &mockProductCache{
                store:   make(map[string]*domain.Product),
                deleted: make(map[string]bool),
        }
}

func (c *mockProductCache) Get(_ context.Context, slug string) (*domain.Product, error) {
        p, ok := c.store[slug]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return p, nil
}

func (c *mockProductCache) Set(_ context.Context, p *domain.Product) error {
        c.store[p.Slug] = p
        return nil
}

func (c *mockProductCache) Delete(_ context.Context, slug string) error {
        c.deleted[slug] = true
        delete(c.store, slug)
        return nil
}

// --- Helpers ---

func newTestService(t *testing.T) (*Service, *mockProductRepo, *mockProductCache) {
        t.Helper()
        repo := newMockProductRepo()
        cache := newMockProductCache()
        svc := NewService(repo, cache)
        return svc, repo, cache
}

func sampleCreateInput() CreateProductInput {
        return CreateProductInput{
                Name:        "Test T-Shirt",
                Slug:        "test-t-shirt",
                Description: "A comfortable cotton t-shirt",
                ShortDesc:   "Cotton t-shirt",
                Brand:       "TestBrand",
                CategoryID:  "cat-123",
                Tags:        []string{"cotton", "tshirt"},
                SEO: domain.SEO{
                        Title:       "Test T-Shirt",
                        Description: "Buy test t-shirt",
                        Keywords:    []string{"tshirt", "test", "cotton"},
                },
                Variants: []CreateVariantInput{
                        {
                                SKU:       "TSH-001",
                                Price:     2999,
                                CostPrice: 1000,
                                Options:   map[string]string{"size": "M", "color": "Blue"},
                        },
                },
        }
}

// --- Tests ---

func TestCreateProduct_Success(t *testing.T) {
        svc, repo, _ := newTestService(t)
        input := sampleCreateInput()

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)

        assert.NotEmpty(t, product.ID)
        assert.Equal(t, "Test T-Shirt", product.Name)
        assert.Equal(t, "test-t-shirt", product.Slug)
        assert.Equal(t, "TestBrand", product.Brand)
        assert.Equal(t, "cat-123", product.CategoryID)
        assert.Equal(t, domain.ProductStatusDraft, product.Status)
        assert.Len(t, product.Variants, 1)
        assert.Equal(t, "TSH-001", product.Variants[0].SKU)
        assert.Equal(t, 2999, product.Variants[0].Price)
        assert.Equal(t, 2999, product.LowestPrice)
        assert.Equal(t, "piece", product.Unit)
        assert.Equal(t, "BDT", product.Currency)

        // Verify stored
        found, _ := repo.FindBySlug(context.Background(), "test-t-shirt")
        assert.NotNil(t, found)
}

func TestCreateProduct_DuplicateSlug(t *testing.T) {
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

func TestCreateProduct_DuplicateSKU(t *testing.T) {
        svc, _, _ := newTestService(t)

        input1 := sampleCreateInput()
        input1.Slug = "product-1"
        _, err := svc.Create(context.Background(), input1)
        require.NoError(t, err)

        input2 := sampleCreateInput()
        input2.Slug = "product-2"
        input2.Variants[0].SKU = "TSH-001" // duplicate
        _, err = svc.Create(context.Background(), input2)
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, 409, appErr.Status)
}

func TestGetBySlug_CacheMissThenHit(t *testing.T) {
        svc, _, cache := newTestService(t)
        input := sampleCreateInput()

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)

        // Activate product so it gets cached
        product.Status = domain.ProductStatusActive
        _, err = svc.Update(context.Background(), product.ID, UpdateProductInput{
                Status: strPtr("ACTIVE"),
        })
        require.NoError(t, err)

        // First call — cache miss (old cache was set but product was updated so cache invalidated)
        // Actually after update, cache is deleted. So this is a cache miss.
        found, err := svc.GetBySlug(context.Background(), "test-t-shirt")
        require.NoError(t, err)
        assert.Equal(t, product.ID, found.ID)

        // Now it should be cached
        _, ok := cache.store["test-t-shirt"]
        assert.True(t, ok, "product should be cached after get")

        // Second call — cache hit
        found2, err := svc.GetBySlug(context.Background(), "test-t-shirt")
        require.NoError(t, err)
        assert.Equal(t, product.ID, found2.ID)
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

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)

        found, err := svc.GetByID(context.Background(), product.ID)
        require.NoError(t, err)
        assert.Equal(t, product.ID, found.ID)
}

func TestUpdateProduct(t *testing.T) {
        svc, _, _ := newTestService(t)
        input := sampleCreateInput()

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)

        newName := "Updated T-Shirt"
        updated, err := svc.Update(context.Background(), product.ID, UpdateProductInput{
                Name:   &newName,
                Status: strPtr("ACTIVE"),
        })
        require.NoError(t, err)
        assert.Equal(t, "Updated T-Shirt", updated.Name)
        assert.Equal(t, domain.ProductStatusActive, updated.Status)
}

func TestUpdateProduct_DuplicateSlug(t *testing.T) {
        svc, _, _ := newTestService(t)

        input1 := sampleCreateInput()
        input1.Slug = "product-a"
        _, err := svc.Create(context.Background(), input1)
        require.NoError(t, err)

        input2 := sampleCreateInput()
        input2.Slug = "product-b"
        input2.Variants[0].SKU = "TSH-002"
        p2, err := svc.Create(context.Background(), input2)
        require.NoError(t, err)

        // Try to update product-b to have product-a's slug
        _, err = svc.Update(context.Background(), p2.ID, UpdateProductInput{
                Slug: strPtr("product-a"),
        })
        assert.Error(t, err)
        appErr, ok := errors.IsAppError(err)
        require.True(t, ok)
        assert.Equal(t, 409, appErr.Status)
}

func TestDeleteProduct(t *testing.T) {
        svc, repo, _ := newTestService(t)
        input := sampleCreateInput()

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)

        err = svc.Delete(context.Background(), product.ID)
        require.NoError(t, err)

        // Verify deleted
        _, err = repo.FindByID(context.Background(), product.ID)
        assert.Equal(t, 404, errors.FromError(err).Status)
}

func TestDeleteProduct_NotFound(t *testing.T) {
        svc, _, _ := newTestService(t)

        err := svc.Delete(context.Background(), "nonexistent-id")
        assert.Error(t, err)
        assert.Equal(t, 404, errors.FromError(err).Status)
}

func TestListProducts_Pagination(t *testing.T) {
        svc, _, _ := newTestService(t)

        // Create 3 products
        for i := 0; i < 3; i++ {
                input := sampleCreateInput()
                input.Slug = "product-" + itoa(i)
                input.Name = "Product " + itoa(i)
                input.Variants[0].SKU = "SKU-" + itoa(i)
                _, err := svc.Create(context.Background(), input)
                require.NoError(t, err)
        }

        // List page 1 with per_page=2
        result, err := svc.List(context.Background(), domain.ProductFilter{}, pagination.Params{
                Page: 1, PerPage: 2, Skip: 0, Limit: 2,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(3), result.Total)
        assert.Len(t, result.Products, 2)
        assert.Equal(t, 2, result.PerPage)
}

func TestCacheInvalidation_OnUpdate(t *testing.T) {
        svc, _, cache := newTestService(t)
        input := sampleCreateInput()

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)

        // Activate and cache it
        product.Status = domain.ProductStatusActive
        _, err = svc.Update(context.Background(), product.ID, UpdateProductInput{Status: strPtr("ACTIVE")})
        require.NoError(t, err)
        _, err = svc.GetBySlug(context.Background(), "test-t-shirt")
        require.NoError(t, err)
        assert.NotNil(t, cache.store["test-t-shirt"])

        // Now update — should invalidate cache
        newSlug := "updated-t-shirt"
        _, err = svc.Update(context.Background(), product.ID, UpdateProductInput{Slug: &newSlug})
        require.NoError(t, err)

        _, deleted := cache.deleted["test-t-shirt"]
        assert.True(t, deleted, "old slug should be deleted from cache")
}

func TestCacheInvalidation_OnDelete(t *testing.T) {
        svc, _, cache := newTestService(t)
        input := sampleCreateInput()

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)

        // Activate and cache
        product.Status = domain.ProductStatusActive
        _, err = svc.Update(context.Background(), product.ID, UpdateProductInput{Status: strPtr("ACTIVE")})
        require.NoError(t, err)
        _, err = svc.GetBySlug(context.Background(), "test-t-shirt")
        require.NoError(t, err)
        assert.NotNil(t, cache.store["test-t-shirt"])

        // Delete should invalidate cache
        err = svc.Delete(context.Background(), product.ID)
        require.NoError(t, err)

        _, deleted := cache.deleted["test-t-shirt"]
        assert.True(t, deleted, "slug should be deleted from cache on product delete")
}

func TestProduct_LowestPrice(t *testing.T) {
        svc, _, _ := newTestService(t)

        input := sampleCreateInput()
        input.Variants = []CreateVariantInput{
                {SKU: "SKU-A", Price: 5000, CostPrice: 2000},
                {SKU: "SKU-B", Price: 2500, CostPrice: 1000},
        }

        product, err := svc.Create(context.Background(), input)
        require.NoError(t, err)
        assert.Equal(t, 2500, product.LowestPrice)
}

func TestCreateProduct_RejectsInvalidVariants(t *testing.T) {
        svc, _, _ := newTestService(t)

        input := sampleCreateInput()
        input.Variants = []CreateVariantInput{{SKU: "A", Price: 0}}
        _, err := svc.Create(context.Background(), input)
        require.Error(t, err)

        input.Variants = []CreateVariantInput{{SKU: "A", Price: 100}, {SKU: "A", Price: 100}}
        _, err = svc.Create(context.Background(), input)
        require.Error(t, err, "duplicate SKUs in one request must be rejected")
}

type fakeStock struct{ avail map[string]int }

func (f *fakeStock) AvailableBySKU(_ context.Context, skus []string) (map[string]int, error) {
        out := map[string]int{}
        for _, s := range skus {
                if v, ok := f.avail[s]; ok {
                        out[s] = v
                }
        }
        return out, nil
}

func (f *fakeStock) InStockSKUs(_ context.Context) ([]string, error) {
        var out []string
        for s, v := range f.avail {
                if v > 0 {
                        out = append(out, s)
                }
        }
        return out, nil
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

// --- Search Tests ---

func TestSearch_ByQuery(t *testing.T) {
        svc, _, _ := newTestService(t)

        input1 := sampleCreateInput()
        input1.Name = "Blue Cotton Shirt"
        input1.Slug = "blue-shirt"
        input1.Variants[0].SKU = "BLS-001"
        p1, err := svc.Create(context.Background(), input1)
        require.NoError(t, err)

        input2 := sampleCreateInput()
        input2.Name = "Red Silk Blouse"
        input2.Slug = "red-blouse"
        input2.Variants[0].SKU = "RED-002"
        p2, err := svc.Create(context.Background(), input2)
        require.NoError(t, err)

        // Activate both products so they're searchable (default filter is ACTIVE)
        _, _ = svc.Update(context.Background(), p1.ID, UpdateProductInput{Status: strPtr("ACTIVE")})
        _, _ = svc.Update(context.Background(), p2.ID, UpdateProductInput{Status: strPtr("ACTIVE")})

        // Search for "blue"
        result, err := svc.Search(context.Background(), domain.SearchInput{
                Query:   "blue",
                Page:     1,
                PerPage:  10,
        })
        require.NoError(t, err)
        // DRAFT products are excluded by default; only active match
        assert.Equal(t, int64(1), result.Total)
        if len(result.Products) > 0 {
                assert.Contains(t, strings.ToLower(result.Products[0].Name), "blue")
        }
}

func TestSearch_BySKU(t *testing.T) {
        svc, _, _ := newTestService(t)

        input := sampleCreateInput()
        input.Name = "Widget"
        input.Slug = "widget"
        input.Variants[0].SKU = "WGT-999"
        prod, err := svc.Create(context.Background(), input)
        require.NoError(t, err)
        _, _ = svc.Update(context.Background(), prod.ID, UpdateProductInput{Status: strPtr("ACTIVE")})

        result, err := svc.Search(context.Background(), domain.SearchInput{
                Query:   "WGT-999",
                Page:     1,
                PerPage:  10,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(1), result.Total)
}

func TestSearch_ByBrand(t *testing.T) {
        svc, _, _ := newTestService(t)

        input := sampleCreateInput()
        input.Brand = "Nike"
        input.Slug = "nike-shoe"
        input.Variants[0].SKU = "NK-001"
        prod, err := svc.Create(context.Background(), input)
        require.NoError(t, err)
        _, _ = svc.Update(context.Background(), prod.ID, UpdateProductInput{Status: strPtr("ACTIVE")})

        result, err := svc.Search(context.Background(), domain.SearchInput{
                Brand:   "Nike",
                Page:    1,
                PerPage: 10,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(1), result.Total)
}

func TestSearch_ByTags(t *testing.T) {
        svc, _, _ := newTestService(t)

        input := sampleCreateInput()
        input.Tags = []string{"cotton", "summer"}
        input.Slug = "summer-dress"
        input.Variants[0].SKU = "SDR-001"
        prod, err := svc.Create(context.Background(), input)
        require.NoError(t, err)
        _, _ = svc.Update(context.Background(), prod.ID, UpdateProductInput{Status: strPtr("ACTIVE")})

        result, err := svc.Search(context.Background(), domain.SearchInput{
                Tags:    []string{"cotton"},
                Page:    1,
                PerPage: 10,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(1), result.Total)
}

func TestSearch_ExcludesDraft(t *testing.T) {
        svc, _, _ := newTestService(t)

        input := sampleCreateInput()
        input.Name = "Draft Product"
        input.Slug = "draft-prod"
        input.Variants[0].SKU = "DFT-001"
        _, err := svc.Create(context.Background(), input)
        require.NoError(t, err)
        // Don't activate — stays DRAFT

        result, err := svc.Search(context.Background(), domain.SearchInput{
                Query:   "Draft",
                Page:     1,
                PerPage:  10,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(0), result.Total)
}

func TestSearch_Pagination(t *testing.T) {
        svc, _, _ := newTestService(t)

        for i := 0; i < 5; i++ {
                input := sampleCreateInput()
                input.Name = "Searchable Item " + itoa(i)
                input.Slug = "searchable-" + itoa(i)
                input.Variants[0].SKU = "SRH-" + itoa(i)
                prod, err := svc.Create(context.Background(), input)
                require.NoError(t, err)
                _, _ = svc.Update(context.Background(), prod.ID, UpdateProductInput{Status: strPtr("ACTIVE")})
        }

        // Search with page 1, per_page 2
        result, err := svc.Search(context.Background(), domain.SearchInput{
                Query:   "Searchable",
                Page:     1,
                PerPage:  2,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(5), result.Total)
        assert.Len(t, result.Products, 2)
}

func TestSearch_InStock(t *testing.T) {
        svc, _, _ := newTestService(t)
        svc.SetStockChecker(&fakeStock{avail: map[string]int{"IS-001": 50, "OOS-002": 0}})

        // Product with stock
        inStock := sampleCreateInput()
        inStock.Slug = "in-stock-item"
        inStock.Variants[0].SKU = "IS-001"
        p1, err := svc.Create(context.Background(), inStock)
        require.NoError(t, err)
        _, _ = svc.Update(context.Background(), p1.ID, UpdateProductInput{Status: strPtr("ACTIVE")})

        // Product out of stock
        oos := sampleCreateInput()
        oos.Slug = "out-of-stock-item"
        oos.Variants[0].SKU = "OOS-002"
        p2, err := svc.Create(context.Background(), oos)
        require.NoError(t, err)
        _, _ = svc.Update(context.Background(), p2.ID, UpdateProductInput{Status: strPtr("ACTIVE")})

        // Search without in_stock filter — both should appear
        result, err := svc.Search(context.Background(), domain.SearchInput{
                Query:   "Item",
                Page:     1,
                PerPage:  10,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(2), result.Total)

        // Search with in_stock=true — only the in-stock product
        result, err = svc.Search(context.Background(), domain.SearchInput{
                Query:   "Item",
                InStock: true,
                Page:     1,
                PerPage:  10,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(1), result.Total)
        if len(result.Products) > 0 {
                assert.Equal(t, "in-stock-item", result.Products[0].Slug)
        }
}

func TestSearch_MultiTerm(t *testing.T) {
        svc, _, _ := newTestService(t)

        input := sampleCreateInput()
        input.Name = "Blue Cotton Shirt"
        input.Slug = "blue-cotton-shirt"
        input.Variants[0].SKU = "BCS-001"
        p, err := svc.Create(context.Background(), input)
        require.NoError(t, err)
        _, _ = svc.Update(context.Background(), p.ID, UpdateProductInput{Status: strPtr("ACTIVE")})

        // Multi-term search: "blue cotton" should match the product
        result, err := svc.Search(context.Background(), domain.SearchInput{
                Query:   "blue cotton",
                Page:     1,
                PerPage:  10,
        })
        require.NoError(t, err)
        assert.Equal(t, int64(1), result.Total)
}
