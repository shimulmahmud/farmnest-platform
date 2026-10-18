package domain

import (
	"context"
	"time"

	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/google/uuid"
)

// AuditAction represents the type of auditable action (admin module legacy actions).
// New code should use pkg/audit.Action constants.
type AuditAction string

const (
	AuditActionUserRoleUpdate    AuditAction = "user.role.update"
	AuditActionUserStatusUpdate  AuditAction = "user.status.update"
	AuditActionReviewApprove     AuditAction = "review.approve"
	AuditActionReviewReject      AuditAction = "review.reject"
	AuditActionReviewHide        AuditAction = "review.hide"
	AuditActionProductCreate     AuditAction = "product.create"
	AuditActionProductUpdate     AuditAction = "product.update"
	AuditActionProductDelete     AuditAction = "product.delete"
	AuditActionProductPriceChange AuditAction = "product.price.change"
	AuditActionCategoryCreate    AuditAction = "category.create"
	AuditActionCategoryUpdate    AuditAction = "category.update"
	AuditActionCategoryDelete    AuditAction = "category.delete"
	AuditActionInventoryAdjust   AuditAction = "inventory.adjust"
	AuditActionOrderCancel       AuditAction = "order.cancel"
	AuditActionOrderRefund       AuditAction = "order.refund"
	AuditActionOrderStatusUpdate AuditAction = "order.status.update"
	AuditActionCustomerUpdate    AuditAction = "customer.update"
	AuditActionPromotionCreate   AuditAction = "promotion.create"
	AuditActionPromotionUpdate   AuditAction = "promotion.update"
	AuditActionPromotionDelete   AuditAction = "promotion.delete"
)

// SensitiveActions is the set of actions that must produce audit records.
var SensitiveActions = map[AuditAction]bool{
	AuditActionUserRoleUpdate:    true,
	AuditActionUserStatusUpdate:  true,
	AuditActionReviewApprove:     true,
	AuditActionReviewReject:      true,
	AuditActionReviewHide:        true,
	AuditActionProductDelete:     true,
	AuditActionCategoryDelete:    true,
	AuditActionOrderCancel:       true,
	AuditActionOrderRefund:       true,
	AuditActionOrderStatusUpdate: true,
	AuditActionPromotionDelete:   true,
}

// AuditLog records a sensitive administrative action for compliance.
// This is the admin module's view for querying; actual writing goes through pkg/audit.
type AuditLog struct {
	ID           string      `bson:"_id" json:"id"`
	ActorID      string      `bson:"actor_id" json:"actor_id"`
	ActorRole    string      `bson:"actor_role" json:"actor_role"`
	Action       AuditAction `bson:"action" json:"action"`
	ResourceType string      `bson:"resource_type" json:"resource_type"`
	ResourceID   string      `bson:"resource_id" json:"resource_id"`
	OldValue     string      `bson:"old_value,omitempty" json:"old_value,omitempty"`
	NewValue     string      `bson:"new_value,omitempty" json:"new_value,omitempty"`
	Details      string      `bson:"details,omitempty" json:"details,omitempty"`
	IPAddress    string      `bson:"ip_address" json:"ip_address"`
	RequestID    string      `bson:"request_id,omitempty" json:"request_id,omitempty"`
	UserAgent    string      `bson:"user_agent" json:"user_agent"`
	CreatedAt    time.Time   `bson:"created_at" json:"created_at"`
}

// NewAuditLog creates an audit log entry.
func NewAuditLog(actorID, actorRole string, action AuditAction, resourceType, resourceID, details string) *AuditLog {
	return &AuditLog{
		ID:           uuid.New().String(),
		ActorID:      actorID,
		ActorRole:    actorRole,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		CreatedAt:    time.Now().UTC(),
	}
}

// SetRequestMetadata sets IP and user agent from the HTTP request context.
func (a *AuditLog) SetRequestMetadata(ip, userAgent string) {
	a.IPAddress = ip
	a.UserAgent = userAgent
}

// AuditRepository defines persistence operations for audit logs (query side).
type AuditRepository interface {
	// Insert persists a new audit log entry.
	Insert(ctx context.Context, log *AuditLog) error

	// List returns paginated audit logs with optional filters.
	List(ctx context.Context, filter *AuditFilter, p pagination.Params) ([]*AuditLog, int64, error)
}

// AuditFilter defines optional filters for listing audit logs.
type AuditFilter struct {
	ActorID      string
	Action       AuditAction
	ResourceType string
	ResourceID   string
	RequestID    string
}