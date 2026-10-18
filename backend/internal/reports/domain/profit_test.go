package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFinalize_ComputesGrossNetAndMargin(t *testing.T) {
	s := ProfitSummary{ItemsRevenue: 10000, CostOfGoods: 6500, ShippingCharged: 600, CourierCost: 800}
	s.Finalize()
	assert.Equal(t, 3500, s.GrossProfit)
	assert.Equal(t, 3300, s.NetProfit, "gross + shipping charged - courier cost")
	assert.Equal(t, 35.0, s.MarginPercent)

	d := ProfitSummary{ItemsRevenue: 10000, Discounts: 1000, CostOfGoods: 6500}
	d.Finalize()
	assert.Equal(t, 2500, d.GrossProfit, "coupon discounts reduce profit")
	assert.Equal(t, 27.8, d.MarginPercent, "margin is on what the customer actually paid for items")
}

func TestMargin_RoundsAndHandlesZeroRevenue(t *testing.T) {
	assert.Equal(t, 0.0, Margin(100, 0))
	assert.Equal(t, 33.3, Margin(1, 3))
	assert.Equal(t, -10.0, Margin(-100, 1000), "a loss shows as a negative margin")
}
