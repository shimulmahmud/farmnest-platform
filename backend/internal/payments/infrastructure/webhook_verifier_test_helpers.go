// Package infrastructure contains webhook signature verification for production use.
//
// WARNING: AlwaysValidVerifier and AlwaysInvalidVerifier in this file are intended
// for testing ONLY. They must NEVER be wired into production — doing so bypasses
// all webhook signature verification and allows arbitrary forged webhook payloads.
package infrastructure

import (
	"context"
	"fmt"

	"github.com/ecommerce/backend/pkg/errors"
)

// AlwaysValidVerifier accepts any signature.
// FOR TESTING ONLY — never use in production.
type AlwaysValidVerifier struct{}

// NewAlwaysValidVerifier creates a verifier that always succeeds.
func NewAlwaysValidVerifier() *AlwaysValidVerifier {
	return &AlwaysValidVerifier{}
}

func (v *AlwaysValidVerifier) Verify(_ context.Context, provider, signature string, payload []byte) error {
	if provider == "" {
		return errors.New("VALIDATION_ERROR", "provider is required", 422)
	}
	return nil
}

// AlwaysInvalidVerifier rejects every signature.
// FOR TESTING ONLY.
type AlwaysInvalidVerifier struct{}

// NewAlwaysInvalidVerifier creates a verifier that always rejects.
func NewAlwaysInvalidVerifier() *AlwaysInvalidVerifier {
	return &AlwaysInvalidVerifier{}
}

func (v *AlwaysInvalidVerifier) Verify(_ context.Context, provider, signature string, payload []byte) error {
	return fmt.Errorf("signature verification failed")
}
