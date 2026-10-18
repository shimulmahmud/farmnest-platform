package domain

import (
	"context"
)

// WishlistRepository defines the persistence operations for wishlists.
type WishlistRepository interface {
	// FindByCustomer retrieves the wishlist for a customer, or nil if none exists.
	FindByCustomer(ctx context.Context, customerID string) (*Wishlist, error)

	// Create inserts a new wishlist.
	Create(ctx context.Context, wishlist *Wishlist) error

	// Update persists changes to an existing wishlist.
	Update(ctx context.Context, wishlist *Wishlist) error

	// AddItem atomically pushes a new item to the wishlist (prevents duplicates at DB level).
	// Returns ErrDuplicateWishlistItem if the item already exists.
	AddItem(ctx context.Context, customerID string, item WishlistItem) error

	// RemoveItem atomically pulls an item from the wishlist by item ID.
	// Returns ErrWishlistItemNotFound if the item doesn't exist.
	RemoveItem(ctx context.Context, customerID, itemID string) error

	// ExistsByCustomerAndProduct checks if a product/variant is already in the customer's wishlist.
	ExistsByCustomerAndProduct(ctx context.Context, customerID, productID, variantID string) (bool, error)
}

// ProductValidator checks if a product or variant exists.
// This is an optional dependency — if nil, product/variant validation is skipped.
type ProductValidator interface {
	ProductExists(ctx context.Context, productID string) (bool, error)
	VariantExists(ctx context.Context, variantID string) (bool, error)
}

// EventPublisher publishes wishlist lifecycle events.
type EventPublisher interface {
	PublishAdded(ctx context.Context, event *WishlistEvent) error
	PublishRemoved(ctx context.Context, event *WishlistEvent) error
}

// WishlistEvent is published on wishlist changes.
type WishlistEvent struct {
	EventType  string `json:"event_type"`
	CustomerID string `json:"customer_id"`
	ProductID  string `json:"product_id"`
	VariantID  string `json:"variant_id,omitempty"`
	ItemID     string `json:"item_id"`
}
