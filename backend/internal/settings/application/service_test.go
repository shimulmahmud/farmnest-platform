package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/settings/domain"
)

type memRepo struct{ saved *domain.Delivery }

func (r *memRepo) GetDelivery(_ context.Context) (*domain.Delivery, error) { return r.saved, nil }
func (r *memRepo) SaveDelivery(_ context.Context, d *domain.Delivery) error {
	cp := *d
	r.saved = &cp
	return nil
}

func TestDelivery_DefaultsUntilSaved_ThenUsesSavedCharges(t *testing.T) {
	svc := NewService(&memRepo{})
	ctx := context.Background()

	d, err := svc.Delivery(ctx)
	require.NoError(t, err)
	_, fee, _ := d.Fee("standard", "inside_dhaka", 100)
	assert.Equal(t, 60, fee, "built-in default")

	next := domain.DefaultDelivery()
	next.Methods[0].FeeInsideDhaka = 80
	next.Methods[0].FeeOutsideDhaka = 150
	next.FreeShippingThreshold = 3000
	saved, err := svc.UpdateDelivery(ctx, next, "admin-1")
	require.NoError(t, err)
	assert.Equal(t, "admin-1", saved.UpdatedBy)
	assert.False(t, saved.UpdatedAt.IsZero())

	d, err = svc.Delivery(ctx)
	require.NoError(t, err)
	_, fee, _ = d.Fee("standard", "outside_dhaka", 100)
	assert.Equal(t, 150, fee, "the shop's new charge is used")
	_, fee, _ = d.Fee("standard", "inside_dhaka", 3000)
	assert.Equal(t, 0, fee, "new free-delivery amount")
}

func TestUpdateDelivery_RejectsBadInput_AndKeepsOldCharges(t *testing.T) {
	repo := &memRepo{}
	svc := NewService(repo)
	bad := domain.DefaultDelivery()
	bad.Methods[0].FeeInsideDhaka = -10
	_, err := svc.UpdateDelivery(context.Background(), bad, "admin-1")
	require.Error(t, err)
	assert.Nil(t, repo.saved, "nothing is saved when the input is invalid")
}
