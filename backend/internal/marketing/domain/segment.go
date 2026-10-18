package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// SegmentType categorizes the customer segment.
type SegmentType string

const (
	SegmentTypeStatic  SegmentType = "STATIC"   // manually curated customer list
	SegmentTypeDynamic SegmentType = "DYNAMIC"  // rule-based, evaluated from analytics events
)

var (
	// ErrSegmentNotFound is returned when a segment cannot be found.
	ErrSegmentNotFound = errors.New("segment not found")
)

// SegmentRule defines a single condition for dynamic segment membership.
type SegmentRule struct {
	Field    string `bson:"field" json:"field"`       // e.g. "event_type", "product_id", "category_id"
	Operator string `bson:"operator" json:"operator"` // "eq", "neq", "gte", "lte", "in"
	Value    any    `bson:"value" json:"value"`
}

// Segment represents a group of customers for marketing targeting.
type Segment struct {
	ID          string        `bson:"_id" json:"id"`
	Name        string        `bson:"name" json:"name"`
	Type        SegmentType   `bson:"type" json:"type"`
	Description string        `bson:"description" json:"description,omitempty"`
	Rules       []SegmentRule `bson:"rules" json:"rules,omitempty"`
	CustomerIDs []string      `bson:"customer_ids" json:"customer_ids,omitempty"` // for STATIC segments
	CustomerCount int64       `bson:"customer_count" json:"customer_count"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

// NewSegment creates a new customer segment.
func NewSegment(name string, segType SegmentType, rules []SegmentRule) *Segment {
	now := time.Now().UTC()
	return &Segment{
		ID:            uuid.New().String(),
		Name:          name,
		Type:          segType,
		Rules:         rules,
		CustomerIDs:   []string{},
		CustomerCount: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// SegmentRepository persists customer segments.
type SegmentRepository interface {
	Create(ctx context.Context, segment *Segment) error
	FindByID(ctx context.Context, id string) (*Segment, error)
	Update(ctx context.Context, segment *Segment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*Segment, int64, error)
	FindMatchingSegments(ctx context.Context, customerID string, eventType string) ([]*Segment, error)
}
