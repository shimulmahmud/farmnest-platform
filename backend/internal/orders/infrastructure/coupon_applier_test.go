package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
	"github.com/ecommerce/backend/internal/orders/domain"
)

var cart = []domain.CouponLine{
	{ProductID: "p1", CategoryID: "electronics", Total: 8000},
	{ProductID: "p2", CategoryID: "fashion", Total: 2000},
}

func promo(t mktDomain.DiscountType, value int) *mktDomain.Promotion {
	return &mktDomain.Promotion{ID: "promo-1", DiscountType: t, DiscountValue: value}
}

func TestApplyPromotion_Percentage_WithCap(t *testing.T) {
	p := promo(mktDomain.DiscountPercentage, 10)
	r, err := applyPromotion(p, "SAVE10", cart, 10000)
	require.NoError(t, err)
	assert.Equal(t, 1000, r.Discount, "10% of 10,000")

	p.MaxDiscountCents = 300
	r, _ = applyPromotion(p, "SAVE10", cart, 10000)
	assert.Equal(t, 300, r.Discount, "capped at the maximum discount")
}

func TestApplyPromotion_Fixed_NeverMoreThanTheItems(t *testing.T) {
	r, err := applyPromotion(promo(mktDomain.DiscountFixed, 100), "FLAT100", cart, 10000)
	require.NoError(t, err)
	assert.Equal(t, 100, r.Discount)

	r, _ = applyPromotion(promo(mktDomain.DiscountFixed, 99999), "BIG", cart, 10000)
	assert.Equal(t, 10000, r.Discount)
}

func TestApplyPromotion_OnlyEligibleCategoriesCount(t *testing.T) {
	p := promo(mktDomain.DiscountPercentage, 10)
	p.AppliesToCategoryIDs = []string{"electronics"}
	r, err := applyPromotion(p, "ELEC10", cart, 10000)
	require.NoError(t, err)
	assert.Equal(t, 800, r.Discount, "10% of the electronics line only")

	p.AppliesToCategoryIDs = []string{"beauty"}
	_, err = applyPromotion(p, "ELEC10", cart, 10000)
	require.Error(t, err, "nothing in the cart is eligible")
}

func TestApplyPromotion_FreeShipping(t *testing.T) {
	r, err := applyPromotion(promo(mktDomain.DiscountFreeShip, 0), "FREESHIP", cart, 10000)
	require.NoError(t, err)
	assert.True(t, r.FreeShipping)
	assert.Equal(t, 0, r.Discount)
}
