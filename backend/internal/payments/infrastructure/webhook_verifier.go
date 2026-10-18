package infrastructure

import (
        "context"
        "crypto/hmac"
        "crypto/sha256"
        "encoding/hex"
        "strings"

        "github.com/ecommerce/backend/pkg/errors"
)

// FakeSignatureVerifier implements domain.SignatureVerifier using HMAC-SHA256.
// It is suitable for both testing and production with a "fake" provider.
type FakeSignatureVerifier struct {
        secrets map[string]string // provider name -> HMAC secret
}

// NewFakeSignatureVerifier creates a verifier with the given provider secrets.
func NewFakeSignatureVerifier(secrets map[string]string) *FakeSignatureVerifier {
        if secrets == nil {
                secrets = make(map[string]string)
        }
        return &FakeSignatureVerifier{secrets: secrets}
}

// Verify checks an HMAC-SHA256 signature of the payload.
// The signature should be prefixed with "sha256=" (e.g. "sha256=abcdef...").
func (v *FakeSignatureVerifier) Verify(_ context.Context, provider, signature string, payload []byte) error {
        secret, ok := v.secrets[provider]
        if !ok {
                return errors.New("VALIDATION_ERROR", "unsupported webhook provider: "+provider, 422)
        }

        if signature == "" {
                return errors.New("UNAUTHORIZED", "missing webhook signature", 401)
        }

        expected := computeHMAC(secret, payload)

        // Accept both raw hex and "sha256=" prefixed format.
        provided := signature
        if strings.HasPrefix(provided, "sha256=") {
                provided = strings.TrimPrefix(provided, "sha256=")
        }

        if !hmac.Equal([]byte(provided), []byte(expected)) {
                return errors.New("UNAUTHORIZED", "invalid webhook signature", 401)
        }

        return nil
}

// SignPayload computes the HMAC-SHA256 signature for a payload (useful for testing).
func (v *FakeSignatureVerifier) SignPayload(provider string, payload []byte) string {
        secret := v.secrets[provider]
        return "sha256=" + computeHMAC(secret, payload)
}

// computeHMAC computes HMAC-SHA256 of payload using the given secret.
func computeHMAC(secret string, payload []byte) string {
        h := hmac.New(sha256.New, []byte(secret))
        h.Write(payload)
        return hex.EncodeToString(h.Sum(nil))
}
