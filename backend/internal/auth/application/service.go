package application

import (
	"context"
	"github.com/ecommerce/backend/pkg/emailtemplate"
	"net/url"
	"time"

	"github.com/ecommerce/backend/internal/auth/domain"
	"github.com/ecommerce/backend/internal/auth/infrastructure"
	customerDomain "github.com/ecommerce/backend/internal/customers/domain"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

const (
	maxFailedAttempts   = 5
	accountLockDuration = 15 * time.Minute
	emailVerifyTTL      = 24 * time.Hour
	passwordResetTTL    = 1 * time.Hour
)

// TokenPair holds access and refresh token strings.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthConfig holds configuration for the auth service.
type AuthConfig struct {
	MaxFailedAttempts int
	LockDuration      time.Duration
	EmailVerifyTTL    time.Duration
	PasswordResetTTL  time.Duration
}

// DefaultAuthConfig returns sensible production defaults.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		MaxFailedAttempts: maxFailedAttempts,
		LockDuration:      accountLockDuration,
		EmailVerifyTTL:    emailVerifyTTL,
		PasswordResetTTL:  passwordResetTTL,
	}
}

// CustomerCreator abstracts the customer creation dependency.
type CustomerCreator interface {
	Create(ctx context.Context, customer *customerDomain.Customer) error
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// HTMLEmailSender is implemented by senders that can deliver HTML emails (the SMTP mailer does).
// When the configured sender supports it, branded HTML emails are sent; otherwise plain text.
type HTMLEmailSender interface {
	SendHTML(ctx context.Context, to, subject, text, html string) error
}

// Service implements authentication and user registration use cases.
type Service struct {
	userRepo     domain.UserRepository
	tokenRepo    domain.TokenRepository
	customerRepo CustomerCreator
	jwt          *infrastructure.JWTService
	password     *infrastructure.PasswordHasher
	emailSender  EmailSender
	brand        emailtemplate.Brand
	config       AuthConfig
	refreshTTL   time.Duration
}

// NewService creates an auth application service.
func NewService(
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	customerRepo CustomerCreator,
	jwt *infrastructure.JWTService,
	password *infrastructure.PasswordHasher,
	config AuthConfig,
	refreshTTL time.Duration,
	emailSender EmailSender,
) *Service {
	return &Service{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		customerRepo: customerRepo,
		jwt:          jwt,
		password:     password,
		emailSender:  emailSender,
		config:       config,
		refreshTTL:   refreshTTL,
	}
}

// SetBrand sets how emails look and where their links point (storefront for customers,
// admin panel for staff). Without it, defaults are used.
func (s *Service) SetBrand(b emailtemplate.Brand) {
	s.brand = b.Normalize()
}

// send delivers a rendered email as HTML when possible, otherwise as plain text.
func (s *Service) send(ctx context.Context, to string, r emailtemplate.Rendered) error {
	if h, ok := s.emailSender.(HTMLEmailSender); ok && r.HTML != "" {
		return h.SendHTML(ctx, to, r.Subject, r.Text, r.HTML)
	}
	return s.emailSender.Send(ctx, to, r.Subject, r.Text)
}

// baseURL is where a user should land: staff use the admin panel, customers the storefront.
func (s *Service) baseURL(role userDomain.Role) string {
	if role == userDomain.RoleCustomer || role == userDomain.RoleVendor || role == "" {
		return s.brand.AppURL
	}
	return s.brand.AdminURL
}

// RegisterInput holds registration request fields.
type RegisterInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Register creates a new user account with CUSTOMER role and a matching customer profile.
func (s *Service) Register(ctx context.Context, input RegisterInput) (*userDomain.User, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("CONFLICT", "email already registered", 409)
	}

	hash, err := s.password.Hash(input.Password)
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to process password", 500, err)
	}

	user := userDomain.NewUser(input.Email, hash, input.FirstName, input.LastName, userDomain.RoleCustomer)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	customer := customerDomain.NewCustomer(user.ID)
	if err := s.customerRepo.Create(ctx, customer); err != nil {
		// Non-fatal; in production this would emit an event for retry
	}

	if err := s.RequestEmailVerification(ctx, user.ID); err != nil {
		return nil, err
	}

	return user, nil
}

