package application

import (
        "context"
        "fmt"
        "time"

        "github.com/ecommerce/backend/internal/products/domain"
        "github.com/ecommerce/backend/pkg/audit"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
)

// Service implements product catalog use cases.
type Service struct {
        repo    domain.ProductRepository
        cache   domain.ProductCache
        auditor audit.Auditor
        events  domain.EventPublisher
        stock   domain.StockChecker
        ratings domain.RatingReader
}

// NewService creates a product application service.
func NewService(repo domain.ProductRepository, cache domain.ProductCache) *Service {
        return &Service{repo: repo, cache: cache, auditor: audit.NopAuditor}
}

// SetAuditor injects an async auditor for cross-cutting audit logging.
func (s *Service) SetAuditor(a audit.Auditor) {
        s.auditor = a
}

// SetRatingReader injects the review-summary reader (optional).
func (s *Service) SetRatingReader(r domain.RatingReader) {
	s.ratings = r
}

// Ratings returns the approved-review summary per product ID.
func (s *Service) Ratings(ctx context.Context, products ...*domain.Product) (map[string]domain.Rating, error) {
	if s.ratings == nil || len(products) == 0 {
		return map[string]domain.Rating{}, nil
	}
	ids := make([]string, len(products))
	for i, p := range products {
		ids[i] = p.ID
	}
	return s.ratings.RatingsByProduct(ctx, ids)
}

// SetStockChecker injects the inventory-backed availability reader.
func (s *Service) SetStockChecker(c domain.StockChecker) {
        s.stock = c
}

// Availability returns available quantity per SKU for the given products.
// Returns an empty map when no stock checker is wired.
func (s *Service) Availability(ctx context.Context, products ...*domain.Product) (map[string]int, error) {
        var skus []string
        for _, p := range products {
                skus = append(skus, p.SKUs()...)
        }
        if s.stock == nil || len(skus) == 0 {
                return map[string]int{}, nil
        }
        return s.stock.AvailableBySKU(ctx, skus)
}

// SetEvents injects an event publisher for product lifecycle events.
func (s *Service) SetEvents(e domain.EventPublisher) {
        s.events = e
}

// CreateProductInput holds the fields for creating a product.
type CreateProductInput struct {
        Status      string               `json:"status"` // optional: DRAFT (default), ACTIVE, ARCHIVED
        VendorID    string               `json:"vendor_id"`
        Name        string               `json:"name"`
        Slug        string               `json:"slug"`
        Description string               `json:"description"`
        ShortDesc   string               `json:"short_description"`
        Brand       string               `json:"brand"`
        CategoryID  string               `json:"category_id"`
        Unit        string               `json:"unit"`
        Currency    string               `json:"currency"`
        Specs       map[string]string    `json:"specs"`
        Tags        []string             `json:"tags"`
        Images      []domain.Image       `json:"images"`
        Variants    []CreateVariantInput `json:"variants"`
        SEO         domain.SEO           `json:"seo"`
        SortOrder   int                  `json:"sort_order"`
}

// CreateVariantInput holds the fields for a variant (create and update).
// Prices are integer minor units; weight is grams; dimensions are millimetres.
type CreateVariantInput struct {
        SKU              string            `json:"sku"`
        Barcode          string            `json:"barcode"`
        Options          map[string]string `json:"options"`
        Price            int               `json:"price"`
        CompareAtPrice   *int              `json:"compare_at_price"`
        CostPrice        int               `json:"cost_price"`
        WeightG          int               `json:"weight_g"`
        LengthMM         int               `json:"length_mm"`
        WidthMM          int               `json:"width_mm"`
        HeightMM         int               `json:"height_mm"`
        RequiresShipping *bool             `json:"requires_shipping"` // nil = true
        Images           []domain.Image    `json:"images"`
}

