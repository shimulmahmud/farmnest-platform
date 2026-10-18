package domain

import (
	"time"

	"github.com/google/uuid"
)

// CategoryStatus represents the visibility of a category.
type CategoryStatus string

const (
	CategoryStatusActive   CategoryStatus = "ACTIVE"
	CategoryStatusInactive CategoryStatus = "INACTIVE"
)

// Category represents a product category in the catalog.
type Category struct {
	ID          string         `bson:"_id" json:"id"`
	Name        string         `bson:"name" json:"name"`
	Slug        string         `bson:"slug" json:"slug"`
	Description string         `bson:"description" json:"description"`
	ParentID    *string        `bson:"parent_id" json:"parent_id,omitempty"`
	Image       string         `bson:"image" json:"image"`
	SortOrder   int            `bson:"sort_order" json:"sort_order"`
	Status      CategoryStatus `bson:"status" json:"status"`
	SEO         CategorySEO    `bson:"seo" json:"seo"`
	CreatedAt   time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `bson:"updated_at" json:"updated_at"`
}

// CategorySEO holds SEO metadata for a category.
type CategorySEO struct {
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description" json:"description"`
	Keywords    string `bson:"keywords" json:"keywords"`
}

// NewCategory creates a new Category with generated ID and timestamps.
func NewCategory(name, slug, description string, parentID *string, image string, sortOrder int, seo CategorySEO) *Category {
	now := time.Now().UTC()
	return &Category{
		ID:          uuid.New().String(),
		Name:        name,
		Slug:        slug,
		Description: description,
		ParentID:    parentID,
		Image:       image,
		SortOrder:   sortOrder,
		Status:      CategoryStatusActive,
		SEO:         seo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsActive returns true if the category is visible.
func (c *Category) IsActive() bool {
	return c.Status == CategoryStatusActive
}