package transport

import "github.com/gofiber/fiber/v2"

// RegisterCategoryRoutes mounts category endpoints on the given router.
func RegisterCategoryRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler) {
	// Public routes
	categories := router.Group("/api/v1/categories")
	categories.Get("/", handler.ListCategories)
	categories.Get("/:slug", handler.GetCategory)

	// Admin routes
	admin := router.Group("/api/v1/admin/categories", authMW, adminMW)
	admin.Post("/", handler.CreateCategory)
	admin.Patch("/:id", handler.UpdateCategory)
	admin.Delete("/:id", handler.DeleteCategory)
}