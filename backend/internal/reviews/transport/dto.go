package transport

import (
	"github.com/ecommerce/backend/internal/reviews/domain"
)

// --- Request DTOs ---

// CreateReviewRequest is the JSON body for POST /api/v1/products/:id/reviews.
type CreateReviewRequest struct {
	Rating  int                `json:"rating" validate:"required"`
	Title   string             `json:"title" validate:"required"`
	Comment string             `json:"comment"`
	Images  []ReviewImageRequest `json:"images"`
}

// ReviewImageRequest is the review image DTO.
type ReviewImageRequest struct {
	URL       string `json:"url" validate:"required"`
	Alt       string `json:"alt"`
	SortOrder int    `json:"sort_order"`
}

// UpdateReviewRequest is the JSON body for PATCH /api/v1/reviews/:id.
type UpdateReviewRequest struct {
	Rating  *int                `json:"rating"`
	Title   *string             `json:"title"`
	Comment *string             `json:"comment"`
	Images  *[]ReviewImageRequest `json:"images"`
}

// --- Response DTOs ---

// ReviewResponse is the review data returned in API responses.
type ReviewResponse struct {
	ID               string              `json:"id"`
	ProductID        string              `json:"product_id"`
	CustomerID       string              `json:"customer_id,omitempty"`
	AuthorName       string              `json:"author_name"`
	Rating           int                 `json:"rating"`
	Title            string              `json:"title"`
	Comment          string              `json:"comment"`
	Images           []ReviewImageResponse `json:"images"`
	VerifiedPurchase bool                `json:"verified_purchase"`
	Status           string              `json:"status"`
	CreatedAt        string              `json:"created_at"`
	UpdatedAt        string              `json:"updated_at"`
}

// ReviewImageResponse is the review image response DTO.
type ReviewImageResponse struct {
	URL       string `json:"url"`
	Alt       string `json:"alt,omitempty"`
	SortOrder int    `json:"sort_order"`
}

// toReviewResponse converts a domain Review to a ReviewResponse.
func toReviewResponse(r *domain.Review) ReviewResponse {
	images := make([]ReviewImageResponse, len(r.Images))
	for i, img := range r.Images {
		images[i] = ReviewImageResponse{
			URL:       img.URL,
			Alt:       img.Alt,
			SortOrder: img.SortOrder,
		}
	}

	return ReviewResponse{
		ID:               r.ID,
		ProductID:        r.ProductID,
		CustomerID:       r.CustomerID,
		AuthorName:       r.AuthorName,
		Rating:           r.Rating,
		Title:            r.Title,
		Comment:          r.Comment,
		Images:           images,
		VerifiedPurchase: r.VerifiedPurchase,
		Status:           string(r.Status),
		CreatedAt:        r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        r.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// toDomainImages converts request image DTOs to domain ReviewImage slice.
func toDomainImages(req []ReviewImageRequest) []domain.ReviewImage {
	if req == nil {
		return nil
	}
	images := make([]domain.ReviewImage, len(req))
	for i, img := range req {
		images[i] = domain.ReviewImage{
			URL:       img.URL,
			Alt:       img.Alt,
			SortOrder: img.SortOrder,
		}
	}
	return images
}
