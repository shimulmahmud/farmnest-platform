package mailer

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMessage_HTMLIsMultipartAndSubjectIsEncoded(t *testing.T) {
	from := mail.Address{Name: "FarmNest", Address: "no-reply@shop.bd"}
	to := mail.Address{Address: "rahim@example.com"}

	msg, err := buildMessage(from, to, "Order Confirmed \u2014 ORD-1", "plain version", "<p>html version</p>")
	require.NoError(t, err)

	assert.Contains(t, msg, "Subject: =?utf-8?q?Order_Confirmed_=E2=80=94_ORD-1?=", "non-ASCII subject must be encoded")
	assert.Contains(t, msg, "multipart/alternative")
	assert.Contains(t, msg, "text/plain; charset=UTF-8")
	assert.Contains(t, msg, "text/html; charset=UTF-8")
	assert.Contains(t, msg, "plain version")
	assert.Contains(t, msg, "<p>html version</p>")
}

func TestBuildMessage_PlainOnly(t *testing.T) {
	msg, err := buildMessage(mail.Address{Address: "a@b.bd"}, mail.Address{Address: "c@d.bd"}, "Hi", "hello", "")
	require.NoError(t, err)
	assert.False(t, strings.Contains(msg, "multipart"))
	assert.Contains(t, msg, "Content-Type: text/plain; charset=UTF-8")
}
