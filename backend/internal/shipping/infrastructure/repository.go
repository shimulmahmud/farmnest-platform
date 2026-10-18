package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/backend/internal/shipping/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type shipmentRepository struct {
	coll *mongo.Collection
}

// NewShipmentRepository creates a MongoDB-backed ShipmentRepository.
func NewShipmentRepository(db *mongo.Database) domain.ShipmentRepository {
	return &shipmentRepository{coll: db.Collection("shipments")}
}

func (r *shipmentRepository) Create(ctx context.Context, shipment *domain.Shipment) error {
	_, err := r.coll.InsertOne(ctx, shipment)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Wrap("CONFLICT", "shipment with this number already exists", 409, err)
		}
		return errors.Wrap("INTERNAL_ERROR", "failed to create shipment", 500, err)
	}
	return nil
}

func (r *shipmentRepository) FindByID(ctx context.Context, id string) (*domain.Shipment, error) {
	var shipment domain.Shipment
	filter := bson.M{"_id": id}
	if err := r.coll.FindOne(ctx, filter).Decode(&shipment); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find shipment", 500, err)
	}
	return &shipment, nil
}

func (r *shipmentRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Shipment, error) {
	var shipment domain.Shipment
	filter := bson.M{"order_id": orderID}
	if err := r.coll.FindOne(ctx, filter).Decode(&shipment); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to find shipment by order", 500, err)
	}
	return &shipment, nil
}

func (r *shipmentRepository) List(ctx context.Context, filter *domain.ShipmentFilter, p pagination.Params) ([]*domain.Shipment, int64, error) {
	bsonFilter := bson.M{}
	if filter != nil {
		if filter.OrderID != "" {
			bsonFilter["order_id"] = filter.OrderID
		}
		if filter.CustomerID != "" {
			bsonFilter["customer_id"] = filter.CustomerID
		}
		if filter.Status != "" {
			bsonFilter["status"] = filter.Status
		}
		if filter.Carrier != "" {
			bsonFilter["carrier"] = filter.Carrier
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
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count shipments", 500, err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(p.Skip)).
		SetLimit(int64(p.Limit))

	cursor, err := r.coll.Find(ctx, bsonFilter, opts)
	if err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list shipments", 500, err)
	}
	defer cursor.Close(ctx)

	var shipments []*domain.Shipment
	if err := cursor.All(ctx, &shipments); err != nil {
		return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode shipments", 500, err)
	}
	return shipments, total, nil
}

// UpdateStatus atomically transitions shipment status using findOneAndUpdate.
func (r *shipmentRepository) UpdateStatus(ctx context.Context, id string, fromStatus, toStatus domain.ShipmentStatus, reason string) (*domain.Shipment, error) {
	now := time.Now().UTC()

	historyEntry := domain.StatusHistoryEntry{
		Status:    toStatus,
		ChangedAt: now,
		Reason:    reason,
	}

	filter := bson.M{
		"_id":    id,
		"status": fromStatus,
	}
	update := bson.M{
		"$set": bson.M{
			"status":     toStatus,
			"updated_at": now,
		},
		"$push": bson.M{
			"status_history": historyEntry,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated domain.Shipment
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("CONFLICT", "shipment not found or status has changed", 409)
		}
		return nil, errors.Wrap("INTERNAL_ERROR", "failed to update shipment status", 500, err)
	}
	return &updated, nil
}

// UpdateTracking appends tracking events and sets the tracking number.
func (r *shipmentRepository) UpdateTracking(ctx context.Context, id string, trackingNumber string, events []domain.TrackingEvent) error {
	now := time.Now().UTC()

	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"tracking_number": trackingNumber,
			"updated_at":    now,
		},
		"$push": bson.M{
			"tracking_events": bson.M{"$each": events},
		},
	}

	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrap("INTERNAL_ERROR", "failed to update tracking", 500, err)
	}
	if result.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// EnsureIndexes creates all required indexes for the shipments collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("shipments")
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "shipment_number", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_shipment_number_unique"),
		},
		{
			Keys:    bson.D{{Key: "order_id", Value: 1}},
			Options: options.Index().SetName("idx_shipment_order_id").SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "customer_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_shipment_customer_created"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_shipment_status_created"),
		},
		{
			Keys:    bson.D{{Key: "carrier", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_shipment_carrier_created"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_shipment_created"),
		},
		{
			Keys:    bson.D{{Key: "tracking_number", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("idx_shipment_tracking_number"),
		},
	}
	if _, err := coll.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("failed to create shipment indexes: %w", err)
	}
	return nil
}
