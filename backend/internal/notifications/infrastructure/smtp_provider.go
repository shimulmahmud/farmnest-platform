package infrastructure

import (
	"context"
	"fmt"

	"github.com/ecommerce/backend/internal/notifications/domain"
	"github.com/ecommerce/backend/pkg/emailtemplate"
	"github.com/ecommerce/backend/pkg/mailer"
)

// SMTPEmailProvider sends the notification emails (order confirmed, shipped...) for real,
// wrapping each plain-text message in the shop's branded layout.
type SMTPEmailProvider struct {
	smtp  *mailer.SMTP
	brand emailtemplate.Brand
}

// NewSMTPEmailProvider creates a domain.EmailProvider that delivers through SMTP.
func NewSMTPEmailProvider(smtp *mailer.SMTP, brand emailtemplate.Brand) *SMTPEmailProvider {
	return &SMTPEmailProvider{smtp: smtp, brand: brand.Normalize()}
}

// SendEmail delivers one email as HTML with a plain-text fallback.
func (p *SMTPEmailProvider) SendEmail(ctx context.Context, msg *domain.EmailMessage) (*domain.EmailResult, error) {
	r := p.brand.FromPlain(msg.Subject, msg.Body)
	if err := p.smtp.SendHTML(ctx, msg.To, msg.Subject, r.Text, r.HTML); err != nil {
		return nil, fmt.Errorf("send email: %w", err)
	}
	return &domain.EmailResult{Success: true, MessageID: "smtp"}, nil
}
