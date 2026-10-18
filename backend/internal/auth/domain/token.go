package domain

import (
	"time"

	"github.com/google/uuid"
)

// TokenType distinguishes access vs refresh tokens.
type TokenType string

const (
	TokenTypeAccess        TokenType = "access"
	TokenTypeRefresh       TokenType = "refresh"
	TokenTypeEmailVerify   TokenType = "email_verify"
	TokenTypePasswordReset TokenType = "password_reset"
)

// RefreshToken represents a stored refresh token for rotation and revocation.
type RefreshToken struct {
	ID        string    `bson:"_id" json:"id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	Token     string    `bson:"token" json:"-"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	Revoked   bool      `bson:"revoked" json:"revoked"`
}

// NewRefreshToken creates a refresh token record.
func NewRefreshToken(userID, token string, ttl time.Duration) *RefreshToken {
	now := time.Now().UTC()
	return &RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		Revoked:   false,
	}
}

// IsExpired checks whether the token has passed its expiration time.
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().UTC().After(rt.ExpiresAt)
}

// VerificationToken represents a token for email verification or password reset.
type VerificationToken struct {
	ID        string    `bson:"_id" json:"id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	Token     string    `bson:"token" json:"-"`
	Type      TokenType `bson:"type" json:"type"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
	Used      bool      `bson:"used" json:"used"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// NewVerificationToken creates an email or password reset token.
func NewVerificationToken(userID, token string, tokenType TokenType, ttl time.Duration) *VerificationToken {
	now := time.Now().UTC()
	return &VerificationToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		Type:      tokenType,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
}

// IsExpired checks whether the verification token has expired.
func (vt *VerificationToken) IsExpired() bool {
	return time.Now().UTC().After(vt.ExpiresAt)
}
