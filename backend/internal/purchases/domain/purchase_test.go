package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeightedAverageCost(t *testing.T) {
	assert.Equal(t, 300, WeightedAverageCost(0, 999, 10, 300), "no stock: new cost wins")
	assert.Equal(t, 330, WeightedAverageCost(10, 300, 10, 360), "equal blend")
	assert.Equal(t, 313, WeightedAverageCost(30, 300, 10, 350), "rounds to whole taka")
}

func TestNewPurchase_TotalsAndNumber(t *testing.T) {
	p := NewPurchase("Dhaka Traders", "INV-1", "wh", "", []Item{
		{SKU: "A", Quantity: 2, UnitCost: 100, LineTotal: 200},
		{SKU: "B", Quantity: 1, UnitCost: 50, LineTotal: 50},
	})
	assert.Equal(t, 250, p.TotalCost)
	assert.Contains(t, p.Number, "PUR-")
}
