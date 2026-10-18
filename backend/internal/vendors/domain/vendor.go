package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ecommerce/backend/pkg/pagination"
)

// Status is the approval state of a vendor.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusApproved  Status = "APPROVED"
	StatusSuspended Status = "SUSPENDED"
)

// IsValid reports whether s is a known vendor status.
func (s Status) IsValid() bool {
	return s == StatusPending || s == StatusApproved || s == StatusSuspended
}

// Vendor is a seller on the marketplace, owned by one user account (role VENDOR).
type Vendor struct {
	ID            string    `bson:"_id" json:"id"`
	OwnerUserID   string    `bson:"owner_user_id" json:"owner_user_id"`
	Name          string    `bson:"name" json:"name"`
	Slug          string    `bson:"slug" json:"slug"`
	Status        Status    `bson:"status" json:"status"`
	CommissionBps int       `bson:"commission_bps" json:"commission_bps"` // 1000 = 10%
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
}

// NewVendor creates a PENDING vendor.
func NewVendor(ownerUserID, name, slug string, commissionBps int) *Vendor {
	now := time.Now().UTC()
	return &Vendor{
		ID: uuid.New().String(), OwnerUserID: ownerUserID, Name: name, Slug: slug,
		Status: StatusPending, CommissionBps: commissionBps, CreatedAt: now, UpdatedAt: now,
	}
}

// Repository persists vendors.
type Repository interface {
	Create(ctx context.Context, v *Vendor) error
	FindByID(ctx context.Context, id string) (*Vendor, error)
	FindByOwner(ctx context.Context, userID string) (*Vendor, error)
	Update(ctx context.Context, v *Vendor) error
	List(ctx context.Context, status string, p pagination.Params) ([]*Vendor, int64, error)
}
