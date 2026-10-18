package audit

import (
        "context"
        "testing"

        "github.com/ecommerce/backend/pkg/pagination"
)

// --- Mock Repository ---

type mockRepo struct {
        entries []*Entry
}

func (m *mockRepo) Insert(_ context.Context, entry *Entry) error {
        m.entries = append(m.entries, entry)
        return nil
}

func (m *mockRepo) List(_ context.Context, filter *Filter, _ pagination.Params) ([]*Entry, int64, error) {
        if filter != nil && filter.ActorID != "" {
                var result []*Entry
                for _, e := range m.entries {
                        if e.ActorID == filter.ActorID {
                                result = append(result, e)
                        }
                }
                return result, int64(len(result)), nil
        }
        return m.entries, int64(len(m.entries)), nil
}

// --- Tests ---

func TestNopAuditor_DoesNotPanic(t *testing.T) {
        NopAuditor.Record("a", ActionProductCreate, "p", "p1", "", "new", "test")
        NopAuditor.RecordWithMetadata(nil, "a", ActionProductUpdate, "p", "p1", "old", "new", "test")
        NopAuditor.Stop()
}

func TestActionConstants(t *testing.T) {
        actions := []Action{
                ActionUserRoleUpdate, ActionUserStatusUpdate,
                ActionProductCreate, ActionProductUpdate, ActionProductDelete, ActionProductPriceChange,
                ActionCategoryCreate, ActionCategoryUpdate, ActionCategoryDelete,
                ActionInventoryAdjust,
                ActionOrderCancel, ActionOrderRefund, ActionOrderStatusChange,
                ActionCustomerUpdate, ActionCustomerProfileUpdate,
                ActionReviewApprove, ActionReviewReject, ActionReviewHide,
                ActionPromotionCreate, ActionPromotionUpdate, ActionPromotionDelete,
                ActionPermissionChange,
        }
        for _, a := range actions {
                if a == "" {
                        t.Errorf("action constant is empty")
                }
        }
}

func TestFilter_Fields(t *testing.T) {
        f := &Filter{
                ActorID:      "admin-1",
                Action:       ActionProductUpdate,
                ResourceType: "product",
                ResourceID:   "p-1",
                RequestID:    "req-123",
        }
        if f.ActorID != "admin-1" {
                t.Errorf("expected actor_id admin-1, got %s", f.ActorID)
        }
        if f.RequestID != "req-123" {
                t.Errorf("expected request_id req-123, got %s", f.RequestID)
        }
}

func TestEntry_Fields(t *testing.T) {
        e := &Entry{
                ID:           "e1",
                ActorID:      "admin-1",
                Action:       ActionProductPriceChange,
                ResourceType: "product",
                ResourceID:   "p-1",
                OldValue:     "10.00",
                NewValue:     "15.00",
                Details:      "price increase",
                IPAddress:    "10.0.0.1",
                RequestID:    "req-abc",
                CreatedAt:    1700000000,
        }
        if e.OldValue != "10.00" {
                t.Errorf("expected old_value 10.00, got %s", e.OldValue)
        }
        if e.NewValue != "15.00" {
                t.Errorf("expected new_value 15.00, got %s", e.NewValue)
        }
        if e.RequestID != "req-abc" {
                t.Errorf("expected request_id req-abc, got %s", e.RequestID)
        }
        if e.Action != ActionProductPriceChange {
                t.Errorf("expected action product.price.change, got %s", e.Action)
        }
}

// --- Sanitize Tests ---

func TestSanitize_RemovesPasswordValues(t *testing.T) {
        input := "password=secret123 and name=John"
        got := sanitize(input)
        if got != "[REDACTED] and name=John" {
                t.Errorf("expected password redacted, got %q", got)
        }
}

func TestSanitize_RemovesTokenValues(t *testing.T) {
        input := "token: abc123def and api_key=xyz789"
        got := sanitize(input)
        if got != "[REDACTED] and [REDACTED]" {
                t.Errorf("expected tokens redacted, got %q", got)
        }
}

func TestSanitize_RemovesCardNumbers(t *testing.T) {
        input := "card_number=4111111111111111, cvv=123"
        got := sanitize(input)
        if got != "[REDACTED], [REDACTED]" {
                t.Errorf("expected card info redacted, got %q", got)
        }
}

func TestSanitize_RemovesBearerTokens(t *testing.T) {
        input := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.abc"
        got := sanitize(input)
        if got != "Authorization: [REDACTED]" {
                t.Errorf("expected bearer token redacted, got %q", got)
        }
}

func TestSanitize_PreservesSafeContent(t *testing.T) {
        input := "price changed from 10.00 to 15.00"
        got := sanitize(input)
        if got != input {
                t.Errorf("expected no change for safe content, got %q", got)
        }
}

func TestSanitize_EmptyString(t *testing.T) {
        got := sanitize("")
        if got != "" {
                t.Errorf("expected empty string, got %q", got)
        }
}

func TestSanitize_CaseInsensitive(t *testing.T) {
        input := "PASSWORD=mysecret TOKEN=abc123"
        got := sanitize(input)
        if got != "[REDACTED] [REDACTED]" {
                t.Errorf("expected case-insensitive redaction, got %q", got)
        }
}

func TestSanitize_PrivateKey(t *testing.T) {
        input := "private_key=MIIEvgIBADANBgkqhkiG9w0BAQEFAAS"
        got := sanitize(input)
        if got != "[REDACTED]" {
                t.Errorf("expected private_key redacted, got %q", got)
        }
}

func TestSanitize_StripsCVV(t *testing.T) {
        input := "cvv = 456 and CVC:789"
        got := sanitize(input)
        if got != "[REDACTED] and [REDACTED]" {
                t.Errorf("expected cvv/cvc redacted, got %q", got)
        }
}

func TestSanitize_PaymentReferenceNotRedacted(t *testing.T) {
        input := "payment_reference=pay_abc123 and order_number=ORD-001"
        got := sanitize(input)
        if got != input {
                t.Errorf("expected safe payment references preserved, got %q", got)
        }
}

func TestNewConstants_NonEmpty(t *testing.T) {
        if ActionCustomerProfileUpdate == "" {
                t.Error("ActionCustomerProfileUpdate is empty")
        }
        if ActionPermissionChange == "" {
                t.Error("ActionPermissionChange is empty")
        }
        if string(ActionCustomerProfileUpdate) != "customer.profile.update" {
                t.Errorf("expected customer.profile.update, got %s", ActionCustomerProfileUpdate)
        }
        if string(ActionPermissionChange) != "permission.change" {
                t.Errorf("expected permission.change, got %s", ActionPermissionChange)
        }
}
