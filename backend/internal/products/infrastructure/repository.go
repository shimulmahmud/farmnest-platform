package infrastructure

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ecommerce/backend/internal/products/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type productRepository struct {
        coll *mongo.Collection
}

// NewProductRepository creates a MongoDB-backed ProductRepository.
func NewProductRepository(db *mongo.Database) domain.ProductRepository {
        return &productRepository{coll: db.Collection("products")}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
        _, err := r.coll.InsertOne(ctx, product)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        if strings.Contains(err.Error(), "slug") {
                                return errors.Wrap("CONFLICT", "product slug already exists", 409, err)
                        }
                        return errors.Wrap("CONFLICT", "product with conflicting data already exists", 409, err)
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to create product", 500, err)
        }
        return nil
}

func (r *productRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
        var product domain.Product
        filter := bson.M{"_id": id}
        if err := r.coll.FindOne(ctx, filter).Decode(&product); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find product", 500, err)
        }
        return &product, nil
}

func (r *productRepository) FindBySlug(ctx context.Context, slug string) (*domain.Product, error) {
        var product domain.Product
        filter := bson.M{"slug": slug}
        if err := r.coll.FindOne(ctx, filter).Decode(&product); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find product by slug", 500, err)
        }
        return &product, nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
        product.UpdatedAt = time.Now().UTC()
        filter := bson.M{"_id": product.ID}
        update := bson.M{"$set": product}
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        if strings.Contains(err.Error(), "slug") {
                                return errors.Wrap("CONFLICT", "product slug already exists", 409, err)
                        }
                        return errors.Wrap("CONFLICT", "product with conflicting data already exists", 409, err)
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to update product", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
        result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to delete product", 500, err)
        }
        if result.DeletedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *productRepository) List(ctx context.Context, filter domain.ProductFilter, p pagination.Params) (*domain.ProductListResult, error) {
        query := bson.M{}

        if filter.VendorID != "" {
                query["vendor_id"] = filter.VendorID
        }
        if filter.CategoryID != "" {
                query["category_id"] = filter.CategoryID
        }
        if filter.Brand != "" {
                query["brand"] = filter.Brand
        }
        if filter.Status != "" {
                query["status"] = filter.Status
        }
        if filter.Tag != "" {
                query["tags"] = filter.Tag
        }
        if filter.Search != "" {
                query["name"] = bson.M{"$regex": regexp.QuoteMeta(filter.Search), "$options": "i"}
        }

        total, err := r.coll.CountDocuments(ctx, query)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to count products", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "sort_order", Value: 1}, {Key: "created_at", Value: -1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        cursor, err := r.coll.Find(ctx, query, opts)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to list products", 500, err)
        }
        defer cursor.Close(ctx)

        var products []*domain.Product
        if err := cursor.All(ctx, &products); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode products", 500, err)
        }

        return &domain.ProductListResult{
                Products: products,
                Total:    total,
                Page:     p.Page,
                PerPage:  p.PerPage,
        }, nil
}

func (r *productRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
        count, err := r.coll.CountDocuments(ctx, bson.M{"slug": slug})
        if err != nil {
                return false, errors.Wrap("INTERNAL_ERROR", "failed to check product slug existence", 500, err)
        }
        return count > 0, nil
}

func (r *productRepository) ExistsBySKU(ctx context.Context, sku string) (bool, error) {
        count, err := r.coll.CountDocuments(ctx, bson.M{"variants.sku": sku})
        if err != nil {
                return false, errors.Wrap("INTERNAL_ERROR", "failed to check product SKU existence", 500, err)
        }
        return count > 0, nil
}

func (r *productRepository) FindAllActive(ctx context.Context) ([]*domain.Product, error) {
        cursor, err := r.coll.Find(ctx, bson.M{"status": domain.ProductStatusActive},
                options.Find().SetSort(bson.D{{Key: "sort_order", Value: 1}}))
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to list active products", 500, err)
        }
        defer cursor.Close(ctx)

        var products []*domain.Product
        if err := cursor.All(ctx, &products); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode active products", 500, err)
        }
        return products, nil
}

