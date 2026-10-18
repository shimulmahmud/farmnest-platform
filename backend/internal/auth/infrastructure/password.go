package infrastructure

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the bcrypt cost factor for production.
const DefaultCost = 12

// PasswordHasher provides bcrypt-based password operations.
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a PasswordHasher with the given bcrypt cost.
func NewPasswordHasher(cost int) *PasswordHasher {
	if cost < bcrypt.MinCost {
		cost = DefaultCost
	}
	return &PasswordHasher{cost: cost}
}

// Hash returns the bcrypt hash of a plaintext password.
func (h *PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// Verify compares a plaintext password against a bcrypt hash.
func (h *PasswordHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
