package application

import (
	"context"
	"testing"
	"time"

	"github.com/ecommerce/backend/internal/auth/domain"
	"github.com/ecommerce/backend/internal/auth/infrastructure"
	customerDomain "github.com/ecommerce/backend/internal/customers/domain"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockUserRepo struct {
	users   map[string]*userDomain.User
	byEmail map[string]*userDomain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[string]*userDomain.User),
		byEmail: make(map[string]*userDomain.User),
	}
}

func (r *mockUserRepo) Create(_ context.Context, u *userDomain.User) error {
	r.users[u.ID] = u
	r.byEmail[u.Email] = u
	return nil
}

func (r *mockUserRepo) FindByID(_ context.Context, id string) (*userDomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.ErrNotFound
	}
	return u, nil
}

func (r *mockUserRepo) FindByEmail(_ context.Context, email string) (*userDomain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, errors.ErrNotFound
	}
	return u, nil
}

func (r *mockUserRepo) Update(_ context.Context, u *userDomain.User) error {
	r.users[u.ID] = u
	r.byEmail[u.Email] = u
	return nil
}

func (r *mockUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := r.byEmail[email]
	return ok, nil
}

type mockTokenRepo struct {
	refreshTokens      map[string]*domain.RefreshToken
	verificationTokens map[string]*domain.VerificationToken
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{
		refreshTokens:      make(map[string]*domain.RefreshToken),
		verificationTokens: make(map[string]*domain.VerificationToken),
	}
}

func (r *mockTokenRepo) StoreRefreshToken(_ context.Context, rt *domain.RefreshToken) error {
	r.refreshTokens[rt.Token] = rt
	return nil
}

func (r *mockTokenRepo) FindRefreshToken(_ context.Context, token string) (*domain.RefreshToken, error) {
	rt, ok := r.refreshTokens[token]
	if !ok || rt.Revoked {
		return nil, errors.ErrUnauthorized
	}
	return rt, nil
}

func (r *mockTokenRepo) RevokeRefreshToken(_ context.Context, token string) error {
	if rt, ok := r.refreshTokens[token]; ok {
		rt.Revoked = true
	}
	return nil
}

func (r *mockTokenRepo) RevokeAllUserRefreshTokens(_ context.Context, userID string) error {
	for _, rt := range r.refreshTokens {
		if rt.UserID == userID {
			rt.Revoked = true
		}
	}
	return nil
}

func (r *mockTokenRepo) StoreVerificationToken(_ context.Context, vt *domain.VerificationToken) error {
	r.verificationTokens[vt.Token] = vt
	return nil
}

func (r *mockTokenRepo) FindVerificationToken(_ context.Context, token string, _ domain.TokenType) (*domain.VerificationToken, error) {
	vt, ok := r.verificationTokens[token]
	if !ok || vt.Used {
		return nil, errors.ErrNotFound
	}
	return vt, nil
}

func (r *mockTokenRepo) MarkVerificationTokenUsed(_ context.Context, id string) error {
	for _, vt := range r.verificationTokens {
		if vt.ID == id {
			vt.Used = true
			break
		}
	}
	return nil
}

type mockCustomerRepo struct{}

func (m *mockCustomerRepo) Create(_ context.Context, _ *customerDomain.Customer) error { return nil }

type mockEmailSender struct {
	messages []string
}

func (m *mockEmailSender) Send(_ context.Context, _ string, subject, body string) error {
	m.messages = append(m.messages, subject+"\n"+body)
	return nil
}

// --- Tests ---

func newTestService(t *testing.T) (*Service, *mockUserRepo, *mockTokenRepo) {
	t.Helper()
	jwtSvc := infrastructure.NewJWTService("test-secret-key", 15*time.Minute, 168*time.Hour, "test-issuer")
	passwordSvc := infrastructure.NewPasswordHasher(infrastructure.DefaultCost)

	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	customerRepo := &mockCustomerRepo{}
	emailSender := &mockEmailSender{}

	svc := NewService(userRepo, tokenRepo, customerRepo, jwtSvc, passwordSvc, DefaultAuthConfig(), 168*time.Hour, emailSender)
	return svc, userRepo, tokenRepo
}

func registerAndVerify(t *testing.T, svc *Service, input RegisterInput) *userDomain.User {
	t.Helper()
	user, err := svc.Register(context.Background(), input)
	require.NoError(t, err)

	var tokenStr string
	for value, token := range svc.tokenRepo.(*mockTokenRepo).verificationTokens {
		if token.UserID == user.ID && token.Type == domain.TokenTypeEmailVerify {
			tokenStr = value
			break
		}
	}
	require.NotEmpty(t, tokenStr)
	require.NoError(t, svc.VerifyEmail(context.Background(), tokenStr))
	return user
}

