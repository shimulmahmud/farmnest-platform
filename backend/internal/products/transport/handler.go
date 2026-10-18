package transport

import (
	"strconv"
	"strings"

	"github.com/ecommerce/backend/internal/products/application"
	"github.com/ecommerce/backend/internal/products/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/ecommerce/backend/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for product endpoints.
type Handler struct {
        service   *application.Service
        validator *validator.Validator
}

// NewHandler creates a product transport handler.
func NewHandler(service *application.Service) *Handler {
        return &Handler{
                service:   service,
                validator: validator.New(),
        }
}

// ListProducts handles GET /api/v1/products.
func (h *Handler) ListProducts(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        filter := domain.ProductFilter{
                CategoryID: c.Query("category_id"),
                Brand:      c.Query("brand"),
                // Public endpoint: always ACTIVE. A client-supplied status is ignored
                // so DRAFT/ARCHIVE products are never exposed.
                Status:     string(domain.ProductStatusActive),
                Tag:        c.Query("tag"),
                Search:     c.Query("search"),
        }

        result, err := h.service.List(c.Context(), filter, p)
        if err != nil {
                return response.Error(c, err)
        }

        avail, err := h.service.Availability(c.Context(), result.Products...)
        if err != nil {
                return response.Error(c, err)
        }
        ratings, err := h.service.Ratings(c.Context(), result.Products...)
        if err != nil {
                return response.Error(c, err)
        }
        products := make([]PublicProductResponse, len(result.Products))
        for i, prod := range result.Products {
                products[i] = withRating(toPublicProductResponse(prod, avail), ratings)
        }

        meta := response.PaginatedMeta(result.Page, result.PerPage, result.Total)
        return response.OKWithMeta(c, products, meta)
}

// SearchProducts handles GET /api/v1/products/search.
func (h *Handler) SearchProducts(c *fiber.Ctx) error {
        query := c.Query("q")
        minPrice := c.Query("min_price")
        maxPrice := c.Query("max_price")
        tags := strings.TrimSpace(c.Query("tags"))

        page, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }
        perPage := page.PerPage
        tagsList := []string{}
        if tags != "" {
                for _, t := range strings.Split(tags, ",") {
                        t = strings.TrimSpace(t)
                        if t != "" {
                                tagsList = append(tagsList, t)
                        }
        }
        }
        sortBy := c.Query("sort_by")
        // Public endpoint: status is always ACTIVE (client value ignored).
        status := string(domain.ProductStatusActive)

        input := domain.SearchInput{
                Query:      query,
                CategoryID: c.Query("category_id"),
                Brand:      c.Query("brand"),
                Tags:       tagsList,
                InStock:    c.QueryBool("in_stock"),
                Status:     status,
                SortBy:     sortBy,
                Page:       page.Page,
                PerPage:    perPage,
        }

        if minPrice != "" {
                v, err := strconv.Atoi(minPrice)
                if err == nil && v >= 0 {
                        input.MinPrice = &v
                }
        }
        if maxPrice != "" {
                v, err := strconv.Atoi(maxPrice)
                if err == nil && v >= 0 {
                        input.MaxPrice = &v
                }
        }

        result, err := h.service.Search(c.Context(), input)
        if err != nil {
                return response.Error(c, err)
        }

        avail, err := h.service.Availability(c.Context(), result.Products...)
        if err != nil {
                return response.Error(c, err)
        }
        ratings, err := h.service.Ratings(c.Context(), result.Products...)
        if err != nil {
                return response.Error(c, err)
        }
        products := make([]PublicProductResponse, len(result.Products))
        for i, p := range result.Products {
                products[i] = withRating(toPublicProductResponse(p, avail), ratings)
        }

        meta := response.PaginatedMeta(result.Page, result.PerPage, result.Total)
        return response.OKWithMeta(c, products, meta)
}

// GetProduct handles GET /api/v1/products/:slug.
func (h *Handler) GetProduct(c *fiber.Ctx) error {
        slug := c.Params("slug")
        if slug == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "slug is required", 422))
        }

        product, err := h.service.GetBySlug(c.Context(), slug)
        if err != nil {
                return response.Error(c, err)
        }
        if !product.IsActive() {
                return response.Error(c, errors.ErrNotFound)
        }

        avail, err := h.service.Availability(c.Context(), product)
        if err != nil {
                return response.Error(c, err)
        }
        ratings, err := h.service.Ratings(c.Context(), product)
        if err != nil {
                return response.Error(c, err)
        }
        return response.OK(c, withRating(toPublicProductResponse(product, avail), ratings))
}

