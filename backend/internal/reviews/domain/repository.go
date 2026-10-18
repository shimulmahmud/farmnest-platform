package domain

import (
        "context"

        "github.com/ecommerce/backend/pkg/pagination"
)

// ReviewRepository defines the persistence operations for reviews.
type ReviewRepository interface {
        // Create inserts a new review.
        Create(ctx context.Context, review *Review) error

        // FindByID retrieves a review by ID.
        FindByID(ctx context.Context, id string) (*Review, error)

        // Update persists changes to an existing review.
        Update(ctx context.Context, review *Review) error

        // Delete removes a review by ID.
        Delete(ctx context.Context, id string) error

        // ListByProduct returns visible (APPROVED) reviews for a product with pagination.
        ListByProduct(ctx context.Context, productID string, p pagination.Params) (*ReviewListResult, error)

        // ListByProductAdmin returns all reviews for a product (any status) with pagination.
        ListByProductAdmin(ctx context.Context, productID string, p pagination.Params) (*ReviewListResult, error)

        // ListAll returns all reviews across all products with optional status filter and pagination.
        // Used by admins for review moderation.
        ListAll(ctx context.Context, status ModerationStatus, p pagination.Params) (*ReviewListResult, error)

        // ExistsByProductAndCustomer checks if a customer has already reviewed a product.
        ExistsByProductAndCustomer(ctx context.Context, productID, customerID string) (bool, error)
}

// VerifiedPurchaseChecker determines if a customer purchased a given product.
// Implemented by querying completed orders.
type VerifiedPurchaseChecker interface {
        IsVerifiedPurchase(ctx context.Context, customerID, productID string) (bool, error)
}
