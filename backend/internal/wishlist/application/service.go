package application

import (
	"context"

	"github.com/ecommerce/backend/internal/wishlist/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

// Service implements wishlist use cases.
type Service struct {
	repo     domain.WishlistRepository
	validator domain.ProductValidator
	publisher domain.EventPublisher
}

// NewService creates a wishlist application service.
func NewService(repo domain.WishlistRepository, validator domain.ProductValidator, publisher domain.EventPublisher) *Service {
	return &Service{
		repo:     repo,
		validator: validator,
		publisher: publisher,
	}
}

// GetWishlist retrieves or creates the customer's wishlist.
func (s *Service) GetWishlist(ctx context.Context, customerID string) (*domain.Wishlist, error) {
	wishlist, err := s.repo.FindByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if wishlist != nil {
		return wishlist, nil
	}

	// Create an empty wishlist on first access
	wishlist = domain.NewWishlist(customerID)
	if err := s.repo.Create(ctx, wishlist); err != nil {
		return nil, err
	}
	return wishlist, nil
}

// AddItemInput holds the fields for adding a wishlist item.
type AddItemInput struct {
	ProductID string
	VariantID string
}

// AddItem validates and adds a product to the wishlist.
// It validates product existence (and variant if provided), prevents duplicates,
// and optionally publishes a wishlist.added event.
func (s *Service) AddItem(ctx context.Context, customerID string, input AddItemInput) (*domain.WishlistItem, error) {
	// Validate product exists
	if s.validator != nil {
		exists, err := s.validator.ProductExists(ctx, input.ProductID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("VALIDATION_ERROR", "product not found", 422)
		}

		// Validate variant if provided
		if input.VariantID != "" {
			vExists, err := s.validator.VariantExists(ctx, input.VariantID)
			if err != nil {
				return nil, err
			}
			if !vExists {
				return nil, errors.New("VALIDATION_ERROR", "variant not found", 422)
			}
		}
	}

	// Check duplicate at application level (DB also enforces via index)
	dup, err := s.repo.ExistsByCustomerAndProduct(ctx, customerID, input.ProductID, input.VariantID)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, errors.New("CONFLICT", "item already in wishlist", 409)
	}

	item := domain.NewWishlistItem(input.ProductID, input.VariantID)

	if err := s.repo.AddItem(ctx, customerID, item); err != nil {
		// Handle race condition: duplicate inserted between our check and the DB write
		if err == domain.ErrDuplicateWishlistItem {
			return nil, errors.New("CONFLICT", "item already in wishlist", 409)
		}
		return nil, err
	}

	// Publish event (non-blocking — don't fail the request)
	if s.publisher != nil {
		_ = s.publisher.PublishAdded(ctx, &domain.WishlistEvent{
			EventType:  "wishlist.added",
			CustomerID: customerID,
			ProductID:  input.ProductID,
			VariantID:  input.VariantID,
			ItemID:     item.ID,
		})
	}

	return &item, nil
}

// RemoveItem removes an item from the customer's wishlist.
// It optionally publishes a wishlist.removed event.
func (s *Service) RemoveItem(ctx context.Context, customerID, itemID string) error {
	// Verify the item exists in the customer's wishlist
	wishlist, err := s.repo.FindByCustomer(ctx, customerID)
	if err != nil {
		return err
	}
	if wishlist == nil {
		return domain.ErrWishlistNotFound
	}

	target := wishlist.FindItemByID(itemID)
	if target == nil {
		return domain.ErrWishlistItemNotFound
	}

	if err := s.repo.RemoveItem(ctx, customerID, itemID); err != nil {
		return err
	}

	// Publish event (non-blocking)
	if s.publisher != nil {
		_ = s.publisher.PublishRemoved(ctx, &domain.WishlistEvent{
			EventType:  "wishlist.removed",
			CustomerID: customerID,
			ProductID:  target.ProductID,
			VariantID:  target.VariantID,
			ItemID:     itemID,
		})
	}

	return nil
}