// UpdateProductInput holds the fields for updating a product.
type UpdateProductInput struct {
        VendorID    *string               `json:"vendor_id"`
        Name        *string               `json:"name"`
        Slug        *string               `json:"slug"`
        Description *string               `json:"description"`
        ShortDesc   *string               `json:"short_description"`
        Brand       *string               `json:"brand"`
        CategoryID  *string               `json:"category_id"`
        Unit        *string               `json:"unit"`
        Currency    *string               `json:"currency"`
        Specs       *map[string]string    `json:"specs"`
        Tags        *[]string             `json:"tags"`
        Images      *[]domain.Image       `json:"images"`
        Variants    *[]CreateVariantInput `json:"variants"`
        Status      *string               `json:"status"`
        SEO         *domain.SEO           `json:"seo"`
        SortOrder   *int                  `json:"sort_order"`
}

// applyVariantInput copies editable fields from input onto v.
func applyVariantInput(v *domain.Variant, in CreateVariantInput) {
        v.Barcode = in.Barcode
        v.Options = in.Options
        v.Price = in.Price
        v.CompareAtPrice = in.CompareAtPrice
        v.CostPrice = in.CostPrice
        v.WeightG = in.WeightG
        v.LengthMM = in.LengthMM
        v.WidthMM = in.WidthMM
        v.HeightMM = in.HeightMM
        v.RequiresShipping = in.RequiresShipping == nil || *in.RequiresShipping
        v.Images = in.Images
}

// validateVariants checks cross-field rules shared by create and update.
func validateVariants(in []CreateVariantInput) error {
        if len(in) == 0 {
                return errors.New("VALIDATION_ERROR", "at least one variant is required", 422)
        }
        seen := map[string]bool{}
        for i, v := range in {
                if v.SKU == "" {
                        return errors.New("VALIDATION_ERROR", fmt.Sprintf("variant[%d].sku is required", i), 422)
                }
                if seen[v.SKU] {
                        return errors.New("VALIDATION_ERROR", fmt.Sprintf("duplicate sku %q in request", v.SKU), 422)
                }
                seen[v.SKU] = true
                if v.Price <= 0 {
                        return errors.New("VALIDATION_ERROR", fmt.Sprintf("variant[%d].price must be positive (minor units)", i), 422)
                }
                if v.CompareAtPrice != nil && *v.CompareAtPrice < v.Price {
                        return errors.New("VALIDATION_ERROR", fmt.Sprintf("variant[%d].compare_at_price must be >= price", i), 422)
                }
                if v.CostPrice < 0 || v.WeightG < 0 {
                        return errors.New("VALIDATION_ERROR", fmt.Sprintf("variant[%d] cost_price and weight_g must not be negative", i), 422)
                }
        }
        return nil
}

// Create creates a new product with uniqueness checks.
func (s *Service) Create(ctx context.Context, input CreateProductInput) (*domain.Product, error) {
        if err := validateVariants(input.Variants); err != nil {
                return nil, err
        }
        exists, err := s.repo.ExistsBySlug(ctx, input.Slug)
        if err != nil {
                return nil, err
        }
        if exists {
                return nil, errors.New("CONFLICT", "product slug already exists", 409)
        }

        for _, v := range input.Variants {
                existsSKU, err := s.repo.ExistsBySKU(ctx, v.SKU)
                if err != nil {
                        return nil, err
                }
                if existsSKU {
                        return nil, errors.New("CONFLICT", "variant SKU \""+v.SKU+"\" already exists", 409)
                }
        }

        variants := make([]domain.Variant, len(input.Variants))
        for i, v := range input.Variants {
                variant := domain.NewVariant(v.SKU, v.Price, v.CostPrice, v.Options)
                applyVariantInput(variant, v)
                variant.Position = i
                variants[i] = *variant
        }

        product := domain.NewProduct(
                input.Name,
                input.Slug,
                input.Description,
                input.ShortDesc,
                input.Brand,
                input.CategoryID,
                input.Tags,
                input.Images,
                input.SEO,
        )
        product.Variants = variants
        product.VendorID = input.VendorID
        product.Specs = input.Specs
        if input.Unit != "" {
                product.Unit = input.Unit
        }
        if input.Currency != "" {
                product.Currency = input.Currency
        }
        if input.SortOrder != 0 {
                product.SortOrder = input.SortOrder
        }
        if input.Status != "" {
                st := domain.ProductStatus(input.Status)
                if !st.IsValid() {
                        return nil, errors.New("VALIDATION_ERROR", "status must be DRAFT, ACTIVE or ARCHIVED", 422)
                }
                product.Status = st
        }
        product.RecomputeDerived()

        if err := s.repo.Create(ctx, product); err != nil {
                return nil, err
        }

        s.auditor.Record("", audit.ActionProductCreate, "product", product.ID, "", input.Name, "product created")

        s.publishEvent(ctx, "product.created", product)

        return product, nil
}

