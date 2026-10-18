package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeVendorTotals(t *testing.T) {
	items := []OrderItem{
		{VendorID: "v1", CommissionBps: 1000, Pricing: OrderItemPricing{Total: 10000}},
		{VendorID: "v1", CommissionBps: 1000, Pricing: OrderItemPricing{Total: 5005}},
		{VendorID: "v2", CommissionBps: 500, Pricing: OrderItemPricing{Total: 2000}},
		{VendorID: "", Pricing: OrderItemPricing{Total: 999}}, // platform-owned: skipped
	}
	got := ComputeVendorTotals(items)
	require.Len(t, got, 2)
	assert.Equal(t, VendorTotal{VendorID: "v1", Subtotal: 15005, Commission: 1501, Payout: 15005 - 1501}, got[0])
	assert.Equal(t, VendorTotal{VendorID: "v2", Subtotal: 2000, Commission: 100, Payout: 1900}, got[1])
}
