package transport

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/products/domain"
)

func TestToPublicProductResponse_HidesInternalFields(t *testing.T) {
	p := domain.NewProduct("Mango", "mango", "d", "s", "Farm", "cat", nil, nil, domain.SEO{})
	p.VendorID = "vendor-1"
	p.Unit = "kg"
	v := domain.NewVariant("MNG-1", 12000, 8000, map[string]string{"grade": "A"})
	archived := domain.NewVariant("MNG-OLD", 10000, 7000, nil)
	archived.Status = domain.VariantStatusArchived
	p.Variants = []domain.Variant{*v, *archived}
	p.RecomputeDerived()

	resp := toPublicProductResponse(p, map[string]int{"MNG-1": 40})
	require.Len(t, resp.Variants, 1, "archived variants must not be public")
	assert.Equal(t, "kg", resp.Unit)
	assert.True(t, resp.InStock)
	assert.True(t, resp.Variants[0].InStock)

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	body := string(raw)
	assert.NotContains(t, body, "cost_price")
	assert.NotContains(t, body, "vendor_id")
	assert.NotContains(t, body, `"stock"`)
	assert.NotContains(t, body, `"status"`)
}

func TestToPublicProductResponse_UnknownSKUIsOutOfStock(t *testing.T) {
	p := domain.NewProduct("n", "s", "d", "sd", "b", "c", nil, nil, domain.SEO{})
	p.Variants = []domain.Variant{*domain.NewVariant("X", 100, 50, nil)}
	resp := toPublicProductResponse(p, map[string]int{})
	assert.False(t, resp.InStock)
}
