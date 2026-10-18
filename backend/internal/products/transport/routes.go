package transport

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterProductRoutes mounts product endpoints on the given router.
//
//	public : GET /api/v1/products, /search, /:slug     (ACTIVE only, no cost/stock)
//	admin  : full CRUD on any product
//	vendor : full CRUD on the caller's OWN products (vendorMW sets Locals("vendor_id"))
func RegisterProductRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler, vendorMW fiber.Handler) {
	products := router.Group("/api/v1/products")
	products.Get("/", handler.ListProducts)
	products.Get("/search", handler.SearchProducts)
	products.Get("/:slug", handler.GetProduct)

	admin := router.Group("/api/v1/admin/products", authMW, adminMW)
	admin.Get("/", handler.ListManagedProducts)
	admin.Get("/:id", handler.GetManagedProduct)
	admin.Post("/", handler.CreateProduct)
	admin.Patch("/:id", handler.UpdateProduct)
	admin.Delete("/:id", handler.DeleteProduct)

	// requireScope is a belt-and-braces guard: a vendor route must never run
	// unscoped, otherwise a vendor would get admin-level access.
	requireScope := func(c *fiber.Ctx) error {
		if vendorScope(c) == "" {
			return fiber.ErrForbidden
		}
		return c.Next()
	}
	vendor := router.Group("/api/v1/vendor/products", authMW, vendorMW, requireScope)
	vendor.Get("/", handler.ListManagedProducts)
	vendor.Get("/:id", handler.GetManagedProduct)
	vendor.Post("/", handler.CreateProduct)
	vendor.Patch("/:id", handler.UpdateProduct)
	vendor.Delete("/:id", handler.DeleteProduct)
}
