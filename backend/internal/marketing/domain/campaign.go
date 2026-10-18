package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CampaignStatus represents the lifecycle state of a marketing campaign.
type CampaignStatus string

const (
	CampaignDraft     CampaignStatus = "DRAFT"
	CampaignActive    CampaignStatus = "ACTIVE"
	CampaignPaused    CampaignStatus = "PAUSED"
	CampaignCompleted CampaignStatus = "COMPLETED"
	CampaignArchived  CampaignStatus = "ARCHIVED"
)

// CampaignType categorizes the campaign.
type CampaignType string

const (
	CampaignTypeAbandonedCart   CampaignType = "ABANDONED_CART"
	CampaignTypeProductBrowse   CampaignType = "PRODUCT_BROWSE"
	CampaignTypeCheckoutRecovery CampaignType = "CHECKOUT_RECOVERY"
	CampaignTypePostPurchase   CampaignType = "POST_PURCHASE"
	CampaignTypePromotional    CampaignType = "PROMOTIONAL"
)

var (
	// ErrCampaignNotFound is returned when a campaign cannot be found.
	ErrCampaignNotFound = errors.New("campaign not found")
	// ErrInvalidCampaignTransition is returned when a status transition is invalid.
	ErrInvalidCampaignTransition = errors.New("invalid campaign status transition")
)

// validCampaignTransitions defines allowed status transitions.
var validCampaignTransitions = map[CampaignStatus][]CampaignStatus{
	CampaignDraft:     {CampaignActive, CampaignArchived},
	CampaignActive:    {CampaignPaused, CampaignCompleted, CampaignArchived},
	CampaignPaused:    {CampaignActive, CampaignArchived},
	CampaignCompleted: {CampaignArchived},
	CampaignArchived:  {},
}

// Campaign represents a marketing campaign.
type Campaign struct {
	ID          string             `bson:"_id" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Type        CampaignType       `bson:"type" json:"type"`
	Status      CampaignStatus     `bson:"status" json:"status"`
	Description string             `bson:"description" json:"description,omitempty"`
	Rules       CampaignRules      `bson:"rules" json:"rules"`
	Metrics     CampaignMetrics    `bson:"metrics" json:"metrics"`
	StartAt     *time.Time         `bson:"start_at,omitempty" json:"start_at,omitempty"`
	EndAt       *time.Time         `bson:"end_at,omitempty" json:"end_at,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// CampaignRules defines the targeting and trigger rules.
type CampaignRules struct {
	SegmentIDs         []string `bson:"segment_ids" json:"segment_ids"`
	EventTypes         []string `bson:"event_types" json:"event_types"`             // which analytics events trigger this campaign
	ProductIDs         []string `bson:"product_ids,omitempty" json:"product_ids,omitempty"`
	CategoryIDs        []string `bson:"category_ids,omitempty" json:"category_ids,omitempty"`
	MinAbandonedMinutes int     `bson:"min_abandoned_minutes,omitempty" json:"min_abandoned_minutes,omitempty"`
	MaxEmailsPerUser   int     `bson:"max_emails_per_user,omitempty" json:"max_emails_per_user,omitempty"`
}

// CampaignMetrics tracks campaign performance (updated atomically).
type CampaignMetrics struct {
	EventsReceived int64 `bson:"events_received" json:"events_received"`
	CustomersReached int64 `bson:"customers_reached" json:"customers_reached"`
	MessagesSent   int64 `bson:"messages_sent" json:"messages_sent"`
	Conversions    int64 `bson:"conversions" json:"conversions"`
	RevenueCents   int64 `bson:"revenue_cents" json:"revenue_cents"`
}

// NewCampaign creates a new campaign in DRAFT status.
func NewCampaign(name string, campaignType CampaignType, rules CampaignRules) *Campaign {
	now := time.Now().UTC()
	return &Campaign{
		ID:        uuid.New().String(),
		Name:      name,
		Type:      campaignType,
		Status:    CampaignDraft,
		Rules:     rules,
		Metrics:   CampaignMetrics{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CanTransitionTo checks if the campaign can move to the target status.
func (c *Campaign) CanTransitionTo(target CampaignStatus) bool {
	targets, ok := validCampaignTransitions[c.Status]
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
func (c *Campaign) TransitionTo(target CampaignStatus) error {
	if !c.CanTransitionTo(target) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidCampaignTransition, c.Status, target)
	}
	now := time.Now().UTC()
	c.Status = target
	c.UpdatedAt = now
	return nil
}

// IsActive returns true if the campaign is in ACTIVE status.
func (c *Campaign) IsActive() bool {
	return c.Status == CampaignActive
}

// MatchesEventType returns true if the campaign is triggered by the given event type.
func (c *Campaign) MatchesEventType(eventType string) bool {
	for _, et := range c.Rules.EventTypes {
		if et == eventType {
			return true
		}
	}
	return false
}

// MatchesProduct returns true if the campaign targets the given product (empty product list = matches all).
func (c *Campaign) MatchesProduct(productID string) bool {
	if len(c.Rules.ProductIDs) == 0 {
		return true
	}
	for _, pid := range c.Rules.ProductIDs {
		if pid == productID {
			return true
		}
	}
	return false
}

// MatchesCategory returns true if the campaign targets the given category (empty = matches all).
func (c *Campaign) MatchesCategory(categoryID string) bool {
	if len(c.Rules.CategoryIDs) == 0 {
		return true
	}
	for _, cid := range c.Rules.CategoryIDs {
		if cid == categoryID {
			return true
		}
	}
	return false
}

// CampaignRepository persists campaigns.
type CampaignRepository interface {
	Create(ctx context.Context, campaign *Campaign) error
	FindByID(ctx context.Context, id string) (*Campaign, error)
	Update(ctx context.Context, campaign *Campaign) error
	FindActiveByEventType(ctx context.Context, eventType string) ([]*Campaign, error)
	List(ctx context.Context, status string, offset, limit int) ([]*Campaign, int64, error)
	IncrementMetrics(ctx context.Context, id string, field string, value int64) error
}
