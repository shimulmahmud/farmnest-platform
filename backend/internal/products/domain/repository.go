package domain

import (
        "context"
        "strings"

        "go.mongodb.org/mongo-driver/v2/bson"

        "github.com/ecommerce/backend/pkg/pagination"
)

// ProductListResult holds a paginated product listing.
type ProductListResult struct {
        Products  []*Product
        Total     int64
        Page      int
        PerPage   int
}

// ProductFilter holds optional filters for listing products.
type ProductFilter struct {
        VendorID   string
        CategoryID string
        Brand      string
        Status     string
        Tag        string
        Search     string
}

// SearchInput holds the parameters for a product search.
type SearchInput struct {
        VendorID   string
        Query      string
        CategoryID string
        Brand      string
        Tags       []string
        MinPrice   *int // whole BDT taka
        MaxPrice   *int
        InStock    bool
        // InStockSKUs is filled by the service from the inventory module when
        // InStock is set; the repository only matches products having an active
        // variant whose SKU is in this list.
        InStockSKUs []string
        Status     string
        SortBy     string // "relevance", "price_asc", "price_desc", "created_at", "name"
        Page       int
        PerPage    int
}

// SearchSort maps sort_by values to MongoDB sort documents.
func SearchSort(sortBy string) bson.D {
        switch strings.ToLower(sortBy) {
        case "price_asc":
                return bson.D{{Key: "lowest_price", Value: 1}, {Key: "created_at", Value: -1}}
        case "price_desc":
                return bson.D{{Key: "lowest_price", Value: -1}, {Key: "created_at", Value: -1}}
        case "created_at", "newest":
                return bson.D{{Key: "created_at", Value: -1}}
        case "name":
                return bson.D{{Key: "name", Value: 1}}
        default:
                // Relevance (text score) — caller must use $meta for this
                return bson.D{{Key: "created_at", Value: -1}}
        }
}
// NormalizeSearchTerms lowercases and splits a query into individual terms,
// stripping empty strings. Used for tag and multi-word search support.
func NormalizeSearchTerms(query string) []string {
        query = strings.ToLower(strings.TrimSpace(query))
        terms := strings.Fields(query)
        result := make([]string, 0, len(terms))
        for _, t := range terms {
                if t != "" {
                        result = append(result, t)
                }
        }
        return result
}
// ProductRepository defines the persistence operations for products.
type ProductRepository interface {
	// FindBySKU returns the product that owns the variant with this SKU.
	FindBySKU(ctx context.Context, sku string) (*Product, error)

	// SetVariantCost updates one variant's cost_price (whole BDT) by SKU.
	SetVariantCost(ctx context.Context, sku string, cost int) error

        Create(ctx context.Context, product *Product) error
        FindByID(ctx context.Context, id string) (*Product, error)
        FindBySlug(ctx context.Context, slug string) (*Product, error)
        Update(ctx context.Context, product *Product) error
        Delete(ctx context.Context, id string) error
        List(ctx context.Context, filter ProductFilter, p pagination.Params) (*ProductListResult, error)
        Search(ctx context.Context, input SearchInput) (*ProductListResult, error)
        ExistsBySlug(ctx context.Context, slug string) (bool, error)
        ExistsBySKU(ctx context.Context, sku string) (bool, error)
        FindAllActive(ctx context.Context) ([]*Product, error)
}

// ProductCache defines the caching operations for products.
type ProductCache interface {
        Get(ctx context.Context, slug string) (*Product, error)
        Set(ctx context.Context, product *Product) error
        Delete(ctx context.Context, slug string) error
}


// Rating is the approved-review summary of a product.
type Rating struct {
	Average float64
	Count   int
}

// RatingReader reads approved-review summaries (from the reviews module).
type RatingReader interface {
	// RatingsByProduct returns the summary per product ID; products with no approved reviews are absent.
	RatingsByProduct(ctx context.Context, productIDs []string) (map[string]Rating, error)
}

// StockChecker reads availability from the inventory module (the single source
// of truth for stock). Products never store stock themselves.
type StockChecker interface {
	// AvailableBySKU returns available quantity (total - reserved) per SKU.
	// SKUs with no inventory record are absent from the map.
	AvailableBySKU(ctx context.Context, skus []string) (map[string]int, error)
	// InStockSKUs returns every SKU that currently has available > 0.
	InStockSKUs(ctx context.Context) ([]string, error)
}
