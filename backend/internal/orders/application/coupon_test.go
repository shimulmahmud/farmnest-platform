package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/orders/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

type fakeCoupons struct {
	redeemed, released []string
	failRedeem         bool
}

func (f *fakeCoupons) Quote(_ context.Context, _, _ string, _ []domain.CouponLine, _ int) (*domain.CouponResult, error) {
	return &domain.CouponResult{}, nil
}
func (f *fakeCoupons) Redeem(_ context.Context, id string) error {
	if f.failRedeem {
		return errors.New("VALIDATION_ERROR", "this coupon has just been fully used", 422)
	}
	f.redeemed = append(f.redeemed, id)
	return nil
}
func (f *fakeCoupons) Release(_ context.Context, id string) error {
	f.released = append(f.released, id)
	return nil
}

func couponOrder(svc *Service, method domain.PaymentMethod) (*domain.Order, error) {
	pricing := testPricing()
	pricing.Discount = 500
	pricing.Total -= 500
	return svc.CreateOrder(context.Background(), CreateOrderInput{
		CustomerID: "cust-1", Items: testItems(), Pricing: pricing,
		ShippingAddress: testAddress(), BillingAddress: testAddress(),
		PaymentMethod: method, CouponCode: "SAVE10", PromotionID: "promo-1",
	})
}

func TestCoupon_UseIsCountedWhenOrderIsPlaced(t *testing.T) {
	svc, _, _ := newTestService(t)
	fc := &fakeCoupons{}
	svc.SetCoupons(fc)

	order, err := couponOrder(svc, domain.PaymentOnline)
	require.NoError(t, err)
	assert.Equal(t, "SAVE10", order.CouponCode)
	assert.Equal(t, 500, order.Pricing.Discount)
	assert.Equal(t, []string{"promo-1"}, fc.redeemed)
}

func TestCoupon_RejectedWhenJustFullyUsed_NoOrderSaved(t *testing.T) {
	svc, repo, _ := newTestService(t)
	svc.SetCoupons(&fakeCoupons{failRedeem: true})

	_, err := couponOrder(svc, domain.PaymentOnline)
	require.Error(t, err)
	assert.Empty(t, repo.orders)
}

func TestCoupon_GivenBackWhenStockIsShort(t *testing.T) {
	svc, repo, _ := newTestService(t)
	fc := &fakeCoupons{}
	svc.SetCoupons(fc)
	svc.SetStock(&fakeStock{failDeduct: true})

	_, err := couponOrder(svc, domain.PaymentCOD)
	require.Error(t, err)
	assert.Empty(t, repo.orders)
	assert.Equal(t, []string{"promo-1"}, fc.released, "the coupon use is returned")
}

func TestCoupon_GivenBackWhenOrderIsCancelled(t *testing.T) {
	svc, _, _ := newTestService(t)
	fc := &fakeCoupons{}
	svc.SetCoupons(fc)

	order, err := couponOrder(svc, domain.PaymentOnline)
	require.NoError(t, err)
	_, err = svc.UpdateOrderStatus(context.Background(), order.ID, UpdateStatusInput{Status: string(domain.OrderCancelled)})
	require.NoError(t, err)
	assert.Equal(t, []string{"promo-1"}, fc.released)
}