// vendorScope returns the vendor ID the request is restricted to, or "" for
// unrestricted (admin) routes. Vendor routes set Locals("vendor_id") in middleware.
func vendorScope(c *fiber.Ctx) string {
	if v, ok := c.Locals("vendor_id").(string); ok {
		return v
	}
	return ""
}

// ownedProduct loads a product and enforces vendor ownership. A vendor asking
// for another vendor's product gets NOT_FOUND (no existence leak).
func (h *Handler) ownedProduct(c *fiber.Ctx, id string) (*domain.Product, error) {
	product, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return nil, err
	}
	if scope := vendorScope(c); scope != "" && product.VendorID != scope {
		return nil, errors.ErrNotFound
	}
	return product, nil
}

// ListManagedProducts handles GET /api/v1/{admin|vendor}/products.
// Vendors only ever see their own products, any status.
func (h *Handler) ListManagedProducts(c *fiber.Ctx) error {
	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}
	filter := domain.ProductFilter{
		VendorID:   c.Query("vendor_id"),
		CategoryID: c.Query("category_id"),
		Brand:      c.Query("brand"),
		Status:     c.Query("status"),
		Tag:        c.Query("tag"),
		Search:     c.Query("search"),
	}
	if scope := vendorScope(c); scope != "" {
		filter.VendorID = scope
	}
	result, err := h.service.List(c.Context(), filter, p)
	if err != nil {
		return response.Error(c, err)
	}
	avail, err := h.service.Availability(c.Context(), result.Products...)
	if err != nil {
		return response.Error(c, err)
	}
	out := make([]ProductResponse, len(result.Products))
	for i, prod := range result.Products {
		out[i] = withStock(toProductResponse(prod), avail)
	}
	return response.OKWithMeta(c, out, response.PaginatedMeta(result.Page, result.PerPage, result.Total))
}

// GetManagedProduct handles GET /api/v1/{admin|vendor}/products/:id.
func (h *Handler) GetManagedProduct(c *fiber.Ctx) error {
	product, err := h.ownedProduct(c, c.Params("id"))
	if err != nil {
		return response.Error(c, err)
	}
	avail, err := h.service.Availability(c.Context(), product)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, withStock(toProductResponse(product), avail))
}

// CreateProduct handles POST /api/v1/{admin|vendor}/products.
func (h *Handler) CreateProduct(c *fiber.Ctx) error {
	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	if err := h.validator.ValidateRequired(map[string]string{
		"name":        req.Name,
		"slug":        req.Slug,
		"category_id": req.CategoryID,
	}); err != nil {
		return response.Error(c, err)
	}

	vendorID := req.VendorID
	if scope := vendorScope(c); scope != "" {
		vendorID = scope // vendors can only create for themselves
	}

	product, err := h.service.Create(c.Context(), application.CreateProductInput{
		Status:      req.Status,
		VendorID:    vendorID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ShortDesc:   req.ShortDesc,
		Brand:       req.Brand,
		CategoryID:  req.CategoryID,
		Unit:        req.Unit,
		Currency:    req.Currency,
		Specs:       req.Specs,
		Tags:        req.Tags,
		Images:      req.Images,
		Variants:    req.Variants,
		SEO:         req.SEO,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, toProductResponse(product))
}

// UpdateProduct handles PATCH /api/v1/{admin|vendor}/products/:id.
func (h *Handler) UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "id is required", 422))
	}
	if _, err := h.ownedProduct(c, id); err != nil {
		return response.Error(c, err)
	}

	var req UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	input := application.UpdateProductInput{
		VendorID:    req.VendorID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ShortDesc:   req.ShortDesc,
		Brand:       req.Brand,
		CategoryID:  req.CategoryID,
		Unit:        req.Unit,
		Currency:    req.Currency,
		Specs:       req.Specs,
		Tags:        req.Tags,
		Images:      req.Images,
		Variants:    req.Variants,
		Status:      req.Status,
		SEO:         req.SEO,
		SortOrder:   req.SortOrder,
	}
	if vendorScope(c) != "" {
		input.VendorID = nil // a vendor can never reassign ownership
	}

	product, err := h.service.Update(c.Context(), id, input)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, toProductResponse(product))
}

