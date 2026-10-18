package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/products/domain"
)

func TestMergeVariants_KeepsIDForExistingSKU(t *testing.T) {
	existing := []domain.Variant{*domain.NewVariant("SKU-A", 1000, 500, map[string]string{"size": "M"})}
	origID := existing[0].ID
	origCreated := existing[0].CreatedAt

	merged := mergeVariants(existing, []CreateVariantInput{
		{SKU: "SKU-A", Price: 1250, CostPrice: 600, Options: map[string]string{"size": "M"}},
	})

	require.Len(t, merged, 1)
	assert.Equal(t, origID, merged[0].ID, "variant ID must be stable across edits")
	assert.Equal(t, origCreated, merged[0].CreatedAt)
	assert.Equal(t, 1250, merged[0].Price)
}

func TestMergeVariants_NewSKUGetsNewID_RemovedSKUIsArchived(t *testing.T) {
	existing := []domain.Variant{*domain.NewVariant("OLD", 1000, 500, nil)}

	merged := mergeVariants(existing, []CreateVariantInput{{SKU: "NEW", Price: 2000, CostPrice: 800}})

	require.Len(t, merged, 2)
	assert.Equal(t, "NEW", merged[0].SKU)
	assert.Equal(t, domain.VariantStatusActive, merged[0].Status)
	assert.Equal(t, "OLD", merged[1].SKU)
	assert.Equal(t, domain.VariantStatusArchived, merged[1].Status, "removed variants are archived, not deleted")
}

func TestRecomputeDerived_IgnoresArchivedVariants(t *testing.T) {
	p := domain.NewProduct("n", "s", "d", "sd", "b", "c", nil, nil, domain.SEO{})
	cheap := domain.NewVariant("CHEAP", 100, 50, nil)
	cheap.Status = domain.VariantStatusArchived
	p.Variants = []domain.Variant{*cheap, *domain.NewVariant("REAL", 900, 400, nil)}
	p.RecomputeDerived()
	assert.Equal(t, 900, p.LowestPrice)
}
