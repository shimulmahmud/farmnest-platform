package infrastructure

import (
	"context"
	"time"

	"github.com/ecommerce/backend/internal/reports/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// tz is Bangladesh time; days and months are cut at local midnight.
const tz = "+06:00"

type repository struct{ coll *mongo.Collection }

// NewRepository creates a profit repository that aggregates the orders collection.
func NewRepository(db *mongo.Database) domain.Repository {
	return &repository{coll: db.Collection("orders")}
}

func match(r domain.Range) bson.D {
	return bson.D{
		{Key: "created_at", Value: bson.D{{Key: "$gte", Value: r.From}, {Key: "$lte", Value: r.To}}},
		{Key: "status", Value: bson.D{{Key: "$in", Value: r.Statuses}}},
	}
}

// itemCost is the cost of one unwound/mapped item: unit_cost * quantity (0 if no cost recorded).
func itemCost(ref string) bson.D {
	return bson.D{{Key: "$multiply", Value: bson.A{
		bson.D{{Key: "$ifNull", Value: bson.A{ref + ".unit_cost", 0}}},
		ref + ".quantity",
	}}}
}

// perOrder adds per-order figures computed from the items array.
func perOrder() bson.D {
	return bson.D{{Key: "$addFields", Value: bson.D{
		{Key: "o_revenue", Value: bson.D{{Key: "$sum", Value: "$items.pricing.total"}}},
		{Key: "o_units", Value: bson.D{{Key: "$sum", Value: "$items.quantity"}}},
		{Key: "o_cogs", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$map", Value: bson.D{
			{Key: "input", Value: "$items"}, {Key: "as", Value: "i"}, {Key: "in", Value: itemCost("$$i")},
		}}}}}},
		{Key: "o_missing", Value: bson.D{{Key: "$size", Value: bson.D{{Key: "$filter", Value: bson.D{
			{Key: "input", Value: "$items"}, {Key: "as", Value: "i"},
			{Key: "cond", Value: bson.D{{Key: "$eq", Value: bson.A{bson.D{{Key: "$ifNull", Value: bson.A{"$$i.unit_cost", 0}}}, 0}}}},
		}}}}}},
		{Key: "o_discount", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$pricing.discount", 0}}}},
		{Key: "o_shipping", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$pricing.shipping", 0}}}},
		{Key: "o_courier", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$courier_cost", 0}}}},
	}}}
}

func dbErr(err error) error {
	return errors.Wrap("INTERNAL_ERROR", "failed to build report", 500, err)
}

func (r *repository) Summary(ctx context.Context, rg domain.Range) (*domain.ProfitSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match(rg)}},
		perOrder(),
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "orders", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "units", Value: bson.D{{Key: "$sum", Value: "$o_units"}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$o_revenue"}}},
			{Key: "cogs", Value: bson.D{{Key: "$sum", Value: "$o_cogs"}}},
			{Key: "discounts", Value: bson.D{{Key: "$sum", Value: "$o_discount"}}},
			{Key: "shipping", Value: bson.D{{Key: "$sum", Value: "$o_shipping"}}},
			{Key: "courier", Value: bson.D{{Key: "$sum", Value: "$o_courier"}}},
			{Key: "missing", Value: bson.D{{Key: "$sum", Value: "$o_missing"}}},
		}}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, dbErr(err)
	}
	defer cur.Close(ctx)

	var rows []struct {
		Orders, Units, Revenue, Discounts, Cogs, Shipping, Courier, Missing int
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, dbErr(err)
	}
	s := &domain.ProfitSummary{}
	if len(rows) > 0 {
		x := rows[0]
		s.Orders, s.UnitsSold, s.ItemsRevenue, s.CostOfGoods = x.Orders, x.Units, x.Revenue, x.Cogs
		s.Discounts = x.Discounts
		s.ShippingCharged, s.CourierCost, s.ItemsMissingCost = x.Shipping, x.Courier, x.Missing
	}
	s.Finalize()
	return s, nil
}