// isSKULike returns true if the query looks like a SKU (contains digits).
// SKU queries additionally search variants.sku via regex.
func isSKULike(query string) bool {
        for _, c := range query {
                if c >= '0' && c <= '9' {
                        return true
                }
        }
        return false
}

// Search performs a multi-field text search with filters, sorting, and pagination.
// It uses $text for full-text search on name, brand, description, short_description, tags.
// When the query contains digits (SKU-like), it also searches variants.sku via regex.
// The in_stock filter uses $elemMatch on variants, wrapped in $and to avoid
// field-path conflicts with the SKU $or branch.
// For future search engine sync: product.created/updated/deleted events are published
// on ECOMMERCE_PRODUCTS stream — a search worker can consume these to maintain
// an external index (e.g., OpenSearch/Meilisearch).
func (r *productRepository) Search(ctx context.Context, input domain.SearchInput) (*domain.ProductListResult, error) {
        query := bson.M{}

        // Status filter
        if input.Status != "" {
                query["status"] = input.Status
        } else {
                query["status"] = domain.ProductStatusActive
        }

        if input.VendorID != "" {
                query["vendor_id"] = input.VendorID
        }

        // Category filter
        if input.CategoryID != "" {
                query["category_id"] = input.CategoryID
        }

        // Brand filter (exact match)
        if input.Brand != "" {
                query["brand"] = input.Brand
        }

        // Tags filter (match all of the normalized tags)
        if len(input.Tags) > 0 {
                normalizedTags := make([]string, 0, len(input.Tags))
                for _, tag := range input.Tags {
                        normalizedTags = append(normalizedTags, strings.ToLower(strings.TrimSpace(tag)))
                }
                query["tags"] = bson.M{"$all": normalizedTags}
        }

        // Price range filter on variant prices
        if input.MinPrice != nil || input.MaxPrice != nil {
                priceFilter := bson.M{}
                if input.MinPrice != nil {
                        priceFilter["$gte"] = *input.MinPrice
                }
                if input.MaxPrice != nil {
                        priceFilter["$lte"] = *input.MaxPrice
                }
                query["variants.price"] = priceFilter
        }

        // In-stock filter: only match products with at least one active variant having stock > 0.
        // We store this as a separate clause to avoid conflicts with $or on variants.sku.
        var inStockClause bson.M
        if input.InStock {
                skus := input.InStockSKUs
                if skus == nil {
                        skus = []string{}
                }
                inStockClause = bson.M{
                        "variants": bson.M{
                                "$elemMatch": bson.M{"status": domain.VariantStatusActive, "sku": bson.M{"$in": skus}},
                        },
                }
        }

        // Text search with SKU regex fallback
        useTextSearch := false
        if input.Query != "" {
                terms := domain.NormalizeSearchTerms(input.Query)
                if len(terms) > 0 {
                        // Join all normalized terms with spaces so MongoDB's $text
                        // searches for the complete phrase (all terms must appear).
                        searchText := strings.Join(terms, " ")
                        if isSKULike(input.Query) {
                                // SKU-like query: use $or to combine $text and SKU regex
                                sanitized := regexp.QuoteMeta(input.Query)
                                query["$or"] = []bson.M{
                                        {"$text": bson.M{"$search": searchText}},
                                        {"variants.sku": bson.M{"$regex": sanitized, "$options": "i"}},
                                }
                                useTextSearch = true
                        } else {
                                query["$text"] = bson.M{"$search": searchText}
                                useTextSearch = true
                        }
                }
        }

        // When inStockClause is set, wrap the query in $and to avoid conflicts
        // between the variants $elemMatch and other conditions on variants (e.g. $or on variants.sku).
        finalQuery := query
        if inStockClause != nil {
                finalQuery = bson.M{"$and": []bson.M{query, inStockClause}}
        }

        total, err := r.coll.CountDocuments(ctx, finalQuery)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to count search results", 500, err)
        }

        p := pagination.Params{
                Page:    input.Page,
                PerPage: input.PerPage,
                Skip:    (input.Page - 1) * input.PerPage,
                Limit:   input.PerPage,
        }
        if p.Page < 1 {
                p.Page = 1
                p.Skip = 0
        }
        if p.PerPage < 1 {
                p.PerPage = 20
        }
        if p.PerPage > 100 {
                p.PerPage = 100
        }
        if p.Limit > 100 {
                p.Limit = 100
        }

        opts := options.Find().
                SetSort(domain.SearchSort(input.SortBy)).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        // For text search with relevance sort, use $meta to project score
        if useTextSearch && (input.SortBy == "" || input.SortBy == "relevance") {
                opts.SetSort(bson.D{{Key: "score", Value: bson.D{{Key: "$meta", Value: "textScore"}}}})
                opts.SetProjection(bson.M{"score": bson.E{Key: "$meta", Value: "textScore"}})
        }

        cursor, err := r.coll.Find(ctx, finalQuery, opts)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to search products", 500, err)
        }
        defer cursor.Close(ctx)

        var products []*domain.Product
        if err := cursor.All(ctx, &products); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode search results", 500, err)
        }

        return &domain.ProductListResult{
                Products: products,
                Total:    total,
                Page:     p.Page,
                PerPage:  p.PerPage,
        }, nil
}

