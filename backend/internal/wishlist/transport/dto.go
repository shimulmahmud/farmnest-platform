package transport

import (
	"github.com/ecommerce/backend/internal/wishlist/domain"
)

// --- Request DTOs ---

// AddWishlistItemRequest is the JSON body for POST /api/v1/wishlist/items.
type AddWishlistItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id"`
}

// --- Response DTOs ---

// WishlistResponse is the wishlist data returned in API responses.
type WishlistResponse struct {
	ID         string              `json:"id"`
	CustomerID string              `json:"customer_id"`
	Items      []WishlistItemResponse `json:"items"`
	CreatedAt  string              `json:"created_at"`
	UpdatedAt  string              `json:"updated_at"`
}

// WishlistItemResponse is a single wishlist item in API responses.
type WishlistItemResponse struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id,omitempty"`
	AddedAt   string `json:"added_at"`
}

// WishlistItemAddedResponse is returned when an item is added.
type WishlistItemAddedResponse struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id,omitempty"`
	AddedAt   string `json:"added_at"`
}

// toWishlistResponse converts a domain Wishlist to a WishlistResponse.
func toWishlistResponse(w *domain.Wishlist) WishlistResponse {
	items := make([]WishlistItemResponse, len(w.Items))
	for i, item := range w.Items {
		items[i] = WishlistItemResponse{
			ID:        item.ID,
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			AddedAt:   item.AddedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	return WishlistResponse{
		ID:         w.ID,
		CustomerID: w.CustomerID,
		Items:      items,
		CreatedAt:  w.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  w.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// toItemAddedResponse converts a domain WishlistItem to an added response.
func toItemAddedResponse(item *domain.WishlistItem) WishlistItemAddedResponse {
	return WishlistItemAddedResponse{
		ID:        item.ID,
		ProductID: item.ProductID,
		VariantID: item.VariantID,
		AddedAt:   item.AddedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
