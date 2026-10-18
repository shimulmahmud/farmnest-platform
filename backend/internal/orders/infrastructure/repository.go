package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/backend/internal/orders/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type orderRepository struct {
	coll *mongo.Collection
}

// NewOrderRepository creates a MongoDB-backed OrderRepository.
func NewOrderRepository(db *mongo.Database) domain.OrderRepository {
	return &orderRepository{coll: db.Collection("orders")}
}

func (r *orderRepository) Create(ctx context.Context, order *domain.Order) error {
	_, err := r.coll.InsertOne(ctx, order)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Wrap("CONFLICT", "order with this number already exists", 409, err)
		}
		return errors.Wrap("INTERNAL_ERROR", "failed to create order", 500, err)
	}
	return nil
}

func (r *orderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	var order domain.Order
	filter := bson.M{"_id": id}
	if err := r.coll.FindOne(ctx, filter).Decode(&order); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find order", 500, err)
	}
	return &order, nil
}

func (r *orderRepository) FindByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	var order domain.Order
	filter := bson.M{"order_number": orderNumber}
	if err := r.coll.FindOne(ctx, filter).Decode(&order); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find order by number", 500, err)
	}
	return &order, nil
}

func (r *orderRepository) List(ctx context.Context, filter *domain.OrderFilter, p pagination.Params) ([]*domain.Order, int64, error) {
	bsonFilter := bson.M{}
	if filter != nil {
		if filter.CustomerID != "" {
			bsonFilter["customer_id"] = filter.CustomerID
		}
		if filter.Status != "" {
			bsonFilter["status"] = filter.Status
		}
		if filter.From != nil || filter.To != nil {
			dateFilter := bson.M{}
			if filter.From != nil {
				dateFilter["$gte"] = filter.From
			}
			if filter.To != nil {
				dateFilter["$lte"] = filter.To
			}
			bsonFilter["created_at"] = dateFilter
		}
	}

	total, err := r.coll.CountDocuments(ctx, bsonFilter)
	if err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count orders", 500, err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(p.Skip)).
		SetLimit(int64(p.Limit))

	cursor, err := r.coll.Find(ctx, bsonFilter, opts)
	if err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list orders", 500, err)
	}
	defer cursor.Close(ctx)

	var orders []*domain.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode orders", 500, err)
	}
	return orders, total, nil
}

// UpdateStatus atomically transitions order status using findOneAndUpdate.
// The filter ensures the current status matches expectedStatus to prevent
// concurrent conflicting transitions.
func (r *orderRepository) UpdateStatus(ctx context.Context, orderID string, expectedStatus, newStatus domain.OrderStatus, reason string) (*domain.Order, error) {
	now := time.Now().UTC()

	historyEntry := domain.StatusHistoryEntry{
		Status:    newStatus,
		ChangedAt: now,
		Reason:    reason,
	}

	filter := bson.M{
		"_id":    orderID,
		"status": expectedStatus,
	}
	update := bson.M{
		"$set": bson.M{
			"status":     newStatus,
			"updated_at": now,
		},
		"$push": bson.M{
			"status_history": historyEntry,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated domain.Order
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("CONFLICT", "order not found or status has changed", 409)
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to update order status", 500, err)
	}
	return &updated, nil
}

// SetReferences persists payment/shipment references; empty values are skipped.
func (r *orderRepository) SetReferences(ctx context.Context, orderID, paymentRef, shipmentRef string) error {
	set := bson.M{}
	if paymentRef != "" {
		set["payment_reference"] = paymentRef
	}
	if shipmentRef != "" {
		set["shipment_reference"] = shipmentRef
	}
	if len(set) == 0 {
		return nil
	}
	set["updated_at"] = time.Now().UTC()
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": orderID}, bson.M{"$set": set})
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to save order references", 500, err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// SetStockAllocations stores where the order's stock was taken from.
func (r *orderRepository) SetStockAllocations(ctx context.Context, orderID string, allocs []domain.StockAllocation) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": orderID}, bson.M{"$set": bson.M{"stock_allocations": allocs, "updated_at": time.Now().UTC()}})
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to save stock allocations", 500, err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// SetPaymentStatus records whether the money has been received.
func (r *orderRepository) SetPaymentStatus(ctx context.Context, orderID string, status domain.PaymentStatus) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": orderID}, bson.M{"$set": bson.M{"payment_status": status, "updated_at": time.Now().UTC()}})
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to save payment status", 500, err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// MarkStockRestored flags that a cancelled order's stock was put back.
func (r *orderRepository) MarkStockRestored(ctx context.Context, orderID string) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": orderID}, bson.M{"$set": bson.M{"stock_restored": true, "updated_at": time.Now().UTC()}})
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to flag stock restored", 500, err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// SetShipment records carrier, tracking number and shipped time.
func (r *orderRepository) SetShipment(ctx context.Context, orderID, carrier, trackingNumber string, courierCost int) (*domain.Order, error) {
	now := time.Now().UTC()
	update := bson.M{"$set": bson.M{
		"carrier":            carrier,
		"tracking_number":    trackingNumber,
		"shipment_reference": trackingNumber,
		"shipped_at":         now,
		"courier_cost":       courierCost,
		"updated_at":         now,
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated domain.Order
	if err := r.coll.FindOneAndUpdate(ctx, bson.M{"_id": orderID}, update, opts).Decode(&updated); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to save shipment", 500, err)
	}
	return &updated, nil
}

// EnsureIndexes creates all required indexes for the orders collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("orders")
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "order_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_order_number_unique"),
		},
		{
			Keys:    bson.D{{Key: "customer_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_order_customer_created"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_order_status_created"),
		},
		{
			Keys:    bson.D{{Key: "customer_id", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_order_customer_status"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_order_created"),
		},
	}
	if _, err := coll.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("failed to create order indexes: %w", err)
	}
	return nil
}
