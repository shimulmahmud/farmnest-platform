package transport

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterReviewRoutes mounts review endpoints on the given router.
//
// Public:  GET /api/v1/products/:id/reviews
// Auth:    POST /api/v1/products/:id/reviews
// Auth:    PATCH /api/v1/reviews/:id
// Auth:    DELETE /api/v1/reviews/:id
// Auth:    GET /api/v1/reviews/:id
func RegisterReviewRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler) {
	// Public: list approved reviews for a product
	router.Get("/api/v1/products/:id/reviews", handler.ListReviews)

	// Authenticated: create a review
	reviewAuth := router.Group("/api/v1/products/:id/reviews", authMW)
	reviewAuth.Post("/", handler.CreateReview)

	// Authenticated: single review operations
	reviewsGroup := router.Group("/api/v1/reviews", authMW)
	reviewsGroup.Get("/:id", handler.GetReview)
	reviewsGroup.Patch("/:id", handler.UpdateReview)
	reviewsGroup.Delete("/:id", handler.DeleteReview)
}
