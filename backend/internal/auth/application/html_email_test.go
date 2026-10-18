package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/auth/infrastructure"
	"github.com/ecommerce/backend/pkg/emailtemplate"
)

type htmlSender struct {
	mockEmailSender
	html []string
}

func (h *htmlSender) SendHTML(_ context.Context, _ string, _ string, _ string, html string) error {
	h.html = append(h.html, html)
	return nil
}

func TestRegister_SendsBrandedHTMLWithStorefrontLink(t *testing.T) {
	userRepo := newMockUserRepo()
	sender := &htmlSender{}
	jwtSvc := infrastructure.NewJWTService("test-secret-key", 15*time.Minute, 168*time.Hour, "test-issuer")
	passwordSvc := infrastructure.NewPasswordHasher(infrastructure.DefaultCost)
	svc := NewService(userRepo, newMockTokenRepo(), &mockCustomerRepo{}, jwtSvc, passwordSvc, DefaultAuthConfig(), 168*time.Hour, sender)
	svc.SetBrand(emailtemplate.Brand{Name: "FarmNest", AppURL: "https://shop.example.bd", AdminURL: "https://admin.example.bd"})

	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@example.com", Password: "password1234", FirstName: "Rahim", LastName: "U"})
	require.NoError(t, err)

	require.Len(t, sender.html, 1, "HTML email is used when the sender supports it")
	assert.Contains(t, sender.html[0], "https://shop.example.bd/verify-email?token=")
	assert.NotContains(t, sender.html[0], "admin.example.bd", "customers are sent to the storefront")
}
