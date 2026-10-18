package transport

import "github.com/ecommerce/backend/internal/categories/domain"

// --- Request DTOs ---

// CreateCategoryRequest is the JSON body for POST /api/v1/admin/categories.
type CreateCategoryRequest struct {
	Name        string             `json:"name" validate:"required"`
	Slug        string             `json:"slug" validate:"required"`
	Description string             `json:"description"`
	ParentID    *string            `json:"parent_id"`
	Image       string             `json:"image"`
	SortOrder   int                `json:"sort_order"`
	SEO         CategorySEORequest `json:"seo"`
}

// UpdateCategoryRequest is the JSON body for PATCH /api/v1/admin/categories/:id.
type UpdateCategoryRequest struct {
	Name        *string             `json:"name"`
	Slug        *string             `json:"slug"`
	Description *string             `json:"description"`
	ParentID    *string             `json:"parent_id"`
	Image       *string             `json:"image"`
	SortOrder   *int                `json:"sort_order"`
	Status      *string             `json:"status"`
	SEO         *CategorySEORequest `json:"seo"`
}

// CategorySEORequest is the category SEO metadata DTO.
type CategorySEORequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
}

// --- Response DTOs ---

// CategoryResponse is the category data returned in API responses.
type CategoryResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Slug        string              `json:"slug"`
	Description string              `json:"description"`
	ParentID    *string             `json:"parent_id,omitempty"`
	Image       string              `json:"image"`
	SortOrder   int                 `json:"sort_order"`
	Status      string              `json:"status"`
	SEO         CategorySEOResponse `json:"seo"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

// CategorySEOResponse is the category SEO metadata response DTO.
type CategorySEOResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
}

// toSEOInput converts a request SEO DTO to domain SEO.
func toSEOInput(r CategorySEORequest) domain.CategorySEO {
	return domain.CategorySEO{
		Title:       r.Title,
		Description: r.Description,
		Keywords:    r.Keywords,
	}
}

// toSEOOutput converts a domain SEO to response SEO DTO.
func toSEOOutput(s domain.CategorySEO) CategorySEOResponse {
	return CategorySEOResponse{
		Title:       s.Title,
		Description: s.Description,
		Keywords:    s.Keywords,
	}
}
