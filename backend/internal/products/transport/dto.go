package transport

import (
	"github.com/ecommerce/backend/internal/products/application"
	"github.com/ecommerce/backend/internal/products/domain"
)

// Conventions: money = integer whole BDT taka (no cents), weight = grams,
// dimensions = millimetres. Currency is per product.

// --- Request DTOs ---

// CreateProductRequest is the JSON body for POST /api/v1/admin/products
// and POST /api/v1/vendor/products (vendor_id is ignored there; it comes from the token).
type CreateProductRequest struct {
	Status      string                        `json:"status"`
	VendorID    string                        `json:"vendor_id"`
	Name        string                        `json:"name"`
	Slug        string                        `json:"slug"`
	Description string                        `json:"description"`
	ShortDesc   string                        `json:"short_description"`
	Brand       string                        `json:"brand"`
	CategoryID  string                        `json:"category_id"`
	Unit        string                        `json:"unit"`
	Currency    string                        `json:"currency"`
	Specs       map[string]string             `json:"specs"`
	Tags        []string                      `json:"tags"`
	Images      []domain.Image                `json:"images"`
	Variants    []application.CreateVariantInput `json:"variants"`
	SEO         domain.SEO                    `json:"seo"`
	SortOrder   int                           `json:"sort_order"`
}

// UpdateProductRequest is the JSON body for PATCH /api/v1/{admin|vendor}/products/:id.
type UpdateProductRequest struct {
	VendorID    *string                           `json:"vendor_id"`
	Name        *string                           `json:"name"`
	Slug        *string                           `json:"slug"`
	Description *string                           `json:"description"`
	ShortDesc   *string                           `json:"short_description"`
	Brand       *string                           `json:"brand"`
	CategoryID  *string                           `json:"category_id"`
	Unit        *string                           `json:"unit"`
	Currency    *string                           `json:"currency"`
	Specs       *map[string]string                `json:"specs"`
	Tags        *[]string                         `json:"tags"`
	Images      *[]domain.Image                   `json:"images"`
	Variants    *[]application.CreateVariantInput `json:"variants"`
	Status      *string                           `json:"status"`
	SEO         *domain.SEO                       `json:"seo"`
	SortOrder   *int                              `json:"sort_order"`
}

// --- Response DTOs ---

// ProductResponse is the FULL product view for admins and the owning vendor.
// Never use it on public endpoints.
type ProductResponse struct {
	ID          string            `json:"id"`
	VendorID    string            `json:"vendor_id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	ShortDesc   string            `json:"short_description"`
	Brand       string            `json:"brand"`
	CategoryID  string            `json:"category_id"`
	Unit        string            `json:"unit"`
	Currency    string            `json:"currency"`
	Specs       map[string]string `json:"specs"`
	Tags        []string          `json:"tags"`
	Images      []domain.Image    `json:"images"`
	Variants    []VariantResponse `json:"variants"`
	Status      string            `json:"status"`
	SEO         domain.SEO        `json:"seo"`
	SortOrder   int               `json:"sort_order"`
	LowestPrice int               `json:"lowest_price"`
	// TotalAvailable is the sellable stock across all variants (admin list/detail only).
	TotalAvailable *int `json:"total_available,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// VariantResponse is the full variant view (includes cost_price).
type VariantResponse struct {
	ID               string            `json:"id"`
	SKU              string            `json:"sku"`
	Barcode          string            `json:"barcode,omitempty"`
	Options          map[string]string `json:"options"`
	Price            int               `json:"price"`
	CompareAtPrice   *int              `json:"compare_at_price,omitempty"`
	CostPrice        int               `json:"cost_price"`
	WeightG          int               `json:"weight_g"`
	LengthMM         int               `json:"length_mm,omitempty"`
	WidthMM          int               `json:"width_mm,omitempty"`
	HeightMM         int               `json:"height_mm,omitempty"`
	RequiresShipping bool              `json:"requires_shipping"`
	Images           []domain.Image    `json:"images"`
	// Available is the sellable stock for this SKU (admin list/detail only).
	Available        *int              `json:"available,omitempty"`
	Status           string            `json:"status"`
	Position         int               `json:"position"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
}

// PublicProductResponse is the storefront view. It deliberately omits
// cost_price, exact stock, vendor_id, status and internal timestamps.
type PublicProductResponse struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	Description string                  `json:"description"`
	ShortDesc   string                  `json:"short_description"`
	Brand       string                  `json:"brand"`
	CategoryID  string                  `json:"category_id"`
	Unit        string                  `json:"unit"`
	Currency    string                  `json:"currency"`
	Specs       map[string]string       `json:"specs"`
	Tags        []string                `json:"tags"`
	Images      []domain.Image          `json:"images"`
	Variants    []PublicVariantResponse `json:"variants"`
	SEO         domain.SEO              `json:"seo"`
	LowestPrice int                     `json:"lowest_price"`
	InStock     bool                    `json:"in_stock"`
	Rating      float64                 `json:"rating"`       // average of approved reviews
	ReviewCount int                     `json:"review_count"` // number of approved reviews
}

// PublicVariantResponse is the storefront view of a variant.
type PublicVariantResponse struct {
	ID               string            `json:"id"`
	SKU              string            `json:"sku"`
	Options          map[string]string `json:"options"`
	Price            int               `json:"price"`
	CompareAtPrice   *int              `json:"compare_at_price,omitempty"`
	WeightG          int               `json:"weight_g"`
	RequiresShipping bool              `json:"requires_shipping"`
	Images           []domain.Image    `json:"images"`
	InStock          bool              `json:"in_stock"`
	Position         int               `json:"position"`
}
