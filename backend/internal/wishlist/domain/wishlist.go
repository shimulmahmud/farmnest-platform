package domain

import (
	"time"

	"github.com/google/uuid"
)

// WishlistItem represents a single item in a customer's wishlist.
type WishlistItem struct {
	ID        string    `bson:"_id" json:"id"`
	ProductID string    `bson:"product_id" json:"product_id"`
	VariantID string    `bson:"variant_id" json:"variant_id,omitempty"`
	AddedAt   time.Time `bson:"added_at" json:"added_at"`
}

// Wishlist represents a customer's wishlist (one per customer).
type Wishlist struct {
	ID        string         `bson:"_id" json:"id"`
	CustomerID string         `bson:"customer_id" json:"customer_id"`
	Items      []WishlistItem `bson:"items" json:"items"`
	CreatedAt  time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time      `bson:"updated_at" json:"updated_at"`
}

// NewWishlist creates a new empty wishlist for a customer.
func NewWishlist(customerID string) *Wishlist {
	now := time.Now().UTC()
	return &Wishlist{
		ID:         uuid.New().String(),
		CustomerID: customerID,
		Items:      []WishlistItem{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewWishlistItem creates a new wishlist item.
func NewWishlistItem(productID, variantID string) WishlistItem {
	return WishlistItem{
		ID:        uuid.New().String(),
		ProductID: productID,
		VariantID: variantID,
		AddedAt:   time.Now().UTC(),
	}
}

// HasItem checks if the wishlist already contains the given product/variant.
func (w *Wishlist) HasItem(productID, variantID string) bool {
	for _, item := range w.Items {
		if item.ProductID == productID && item.VariantID == variantID {
			return true
		}
	}
	return false
}

// AddItem appends an item and updates the timestamp.
// Returns false if the item already exists.
func (w *Wishlist) AddItem(productID, variantID string) (WishlistItem, bool) {
	if w.HasItem(productID, variantID) {
		return WishlistItem{}, false
	}
	item := NewWishlistItem(productID, variantID)
	w.Items = append(w.Items, item)
	w.UpdatedAt = time.Now().UTC()
	return item, true
}

// RemoveItem removes an item by ID and updates the timestamp.
// Returns true if the item was found and removed.
func (w *Wishlist) RemoveItem(itemID string) bool {
	for i, item := range w.Items {
		if item.ID == itemID {
			w.Items = append(w.Items[:i], w.Items[i+1:]...)
			w.UpdatedAt = time.Now().UTC()
			return true
		}
	}
	return false
}

// FindItemByID returns the item with the given ID, or nil.
func (w *Wishlist) FindItemByID(itemID string) *WishlistItem {
	for i := range w.Items {
		if w.Items[i].ID == itemID {
			return &w.Items[i]
		}
	}
	return nil
}

// ItemCount returns the number of items in the wishlist.
func (w *Wishlist) ItemCount() int {
	return len(w.Items)
}

// --- Domain errors ---

var (
	ErrDuplicateWishlistItem = &WishlistError{Code: "DUPLICATE_ITEM", Message: "item already in wishlist"}
	ErrWishlistNotFound   = &WishlistError{Code: "WISHLIST_NOT_FOUND", Message: "wishlist not found"}
	ErrWishlistItemNotFound = &WishlistError{Code: "ITEM_NOT_FOUND", Message: "wishlist item not found"}
)

// WishlistError represents a domain-level wishlist error.
type WishlistError struct {
	Code    string
	Message string
}

func (e *WishlistError) Error() string {
	return e.Code + ": " + e.Message
}
