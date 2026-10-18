package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	StartTLS bool
	Timeout  time.Duration
}

type SMTP struct {
	config SMTPConfig
}

func NewSMTP(config SMTPConfig) *SMTP {
	return &SMTP{config: config}
}

// Send delivers a plain-text email.
func (s *SMTP) Send(ctx context.Context, to, subject, body string) error {
	return s.deliver(ctx, to, subject, body, "")
}

// SendHTML delivers an email with an HTML version and a plain-text fallback.
func (s *SMTP) SendHTML(ctx context.Context, to, subject, text, htmlBody string) error {
	return s.deliver(ctx, to, subject, text, htmlBody)
}

// buildMessage builds the raw message. The subject is RFC 2047 encoded (so characters like
// the dash or Bengali text display correctly) and bodies are quoted-printable UTF-8.
func buildMessage(from, to mail.Address, subject, text, htmlBody string) (string, error) {
	var msg strings.Builder
	fmt.Fprintf(&msg, "From: %s\r\n", from.String())
	fmt.Fprintf(&msg, "To: %s\r\n", to.String())
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	msg.WriteString("MIME-Version: 1.0\r\n")

	qp := func(s string) (string, error) {
		var b bytes.Buffer
		w := quotedprintable.NewWriter(&b)
		if _, err := w.Write([]byte(s)); err != nil {
			return "", err
		}
		if err := w.Close(); err != nil {
			return "", err
		}
		return b.String(), nil
	}

	textQP, err := qp(text)
	if err != nil {
		return "", err
	}
	if htmlBody == "" {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n")
		msg.WriteString(textQP + "\r\n")
		return msg.String(), nil
	}

	htmlQP, err := qp(htmlBody)
	if err != nil {
		return "", err
	}
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	boundary := "mime-" + hex.EncodeToString(raw)
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
	fmt.Fprintf(&msg, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n%s\r\n", boundary, textQP)
	fmt.Fprintf(&msg, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n%s\r\n", boundary, htmlQP)
	fmt.Fprintf(&msg, "--%s--\r\n", boundary)
	return msg.String(), nil
}

func (s *SMTP) deliver(ctx context.Context, to, subject, text, htmlBody string) error {
	fromAddress, err := mail.ParseAddress(s.config.From)
	if err != nil {
		return fmt.Errorf("invalid SMTP_FROM address: %w", err)
	}
	toAddress, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("email subject contains a line break")
	}

	timeout := s.config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("start SMTP client: %w", err)
	}
	defer client.Close()

	if s.config.StartTLS {
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.config.Host}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if s.config.Username != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate with SMTP server: %w", err)
		}
	}
	if err := client.Mail(fromAddress.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(toAddress.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}
	message, err := buildMessage(*fromAddress, *toAddress, subject, text, htmlBody)
	if err != nil {
		_ = writer.Close()
		return fmt.Errorf("build email: %w", err)
	}
	if _, err := io.WriteString(writer, message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}
	return nil
}