func (r *repository) Trend(ctx context.Context, rg domain.Range, byMonth bool) ([]domain.ProfitPoint, error) {
	format := "%Y-%m-%d"
	if byMonth {
		format = "%Y-%m"
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match(rg)}},
		perOrder(),
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{
				{Key: "format", Value: format}, {Key: "date", Value: "$created_at"}, {Key: "timezone", Value: tz},
			}}}},
			{Key: "orders", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$subtract", Value: bson.A{"$o_revenue", "$o_discount"}}}}}},
			{Key: "cogs", Value: bson.D{{Key: "$sum", Value: "$o_cogs"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, dbErr(err)
	}
	defer cur.Close(ctx)

	var rows []struct {
		Period  string `bson:"_id"`
		Orders  int    `bson:"orders"`
		Revenue int    `bson:"revenue"`
		Cogs    int    `bson:"cogs"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, dbErr(err)
	}
	out := make([]domain.ProfitPoint, len(rows))
	for i, x := range rows {
		gp := x.Revenue - x.Cogs
		out[i] = domain.ProfitPoint{Period: x.Period, Orders: x.Orders, Revenue: x.Revenue,
			CostOfGoods: x.Cogs, GrossProfit: gp, Margin: domain.Margin(gp, x.Revenue)}
	}
	return out, nil
}

func (r *repository) Products(ctx context.Context, rg domain.Range, limit int, worstFirst bool) ([]domain.ProductProfit, error) {
	dir := -1
	if worstFirst {
		dir = 1
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match(rg)}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$items.sku"},
			{Key: "name", Value: bson.D{{Key: "$first", Value: "$items.name"}}},
			{Key: "units", Value: bson.D{{Key: "$sum", Value: "$items.quantity"}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$items.pricing.total"}}},
			{Key: "cogs", Value: bson.D{{Key: "$sum", Value: itemCost("$items")}}},
		}}},
		{{Key: "$addFields", Value: bson.D{{Key: "profit", Value: bson.D{{Key: "$subtract", Value: bson.A{"$revenue", "$cogs"}}}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "profit", Value: dir}}}},
		{{Key: "$limit", Value: int64(limit)}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, dbErr(err)
	}
	defer cur.Close(ctx)

	var rows []struct {
		SKU     string `bson:"_id"`
		Name    string `bson:"name"`
		Units   int    `bson:"units"`
		Revenue int    `bson:"revenue"`
		Cogs    int    `bson:"cogs"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, dbErr(err)
	}
	out := make([]domain.ProductProfit, len(rows))
	for i, x := range rows {
		gp := x.Revenue - x.Cogs
		out[i] = domain.ProductProfit{SKU: x.SKU, Name: x.Name, UnitsSold: x.Units, Revenue: x.Revenue,
			CostOfGoods: x.Cogs, GrossProfit: gp, Margin: domain.Margin(gp, x.Revenue)}
	}
	return out, nil
}

func (r *repository) Orders(ctx context.Context, rg domain.Range, skip, limit int) ([]domain.OrderProfit, int64, error) {
	total, err := r.coll.CountDocuments(ctx, match(rg))
	if err != nil {
		return nil, 0, dbErr(err)
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match(rg)}},
		perOrder(),
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$skip", Value: int64(skip)}},
		{{Key: "$limit", Value: int64(limit)}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline, options.Aggregate())
	if err != nil {
		return nil, 0, dbErr(err)
	}
	defer cur.Close(ctx)

	var rows []struct {
		ID            string    `bson:"_id"`
		OrderNumber   string    `bson:"order_number"`
		CreatedAt     time.Time `bson:"created_at"`
		Status        string    `bson:"status"`
		PaymentMethod string    `bson:"payment_method"`
		PaymentStatus string    `bson:"payment_status"`
		Revenue       int       `bson:"o_revenue"`
		Cogs          int       `bson:"o_cogs"`
		Shipping      int       `bson:"o_shipping"`
		Courier       int       `bson:"o_courier"`
		Discount      int       `bson:"o_discount"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, dbErr(err)
	}
	out := make([]domain.OrderProfit, len(rows))
	for i, x := range rows {
		gp := x.Revenue - x.Discount - x.Cogs
		out[i] = domain.OrderProfit{
			OrderID: x.ID, OrderNumber: x.OrderNumber, CreatedAt: x.CreatedAt, Status: x.Status,
			PaymentMethod: x.PaymentMethod, PaymentStatus: x.PaymentStatus,
			ItemsRevenue: x.Revenue, Discount: x.Discount, CostOfGoods: x.Cogs, ShippingCharged: x.Shipping, CourierCost: x.Courier,
			GrossProfit: gp, NetProfit: gp + x.Shipping - x.Courier,
		}
	}
	return out, total, nil
}
