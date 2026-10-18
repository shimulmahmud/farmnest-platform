package infrastructure

import (
	"context"

	"github.com/ecommerce/backend/internal/orders/domain"
	productDomain "github.com/ecommerce/backend/internal/products/domain"
	vendorDomain "github.com/ecommerce/backend/internal/vendors/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

// catalogResolver resolves order lines from the live catalog so that price,
// name, SKU and vendor always come from the server, never from the client.
type catalogResolver struct {
	products productDomain.ProductRepository
	vendors  vendorDomain.Repository // optional
}

// NewCatalogResolver creates a domain.CatalogResolver backed by the product repository.
func NewCatalogResolver(products productDomain.ProductRepository, vendors vendorDomain.Repository) domain.CatalogResolver {
	return &catalogResolver{products: products, vendors: vendors}
}

func (r *catalogResolver) ResolveForOrder(ctx context.Context, productID, variantID string) (*domain.ResolvedItem, error) {
	product, err := r.products.FindByID(ctx, productID)
	if err != nil {
		if appErr, ok := errors.IsAppError(err); ok && appErr.Code == "NOT_FOUND" {
			return nil, errors.New("VALIDATION_ERROR", "product not found: "+productID, 422)
		}
		return nil, err
	}
	if !product.IsActive() {
		return nil, errors.New("VALIDATION_ERROR", "product is not available: "+productID, 422)
	}

	variant := product.FindVariant(variantID)
	if variant == nil || variant.Status != productDomain.VariantStatusActive {
		return nil, errors.New("VALIDATION_ERROR", "variant is not available: "+variantID, 422)
	}

	commission := 0
	if product.VendorID != "" && r.vendors != nil {
		vendor, err := r.vendors.FindByID(ctx, product.VendorID)
		if err != nil || vendor.Status != vendorDomain.StatusApproved {
			return nil, errors.New("VALIDATION_ERROR", "seller is not available for product: "+productID, 422)
		}
		commission = vendor.CommissionBps
	}

	return &domain.ResolvedItem{
		VendorID:  product.VendorID,
		CommissionBps: commission,
		ProductID: product.ID,
		CategoryID: product.CategoryID,
		VariantID: variant.ID,
		Name:      product.Name,
		SKU:       variant.SKU,
		Barcode:   variant.Barcode,
		ImageURL:  firstImage(product, variant),
		Options:   variant.Options,
		Currency:  product.Currency,
		UnitPrice: variant.Price,
		UnitCost:  variant.CostPrice,
		WeightG:   variant.WeightG,
		RequiresShipping: variant.RequiresShipping,
	}, nil
}

// firstImage picks the variant image if it has one, else the product's first image.
func firstImage(p *productDomain.Product, v *productDomain.Variant) string {
	if len(v.Images) > 0 {
		return v.Images[0].URL
	}
	if len(p.Images) > 0 {
		return p.Images[0].URL
	}
	return ""
}
