package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ecommerce/backend/internal/orders/domain"
	mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// couponApplier reads and updates the "promotions" collection directly (the marketing
// module owns the data and the admin screens; this only applies codes at checkout).
type couponApplier struct {
	promos *mongo.Collection
	orders *mongo.Collection
}

// NewCouponApplier creates a domain.CouponApplier backed by MongoDB.
func NewCouponApplier(db *mongo.Database) domain.CouponApplier {
	return &couponApplier{promos: db.Collection("promotions"), orders: db.Collection("orders")}
}

func invalid(msg string) error { return errors.New("VALIDATION_ERROR", msg, 422) }

func (c *couponApplier) Quote(ctx context.Context, code, customerID string, lines []domain.CouponLine, subtotal int) (*domain.CouponResult, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, invalid("enter a coupon code")
	}

	var promo mktDomain.Promotion
	if err := c.promos.FindOne(ctx, bson.M{"code": code}).Decode(&promo); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, invalid("this coupon code is not valid")
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to check coupon", 500, err)
	}

	now := time.Now().UTC()
	switch {
	case promo.Status != mktDomain.PromotionActive:
		return nil, invalid("this coupon is not active")
	case promo.StartAt != nil && now.Before(*promo.StartAt):
		return nil, invalid("this coupon is not active yet")
	case promo.EndAt != nil && now.After(*promo.EndAt):
		return nil, invalid("this coupon has expired")
	case promo.UsageLimit > 0 && promo.UsageCount >= promo.UsageLimit:
		return nil, invalid("this coupon has been fully used")
	}
	// Money fields keep their old "cents" names but hold whole BDT taka.
	if promo.MinOrderCents > 0 && subtotal < promo.MinOrderCents {
		return nil, invalid(fmt.Sprintf("this coupon needs an order of at least ৳%d", promo.MinOrderCents))
	}

	// One use per customer.
	used, err := c.orders.CountDocuments(ctx, bson.M{
		"customer_id": customerID,
		"coupon_code": code,
		"status":      bson.M{"$nin": bson.A{"CANCELLED", "REFUNDED"}},
	})
	if err != nil {
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to check coupon use", 500, err)
	}
	if used > 0 {
		return nil, invalid("you have already used this coupon")
	}

	return applyPromotion(&promo, code, lines, subtotal)
}

// applyPromotion works out the discount of a usable promotion for a cart (pure: no database).
func applyPromotion(promo *mktDomain.Promotion, code string, lines []domain.CouponLine, subtotal int) (*domain.CouponResult, error) {
	// Only lines the coupon applies to count toward the discount.
	eligible := 0
	for _, l := range lines {
		if promo.AppliesToProduct(l.ProductID) && promo.AppliesToCategory(l.CategoryID) {
			eligible += l.Total
		}
	}
	if eligible == 0 {
		return nil, invalid("this coupon does not apply to the items in your cart")
	}

	res := &domain.CouponResult{PromotionID: promo.ID, Code: code}
	switch promo.DiscountType {
	case mktDomain.DiscountPercentage:
		res.Discount = eligible * promo.DiscountValue / 100
		if promo.MaxDiscountCents > 0 && res.Discount > promo.MaxDiscountCents {
			res.Discount = promo.MaxDiscountCents
		}
		res.Message = fmt.Sprintf("%d%% off applied", promo.DiscountValue)
	case mktDomain.DiscountFixed:
		res.Discount = promo.DiscountValue
		if res.Discount > eligible {
			res.Discount = eligible
		}
		res.Message = fmt.Sprintf("৳%d off applied", res.Discount)
	case mktDomain.DiscountFreeShip:
		res.FreeShipping = true
		res.Message = "Free delivery applied"
	default:
		return nil, invalid("this coupon cannot be used")
	}
	if res.Discount > subtotal {
		res.Discount = subtotal
	}
	return res, nil
}

func (c *couponApplier) Redeem(ctx context.Context, promotionID string) error {
	res, err := c.promos.UpdateOne(ctx,
		bson.M{
			"_id":    promotionID,
			"status": mktDomain.PromotionActive,
			"$or": bson.A{
				bson.M{"usage_limit": bson.M{"$in": bson.A{0, nil}}},
				bson.M{"$expr": bson.M{"$lt": bson.A{"$usage_count", "$usage_limit"}}},
			},
		},
		bson.M{"$inc": bson.M{"usage_count": 1}},
	)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to use coupon", 500, err)
	}
	if res.MatchedCount == 0 {
		return invalid("this coupon has just been fully used")
	}
	return nil
}

func (c *couponApplier) Release(ctx context.Context, promotionID string) error {
	_, err := c.promos.UpdateOne(ctx,
		bson.M{"_id": promotionID, "usage_count": bson.M{"$gt": 0}},
		bson.M{"$inc": bson.M{"usage_count": -1}},
	)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to release coupon", 500, err)
	}
	return nil
}