// FindBySKU returns the product owning the variant with the given SKU.
func (r *productRepository) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	var p domain.Product
	if err := r.coll.FindOne(ctx, bson.M{"variants.sku": sku}).Decode(&p); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find product by SKU", 500, err)
	}
	return &p, nil
}

// SetVariantCost updates a variant's cost_price by SKU.
func (r *productRepository) SetVariantCost(ctx context.Context, sku string, cost int) error {
	now := time.Now().UTC()
	res, err := r.coll.UpdateOne(ctx, bson.M{"variants.sku": sku}, bson.M{"$set": bson.M{
		"variants.$.cost_price": cost,
		"variants.$.updated_at": now,
		"updated_at":            now,
	}})
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to update variant cost", 500, err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// EnsureIndexes creates required indexes on the products collection.
// The text index covers name (weight 10), brand (5), description (3), short_description (4), tags (2)
// for full-text search. SKU search uses regex on the existing unique SKU index.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("products")
        indexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "slug", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_product_slug_unique"),
                },
                {
                        Keys:    bson.D{{Key: "category_id", Value: 1}},
                        Options: options.Index().SetName("idx_product_category"),
                },
                {
                        Keys:    bson.D{{Key: "brand", Value: 1}},
                        Options: options.Index().SetName("idx_product_brand"),
                },
                {
                        Keys:    bson.D{{Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_product_status"),
                },
                {
                        Keys:    bson.D{{Key: "vendor_id", Value: 1}, {Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_product_vendor_status"),
                },
                {
                        Keys:    bson.D{{Key: "lowest_price", Value: 1}},
                        Options: options.Index().SetName("idx_product_lowest_price"),
                },
                {
                        Keys:    bson.D{{Key: "tags", Value: 1}},
                        Options: options.Index().SetName("idx_product_tags"),
                },
                {
                        Keys:    bson.D{{Key: "variants.sku", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_variant_sku_unique"),
                },
                {
                        Keys: bson.D{
                                {Key: "sort_order", Value: 1},
                                {Key: "created_at", Value: -1},
                        },
                        Options: options.Index().SetName("idx_product_sort"),
                },
                // Single compound text index: name, brand, description, short_description, tags.
                // Weights prioritize name > brand > description > short_description > tags.
                // SKU is NOT included (only one text index per collection);
                // SKU search uses regex on the existing unique SKU index.
                {
                        Keys: bson.D{
                                {Key: "name", Value: "text"},
                                {Key: "brand", Value: "text"},
                                {Key: "description", Value: "text"},
                                {Key: "short_description", Value: "text"},
                                {Key: "tags", Value: "text"},
                        },
                        Options: options.Index().SetName("idx_product_search").
                               SetWeights(bson.M{"name": 10, "brand": 5, "description": 3, "short_description": 4, "tags": 2}),
                },
        }
        _, err := coll.Indexes().CreateMany(ctx, indexes)
        if err != nil {
                return fmt.Errorf("failed to create product indexes: %w", err)
        }
        return nil
}
