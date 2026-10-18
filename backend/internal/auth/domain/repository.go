package domain

import (
	"context"

	"github.com/ecommerce/backend/internal/users/domain"
)

// TokenRepository handles persistence of refresh and verification tokens.
// It is implemented in infrastructure and MUST NOT depend on transport or Fiber.
type TokenRepository interface {
	// Refresh tokens
	StoreRefreshToken(ctx context.Context, token *RefreshToken) error
	FindRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string) error

	// Verification tokens
	StoreVerificationToken(ctx context.Context, token *VerificationToken) error
	FindVerificationToken(ctx context.Context, token string, tokenType TokenType) (*VerificationToken, error)
	MarkVerificationTokenUsed(ctx context.Context, id string) error
}

// UserRepository is an alias for the user domain repository.
// Auth service depends on this to look up users during login/registration.
// This keeps the dependency on the users domain, which is acceptable since
// auth is a cross-cutting concern that needs to read user data.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
