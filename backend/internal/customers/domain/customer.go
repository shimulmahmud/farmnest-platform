package domain

import (
	"time"

	"github.com/google/uuid"
)

// Customer represents a customer profile linked to a user account.
type Customer struct {
	ID          string    `bson:"_id" json:"id"`
	UserID      string    `bson:"user_id" json:"user_id"`
	Phone       string    `bson:"phone" json:"phone,omitempty"`
	AvatarURL   string    `bson:"avatar_url" json:"avatar_url,omitempty"`
	DateOfBirth *string   `bson:"date_of_birth" json:"date_of_birth,omitempty"`
	Addresses   []Address `bson:"addresses" json:"addresses"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

// Address represents a shipping/billing address.
type Address struct {
	ID         string `bson:"id" json:"id"`
	Label      string `bson:"label" json:"label"`
	FirstName  string `bson:"first_name" json:"first_name"`
	LastName   string `bson:"last_name" json:"last_name"`
	Line1      string `bson:"line1" json:"line1"`
	Line2      string `bson:"line2" json:"line2,omitempty"`
	City       string `bson:"city" json:"city"`
	State      string `bson:"state" json:"state"`
	PostalCode string `bson:"postal_code" json:"postal_code"`
	Country    string `bson:"country" json:"country"`
	IsDefault  bool   `bson:"is_default" json:"is_default"`
}

// NewCustomer creates a customer profile for a given user.
func NewCustomer(userID string) *Customer {
	now := time.Now().UTC()
	return &Customer{
		ID:        uuid.New().String(),
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddAddress appends an address and marks it default if it's the first.
func (c *Customer) AddAddress(addr Address) {
	if addr.ID == "" {
		addr.ID = uuid.New().String()
	}
	if len(c.Addresses) == 0 {
		addr.IsDefault = true
	}
	c.Addresses = append(c.Addresses, addr)
	c.UpdatedAt = time.Now().UTC()
}

// DefaultAddress returns the default shipping address, or nil.
func (c *Customer) DefaultAddress() *Address {
	for i := range c.Addresses {
		if c.Addresses[i].IsDefault {
			return &c.Addresses[i]
		}
	}
	return nil
}
