package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a domain/application error with HTTP status code mapping.
type AppError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Status  int            `json:"-"`
	Err     error          `json:"-"`
	Meta    map[string]any `json:"meta,omitempty"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// --- Constructors ---

func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

func Wrap(code, message string, status int, err error) *AppError {
	return &AppError{Code: code, Message: message, Status: status, Err: err}
}

func (e *AppError) WithMeta(key string, value any) *AppError {
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}
	e.Meta[key] = value
	return e
}

// --- Common errors ---

var (
	ErrBadRequest         = New("BAD_REQUEST", "Invalid request parameters", http.StatusBadRequest)
	ErrUnauthorized       = New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
	ErrForbidden          = New("FORBIDDEN", "Insufficient permissions", http.StatusForbidden)
	ErrNotFound           = New("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrConflict           = New("CONFLICT", "Resource already exists", http.StatusConflict)
	ErrInternal           = New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrServiceUnavailable = New("SERVICE_UNAVAILABLE", "Service temporarily unavailable", http.StatusServiceUnavailable)
	ErrValidation         = New("VALIDATION_ERROR", "Validation failed", http.StatusUnprocessableEntity)
	ErrRateLimit          = New("RATE_LIMITED", "Too many requests", http.StatusTooManyRequests)
)

// IsAppError checks if an error is an AppError and returns it.
func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// FromError converts a generic error to an AppError with internal status.
func FromError(err error) *AppError {
	if appErr, ok := IsAppError(err); ok {
		return appErr
	}
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: "An unexpected error occurred",
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}
