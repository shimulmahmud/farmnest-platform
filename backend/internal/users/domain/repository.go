package domain

import (
        "context"

        "github.com/ecommerce/backend/pkg/pagination"
)

// UserRepository defines the persistence operations for users.
type UserRepository interface {
        Create(ctx context.Context, user *User) error
        FindByID(ctx context.Context, id string) (*User, error)
        FindByEmail(ctx context.Context, email string) (*User, error)
        Update(ctx context.Context, user *User) error
        ExistsByEmail(ctx context.Context, email string) (bool, error)

        // List returns users with optional role/status filters and pagination.
        List(ctx context.Context, filter *UserFilter, p pagination.Params) ([]*User, int64, error)

        // Count returns total users matching the optional filter.
        Count(ctx context.Context, filter *UserFilter) (int64, error)

        // UpdateRole changes a user's role. Returns ErrNotFound if user doesn't exist.
        UpdateRole(ctx context.Context, userID string, role Role) error

        // UpdateStatus changes a user's account status. Returns ErrNotFound if user doesn't exist.
        UpdateStatus(ctx context.Context, userID string, status AccountStatus) error
}

// UserFilter defines optional filters for listing users.
type UserFilter struct {
        Role   Role
        Status AccountStatus
        Search string // matches email, first_name, or last_name (case-insensitive)
}
