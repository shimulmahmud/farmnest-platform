package emailtemplate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var brand = Brand{Name: "FarmNest", AppURL: "https://shop.example.bd/", SupportEmail: "help@shop.example.bd"}

func TestVerifyEmail_HasLinkCodeAndBothVersions(t *testing.T) {
	r := brand.VerifyEmail("Rahim", "https://shop.example.bd/verify-email?token=abc123", "abc123")
	assert.Contains(t, r.Subject, "FarmNest")
	assert.Contains(t, r.HTML, `href="https://shop.example.bd/verify-email?token=abc123"`)
	assert.Contains(t, r.HTML, "Verify my email")
	assert.Contains(t, r.HTML, "abc123")
	assert.Contains(t, r.Text, "https://shop.example.bd/verify-email?token=abc123")
	assert.Contains(t, r.Text, "Hello Rahim,")
	assert.NotContains(t, r.Text, "POST /api", "no developer instructions in customer emails")
}

func TestNotice_EscapesUserText(t *testing.T) {
	r := brand.Notice("Subject", "Title", "<script>alert(1)</script>", []string{"<b>bold</b> & more"}, "", "")
	assert.NotContains(t, r.HTML, "<script>")
	assert.NotContains(t, r.HTML, "<b>bold</b>")
	assert.Contains(t, r.HTML, "&lt;b&gt;bold&lt;/b&gt;")
}

func TestFromPlain_SplitsParagraphs(t *testing.T) {
	r := brand.FromPlain("Order Confirmed", "Hi Rahim,\n\nYour order ORD-1 is confirmed.\n\nThanks!")
	assert.Equal(t, 3, strings.Count(r.HTML, `font-size:15px;line-height:1.6;">`))
	assert.Contains(t, r.Text, "Your order ORD-1 is confirmed.")
}

func TestBrand_NormalizeDefaults(t *testing.T) {
	b := Brand{AppURL: "https://x.bd/"}.Normalize()
	assert.Equal(t, "FarmNest", b.Name)
	assert.Equal(t, "https://x.bd", b.AppURL)
	assert.NotEmpty(t, b.Color)
}
