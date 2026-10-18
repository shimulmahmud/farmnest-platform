package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	err := New("TEST_CODE", "test message", http.StatusBadRequest)
	assert.Equal(t, "TEST_CODE", err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Nil(t, err.Err)
	assert.Contains(t, err.Error(), "TEST_CODE")
	assert.Contains(t, err.Error(), "test message")
}

func TestWrap(t *testing.T) {
	inner := errors.New("db failure")
	err := Wrap("DB_ERROR", "database operation failed", http.StatusInternalServerError, inner)
	assert.Equal(t, "DB_ERROR", err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, inner, err.Unwrap())
	assert.Contains(t, err.Error(), "db failure")
}

func TestWithMeta(t *testing.T) {
	err := New("VALIDATION_ERROR", "invalid field", http.StatusUnprocessableEntity)
	err = err.WithMeta("field", "email").WithMeta("reason", "invalid format")
	assert.Equal(t, "email", err.Meta["field"])
	assert.Equal(t, "invalid format", err.Meta["reason"])
}

func TestIsAppError(t *testing.T) {
	appErr := New("TEST", "test", 400)
	result, ok := IsAppError(appErr)
	assert.True(t, ok)
	assert.Equal(t, appErr, result)

	result, ok = IsAppError(errors.New("plain error"))
	assert.False(t, ok)
	assert.Nil(t, result)

	result, ok = IsAppError(nil)
	assert.False(t, ok)
}

func TestFromError(t *testing.T) {
	// AppError passes through
	appErr := New("CUSTOM", "custom error", http.StatusConflict)
	result := FromError(appErr)
	assert.Equal(t, "CUSTOM", result.Code)

	// Generic error wrapped
	generic := errors.New("something failed")
	result = FromError(generic)
	assert.Equal(t, "INTERNAL_ERROR", result.Code)
	assert.Equal(t, http.StatusInternalServerError, result.Status)
}

func TestCommonErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *AppError
		status int
	}{
		{"BadRequest", ErrBadRequest, http.StatusBadRequest},
		{"Unauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"Forbidden", ErrForbidden, http.StatusForbidden},
		{"NotFound", ErrNotFound, http.StatusNotFound},
		{"Conflict", ErrConflict, http.StatusConflict},
		{"Internal", ErrInternal, http.StatusInternalServerError},
		{"Validation", ErrValidation, http.StatusUnprocessableEntity},
		{"RateLimit", ErrRateLimit, http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.status, tt.err.Status)
			assert.NotEmpty(t, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}
