package worker

import (
	"context"
	"fmt"

	custDomain "github.com/ecommerce/backend/internal/customers/domain"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
)

// UserFinder retrieves a user by ID.
type UserFinder interface {
	FindByID(ctx context.Context, id string) (*userDomain.User, error)
}

// CustomerFinder retrieves a customer by user ID.
type CustomerFinder interface {
	FindByUserID(ctx context.Context, userID string) (*custDomain.Customer, error)
}

// OrderCustomerResolver resolves an order's customer ID to contact information.
// For order events: aggregateID is the customer's user ID (from OrderEvent.CustomerID).
// For payment events: the caller must first resolve the order to get customer_id.
type OrderCustomerResolver struct {
	UserFinder     UserFinder
	CustomerFinder CustomerFinder
}

// NewOrderCustomerResolver creates a resolver that looks up user email + customer phone.
func NewOrderCustomerResolver(userFinder UserFinder, customerFinder CustomerFinder) *OrderCustomerResolver {
	return &OrderCustomerResolver{
		UserFinder:     userFinder,
		CustomerFinder: customerFinder,
	}
}

// ResolveContact resolves contact info for an aggregate ID.
// The aggregateID is expected to be a user_id (matching Order.CustomerID).
func (r *OrderCustomerResolver) ResolveContact(ctx context.Context, aggregateID string) (*ContactInfo, error) {
	user, err := r.UserFinder.FindByID(ctx, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user %s: %w", aggregateID, err)
	}

	contact := &ContactInfo{
		Email: user.Email,
	}

	// Look up customer profile for phone number.
	customer, err := r.CustomerFinder.FindByUserID(ctx, aggregateID)
	if err == nil && customer != nil {
		contact.Phone = customer.Phone
	}
	// Customer not found is OK — we still have email.

	return contact, nil
}
