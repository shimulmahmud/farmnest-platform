package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DefaultCurrency is applied when a product is created without a currency.
const DefaultCurrency = "BDT"

// DefaultUnit is what one quantity means when a product does not set a unit.
const DefaultUnit = "piece"

// ProductStatus represents the publication state of a product.
type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "DRAFT"
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusArchived ProductStatus = "ARCHIVED"
)

// IsValid reports whether s is a known product status.
func (s ProductStatus) IsValid() bool {
	switch s {
	case ProductStatusDraft, ProductStatusActive, ProductStatusArchived:
		return true
	}
	return false
}

// VariantStatus represents the availability of a specific variant.
type VariantStatus string

const (
	VariantStatusActive   VariantStatus = "ACTIVE"
	VariantStatusInactive VariantStatus = "INACTIVE"
	VariantStatusArchived VariantStatus = "ARCHIVED"
)

// SEO holds search-engine-optimization metadata.
type SEO struct {
	Title       string   `bson:"title" json:"title"`
	Description string   `bson:"description" json:"description"`
	Keywords    []string `bson:"keywords" json:"keywords"`
}

// Image represents a product or variant image. Width/Height are 0 when unknown.
type Image struct {
	URL       string `bson:"url" json:"url"`
	Alt       string `bson:"alt" json:"alt"`
	Width     int    `bson:"width,omitempty" json:"width,omitempty"`
	Height    int    `bson:"height,omitempty" json:"height,omitempty"`
	SortOrder int    `bson:"sort_order" json:"sort_order"`
}

// Variant is a sellable, stockable, shippable unit of a product.
//
// Conventions (platform-wide):
//   - Money is an integer number of whole currency units (BDT taka, no paisa/cents). Currency lives on the Product.
//   - Weight is integer grams, dimensions are integer millimetres.
//   - Stock is NOT stored here: the inventory module is the single source of
//     truth, keyed by SKU.
type Variant struct {
	ID               string            `bson:"_id" json:"id"`
	SKU              string            `bson:"sku" json:"sku"`
	Barcode          string            `bson:"barcode,omitempty" json:"barcode,omitempty"`
	Options          map[string]string `bson:"options" json:"options"` // what the buyer picks: color, size, grade...
	Price            int               `bson:"price" json:"price"`
	CompareAtPrice   *int              `bson:"compare_at_price,omitempty" json:"compare_at_price,omitempty"` // display only
	CostPrice        int               `bson:"cost_price" json:"cost_price"`
	WeightG          int               `bson:"weight_g" json:"weight_g"`
	LengthMM         int               `bson:"length_mm,omitempty" json:"length_mm,omitempty"`
	WidthMM          int               `bson:"width_mm,omitempty" json:"width_mm,omitempty"`
	HeightMM         int               `bson:"height_mm,omitempty" json:"height_mm,omitempty"`
	RequiresShipping bool              `bson:"requires_shipping" json:"requires_shipping"`
	Images           []Image           `bson:"images" json:"images"`
	Status           VariantStatus     `bson:"status" json:"status"`
	Position         int               `bson:"position" json:"position"`
	CreatedAt        time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time         `bson:"updated_at" json:"updated_at"`
}