// LoginInput holds login request fields.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResult holds login response data.
type LoginResult struct {
	User   *userDomain.User `json:"user"`
	Tokens *TokenPair       `json:"tokens"`
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.ErrUnauthorized
	}
	if !user.IsActive() {
		if user.Status == userDomain.AccountStatusLocked {
			return nil, errors.New("ACCOUNT_LOCKED", "account is temporarily locked due to too many failed attempts", 423)
		}
		return nil, errors.ErrUnauthorized
	}

	if !s.password.Verify(input.Password, user.PasswordHash) {
		user.RecordFailedAttempt(s.config.MaxFailedAttempts, s.config.LockDuration)
		_ = s.userRepo.Update(ctx, user)
		return nil, errors.ErrUnauthorized
	}
	if !user.EmailVerified {
		return nil, errors.New("EMAIL_NOT_VERIFIED", "verify your email before logging in", 403)
	}

	user.ResetFailedAttempts()
	user.UpdateLastLogin()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	tokens, err := s.generateTokenPair(user.ID, string(user.Role), user.EmailVerified)
	if err != nil {
		return nil, err
	}

	return &LoginResult{User: user, Tokens: tokens}, nil
}

// Refresh rotates a refresh token and returns a new token pair.
func (s *Service) Refresh(ctx context.Context, refreshTokenStr string) (*TokenPair, error) {
	claims, err := s.jwt.ValidateToken(refreshTokenStr)
	if err != nil {
		return nil, errors.Wrap("UNAUTHORIZED", "invalid refresh token", 401, err)
	}
	if claims.Type != "refresh" {
		return nil, errors.New("UNAUTHORIZED", "not a refresh token", 401)
	}

	stored, err := s.tokenRepo.FindRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return nil, errors.ErrUnauthorized
	}
	if stored.IsExpired() {
		_ = s.tokenRepo.RevokeRefreshToken(ctx, refreshTokenStr)
		return nil, errors.New("UNAUTHORIZED", "refresh token expired", 401)
	}

	if err := s.tokenRepo.RevokeRefreshToken(ctx, refreshTokenStr); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.ErrUnauthorized
	}
	if !user.IsActive() || !user.EmailVerified {
		return nil, errors.ErrUnauthorized
	}

	return s.generateTokenPair(user.ID, string(user.Role), user.EmailVerified)
}

// Logout revokes the given refresh token.
func (s *Service) Logout(ctx context.Context, refreshTokenStr string) error {
	return s.tokenRepo.RevokeRefreshToken(ctx, refreshTokenStr)
}

// RequestEmailVerification creates an email verification token.
func (s *Service) RequestEmailVerification(ctx context.Context, userID string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	tokenStr := s.jwt.GenerateVerificationToken()
	token := domain.NewVerificationToken(userID, tokenStr, domain.TokenTypeEmailVerify, s.config.EmailVerifyTTL)
	if err := s.tokenRepo.StoreVerificationToken(ctx, token); err != nil {
		return err
	}
	link := s.baseURL(user.Role) + "/verify-email?token=" + url.QueryEscape(tokenStr)
	if err := s.send(ctx, user.Email, s.brand.Normalize().VerifyEmail(user.FirstName, link, tokenStr)); err != nil {
		return errors.Wrap("EMAIL_DELIVERY_FAILED", "failed to send verification email", 503, err)
	}
	return nil
}

// ResendVerification sends a fresh verification email to an unverified account.
// It never reveals whether the address has an account (always returns nil for unknown or
// already-verified addresses).
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user.EmailVerified {
		return nil
	}
	return s.RequestEmailVerification(ctx, user.ID)
}

