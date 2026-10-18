package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTService_GenerateAndValidate(t *testing.T) {
	svc := NewJWTService("test-secret-key", 15*time.Minute, 168*time.Hour, "test-issuer")

	pair, err := svc.GenerateTokenPair("user-123", "CUSTOMER", true)
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.False(t, pair.ExpiresAt.IsZero())

	// Validate access token
	claims, err := svc.ValidateToken(pair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "CUSTOMER", claims.Role)
	assert.Equal(t, "access", claims.Type)
	assert.True(t, claims.EmailVerified)

	// Validate refresh token
	refreshClaims, err := svc.ValidateToken(pair.RefreshToken)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", refreshClaims.UserID)
	assert.Equal(t, "refresh", refreshClaims.Type)
}

func TestJWTService_InvalidToken(t *testing.T) {
	svc := NewJWTService("test-secret-key", 15*time.Minute, 168*time.Hour, "test-issuer")

	_, err := svc.ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestJWTService_WrongSecret(t *testing.T) {
	svc1 := NewJWTService("secret-one", 15*time.Minute, 168*time.Hour, "test")
	svc2 := NewJWTService("secret-two", 15*time.Minute, 168*time.Hour, "test")

	pair, _ := svc1.GenerateTokenPair("user-1", "ADMIN", true)

	_, err := svc2.ValidateToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestJWTService_ExpiredToken(t *testing.T) {
	svc := NewJWTService("test-secret", -1*time.Second, -1*time.Second, "test")

	pair, err := svc.GenerateTokenPair("user-1", "CUSTOMER", true)
	assert.NoError(t, err)

	_, err = svc.ValidateToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestJWTService_VerificationToken(t *testing.T) {
	svc := NewJWTService("test", 15*time.Minute, 168*time.Hour, "test")

	token := svc.GenerateVerificationToken()
	assert.NotEmpty(t, token)
	assert.Len(t, token, 64) // 32 bytes = 64 hex chars
}

func TestJWTService_DifferentVerificationTokens(t *testing.T) {
	svc := NewJWTService("test", 15*time.Minute, 168*time.Hour, "test")

	t1 := svc.GenerateVerificationToken()
	t2 := svc.GenerateVerificationToken()
	assert.NotEqual(t, t1, t2)
}
