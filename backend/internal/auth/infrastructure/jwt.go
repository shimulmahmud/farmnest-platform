package infrastructure

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims holds the claims stored in access and refresh JWTs.
type JWTClaims struct {
        UserID        string `json:"uid"`
        Role          string `json:"role"`
        Type          string `json:"type"`
        EmailVerified bool   `json:"email_verified"`
        jwt.RegisteredClaims
}

// TokenPair holds an access and refresh token string.
type TokenPair struct {
        AccessToken  string    `json:"access_token"`
        RefreshToken string    `json:"refresh_token"`
        ExpiresAt    time.Time `json:"expires_at"`
}

// JWTService generates and validates JWT tokens.
type JWTService struct {
        secret     []byte
        accessTTL  time.Duration
        refreshTTL time.Duration
        issuer     string
}

// NewJWTService creates a JWTService from the given config.
func NewJWTService(secret string, accessTTL, refreshTTL time.Duration, issuer string) *JWTService {
        return &JWTService{
                secret:     []byte(secret),
                accessTTL:  accessTTL,
                refreshTTL: refreshTTL,
                issuer:     issuer,
        }
}

// GenerateTokenPair creates a new access+refresh token pair.
func (s *JWTService) GenerateTokenPair(userID string, role string, emailVerified bool) (*TokenPair, error) {
        now := time.Now().UTC()
        accessExpiry := now.Add(s.accessTTL)

        accessClaims := JWTClaims{
                UserID: userID,
                Role:   role,
                Type:   "access",
                        EmailVerified: emailVerified,
                RegisteredClaims: jwt.RegisteredClaims{
                        Issuer:    s.issuer,
                        Subject:   userID,
                        ExpiresAt: jwt.NewNumericDate(accessExpiry),
                        IssuedAt:  jwt.NewNumericDate(now),
                        NotBefore: jwt.NewNumericDate(now),
                        ID:        newJTI(),
                },
        }

        accessToken, err := s.signToken(accessClaims)
        if err != nil {
                return nil, fmt.Errorf("failed to generate access token: %w", err)
        }

        refreshExpiry := now.Add(s.refreshTTL)
        refreshClaims := JWTClaims{
                UserID: userID,
                Role:   role,
                Type:   "refresh",
                        EmailVerified: emailVerified,
                RegisteredClaims: jwt.RegisteredClaims{
                        Issuer:    s.issuer,
                        Subject:   userID,
                        ExpiresAt: jwt.NewNumericDate(refreshExpiry),
                        IssuedAt:  jwt.NewNumericDate(now),
                        NotBefore: jwt.NewNumericDate(now),
                        ID:        newJTI(),
                },
        }

        refreshToken, err := s.signToken(refreshClaims)
        if err != nil {
                return nil, fmt.Errorf("failed to generate refresh token: %w", err)
        }

        return &TokenPair{
                AccessToken:  accessToken,
                RefreshToken: refreshToken,
                ExpiresAt:    accessExpiry,
        }, nil
}

// ValidateToken parses and validates a JWT, returning its claims.
func (s *JWTService) ValidateToken(tokenStr string) (*JWTClaims, error) {
        token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
                }
                return s.secret, nil
        })
        if err != nil {
                return nil, fmt.Errorf("invalid token: %w", err)
        }

        claims, ok := token.Claims.(*JWTClaims)
        if !ok || !token.Valid {
                return nil, fmt.Errorf("invalid token claims")
        }
        if s.issuer != "" && claims.Issuer != s.issuer {
                return nil, fmt.Errorf("invalid token issuer: got %q, expected %q", claims.Issuer, s.issuer)
        }
        return claims, nil
}

// GenerateVerificationToken creates a random token string for email verification or password reset.
func (s *JWTService) GenerateVerificationToken() string {
        return randomHex(32)
}

func (s *JWTService) signToken(claims JWTClaims) (string, error) {
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        return token.SignedString(s.secret)
}

func newJTI() string {
        return randomHex(16)
}

func randomHex(n int) string {
        b := make([]byte, n)
        _, _ = rand.Read(b)
        return fmt.Sprintf("%x", b)
}