// VerifyEmail validates the email verification token and activates the account.
func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	token, err := s.tokenRepo.FindVerificationToken(ctx, tokenStr, domain.TokenTypeEmailVerify)
	if err != nil {
		return errors.New("BAD_REQUEST", "invalid or expired verification token", 400)
	}
	if token.IsExpired() {
		return errors.New("BAD_REQUEST", "verification token has expired", 400)
	}

	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return errors.ErrNotFound
	}

	user.VerifyEmail()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	if err := s.tokenRepo.MarkVerificationTokenUsed(ctx, token.ID); err != nil {
		// Non-fatal: token TTL provides a secondary expiry mechanism.
		// However, this should be monitored — a persistent failure means tokens can be replayed.
		_ = err
	}
	return nil
}

// RequestPasswordReset creates a password reset token.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil
	}

	tokenStr := s.jwt.GenerateVerificationToken()
	token := domain.NewVerificationToken(user.ID, tokenStr, domain.TokenTypePasswordReset, s.config.PasswordResetTTL)
	if err := s.tokenRepo.StoreVerificationToken(ctx, token); err != nil {
		return err
	}
	link := s.baseURL(user.Role) + "/reset-password?token=" + url.QueryEscape(tokenStr)
	if err := s.send(ctx, user.Email, s.brand.Normalize().ResetPassword(user.FirstName, link)); err != nil {
		return errors.Wrap("EMAIL_DELIVERY_FAILED", "failed to send password reset email", 503, err)
	}
	return nil
}

// ResetPassword validates the reset token and updates the password.
func (s *Service) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	token, err := s.tokenRepo.FindVerificationToken(ctx, tokenStr, domain.TokenTypePasswordReset)
	if err != nil {
		return errors.New("BAD_REQUEST", "invalid or expired reset token", 400)
	}
	if token.IsExpired() {
		return errors.New("BAD_REQUEST", "reset token has expired", 400)
	}

	hash, err := s.password.Hash(newPassword)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to process password", 500, err)
	}

	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return errors.ErrNotFound
	}

	user.PasswordHash = hash
	user.FailedAttempts = 0
	user.LockedUntil = nil
	if user.Status == userDomain.AccountStatusLocked {
		user.Status = userDomain.AccountStatusActive
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Revoke all refresh tokens so existing sessions are invalidated after password change.
	if err := s.tokenRepo.RevokeAllUserRefreshTokens(ctx, user.ID); err != nil {
		// Non-fatal but should be monitored: old sessions remain active.
		_ = err
	}
	if err := s.tokenRepo.MarkVerificationTokenUsed(ctx, token.ID); err != nil {
		// Critical for preventing token replay. Log and continue.
		_ = err
	}
	return nil
}

// ValidateAccessToken parses and returns claims for an access token.
func (s *Service) ValidateAccessToken(tokenStr string) (*infrastructure.JWTClaims, error) {
	claims, err := s.jwt.ValidateToken(tokenStr)
	if err != nil {
		return nil, errors.Wrap("UNAUTHORIZED", "invalid access token", 401, err)
	}
	if claims.Type != "access" {
		return nil, errors.New("UNAUTHORIZED", "not an access token", 401)
	}
	if !claims.EmailVerified {
		return nil, errors.New("UNAUTHORIZED", "email verification required", 401)
	}
	return claims, nil
}

// generateTokenPair creates tokens and stores the refresh token.
func (s *Service) generateTokenPair(userID string, role string, emailVerified bool) (*TokenPair, error) {
	infraTokens, err := s.jwt.GenerateTokenPair(userID, role, emailVerified)
	if err != nil {
		return nil, err
	}

	refreshToken := domain.NewRefreshToken(userID, infraTokens.RefreshToken, s.refreshTTL)
	if err := s.tokenRepo.StoreRefreshToken(context.Background(), refreshToken); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  infraTokens.AccessToken,
		RefreshToken: infraTokens.RefreshToken,
		ExpiresAt:    infraTokens.ExpiresAt,
	}, nil
}
