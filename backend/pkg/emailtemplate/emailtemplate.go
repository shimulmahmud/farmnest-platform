// Package emailtemplate renders the shop's branded emails (HTML + plain-text versions).
//
// Every email uses one layout: logo/name header, a title, paragraphs, an optional button,
// and a footer. The HTML uses tables and inline styles because that is what mail apps support.
package emailtemplate

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"strings"
)

// Brand holds what makes the emails look like they come from this shop.
type Brand struct {
	Name         string // shown in the header and footer
	LogoURL      string // optional absolute URL of a logo image
	Color        string // button / accent colour, e.g. "#1f5c3a"
	AppURL       string // storefront address (no trailing slash), used for links
	AdminURL     string // admin panel address (no trailing slash)
	SupportEmail string // shown in the footer
}

// Rendered is a finished email.
type Rendered struct {
	Subject string
	Text    string
	HTML    string
}

// Normalize fills in sensible defaults.
func (b Brand) Normalize() Brand {
	if strings.TrimSpace(b.Name) == "" {
		b.Name = "FarmNest"
	}
	if strings.TrimSpace(b.Color) == "" {
		b.Color = "#1f5c3a"
	}
	b.AppURL = strings.TrimRight(b.AppURL, "/")
	b.AdminURL = strings.TrimRight(b.AdminURL, "/")
	return b
}

type content struct {
	Brand      Brand
	Preheader  string
	Title      string
	Greeting   string
	Paragraphs []string
	Button     string
	ButtonURL  string
	Code       string
	CodeLabel  string
	Note       string
}