// GetBySlug returns a product by slug, using cache when available.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
        if s.cache != nil {
                cached, err := s.cache.Get(ctx, slug)
                if err == nil && cached != nil {
                        return cached, nil
                }
        }

        product, err := s.repo.FindBySlug(ctx, slug)
        if err != nil {
                return nil, err
        }

        if s.cache != nil && product.IsActive() {
                _ = s.cache.Set(ctx, product)
        }

        return product, nil
}

// GetByID returns a product by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Product, error) {
        return s.repo.FindByID(ctx, id)
}

// Update updates an existing product, invalidates cache, and records audit.
func (s *Service) Update(ctx context.Context, id string, input UpdateProductInput) (*domain.Product, error) {
        product, err := s.repo.FindByID(ctx, id)
        if err != nil {
                return nil, err
        }

        oldSlug := product.Slug
        oldPrice := fmt.Sprintf("%d", product.ComputeLowestPrice())

        if input.Slug != nil && *input.Slug != product.Slug {
                exists, err := s.repo.ExistsBySlug(ctx, *input.Slug)
                if err != nil {
                        return nil, err
                }
                if exists {
                        return nil, errors.New("CONFLICT", "product slug already exists", 409)
                }
                product.Slug = *input.Slug
        }

        if input.Name != nil {
                product.Name = *input.Name
        }
        if input.VendorID != nil {
                product.VendorID = *input.VendorID
        }
        if input.Unit != nil {
                product.Unit = *input.Unit
        }
        if input.Currency != nil {
                product.Currency = *input.Currency
        }
        if input.Specs != nil {
                product.Specs = *input.Specs
        }
        if input.Description != nil {
                product.Description = *input.Description
        }
        if input.ShortDesc != nil {
                product.ShortDesc = *input.ShortDesc
        }
        if input.Brand != nil {
                product.Brand = *input.Brand
        }
        if input.CategoryID != nil {
                product.CategoryID = *input.CategoryID
        }
        if input.Tags != nil {
                product.Tags = *input.Tags
        }
        if input.Images != nil {
                product.Images = *input.Images
        }
        if input.Status != nil {
                st := domain.ProductStatus(*input.Status)
                if !st.IsValid() {
                        return nil, errors.New("VALIDATION_ERROR", "status must be DRAFT, ACTIVE or ARCHIVED", 422)
                }
                product.Status = st
        }
        if input.SEO != nil {
                product.SEO = *input.SEO
        }
        if input.SortOrder != nil {
                product.SortOrder = *input.SortOrder
        }
        if input.Variants != nil {
                if err := validateVariants(*input.Variants); err != nil {
                        return nil, err
                }
                product.Variants = mergeVariants(product.Variants, *input.Variants)
        }
        product.RecomputeDerived()

        if err := s.repo.Update(ctx, product); err != nil {
                return nil, err
        }

        // Audit: record update
        newPrice := fmt.Sprintf("%d", product.ComputeLowestPrice())
        details := "product updated"
        action := audit.ActionProductUpdate
        if oldPrice != newPrice {
                action = audit.ActionProductPriceChange
                details = fmt.Sprintf("price changed from %s to %s", oldPrice, newPrice)
        }
        s.auditor.Record("", action, "product", id, oldPrice, newPrice, details)

        s.publishEvent(ctx, "product.updated", product)

        // Invalidate cache
        if s.cache != nil {
                _ = s.cache.Delete(ctx, oldSlug)
        }

        return product, nil
}