func TestRegister_Success(t *testing.T) {
	svc, userRepo, _ := newTestService(t)

	user, err := svc.Register(context.Background(), RegisterInput{
		Email:     "john@example.com",
		Password:  "securepass123",
		FirstName: "John",
		LastName:  "Doe",
	})
	require.NoError(t, err)

	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, userDomain.RoleCustomer, user.Role)
	assert.Equal(t, userDomain.AccountStatusPendingVerification, user.Status)
	assert.False(t, user.EmailVerified)
	assert.Empty(t, svc.tokenRepo.(*mockTokenRepo).refreshTokens)
	assert.Len(t, svc.emailSender.(*mockEmailSender).messages, 1)
	assert.Contains(t, svc.emailSender.(*mockEmailSender).messages[0], "Confirm your email")
	assert.Contains(t, svc.emailSender.(*mockEmailSender).messages[0], "/verify-email?token=", "the email carries a clickable link")

	exists, _ := userRepo.ExistsByEmail(context.Background(), "john@example.com")
	assert.True(t, exists)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _, _ := newTestService(t)

	input := RegisterInput{Email: "dup@example.com", Password: "securepass123", FirstName: "A", LastName: "B"}
	_, err := svc.Register(context.Background(), input)
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), input)
	assert.Error(t, err)
	appErr, ok := errors.IsAppError(err)
	assert.True(t, ok)
	assert.Equal(t, 409, appErr.Status)
}

func TestLogin_Success(t *testing.T) {
	svc, _, _ := newTestService(t)

	registerAndVerify(t, svc, RegisterInput{
		Email: "login@example.com", Password: "mypassword123", FirstName: "Jane", LastName: "Smith",
	})

	result, err := svc.Login(context.Background(), LoginInput{Email: "login@example.com", Password: "mypassword123"})
	require.NoError(t, err)

	assert.Equal(t, "login@example.com", result.User.Email)
	assert.True(t, result.User.IsActive())
	assert.NotNil(t, result.User.LastLoginAt)
	assert.NotEmpty(t, result.Tokens.AccessToken)
	assert.NotEmpty(t, result.Tokens.RefreshToken)
}

func TestLogin_RequiresEmailVerification(t *testing.T) {
	svc, _, _ := newTestService(t)
	_, err := svc.Register(context.Background(), RegisterInput{
		Email: "unverified@example.com", Password: "mypassword123", FirstName: "U", LastName: "V",
	})
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), LoginInput{Email: "unverified@example.com", Password: "mypassword123"})
	appErr, ok := errors.IsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "EMAIL_NOT_VERIFIED", appErr.Code)
	assert.Equal(t, 403, appErr.Status)
}

func TestLogin_InvalidPassword(t *testing.T) {
	svc, _, _ := newTestService(t)

	registerAndVerify(t, svc, RegisterInput{
		Email: "badpass@example.com", Password: "correctpass123", FirstName: "X", LastName: "Y",
	})

	_, err := svc.Login(context.Background(), LoginInput{Email: "badpass@example.com", Password: "wrongpass123"})
	assert.Error(t, err)

	user, _ := svc.userRepo.FindByEmail(context.Background(), "badpass@example.com")
	assert.Equal(t, 1, user.FailedAttempts)
}

func TestLogin_EmailNotFound(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.Login(context.Background(), LoginInput{Email: "nonexistent@example.com", Password: "any"})
	assert.Error(t, err)
	assert.Equal(t, 401, errors.FromError(err).Status)
}

func TestLogin_AccountLocked(t *testing.T) {
	svc, _, _ := newTestService(t)

	registerAndVerify(t, svc, RegisterInput{
		Email: "locked@example.com", Password: "lockpass123", FirstName: "L", LastName: "O",
	})

	for i := 0; i < 5; i++ {
		svc.Login(context.Background(), LoginInput{Email: "locked@example.com", Password: "wrong"})
	}

	_, err := svc.Login(context.Background(), LoginInput{Email: "locked@example.com", Password: "lockpass123"})
	assert.Error(t, err)
	appErr, ok := errors.IsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 423, appErr.Status)
}

