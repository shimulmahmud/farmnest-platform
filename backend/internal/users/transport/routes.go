package transport

import "github.com/gofiber/fiber/v2"

// RegisterUserRoutes mounts user profile endpoints on the given router.
func RegisterUserRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler) {
	me := router.Group("/api/v1/me", authMW)
	me.Get("/", handler.GetMe)
	me.Patch("/", handler.UpdateMe)
}
