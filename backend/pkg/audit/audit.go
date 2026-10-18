package audit

import (
        "context"

        "github.com/ecommerce/backend/pkg/pagination"
)

// Action represents the type of auditable action.
type Action string

const (
        // User actions
        ActionUserRoleUpdate   Action = "user.role.update"
        ActionUserStatusUpdate Action = "user.status.update"

        // Product actions
        ActionProductCreate      Action = "product.create"
        ActionProductUpdate      Action = "product.update"
        ActionProductDelete      Action = "product.delete"
        ActionProductPriceChange Action = "product.price.change"

        // Category actions
        ActionCategoryCreate Action = "category.create"
        ActionCategoryUpdate Action = "category.update"
        ActionCategoryDelete Action = "category.delete"

        // Inventory actions
        ActionInventoryAdjust Action = "inventory.adjust"

        // Order actions
        ActionOrderCancel       Action = "order.cancel"
        ActionOrderRefund       Action = "order.refund"
        ActionOrderStatusChange Action = "order.status.update"

        // Customer actions
        ActionCustomerUpdate       Action = "customer.update"
        ActionCustomerProfileUpdate Action = "customer.profile.update"

        // Review actions
        ActionReviewApprove Action = "review.approve"
        ActionReviewReject  Action = "review.reject"
        ActionReviewHide    Action = "review.hide"

        // Promotion/coupon actions
        ActionPromotionCreate Action = "promotion.create"
        ActionPromotionUpdate Action = "promotion.update"
        ActionPromotionDelete Action = "promotion.delete"

        // RBAC actions
        ActionPermissionChange Action = "permission.change"
)

// Entry records a single auditable action. It is append-only.
type Entry struct {
        ID           string `bson:"_id" json:"id"`
        ActorID      string `bson:"actor_id" json:"actor_id"`
        Action       Action `bson:"action" json:"action"`
        ResourceType string `bson:"resource_type" json:"resource_type"`
        ResourceID   string `bson:"resource_id" json:"resource_id"`
        OldValue     string `bson:"old_value,omitempty" json:"old_value,omitempty"`
        NewValue     string `bson:"new_value,omitempty" json:"new_value,omitempty"`
        Details      string `bson:"details,omitempty" json:"details,omitempty"`
        IPAddress    string `bson:"ip_address" json:"ip_address"`
        RequestID    string `bson:"request_id" json:"request_id"`
        CreatedAt    int64  `bson:"created_at" json:"created_at"`
}

// Filter defines optional filters for listing audit entries.
type Filter struct {
        ActorID      string
        Action       Action
        ResourceType string
        ResourceID   string
        RequestID    string
}

// Repository defines persistence for audit entries (append-only).
type Repository interface {
        Insert(ctx context.Context, entry *Entry) error
        List(ctx context.Context, filter *Filter, p pagination.Params) ([]*Entry, int64, error)
}

// Auditor provides a fire-and-forget audit recording capability.
// Services accept this as an optional dependency to record cross-cutting audit trails.
// Implementations must be async, failure-safe, and never block the caller.
type Auditor interface {
        // Record asynchronously persists an audit entry.
        // Implementations should not block and should swallow errors.
        Record(actorID string, action Action, resourceType, resourceID, oldValue, newValue, details string)

        // RecordWithMetadata asynchronously persists an audit entry with request-scoped metadata.
        RecordWithMetadata(meta *RequestMetadata, actorID string, action Action, resourceType, resourceID, oldValue, newValue, details string)

        // Stop drains the buffer and shuts down the background goroutine.
        Stop()
}

// NopAuditor is a zero-dependency no-op auditor used when audit is not configured.
var NopAuditor Auditor = nopAuditor{}

type nopAuditor struct{}

func (nopAuditor) Record(_ string, _ Action, _, _, _, _, _ string) {}
func (nopAuditor) RecordWithMetadata(_ *RequestMetadata, _ string, _ Action, _, _, _, _, _ string) {}
func (nopAuditor) Stop() {}
