package transport

import (
	"regexp"
	"unicode"

	"github.com/ecommerce/backend/internal/auth/application"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/ecommerce/backend/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler implements the HTTP handlers for auth endpoints.
type Handler struct {
        service   *application.Service
        validator *validator.Validator
}

// NewHandler creates an auth transport handler.
func NewHandler(service *application.Service) *Handler {
        return &Handler{
                service:   service,
                validator: validator.New(),
        }
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(c *fiber.Ctx) error {
        var req RegisterRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "email":      req.Email,
                "password":   req.Password,
                "first_name": req.FirstName,
                "last_name":  req.LastName,
        }); err != nil {
                return response.Error(c, err)
        }
        if !isValidEmailFormat(req.Email) {
                return response.Error(c, errors.New("VALIDATION_ERROR", "email must be a valid email address", 422))
        }
        if len(req.Password) < 8 {
                return response.Error(c, errors.New("VALIDATION_ERROR", "password must be at least 8 characters", 422))
        }
        if len(req.Password) > 128 {
                return response.Error(c, errors.New("VALIDATION_ERROR", "password must be at most 128 characters", 422))
        }
        if err := validatePasswordComplexity(req.Password); err != nil {
                return response.Error(c, err)
        }

        user, err := h.service.Register(c.Context(), application.RegisterInput{
                Email:     req.Email,
                Password:  req.Password,
                FirstName: req.FirstName,
                LastName:  req.LastName,
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.Created(c, RegisterResponse{
                User:    toUserResponse(user),
                Message: "registration successful; verify your email before logging in",
        })
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c *fiber.Ctx) error {
        var req LoginRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "email":    req.Email,
                "password": req.Password,
        }); err != nil {
                return response.Error(c, err)
        }

        result, err := h.service.Login(c.Context(), application.LoginInput{
                Email:    req.Email,
                Password: req.Password,
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, AuthResponse{
                User:   toUserResponse(result.User),
                Tokens: toTokenResponse(result.Tokens),
        })
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *Handler) Refresh(c *fiber.Ctx) error {
        var req RefreshRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if req.RefreshToken == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "refresh_token is required", 422))
        }

        tokens, err := h.service.Refresh(c.Context(), req.RefreshToken)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toTokenResponse(tokens))
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(c *fiber.Ctx) error {
        var req LogoutRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if req.RefreshToken == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "refresh_token is required", 422))
        }

        err := h.service.Logout(c.Context(), req.RefreshToken)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, MessageResponse{Message: "logged out"})
}

// VerifyEmail handles POST /api/v1/auth/verify-email.
func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
        var req VerifyEmailRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if req.Token == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "token is required", 422))
        }

        err := h.service.VerifyEmail(c.Context(), req.Token)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, MessageResponse{Message: "email verified successfully"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password.
func (h *Handler) ForgotPassword(c *fiber.Ctx) error {
        var req ForgotPasswordRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if req.Email == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "email is required", 422))
        }

        err := h.service.RequestPasswordReset(c.Context(), req.Email)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, MessageResponse{Message: "if the email exists, a reset link has been sent"})
}

// ResendVerification handles POST /api/v1/auth/resend-verification.
func (h *Handler) ResendVerification(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}
	if req.Email == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "email is required", 422))
	}
	if err := h.service.ResendVerification(c.Context(), req.Email); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, MessageResponse{Message: "if the account needs verification, a new email has been sent"})
}

// ResetPassword handles POST /api/v1/auth/reset-password.
func (h *Handler) ResetPassword(c *fiber.Ctx) error {
        var req ResetPasswordRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "token":        req.Token,
                "new_password": req.NewPassword,
        }); err != nil {
                return response.Error(c, err)
        }
        if len(req.NewPassword) < 8 {
                return response.Error(c, errors.New("VALIDATION_ERROR", "new_password must be at least 8 characters", 422))
        }
        if len(req.NewPassword) > 128 {
                return response.Error(c, errors.New("VALIDATION_ERROR", "new_password must be at most 128 characters", 422))
        }
        if err := validatePasswordComplexity(req.NewPassword); err != nil {
                return response.Error(c, err)
        }

        err := h.service.ResetPassword(c.Context(), req.Token, req.NewPassword)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, MessageResponse{Message: "password reset successfully"})
}

func toUserResponse(u *userDomain.User) *UserResponse {
        return &UserResponse{
                ID:            u.ID,
                Email:         u.Email,
                FirstName:     u.FirstName,
                LastName:      u.LastName,
                Role:          string(u.Role),
                Status:        string(u.Status),
                EmailVerified: u.EmailVerified,
                CreatedAt:     u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
        }
}

func toTokenResponse(t *application.TokenPair) *TokenResponse {
        return &TokenResponse{
                AccessToken:  t.AccessToken,
                RefreshToken: t.RefreshToken,
                ExpiresAt:    t.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
        }
}

// emailRegex is a basic but practical email format validation pattern.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// isValidEmailFormat checks that the email has a valid format.
func isValidEmailFormat(email string) bool {
        return emailRegex.MatchString(email)
}

// validatePasswordComplexity ensures passwords have mixed character types.
func validatePasswordComplexity(password string) error {
        var hasUpper, hasLower, hasDigit, hasSpecial bool
        for _, r := range password {
                switch {
                case unicode.IsUpper(r):
                        hasUpper = true
                case unicode.IsLower(r):
                        hasLower = true
                case unicode.IsDigit(r):
                        hasDigit = true
                case unicode.IsPunct(r) || unicode.IsSymbol(r):
                        hasSpecial = true
                }
        }
        missing := 0
        if !hasUpper {
                missing++
        }
        if !hasLower {
                missing++
        }
        if !hasDigit {
                missing++
        }
        if !hasSpecial {
                missing++
        }
        // Require at least 3 of 4 character types (allows passphrases without special chars)
        if missing > 1 {
                return errors.New("VALIDATION_ERROR", "password must contain at least 3 of: uppercase, lowercase, digit, special character", 422)
        }
        return nil
}
