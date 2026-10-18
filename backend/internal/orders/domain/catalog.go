package domain

import "context"

// ResolvedItem is the server-authoritative data for an order line.
type ResolvedItem struct {
	VendorID  string
	CommissionBps int
	ProductID string
	CategoryID string
	VariantID string
	Name      string
	SKU       string
	Barcode   string
	ImageURL  string
	Options   map[string]string
	Currency  string
	UnitPrice int // whole BDT taka (no cents)
	UnitCost  int // our purchase cost at order time, for profit reports
	WeightG   int
	RequiresShipping bool
}

// CatalogResolver looks up trusted order-line data from the product catalog,
// so clients can never dictate price, name or SKU.
type CatalogResolver interface {
	ResolveForOrder(ctx context.Context, productID, variantID string) (*ResolvedItem, error)
}
