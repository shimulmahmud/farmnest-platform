package domain

import "context"

// CouponLine is one cart line as the coupon engine needs it. Total is whole BDT after item discounts.
type CouponLine struct {
	ProductID  string
	CategoryID string
	Total      int
}

// CouponResult is the outcome of checking a coupon code against a cart.
type CouponResult struct {
	PromotionID  string
	Code         string
	Discount     int  // whole BDT taken off the items
	FreeShipping bool // delivery fee waived
	Message      string
}

// CouponApplier validates coupon codes and tracks their usage. Optional: when nil, coupons are off.
type CouponApplier interface {
	// Quote checks the code for this customer and cart WITHOUT using it. Errors are
	// VALIDATION_ERROR (422) with a customer-readable message (expired, minimum order, already used...).
	Quote(ctx context.Context, code, customerID string, lines []CouponLine, subtotal int) (*CouponResult, error)
	// Redeem atomically counts one use (fails if the usage limit was just reached).
	Redeem(ctx context.Context, promotionID string) error
	// Release gives a use back (the order failed or was cancelled).
	Release(ctx context.Context, promotionID string) error
}