var layout = template.Must(template.New("email").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
</head>
<body style="margin:0;padding:0;background:#f4f5f3;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">{{.Preheader}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f5f3;padding:24px 12px;">
<tr><td align="center">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;background:#ffffff;border-radius:14px;overflow:hidden;border:1px solid #e5e7e2;">
    <tr><td style="background:{{.Brand.Color}};padding:22px 28px;">
      {{if .Brand.LogoURL}}<img src="{{.Brand.LogoURL}}" alt="{{.Brand.Name}}" height="36" style="display:inline-block;vertical-align:middle;border-radius:8px;margin-right:10px;">{{end}}
      <span style="font-family:Arial,Helvetica,sans-serif;font-size:20px;font-weight:bold;color:#ffffff;vertical-align:middle;">{{.Brand.Name}}</span>
    </td></tr>
    <tr><td style="padding:30px 28px 8px 28px;font-family:Arial,Helvetica,sans-serif;color:#1f2937;">
      <h1 style="margin:0 0 14px 0;font-size:22px;line-height:1.3;color:#111827;">{{.Title}}</h1>
      {{if .Greeting}}<p style="margin:0 0 14px 0;font-size:15px;line-height:1.6;">{{.Greeting}}</p>{{end}}
      {{range .Paragraphs}}<p style="margin:0 0 14px 0;font-size:15px;line-height:1.6;">{{.}}</p>{{end}}
      {{if .ButtonURL}}<table role="presentation" cellpadding="0" cellspacing="0" style="margin:22px 0;"><tr><td style="background:{{.Brand.Color}};border-radius:10px;">
        <a href="{{.ButtonURL}}" style="display:inline-block;padding:13px 26px;font-family:Arial,Helvetica,sans-serif;font-size:15px;font-weight:bold;color:#ffffff;text-decoration:none;">{{.Button}}</a>
      </td></tr></table>
      <p style="margin:0 0 14px 0;font-size:12px;line-height:1.5;color:#6b7280;">If the button does not work, copy this link into your browser:<br><a href="{{.ButtonURL}}" style="color:{{.Brand.Color}};word-break:break-all;">{{.ButtonURL}}</a></p>{{end}}
      {{if .Code}}<p style="margin:18px 0 6px 0;font-size:13px;color:#6b7280;">{{.CodeLabel}}</p>
      <p style="margin:0 0 14px 0;padding:12px 14px;background:#f3f4f6;border-radius:8px;font-family:Consolas,Menlo,monospace;font-size:14px;word-break:break-all;color:#111827;">{{.Code}}</p>{{end}}
      {{if .Note}}<p style="margin:0 0 14px 0;font-size:13px;line-height:1.6;color:#6b7280;">{{.Note}}</p>{{end}}
    </td></tr>
    <tr><td style="padding:18px 28px 26px 28px;border-top:1px solid #eef0ec;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.6;color:#9ca3af;">
      You are receiving this email because of activity on your {{.Brand.Name}} account.{{if .Brand.SupportEmail}} Questions? Write to <a href="mailto:{{.Brand.SupportEmail}}" style="color:#6b7280;">{{.Brand.SupportEmail}}</a>.{{end}}<br>
      &copy; {{.Brand.Name}}
    </td></tr>
  </table>
</td></tr>
</table>
</body>
</html>`))

func render(c content, subject string) Rendered {
	var buf bytes.Buffer
	if err := layout.Execute(&buf, c); err != nil {
		// The layout is static, so this cannot happen in practice; fall back to plain text.
		return Rendered{Subject: subject, Text: plainText(c), HTML: ""}
	}
	return Rendered{Subject: subject, Text: plainText(c), HTML: buf.String()}
}

func plainText(c content) string {
	var b strings.Builder
	b.WriteString(c.Title + "\n\n")
	if c.Greeting != "" {
		b.WriteString(c.Greeting + "\n\n")
	}
	for _, p := range c.Paragraphs {
		b.WriteString(p + "\n\n")
	}
	if c.ButtonURL != "" {
		b.WriteString(c.Button + ":\n" + c.ButtonURL + "\n\n")
	}
	if c.Code != "" {
		b.WriteString(c.CodeLabel + "\n" + c.Code + "\n\n")
	}
	if c.Note != "" {
		b.WriteString(c.Note + "\n\n")
	}
	b.WriteString("-- " + c.Brand.Name + "\n")
	return b.String()
}

func hello(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Hello,"
	}
	return "Hello " + name + ","
}

// VerifyEmail is sent after sign-up. link opens the storefront and verifies automatically;
// code is the same token for people who prefer to paste it.
func (b Brand) VerifyEmail(name, link, code string) Rendered {
	b = b.Normalize()
	return render(content{
		Brand:      b,
		Preheader:  "One click to confirm your email address.",
		Title:      "Confirm your email address",
		Greeting:   hello(name),
		Paragraphs: []string{fmt.Sprintf("Welcome to %s! Please confirm your email address to activate your account and start ordering.", b.Name)},
		Button:     "Verify my email",
		ButtonURL:  link,
		Code:       code,
		CodeLabel:  "Or paste this verification code on the sign-up page:",
		Note:       "This link works for 24 hours. If you did not create an account, you can safely ignore this email.",
	}, "Confirm your email for "+b.Name)
}

// ResetPassword is sent after a "forgot password" request.
func (b Brand) ResetPassword(name, link string) Rendered {
	b = b.Normalize()
	return render(content{
		Brand:      b,
		Preheader:  "Choose a new password for your account.",
		Title:      "Reset your password",
		Greeting:   hello(name),
		Paragraphs: []string{"We received a request to reset the password of your account. Click the button below to choose a new one."},
		Button:     "Choose a new password",
		ButtonURL:  link,
		Note:       "This link works for a short time and can be used once. If you did not ask for this, ignore this email: your password stays the same.",
	}, "Reset your "+b.Name+" password")
}

// Notice is a general-purpose message (order updates, support replies...).
// paragraphs are plain text and are HTML-escaped. buttonURL may be empty.
func (b Brand) Notice(subject, title, name string, paragraphs []string, button, buttonURL string) Rendered {
	b = b.Normalize()
	return render(content{
		Brand:      b,
		Preheader:  firstOr(paragraphs, title),
		Title:      title,
		Greeting:   hello(name),
		Paragraphs: paragraphs,
		Button:     button,
		ButtonURL:  buttonURL,
	}, subject)
}

// FromPlain wraps an existing plain-text email body (for example the order emails) in the
// branded layout. Blank lines separate paragraphs; the first line is not repeated as a title.
func (b Brand) FromPlain(subject, body string) Rendered {
	b = b.Normalize()
	var paras []string
	for _, p := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			paras = append(paras, p)
		}
	}
	return render(content{
		Brand:      b,
		Preheader:  firstOr(paras, subject),
		Title:      subject,
		Paragraphs: paras,
	}, subject)
}

func firstOr(items []string, fallback string) string {
	if len(items) > 0 {
		s := items[0]
		if len(s) > 110 {
			s = s[:110]
		}
		return s
	}
	return fallback
}

// Escape is exported for callers that build text from user input.
func Escape(s string) string { return html.EscapeString(s) }
