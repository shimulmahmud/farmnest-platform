package transport

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterWishlistRoutes mounts wishlist endpoints on the given router.
//
// Auth: GET  /api/v1/wishlist
// Auth: POST /api/v1/wishlist/items
// Auth: DELETE /api/v1/wishlist/items/:id
func RegisterWishlistRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler) {
	wishlist := router.Group("/api/v1/wishlist", authMW)
	wishlist.Get("/", handler.GetWishlist)
	wishlist.Post("/items", handler.AddItem)
	wishlist.Delete("/items/:id", handler.RemoveItem)
}
