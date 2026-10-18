package transport

import (
	"strings"

	"github.com/ecommerce/backend/internal/auth/application"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

const AuthContextKey = "auth"

// AuthMiddleware validates the Bearer token and injects user info into context.
func AuthMiddleware(service *application.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Error(c, errors.ErrUnauthorized)
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return response.Error(c, errors.ErrUnauthorized)
		}

		claims, err := service.ValidateAccessToken(parts[1])
		if err != nil {
			return response.Error(c, err)
		}

		c.Locals(AuthContextKey, &AuthContext{
			UserID: claims.UserID,
			Role:   userDomain.Role(claims.Role),
		})
		return c.Next()
	}
}

// RequireRole returns middleware that checks if the authenticated user has one of the allowed roles.
func RequireRole(allowedRoles ...userDomain.Role) fiber.Handler {
	allowed := make(map[userDomain.Role]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = true
	}

	return func(c *fiber.Ctx) error {
		auth, ok := c.Locals(AuthContextKey).(*AuthContext)
		if !ok || auth == nil {
			return response.Error(c, errors.ErrUnauthorized)
		}

		if !allowed[auth.Role] {
			return response.Error(c, errors.ErrForbidden)
		}

		return c.Next()
	}
}

// AuthContext holds authenticated user information extracted from JWT.
type AuthContext struct {
	UserID string
	Role   userDomain.Role
}
