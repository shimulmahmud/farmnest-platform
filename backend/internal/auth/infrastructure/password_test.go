package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordHasher_HashAndVerify(t *testing.T) {
	h := NewPasswordHasher(DefaultCost)

	hash, err := h.Hash("securePassword123")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "securePassword123", hash)

	assert.True(t, h.Verify("securePassword123", hash))
	assert.False(t, h.Verify("wrongPassword", hash))
}

func TestPasswordHasher_DifferentHashes(t *testing.T) {
	h := NewPasswordHasher(DefaultCost)

	h1, _ := h.Hash("password")
	h2, _ := h.Hash("password")

	// bcrypt uses salt, so hashes should differ
	assert.NotEqual(t, h1, h2)
}

func TestPasswordHasher_LowCost(t *testing.T) {
	h := NewPasswordHasher(0)            // Below MinCost
	assert.Equal(t, DefaultCost, h.cost) // Should use default
}

func TestPasswordHasher_VerifyEmpty(t *testing.T) {
	h := NewPasswordHasher(DefaultCost)
	hash, _ := h.Hash("")
	assert.True(t, h.Verify("", hash))
	assert.False(t, h.Verify("x", hash))
}
