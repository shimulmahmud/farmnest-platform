package domain

import (
        "context"
        "errors"
        "time"

        "github.com/google/uuid"
)

// PromotionStatus represents the lifecycle state of a promotion.
type PromotionStatus string

const (
        PromotionDraft     PromotionStatus = "DRAFT"
        PromotionActive    PromotionStatus = "ACTIVE"
        PromotionPaused    PromotionStatus = "PAUSED"
        PromotionExpired   PromotionStatus = "EXPIRED"
        PromotionArchived  PromotionStatus = "ARCHIVED"
)

// DiscountType defines how the promotion discount is applied.
type DiscountType string

const (
        DiscountPercentage DiscountType = "PERCENTAGE" // e.g. 20% off
        DiscountFixed     DiscountType = "FIXED"      // e.g. $10 off
        DiscountFreeShip  DiscountType = "FREE_SHIPPING"
)

var (
        // ErrPromotionNotFound is returned when a promotion cannot be found.
        ErrPromotionNotFound = errors.New("promotion not found")
        // ErrPromotionExpired is returned when a promotion has passed its end date.
        ErrPromotionExpired = errors.New("promotion expired")
        // ErrPromotionNotActive is returned when a promotion is not in ACTIVE status.
        ErrPromotionNotActive = errors.New("promotion not active")
        // ErrInvalidPromotionTransition is returned for invalid status transitions.
        ErrInvalidPromotionTransition = errors.New("invalid promotion status transition")
)

// validPromotionTransitions defines allowed status transitions.
var validPromotionTransitions = map[PromotionStatus][]PromotionStatus{
        PromotionDraft:    {PromotionActive, PromotionArchived},
        PromotionActive:   {PromotionPaused, PromotionExpired, PromotionArchived},
        PromotionPaused:   {PromotionActive, PromotionArchived},
        PromotionExpired:  {PromotionArchived},
        PromotionArchived: {},
}

// Promotion represents a discount or offer.
type Promotion struct {
        ID             string           `bson:"_id" json:"id"`
        Name           string           `bson:"name" json:"name"`
        Code           string           `bson:"code" json:"code"`                           // e.g. "SAVE20"
        Description    string           `bson:"description" json:"description,omitempty"`
        Status         PromotionStatus  `bson:"status" json:"status"`
        DiscountType   DiscountType     `bson:"discount_type" json:"discount_type"`
        DiscountValue  int              `bson:"discount_value" json:"discount_value"`      // percentage (0-100) or fixed cents
        MinOrderCents  int              `bson:"min_order_cents" json:"min_order_cents,omitempty"`
        MaxDiscountCents int            `bson:"max_discount_cents" json:"max_discount_cents,omitempty"`
        UsageLimit     int              `bson:"usage_limit" json:"usage_limit,omitempty"` // 0 = unlimited
        UsageCount     int              `bson:"usage_count" json:"usage_count"`
        AppliesToProductIDs  []string  `bson:"applies_to_product_ids" json:"applies_to_product_ids,omitempty"`
        AppliesToCategoryIDs []string  `bson:"applies_to_category_ids" json:"applies_to_category_ids,omitempty"`
        CampaignID     string           `bson:"campaign_id,omitempty" json:"campaign_id,omitempty"`
        StartAt        *time.Time       `bson:"start_at,omitempty" json:"start_at,omitempty"`
        EndAt          *time.Time       `bson:"end_at,omitempty" json:"end_at,omitempty"`
        CreatedAt      time.Time        `bson:"created_at" json:"created_at"`
        UpdatedAt      time.Time        `bson:"updated_at" json:"updated_at"`
}

// NewPromotion creates a new promotion in DRAFT status.
func NewPromotion(name, code string, discountType DiscountType, discountValue int) *Promotion {
        now := time.Now().UTC()
        return &Promotion{
                ID:            uuid.New().String(),
                Name:          name,
                Code:          code,
                Status:        PromotionDraft,
                DiscountType:  discountType,
                DiscountValue: discountValue,
                CreatedAt:     now,
                UpdatedAt:     now,
        }
}

// CanTransitionTo checks if the promotion can move to the target status.
func (p *Promotion) CanTransitionTo(target PromotionStatus) bool {
        targets, ok := validPromotionTransitions[p.Status]
        if !ok {
                return false
        }
        for _, t := range targets {
                if t == target {
                        return true
                }
        }
        return false
}

// TransitionTo validates and applies a status transition.
func (p *Promotion) TransitionTo(target PromotionStatus) error {
        targets, ok := validPromotionTransitions[p.Status]
        if !ok {
                targets = []PromotionStatus{}
        }
        found := false
        for _, t := range targets {
                if t == target {
                        found = true
                        break
                }
        }
        if !found {
                return ErrInvalidPromotionTransition
        }
        now := time.Now().UTC()
        p.Status = target
        p.UpdatedAt = now
        return nil
}

// IsActive returns true if the promotion is in ACTIVE status and within its date range.
func (p *Promotion) IsActive() bool {
        if p.Status != PromotionActive {
                return false
        }
        now := time.Now().UTC()
        if p.StartAt != nil && now.Before(*p.StartAt) {
                return false
        }
        if p.EndAt != nil && now.After(*p.EndAt) {
                return false
        }
        return true
}

// IsUsable returns true if the promotion can be applied to an order.
func (p *Promotion) IsUsable() bool {
        if !p.IsActive() {
                return false
        }
        if p.UsageLimit > 0 && p.UsageCount >= p.UsageLimit {
                return false
        }
        return true
}

// AppliesToProduct returns true if the promotion applies to the given product.
func (p *Promotion) AppliesToProduct(productID string) bool {
        if len(p.AppliesToProductIDs) == 0 {
                return true // applies to all products
        }
        for _, pid := range p.AppliesToProductIDs {
                if pid == productID {
                        return true
                }
        }
        return false
}

// AppliesToCategory returns true if the promotion applies to the given category.
func (p *Promotion) AppliesToCategory(categoryID string) bool {
        if len(p.AppliesToCategoryIDs) == 0 {
                return true
        }
        for _, cid := range p.AppliesToCategoryIDs {
                if cid == categoryID {
                        return true
                }
        }
        return false
}

// PromotionRepository persists promotions.
type PromotionRepository interface {
        Create(ctx context.Context, promotion *Promotion) error
        FindByID(ctx context.Context, id string) (*Promotion, error)
        FindByCode(ctx context.Context, code string) (*Promotion, error)
        Update(ctx context.Context, promotion *Promotion) error
        List(ctx context.Context, status string, offset, limit int) ([]*Promotion, int64, error)
        RedeemPromotion(ctx context.Context, id string) (*Promotion, error)
}
