package domain

import (
	"context"
	"math"
	"time"
)

// All money is whole BDT taka.

// Range is the period a report covers (inclusive), by order date.
type Range struct {
	From     time.Time
	To       time.Time
	Statuses []string // orders counted; default SHIPPED, DELIVERED
}

// ProfitSummary is the headline profit picture for a period.
type ProfitSummary struct {
	Orders           int     `json:"orders"`
	UnitsSold        int     `json:"units_sold"`
	ItemsRevenue     int     `json:"items_revenue"`     // sum of item totals
	Discounts        int     `json:"discounts"`         // coupon discounts given
	ShippingCharged  int     `json:"shipping_charged"`  // shipping fees customers paid us
	CostOfGoods      int     `json:"cost_of_goods"`     // what the goods cost us
	CourierCost      int     `json:"courier_cost"`      // what couriers charged us
	GrossProfit      int     `json:"gross_profit"`      // items_revenue - cost_of_goods
	NetProfit        int     `json:"net_profit"`        // gross + shipping_charged - courier_cost
	MarginPercent    float64 `json:"margin_percent"`    // gross_profit / (items_revenue - discounts) * 100
	ItemsMissingCost int     `json:"items_missing_cost"` // order items with no recorded cost: profit is overstated
}

// Finalize derives profit and margin from the summed fields.
func (s *ProfitSummary) Finalize() {
	s.GrossProfit = s.ItemsRevenue - s.Discounts - s.CostOfGoods
	s.NetProfit = s.GrossProfit + s.ShippingCharged - s.CourierCost
	s.MarginPercent = Margin(s.GrossProfit, s.ItemsRevenue-s.Discounts)
}

// Margin returns profit as a percentage of revenue, rounded to 1 decimal (0 if no revenue).
func Margin(profit, revenue int) float64 {
	if revenue <= 0 {
		return 0
	}
	return math.Round(float64(profit)/float64(revenue)*1000) / 10
}

// ProfitPoint is one day or month in the profit trend.
type ProfitPoint struct {
	Period      string  `json:"period"`
	Orders      int     `json:"orders"`
	Revenue     int     `json:"revenue"`
	CostOfGoods int     `json:"cost_of_goods"`
	GrossProfit int     `json:"gross_profit"`
	Margin      float64 `json:"margin_percent"`
}

// ProductProfit is the profit of one SKU in the period.
type ProductProfit struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	UnitsSold   int     `json:"units_sold"`
	Revenue     int     `json:"revenue"`
	CostOfGoods int     `json:"cost_of_goods"`
	GrossProfit int     `json:"gross_profit"`
	Margin      float64 `json:"margin_percent"`
}

// OrderProfit is the profit of a single order.
type OrderProfit struct {
	OrderID       string    `json:"order_id"`
	OrderNumber   string    `json:"order_number"`
	CreatedAt     time.Time `json:"created_at"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"payment_method"`
	PaymentStatus string    `json:"payment_status"`
	ItemsRevenue  int       `json:"items_revenue"`
	Discount      int       `json:"discount"`
	CostOfGoods   int       `json:"cost_of_goods"`
	ShippingCharged int     `json:"shipping_charged"`
	CourierCost   int       `json:"courier_cost"`
	GrossProfit   int       `json:"gross_profit"`
	NetProfit     int       `json:"net_profit"`
}

// Repository reads profit figures from order data.
type Repository interface {
	Summary(ctx context.Context, r Range) (*ProfitSummary, error)
	Trend(ctx context.Context, r Range, byMonth bool) ([]ProfitPoint, error)
	Products(ctx context.Context, r Range, limit int, worstFirst bool) ([]ProductProfit, error)
	Orders(ctx context.Context, r Range, skip, limit int) ([]OrderProfit, int64, error)
}
