package transport

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RegisterAuthRoutes mounts auth endpoints on the given router.
func RegisterAuthRoutes(router fiber.Router, handler *Handler) {
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 60,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})

	auth := router.Group("/api/v1/auth")
	auth.Post("/register", authLimiter, handler.Register)
	auth.Post("/login", authLimiter, handler.Login)
	auth.Post("/refresh", authLimiter, handler.Refresh)
	auth.Post("/logout", authLimiter, handler.Logout)
	auth.Post("/verify-email", authLimiter, handler.VerifyEmail)
	auth.Post("/forgot-password", authLimiter, handler.ForgotPassword)
	auth.Post("/resend-verification", authLimiter, handler.ResendVerification)
	auth.Post("/reset-password", authLimiter, handler.ResetPassword)
}
