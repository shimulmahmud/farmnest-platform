package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ModerationStatus represents the review moderation state.
type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "PENDING"
	ModerationApproved ModerationStatus = "APPROVED"
	ModerationRejected ModerationStatus = "REJECTED"
	ModerationHidden   ModerationStatus = "HIDDEN"
)

// ReviewImage represents an image attached to a review.
type ReviewImage struct {
	URL       string `bson:"url" json:"url"`
	Alt       string `bson:"alt" json:"alt,omitempty"`
	SortOrder int    `bson:"sort_order" json:"sort_order"`
}

// Review represents a product review.
type Review struct {
	ID               string           `bson:"_id" json:"id"`
	ProductID        string           `bson:"product_id" json:"product_id"`
	CustomerID       string           `bson:"customer_id" json:"customer_id"`
	AuthorName       string           `bson:"author_name" json:"author_name"` // e.g. "Rahim U."
	Rating           int              `bson:"rating" json:"rating"`
	Title            string           `bson:"title" json:"title"`
	Comment          string           `bson:"comment" json:"comment"`
	Images           []ReviewImage    `bson:"images" json:"images"`
	VerifiedPurchase bool             `bson:"verified_purchase" json:"verified_purchase"`
	Status           ModerationStatus `bson:"status" json:"status"`
	CreatedAt        time.Time        `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time        `bson:"updated_at" json:"updated_at"`
}

// ReviewListResult holds a paginated review listing.
type ReviewListResult struct {
	Reviews  []*Review
	Total    int64
	Page     int
	PerPage  int
}

// NewReview creates a new review in PENDING moderation status.
func NewReview(productID, customerID, title, comment string, rating int, images []ReviewImage, verifiedPurchase bool) (*Review, error) {
	if rating < 1 || rating > 5 {
		return nil, ErrInvalidRating
	}
	if title == "" {
		return nil, ErrTitleRequired
	}
	now := time.Now().UTC()
	return &Review{
		ID:               uuid.New().String(),
		ProductID:        productID,
		CustomerID:       customerID,
		Rating:           rating,
		Title:            title,
		Comment:          comment,
		Images:           images,
		VerifiedPurchase: verifiedPurchase,
		Status:           ModerationPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// Approve transitions the review to APPROVED status.
func (r *Review) Approve() {
	r.Status = ModerationApproved
	r.UpdatedAt = time.Now().UTC()
}

// Reject transitions the review to REJECTED status.
func (r *Review) Reject() {
	r.Status = ModerationRejected
	r.UpdatedAt = time.Now().UTC()
}

// Hide transitions the review to HIDDEN status.
func (r *Review) Hide() {
	r.Status = ModerationHidden
	r.UpdatedAt = time.Now().UTC()
}

// IsVisible returns true if the review is approved.
func (r *Review) IsVisible() bool {
	return r.Status == ModerationApproved
}

// --- Domain errors ---

var (
	ErrInvalidRating   = &ReviewError{Code: "INVALID_RATING", Message: "rating must be between 1 and 5"}
	ErrTitleRequired   = &ReviewError{Code: "TITLE_REQUIRED", Message: "review title is required"}
	ErrDuplicateReview = &ReviewError{Code: "DUPLICATE_REVIEW", Message: "you have already reviewed this product"}
	ErrReviewNotFound  = &ReviewError{Code: "REVIEW_NOT_FOUND", Message: "review not found"}
)

// ReviewError represents a domain-level review error.
type ReviewError struct {
	Code    string
	Message string
}

func (e *ReviewError) Error() string {
	return e.Code + ": " + e.Message
}

// AuthorNameResolver turns a user ID into a short public display name (e.g. "Rahim U.").
type AuthorNameResolver interface {
	DisplayName(ctx context.Context, userID string) string
}