func TestRefresh_TokenRotation(t *testing.T) {
	svc, _, _ := newTestService(t)

	registerAndVerify(t, svc, RegisterInput{
		Email: "refresh@example.com", Password: "refreshpass123", FirstName: "R", LastName: "E",
	})
	login, err := svc.Login(context.Background(), LoginInput{Email: "refresh@example.com", Password: "refreshpass123"})
	require.NoError(t, err)
	tokens := login.Tokens

	newTokens, err := svc.Refresh(context.Background(), tokens.RefreshToken)
	require.NoError(t, err)

	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken)

	_, err = svc.Refresh(context.Background(), tokens.RefreshToken)
	assert.Error(t, err)
}

func TestLogout(t *testing.T) {
	svc, _, _ := newTestService(t)

	registerAndVerify(t, svc, RegisterInput{
		Email: "logout@example.com", Password: "logoutpass123", FirstName: "O", LastName: "U",
	})
	login, err := svc.Login(context.Background(), LoginInput{Email: "logout@example.com", Password: "logoutpass123"})
	require.NoError(t, err)
	tokens := login.Tokens

	err = svc.Logout(context.Background(), tokens.RefreshToken)
	require.NoError(t, err)

	_, err = svc.Refresh(context.Background(), tokens.RefreshToken)
	assert.Error(t, err)
}

func TestVerifyEmail(t *testing.T) {
	svc, _, _ := newTestService(t)

	user, err := svc.Register(context.Background(), RegisterInput{
		Email: "verify@example.com", Password: "verifypass123", FirstName: "V", LastName: "E",
	})
	require.NoError(t, err)

	err = svc.VerifyEmail(context.Background(), "nonexistent-token")
	assert.Error(t, err)

	var tokenStr string
	for tok, token := range svc.tokenRepo.(*mockTokenRepo).verificationTokens {
		if token.Type == domain.TokenTypeEmailVerify {
			tokenStr = tok
			break
		}
	}
	require.NotEmpty(t, tokenStr)
	assert.Contains(t, svc.emailSender.(*mockEmailSender).messages[0], tokenStr)

	err = svc.VerifyEmail(context.Background(), tokenStr)
	require.NoError(t, err)

	updatedUser, _ := svc.userRepo.FindByID(context.Background(), user.ID)
	assert.True(t, updatedUser.EmailVerified)
	assert.Equal(t, userDomain.AccountStatusActive, updatedUser.Status)
}

func TestResetPassword(t *testing.T) {
	svc, _, _ := newTestService(t)

	registerAndVerify(t, svc, RegisterInput{
		Email: "reset@example.com", Password: "oldpass12345", FirstName: "R", LastName: "S",
	})

	err := svc.RequestPasswordReset(context.Background(), "reset@example.com")
	require.NoError(t, err)

	var tokenStr string
	for tok, token := range svc.tokenRepo.(*mockTokenRepo).verificationTokens {
		if token.Type == domain.TokenTypePasswordReset {
			tokenStr = tok
			break
		}
	}
	require.NotEmpty(t, tokenStr)
	assert.Contains(t, svc.emailSender.(*mockEmailSender).messages[1], tokenStr)

	err = svc.ResetPassword(context.Background(), tokenStr, "brandnewpass99")
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), LoginInput{Email: "reset@example.com", Password: "oldpass12345"})
	assert.Error(t, err)

	_, err = svc.Login(context.Background(), LoginInput{Email: "reset@example.com", Password: "brandnewpass99"})
	assert.NoError(t, err)
}

func TestResetPassword_NonexistentEmail(t *testing.T) {
	svc, _, _ := newTestService(t)
	err := svc.RequestPasswordReset(context.Background(), "nonexistent@example.com")
	assert.NoError(t, err)
}

func TestValidateAccessToken(t *testing.T) {
	svc, _, _ := newTestService(t)

	user, err := svc.Register(context.Background(), RegisterInput{
		Email: "token@example.com", Password: "tokenpass123", FirstName: "T", LastName: "K",
	})
	require.NoError(t, err)
	legacyTokens, err := svc.generateTokenPair(user.ID, string(user.Role), false)
	require.NoError(t, err)
	_, err = svc.ValidateAccessToken(legacyTokens.AccessToken)
	assert.Error(t, err)

	var verificationToken string
	for value, token := range svc.tokenRepo.(*mockTokenRepo).verificationTokens {
		if token.Type == domain.TokenTypeEmailVerify {
			verificationToken = value
			break
		}
	}
	require.NotEmpty(t, verificationToken)
	require.NoError(t, svc.VerifyEmail(context.Background(), verificationToken))
	login, err := svc.Login(context.Background(), LoginInput{Email: user.Email, Password: "tokenpass123"})
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(login.Tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, "CUSTOMER", claims.Role)
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.ValidateAccessToken("garbage")
	assert.Error(t, err)
}
