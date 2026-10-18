package application

import (
	"context"

	"github.com/ecommerce/backend/internal/reviews/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
)

// Service implements review use cases.
type Service struct {
	repo        domain.ReviewRepository
	vpChecker   domain.VerifiedPurchaseChecker
	names       domain.AuthorNameResolver
	// requireVerified: only customers with a DELIVERED order containing the product may review it.
	requireVerified bool
}

// SetAuthorNames enables public author names on new reviews.
func (s *Service) SetAuthorNames(r domain.AuthorNameResolver) { s.names = r }

// RequireVerifiedPurchase restricts reviewing to customers who received the product.
func (s *Service) RequireVerifiedPurchase(on bool) { s.requireVerified = on }

// NewService creates a review application service.
func NewService(repo domain.ReviewRepository, vpChecker domain.VerifiedPurchaseChecker) *Service {
	return &Service{repo: repo, vpChecker: vpChecker}
}

// CreateReviewInput holds the fields for creating a review.
type CreateReviewInput struct {
	ProductID string
	CustomerID string
	Rating    int
	Title     string
	Comment   string
	Images    []domain.ReviewImage
}

// UpdateReviewInput holds the fields for updating a review.
type UpdateReviewInput struct {
	Rating  *int
	Title   *string
	Comment *string
	Images  *[]domain.ReviewImage
}

// CreateReview validates and creates a new review.
// It checks for duplicate reviews and determines verified purchase status.
func (s *Service) CreateReview(ctx context.Context, input CreateReviewInput) (*domain.Review, error) {
	// Check duplicate: one review per customer per product
	exists, err := s.repo.ExistsByProductAndCustomer(ctx, input.ProductID, input.CustomerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("CONFLICT", "you have already reviewed this product", 409)
	}

	// Determine verified purchase
	verifiedPurchase := false
	if s.vpChecker != nil {
		vp, err := s.vpChecker.IsVerifiedPurchase(ctx, input.CustomerID, input.ProductID)
		if err != nil {
			// Log but don't block — verified purchase is enrichment, not a gate
			vp = false
		}
		verifiedPurchase = vp
	}
	if s.requireVerified && !verifiedPurchase {
		return nil, errors.New("FORBIDDEN", "only customers who have received this product can review it", 403)
	}

	review, err := domain.NewReview(
		input.ProductID,
		input.CustomerID,
		input.Title,
		input.Comment,
		input.Rating,
		input.Images,
		verifiedPurchase,
	)
	if err != nil {
		return nil, errors.New("VALIDATION_ERROR", err.Error(), 422)
	}

	if s.names != nil {
		review.AuthorName = s.names.DisplayName(ctx, input.CustomerID)
	}

	if err := s.repo.Create(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// GetReview retrieves a single review by ID.
func (s *Service) GetReview(ctx context.Context, id string) (*domain.Review, error) {
	return s.repo.FindByID(ctx, id)
}

// ListReviewsByProduct returns approved reviews for a product (public view).
func (s *Service) ListReviewsByProduct(ctx context.Context, productID string, p pagination.Params) (*domain.ReviewListResult, error) {
	return s.repo.ListByProduct(ctx, productID, p)
}

// UpdateReview allows the review author to update their review (rating, title, comment, images).
// Only PENDING or APPROVED reviews can be updated.
// After update, the review goes back to PENDING for re-moderation.
func (s *Service) UpdateReview(ctx context.Context, id, customerID string, input UpdateReviewInput) (*domain.Review, error) {
	review, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Ownership check
	if review.CustomerID != customerID {
		return nil, errors.ErrForbidden
	}

	// Only PENDING or APPROVED reviews can be edited
	if review.Status != domain.ModerationPending && review.Status != domain.ModerationApproved {
		return nil, errors.New("VALIDATION_ERROR", "cannot update a review in "+string(review.Status)+" status", 422)
	}

	// Apply updates
	if input.Rating != nil {
		if *input.Rating < 1 || *input.Rating > 5 {
			return nil, errors.New("VALIDATION_ERROR", "rating must be between 1 and 5", 422)
		}
		review.Rating = *input.Rating
	}
	if input.Title != nil {
		if *input.Title == "" {
			return nil, errors.New("VALIDATION_ERROR", "review title is required", 422)
		}
		review.Title = *input.Title
	}
	if input.Comment != nil {
		review.Comment = *input.Comment
	}
	if input.Images != nil {
		review.Images = *input.Images
	}

	// Reset to PENDING for re-moderation after edit
	if review.Status == domain.ModerationApproved {
		review.Status = domain.ModerationPending
	}

	if err := s.repo.Update(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// DeleteReview removes a review. Only the author or an admin can delete.
func (s *Service) DeleteReview(ctx context.Context, id, customerID string, isAdmin bool) error {
	review, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !isAdmin && review.CustomerID != customerID {
		return errors.ErrForbidden
	}

	return s.repo.Delete(ctx, id)
}