// mergeVariants applies the submitted variant list to the existing variants.
// A submitted variant whose SKU matches an existing one is updated in place and
// KEEPS its ID and CreatedAt, so order items, wishlists and carts that
// reference variant_id stay valid. New SKUs get a new variant. Existing
// variants missing from the submitted list are archived, not deleted, because
// past orders may reference them.
func mergeVariants(existing []domain.Variant, submitted []CreateVariantInput) []domain.Variant {
        bySKU := make(map[string]domain.Variant, len(existing))
        for _, v := range existing {
                bySKU[v.SKU] = v
        }

        seen := make(map[string]bool, len(submitted))
        merged := make([]domain.Variant, 0, len(submitted)+len(existing))
        for i, in := range submitted {
                seen[in.SKU] = true
                v, ok := bySKU[in.SKU]
                if !ok {
                        nv := domain.NewVariant(in.SKU, in.Price, in.CostPrice, in.Options)
                        v = *nv
                } else {
                        v.Status = domain.VariantStatusActive
                        v.UpdatedAt = time.Now().UTC()
                }
                applyVariantInput(&v, in)
                v.Position = i
                merged = append(merged, v)
        }

        for _, v := range existing {
                if !seen[v.SKU] {
                        v.Status = domain.VariantStatusArchived
                        v.UpdatedAt = time.Now().UTC()
                        merged = append(merged, v)
                }
        }
        return merged
}

// Delete removes a product, invalidates cache, and records audit.
func (s *Service) Delete(ctx context.Context, id string) error {
        product, err := s.repo.FindByID(ctx, id)
        if err != nil {
                return err
        }

        if err := s.repo.Delete(ctx, id); err != nil {
                return err
        }

        s.auditor.Record("", audit.ActionProductDelete, "product", id, product.Name, "", "product deleted")

        s.publishEvent(ctx, "product.deleted", product)

        if s.cache != nil {
                _ = s.cache.Delete(ctx, product.Slug)
        }

        return nil
}

// Search performs a full-text product search with filters, sorting, and pagination.
func (s *Service) Search(ctx context.Context, input domain.SearchInput) (*domain.ProductListResult, error) {
        if input.InStock {
                if s.stock == nil {
                        return &domain.ProductListResult{Products: nil, Total: 0, Page: input.Page, PerPage: input.PerPage}, nil
                }
                skus, err := s.stock.InStockSKUs(ctx)
                if err != nil {
                        return nil, err
                }
                if len(skus) == 0 {
                        return &domain.ProductListResult{Products: nil, Total: 0, Page: input.Page, PerPage: input.PerPage}, nil
                }
                input.InStockSKUs = skus
        }
        return s.repo.Search(ctx, input)
}

// List returns a paginated product listing.
func (s *Service) List(ctx context.Context, filter domain.ProductFilter, p pagination.Params) (*domain.ProductListResult, error) {
        return s.repo.List(ctx, filter, p)
}

// publishEvent sends a product lifecycle event via the event publisher (fire-and-forget).
// A future search worker can consume these events to maintain an external index.
func (s *Service) publishEvent(ctx context.Context, eventType string, product *domain.Product) {
        if s.events == nil {
                return
        }
        evt := product.SearchableProduct()
        evt.EventType = eventType
        _ = s.events.Publish(ctx, "ECOMMERCE_PRODUCTS."+eventType, evt)
}