// DeleteProduct handles DELETE /api/v1/{admin|vendor}/products/:id.
func (h *Handler) DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "id is required", 422))
	}
	if _, err := h.ownedProduct(c, id); err != nil {
		return response.Error(c, err)
	}
	if err := h.service.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.NoContent(c)
}

const timeFmt = "2006-01-02T15:04:05Z"

func toProductResponse(p *domain.Product) ProductResponse {
	variants := make([]VariantResponse, len(p.Variants))
	for i, v := range p.Variants {
		variants[i] = VariantResponse{
			ID:               v.ID,
			SKU:              v.SKU,
			Barcode:          v.Barcode,
			Options:          v.Options,
			Price:            v.Price,
			CompareAtPrice:   v.CompareAtPrice,
			CostPrice:        v.CostPrice,
			WeightG:          v.WeightG,
			LengthMM:         v.LengthMM,
			WidthMM:          v.WidthMM,
			HeightMM:         v.HeightMM,
			RequiresShipping: v.RequiresShipping,
			Images:           v.Images,
			Status:           string(v.Status),
			Position:         v.Position,
			CreatedAt:        v.CreatedAt.UTC().Format(timeFmt),
			UpdatedAt:        v.UpdatedAt.UTC().Format(timeFmt),
		}
	}
	return ProductResponse{
		ID:          p.ID,
		VendorID:    p.VendorID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		ShortDesc:   p.ShortDesc,
		Brand:       p.Brand,
		CategoryID:  p.CategoryID,
		Unit:        p.Unit,
		Currency:    p.Currency,
		Specs:       p.Specs,
		Tags:        p.Tags,
		Images:      p.Images,
		Variants:    variants,
		Status:      string(p.Status),
		SEO:         p.SEO,
		SortOrder:   p.SortOrder,
		LowestPrice: p.LowestPrice,
		CreatedAt:   p.CreatedAt.UTC().Format(timeFmt),
		UpdatedAt:   p.UpdatedAt.UTC().Format(timeFmt),
	}
}

// toPublicProductResponse builds the storefront view. avail maps SKU -> available
// quantity from inventory; a SKU missing from the map is treated as out of stock.
func toPublicProductResponse(p *domain.Product, avail map[string]int) PublicProductResponse {
	variants := make([]PublicVariantResponse, 0, len(p.Variants))
	anyInStock := false
	for _, v := range p.Variants {
		if v.Status != domain.VariantStatusActive {
			continue
		}
		in := avail[v.SKU] > 0
		anyInStock = anyInStock || in
		variants = append(variants, PublicVariantResponse{
			ID:               v.ID,
			SKU:              v.SKU,
			Options:          v.Options,
			Price:            v.Price,
			CompareAtPrice:   v.CompareAtPrice,
			WeightG:          v.WeightG,
			RequiresShipping: v.RequiresShipping,
			Images:           v.Images,
			InStock:          in,
			Position:         v.Position,
		})
	}
	return PublicProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		ShortDesc:   p.ShortDesc,
		Brand:       p.Brand,
		CategoryID:  p.CategoryID,
		Unit:        p.Unit,
		Currency:    p.Currency,
		Specs:       p.Specs,
		Tags:        p.Tags,
		Images:      p.Images,
		Variants:    variants,
		SEO:         p.SEO,
		LowestPrice: p.LowestPrice,
		InStock:     anyInStock,
	}
}

// withStock fills in the sellable stock per variant and in total (admin views).
func withStock(r ProductResponse, avail map[string]int) ProductResponse {
	total := 0
	for i := range r.Variants {
		a := avail[r.Variants[i].SKU]
		r.Variants[i].Available = &a
		if r.Variants[i].Status == string(domain.VariantStatusActive) {
			total += a
		}
	}
	r.TotalAvailable = &total
	return r
}

// withRating adds the approved-review summary to a public product.
func withRating(r PublicProductResponse, ratings map[string]domain.Rating) PublicProductResponse {
	if rt, ok := ratings[r.ID]; ok {
		r.Rating = rt.Average
		r.ReviewCount = rt.Count
	}
	return r
}
