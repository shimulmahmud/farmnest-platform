package domain

import (
        "context"

        "github.com/ecommerce/backend/pkg/pagination"
)

// CustomerRepository defines the persistence operations for customer profiles.
type CustomerRepository interface {
        Create(ctx context.Context, customer *Customer) error
        FindByUserID(ctx context.Context, userID string) (*Customer, error)
        FindByID(ctx context.Context, id string) (*Customer, error)
        Update(ctx context.Context, customer *Customer) error
        List(ctx context.Context, p pagination.Params) ([]*Customer, int64, error)
        Count(ctx context.Context) (int64, error)
}