// NewVariant creates a new Variant with generated ID and timestamps.
func NewVariant(sku string, price, costPrice int, options map[string]string) *Variant {
	now := time.Now().UTC()
	return &Variant{
		ID:               uuid.New().String(),
		SKU:              sku,
		Price:            price,
		CostPrice:        costPrice,
		Options:          options,
		RequiresShipping: true,
		Status:           VariantStatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// Product represents a catalog product. It works for any kind of goods
// (electronics, agricultural, clothing...): type-specific details go in Specs,
// what a quantity means goes in Unit.
type Product struct {
	ID          string            `bson:"_id" json:"id"`
	VendorID    string            `bson:"vendor_id" json:"vendor_id"` // owning seller; "" = platform-owned
	Name        string            `bson:"name" json:"name"`
	Slug        string            `bson:"slug" json:"slug"`
	Description string            `bson:"description" json:"description"`
	ShortDesc   string            `bson:"short_description" json:"short_description"`
	Brand       string            `bson:"brand" json:"brand"`
	CategoryID  string            `bson:"category_id" json:"category_id"`
	Unit        string            `bson:"unit" json:"unit"`
	Specs       map[string]string `bson:"specs" json:"specs"`
	Tags        []string          `bson:"tags" json:"tags"`
	Images      []Image           `bson:"images" json:"images"`
	Currency    string            `bson:"currency" json:"currency"`
	Variants    []Variant         `bson:"variants" json:"variants"`
	Status      ProductStatus     `bson:"status" json:"status"`
	SEO         SEO               `bson:"seo" json:"seo"`
	SortOrder   int               `bson:"sort_order" json:"sort_order"`
	// LowestPrice is derived (min active variant price) and stored so price
	// sorting works in the database. Always set via RecomputeDerived.
	LowestPrice int       `bson:"lowest_price" json:"lowest_price"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

// NewProduct creates a new Product with generated ID and timestamps.
func NewProduct(name, slug, description, shortDesc, brand, categoryID string, tags []string, images []Image, seo SEO) *Product {
	now := time.Now().UTC()
	return &Product{
		ID:          uuid.New().String(),
		Name:        name,
		Slug:        slug,
		Description: description,
		ShortDesc:   shortDesc,
		Brand:       brand,
		CategoryID:  categoryID,
		Unit:        DefaultUnit,
		Currency:    DefaultCurrency,
		Tags:        tags,
		Images:      images,
		Status:      ProductStatusDraft,
		SEO:         seo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// FindVariant returns the variant with the given ID, or nil.
func (p *Product) FindVariant(id string) *Variant {
	for i := range p.Variants {
		if p.Variants[i].ID == id {
			return &p.Variants[i]
		}
	}
	return nil
}

// IsActive returns true if the product is published.
func (p *Product) IsActive() bool {
	return p.Status == ProductStatusActive
}

// ActiveVariants returns the variants that can currently be sold.
func (p *Product) ActiveVariants() []Variant {
	out := make([]Variant, 0, len(p.Variants))
	for _, v := range p.Variants {
		if v.Status == VariantStatusActive {
			out = append(out, v)
		}
	}
	return out
}

// SKUs returns every variant SKU.
func (p *Product) SKUs() []string {
	skus := make([]string, len(p.Variants))
	for i, v := range p.Variants {
		skus[i] = v.SKU
	}
	return skus
}

// ComputeLowestPrice returns the minimum price among active variants (0 if none).
func (p *Product) ComputeLowestPrice() int {
	min := -1
	for _, v := range p.Variants {
		if v.Status == VariantStatusActive && (min < 0 || v.Price < min) {
			min = v.Price
		}
	}
	if min < 0 {
		return 0
	}
	return min
}

// RecomputeDerived refreshes stored derived fields. Call before every save.
func (p *Product) RecomputeDerived() {
	p.LowestPrice = p.ComputeLowestPrice()
	if p.Unit == "" {
		p.Unit = DefaultUnit
	}
	if p.Currency == "" {
		p.Currency = DefaultCurrency
	}
}

// ProductEvent represents a product lifecycle event published to JetStream.
// A future search worker can consume these events to synchronize
// an external search engine (e.g., OpenSearch, Meilisearch).
// Fields are designed so a search worker has all data needed to build
// a complete search document without querying the primary database.
type ProductEvent struct {
        EventType        string   `json:"event_type"`
        ProductID       string   `json:"product_id"`
        Name             string   `json:"name"`
        Slug             string   `json:"slug"`
        ShortDescription string   `json:"short_description"`
        Brand            string   `json:"brand"`
        CategoryID       string   `json:"category_id"`
        Tags             []string `json:"tags"`
        SEOKeywords      []string `json:"seo_keywords"`
        Status           string   `json:"status"`
        SKUs             []string `json:"skus"`
        LowestPrice      int      `json:"lowest_price"`
}

// EventPublisher publishes product domain events.
// Implementations typically write to the outbox for at-least-once delivery.
type EventPublisher interface {
        Publish(ctx context.Context, subject string, event *ProductEvent) error
}

// SearchableProduct returns a lightweight snapshot for search indexing.
// This is the data a future search worker would extract from the event.
// Includes short_description and seo.keywords so the search worker can
// index all customer-facing text fields without a separate DB lookup.
func (p *Product) SearchableProduct() *ProductEvent {
        skus := make([]string, len(p.Variants))
        for i, v := range p.Variants {
                skus[i] = v.SKU
        }
        return &ProductEvent{
                ProductID:        p.ID,
                Name:             p.Name,
                Slug:             p.Slug,
                ShortDescription: p.ShortDesc,
                Brand:            p.Brand,
                CategoryID:       p.CategoryID,
                Tags:             p.Tags,
                SEOKeywords:      p.SEO.Keywords,
                Status:           string(p.Status),
                SKUs:             skus,
                LowestPrice:      p.ComputeLowestPrice(),
        }
}
