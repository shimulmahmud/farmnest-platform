package domain

import (
        "testing"

        "github.com/stretchr/testify/assert"
)

func TestNormalizeSearchTerms_Simple(t *testing.T) {
        terms := NormalizeSearchTerms("blue shirt")
        assert.Equal(t, []string{"blue", "shirt"}, terms)
}

func TestNormalizeSearchTerms_LeadingTrailingSpace(t *testing.T) {
        terms := NormalizeSearchTerms("  blue shirt  ")
        assert.Equal(t, []string{"blue", "shirt"}, terms)
}

func TestNormalizeSearchTerms_Empty(t *testing.T) {
        terms := NormalizeSearchTerms("")
        assert.Empty(t, terms)
}

func TestNormalizeSearchTerms_OnlySpaces(t *testing.T) {
        terms := NormalizeSearchTerms("   ")
        assert.Empty(t, terms)
}

func TestNormalizeSearchTerms_MultipleSpaces(t *testing.T) {
        terms := NormalizeSearchTerms("blue   shirt")
        assert.Equal(t, []string{"blue", "shirt"}, terms)
}

func TestNormalizeSearchTerms_CaseLowered(t *testing.T) {
        terms := NormalizeSearchTerms("BLUE SHIRT")
        assert.Equal(t, []string{"blue", "shirt"}, terms)
}

func TestSearchSort_Relevance(t *testing.T) {
        sort := SearchSort("relevance")
        // Relevance falls through to default (created_at desc)
        assert.NotEmpty(t, sort)
}

func TestSearchSort_PriceAsc(t *testing.T) {
        sort := SearchSort("price_asc")
        assert.Equal(t, "lowest_price", sort[0].Key)
        assert.Equal(t, int32(1), sort[0].Value)
}

func TestSearchSort_PriceDesc(t *testing.T) {
        sort := SearchSort("price_desc")
        assert.Equal(t, "lowest_price", sort[0].Key)
        assert.Equal(t, int32(-1), sort[0].Value)
}

func TestSearchSort_CreatedAt(t *testing.T) {
        sort := SearchSort("created_at")
        assert.Equal(t, "created_at", sort[0].Key)
        assert.Equal(t, int32(-1), sort[0].Value)
}

func TestSearchSort_Newest(t *testing.T) {
        sort := SearchSort("newest")
        assert.Equal(t, "created_at", sort[0].Key)
}

func TestSearchSort_Name(t *testing.T) {
        sort := SearchSort("name")
        assert.Equal(t, "name", sort[0].Key)
        assert.Equal(t, int32(1), sort[0].Value)
}

func TestSearchSort_CaseInsensitive(t *testing.T) {
        sort1 := SearchSort("PRICE_ASC")
        sort2 := SearchSort("price_asc")
        assert.Equal(t, sort1[0].Key, sort2[0].Key)
}

func TestSearchSort_Unknown(t *testing.T) {
        sort := SearchSort("unknown_field")
        // Falls through to default
        assert.NotEmpty(t, sort)
}

func TestSearchableProduct_Fields(t *testing.T) {
        product := NewProduct(
                "Test Product", "test-product", "A test product",
                "short", "TestBrand", "cat-1",
                []string{"tag1", "tag2"}, nil, SEO{Title: "Test", Description: "Buy", Keywords: []string{"test", "product"}},
        )
        product.Variants = []Variant{
                *NewVariant("SKU-A", 1000, 500, map[string]string{"size": "M"}),
                *NewVariant("SKU-B", 2000, 800, map[string]string{"size": "L"}),
        }

        evt := product.SearchableProduct()
        assert.Equal(t, product.ID, evt.ProductID)
        assert.Equal(t, "Test Product", evt.Name)
        assert.Equal(t, "test-product", evt.Slug)
        assert.Equal(t, "short", evt.ShortDescription)
        assert.Equal(t, "TestBrand", evt.Brand)
        assert.Equal(t, "cat-1", evt.CategoryID)
        assert.Equal(t, []string{"tag1", "tag2"}, evt.Tags)
        assert.Equal(t, []string{"test", "product"}, evt.SEOKeywords)
        assert.Equal(t, string(ProductStatusDraft), evt.Status)
        assert.Equal(t, []string{"SKU-A", "SKU-B"}, evt.SKUs)
        assert.Equal(t, 1000, evt.LowestPrice)
}

func TestSearchableProduct_EmptyVariants(t *testing.T) {
        product := NewProduct(
                "No Variants", "no-variants", "desc", "",
                "Brand", "cat-1", nil, nil, SEO{},
        )
        evt := product.SearchableProduct()
        assert.Equal(t, "no-variants", evt.Slug)
        assert.Empty(t, evt.SKUs)
        assert.Equal(t, 0, evt.LowestPrice) // no active variants
}

func TestProductEvent_Struct(t *testing.T) {
        evt := &ProductEvent{
                EventType:        "product.created",
                ProductID:       "p-1",
                Name:             "Widget",
                Slug:             "widget",
                ShortDescription: "A useful widget",
                Brand:            "Acme",
                CategoryID:       "cat-1",
                Tags:             []string{"gadget"},
                SEOKeywords:      []string{"widget", "gadget", "acme"},
                Status:           "ACTIVE",
                SKUs:             []string{"W-001"},
                LowestPrice:      999,
        }
        assert.Equal(t, "product.created", evt.EventType)
        assert.Equal(t, "p-1", evt.ProductID)
        assert.Equal(t, "A useful widget", evt.ShortDescription)
        assert.Equal(t, "Acme", evt.Brand)
        assert.Equal(t, []string{"widget", "gadget", "acme"}, evt.SEOKeywords)
        assert.Equal(t, 999, evt.LowestPrice)
}

func TestSearchInput_HasInStock(t *testing.T) {
        input := SearchInput{
                Query:   "blue shirt",
                InStock: true,
                Page:    1,
                PerPage: 20,
        }
        assert.True(t, input.InStock)
        assert.Equal(t, "blue shirt", input.Query)
}

func TestSearchInput_InStockDefault(t *testing.T) {
        input := SearchInput{Query: "test"}
        assert.False(t, input.InStock)
}

func TestNormalizeSearchTerms_SpecialChars(t *testing.T) {
        terms := NormalizeSearchTerms("hello-world test_product")
        assert.Equal(t, []string{"hello-world", "test_product"}, terms)
}

func TestNormalizeSearchTerms_SingleTerm(t *testing.T) {
        terms := NormalizeSearchTerms("blue")
        assert.Equal(t, []string{"blue"}, terms)
}
