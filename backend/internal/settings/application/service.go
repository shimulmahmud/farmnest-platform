package application

import (
	"context"
	"time"

	"github.com/ecommerce/backend/internal/settings/domain"
)

// Service reads and updates the shop's settings.
type Service struct{ repo domain.Repository }

// NewService creates a settings service.
func NewService(repo domain.Repository) *Service { return &Service{repo: repo} }

// Delivery returns the saved delivery charges, or the defaults until the shop saves its own.
func (s *Service) Delivery(ctx context.Context) (domain.Delivery, error) {
	d, err := s.repo.GetDelivery(ctx)
	if err != nil {
		return domain.Delivery{}, err
	}
	if d == nil {
		return domain.DefaultDelivery(), nil
	}
	return *d, nil
}

// UpdateDelivery validates and saves new delivery charges. It affects NEW orders only:
// existing orders keep the delivery fee they were placed with.
func (s *Service) UpdateDelivery(ctx context.Context, d domain.Delivery, byUserID string) (*domain.Delivery, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	d.UpdatedAt = time.Now().UTC()
	d.UpdatedBy = byUserID
	if err := s.repo.SaveDelivery(ctx, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
