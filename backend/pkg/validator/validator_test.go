package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	ID    string `json:"id"`
	Age   int    `json:"age"`
}

func TestValidate_RequiredField(t *testing.T) {
	v := New()
	p := testPayload{Name: "", Email: "test@example.com"}
	err := v.Validate(p, map[string]string{"Name": "required"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidate_RequiredPass(t *testing.T) {
	v := New()
	p := testPayload{Name: "John", Email: "test@example.com"}
	err := v.Validate(p, map[string]string{"Name": "required"})
	assert.NoError(t, err)
}

func TestValidate_Email(t *testing.T) {
	v := New()
	p := testPayload{Name: "John", Email: "invalid"}
	err := v.Validate(p, map[string]string{"Email": "email"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid email")
}

func TestValidate_EmailValid(t *testing.T) {
	v := New()
	p := testPayload{Name: "John", Email: "john@example.com"}
	err := v.Validate(p, map[string]string{"Email": "email"})
	assert.NoError(t, err)
}

func TestValidate_UUID(t *testing.T) {
	v := New()
	p := testPayload{ID: "not-a-uuid"}
	err := v.Validate(p, map[string]string{"ID": "uuid"})
	assert.Error(t, err)
}

func TestValidate_UUIDValid(t *testing.T) {
	v := New()
	p := testPayload{ID: "550e8400-e29b-41d4-a716-446655440000"}
	err := v.Validate(p, map[string]string{"ID": "uuid"})
	assert.NoError(t, err)
}

func TestValidateRequired(t *testing.T) {
	v := New()
	err := v.ValidateRequired(map[string]string{"name": "", "email": "test@test.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidateRequired_AllPresent(t *testing.T) {
	v := New()
	err := v.ValidateRequired(map[string]string{"name": "John", "email": "test@test.com"})
	assert.NoError(t, err)
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"@domain.com", false},
		{"user@", false},
		{"no-at-sign", false},
		{"user@domain", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.valid, isValidEmail(tt.email), "email: %s", tt.email)
	}
}

func TestIsValidUUID(t *testing.T) {
	assert.True(t, isValidUUID("550e8400-e29b-41d4-a716-446655440000"))
	assert.False(t, isValidUUID("not-a-uuid"))
	assert.False(t, isValidUUID("550e8400-e29b-41d4-a716"))
	assert.False(t, isValidUUID(""))
}
