package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ecommerce/backend/internal/categories/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type categoryRepository struct {
	coll *mongo.Collection
}

// NewCategoryRepository creates a MongoDB-backed CategoryRepository.
func NewCategoryRepository(db *mongo.Database) domain.CategoryRepository {
	return &categoryRepository{coll: db.Collection("categories")}
}

func (r *categoryRepository) Create(ctx context.Context, category *domain.Category) error {
	_, err := r.coll.InsertOne(ctx, category)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			if strings.Contains(err.Error(), "slug") {
				return errors.Wrap("CONFLICT", "category slug already exists", 409, err)
			}
			return errors.Wrap("CONFLICT", "category with conflicting data already exists", 409, err)
		}
		return errors.Wrap("INTERNAL_ERROR", "failed to create category", 500, err)
	}
	return nil
}

func (r *categoryRepository) FindByID(ctx context.Context, id string) (*domain.Category, error) {
	var category domain.Category
	filter := bson.M{"_id": id}
	if err := r.coll.FindOne(ctx, filter).Decode(&category); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find category", 500, err)
	}
	return &category, nil
}

func (r *categoryRepository) FindBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	var category domain.Category
	filter := bson.M{"slug": slug}
	if err := r.coll.FindOne(ctx, filter).Decode(&category); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find category by slug", 500, err)
	}
	return &category, nil
}

func (r *categoryRepository) Update(ctx context.Context, category *domain.Category) error {
	category.UpdatedAt = time.Now().UTC()
	filter := bson.M{"_id": category.ID}
	update := bson.M{"$set": category}
	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			if strings.Contains(err.Error(), "slug") {
				return errors.Wrap("CONFLICT", "category slug already exists", 409, err)
			}
			return errors.Wrap("CONFLICT", "category with conflicting data already exists", 409, err)
		}
		return errors.Wrap("INTERNAL_ERROR", "failed to update category", 500, err)
	}
	if result.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *categoryRepository) Delete(ctx context.Context, id string) error {
	result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to delete category", 500, err)
	}
	if result.DeletedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *categoryRepository) List(ctx context.Context, status string, p pagination.Params) (*domain.CategoryListResult, error) {
	query := bson.M{}
	if status != "" {
		query["status"] = status
	}

	total, err := r.coll.CountDocuments(ctx, query)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to count categories", 500, err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "sort_order", Value: 1}, {Key: "name", Value: 1}}).
		SetSkip(int64(p.Skip)).
		SetLimit(int64(p.Limit))

	cursor, err := r.coll.Find(ctx, query, opts)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to list categories", 500, err)
	}
	defer cursor.Close(ctx)

	var categories []*domain.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode categories", 500, err)
	}

	return &domain.CategoryListResult{
		Categories: categories,
		Total:      total,
		Page:       p.Page,
		PerPage:    p.PerPage,
	}, nil
}

func (r *categoryRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	count, err := r.coll.CountDocuments(ctx, bson.M{"slug": slug})
	if err != nil {
		return false, errors.Wrap("INTERNAL_ERROR", "failed to check category slug existence", 500, err)
	}
	return count > 0, nil
}

// EnsureIndexes creates required indexes on the categories collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("categories")
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_category_slug_unique"),
		},
		{
			Keys:    bson.D{{Key: "parent_id", Value: 1}},
			Options: options.Index().SetName("idx_category_parent"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_category_status"),
		},
		{
			Keys: bson.D{
				{Key: "sort_order", Value: 1},
				{Key: "name", Value: 1},
			},
			Options: options.Index().SetName("idx_category_sort"),
		},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create category indexes: %w", err)
	}
	return nil
}
