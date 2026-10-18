package application

import (
	"context"

	userDomain "github.com/ecommerce/backend/internal/users/domain"
	"github.com/ecommerce/backend/internal/vendors/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
)

// Service implements vendor use cases.
type Service struct {
	repo  domain.Repository
	users userDomain.UserRepository
}

// NewService creates a vendor service.
func NewService(repo domain.Repository, users userDomain.UserRepository) *Service {
	return &Service{repo: repo, users: users}
}

// Create registers a vendor for an existing user and grants that user the VENDOR role.
func (s *Service) Create(ctx context.Context, ownerUserID, name, slug string, commissionBps int) (*domain.Vendor, error) {
	if ownerUserID == "" || name == "" || slug == "" {
		return nil, errors.New("VALIDATION_ERROR", "owner_user_id, name and slug are required", 422)
	}
	if commissionBps < 0 || commissionBps > 10000 {
		return nil, errors.New("VALIDATION_ERROR", "commission_bps must be between 0 and 10000", 422)
	}
	user, err := s.users.FindByID(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	if user.Role != userDomain.RoleCustomer && user.Role != userDomain.RoleVendor {
		return nil, errors.New("VALIDATION_ERROR", "owner must be a customer account", 422)
	}
	v := domain.NewVendor(ownerUserID, name, slug, commissionBps)
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	if user.Role != userDomain.RoleVendor {
		if err := s.users.UpdateRole(ctx, ownerUserID, userDomain.RoleVendor); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// SetStatus approves or suspends a vendor.
func (s *Service) SetStatus(ctx context.Context, id string, status domain.Status) (*domain.Vendor, error) {
	if !status.IsValid() {
		return nil, errors.New("VALIDATION_ERROR", "status must be PENDING, APPROVED or SUSPENDED", 422)
	}
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	v.Status = status
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

// Get returns a vendor by ID.
func (s *Service) Get(ctx context.Context, id string) (*domain.Vendor, error) {
	return s.repo.FindByID(ctx, id)
}

// GetByOwner returns the vendor owned by a user.
func (s *Service) GetByOwner(ctx context.Context, userID string) (*domain.Vendor, error) {
	return s.repo.FindByOwner(ctx, userID)
}

// List returns vendors, optionally filtered by status.
func (s *Service) List(ctx context.Context, status string, p pagination.Params) ([]*domain.Vendor, int64, error) {
	return s.repo.List(ctx, status, p)
}
