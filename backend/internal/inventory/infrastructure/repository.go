package infrastructure

import (
        "context"
        "fmt"
        "strings"
        "time"

        "github.com/ecommerce/backend/internal/inventory/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

// --- Stock Repository ---

type stockRepository struct {
        coll *mongo.Collection
}

// NewStockRepository creates a MongoDB-backed StockRepository.
func NewStockRepository(db *mongo.Database) domain.StockRepository {
        return &stockRepository{coll: db.Collection("inventory")}
}

func (r *stockRepository) CreateStock(ctx context.Context, stock *domain.StockItem) error {
        _, err := r.coll.InsertOne(ctx, stock)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        return errors.Wrap("CONFLICT", "stock item already exists for this SKU and warehouse", 409, err)
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to create stock item", 500, err)
        }
        return nil
}

func (r *stockRepository) GetStock(ctx context.Context, sku, warehouseID string) (*domain.StockItem, error) {
        var stock domain.StockItem
        filter := bson.M{"sku": sku, "warehouse_id": warehouseID}
        if err := r.coll.FindOne(ctx, filter).Decode(&stock); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to get stock", 500, err)
        }
        return &stock, nil
}

// AdjustStock uses findOneAndUpdate with atomic condition to prevent negative stock.
func (r *stockRepository) AdjustStock(ctx context.Context, sku, warehouseID string, quantity int, reason string) (*domain.StockItem, error) {
        if quantity == 0 {
                return r.GetStock(ctx, sku, warehouseID)
        }

        var updated domain.StockItem
        filter := bson.M{"sku": sku, "warehouse_id": warehouseID}
        var update bson.M

        if quantity > 0 {
                update = bson.M{
                        "$inc":       bson.M{"total_stock": quantity},
                        "$set":       bson.M{"updated_at": time.Now().UTC()},
                }
        } else {
                // Negative: only allow if total_stock + quantity >= 0
                update = bson.M{
                        "$inc":  bson.M{"total_stock": quantity},
                        "$set":  bson.M{"updated_at": time.Now().UTC()},
                }
                // Prevent going negative
                filter["total_stock"] = bson.M{"$gte": -quantity}
        }

        opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
        err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.New("BAD_REQUEST", "adjustment would result in negative stock", 400)
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to adjust stock", 500, err)
        }
        return &updated, nil
}

// ReserveStock atomically increments reserved_stock only if available >= quantity.
func (r *stockRepository) ReserveStock(ctx context.Context, sku, warehouseID string, quantity int) (*domain.StockItem, error) {
        filter := bson.M{
                "sku":          sku,
                "warehouse_id": warehouseID,
                // Ensure available (total - reserved) >= quantity
                "$expr": bson.M{
                        "$gte": bson.A{
                                bson.M{"$subtract": bson.A{"$total_stock", "$reserved_stock"}},
                                quantity,
                        },
                },
        }

        update := bson.M{
                "$inc": bson.M{"reserved_stock": quantity},
                "$set": bson.M{"updated_at": time.Now().UTC()},
        }

        opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

        var updated domain.StockItem
        err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.New("INSUFFICIENT_STOCK", "not enough available stock to reserve", 409)
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to reserve stock", 500, err)
        }
        return &updated, nil
}

// ReleaseStock atomically decrements reserved_stock.
func (r *stockRepository) ReleaseStock(ctx context.Context, sku, warehouseID string, quantity int) (*domain.StockItem, error) {
        filter := bson.M{
                "sku":          sku,
                "warehouse_id": warehouseID,
                "reserved_stock": bson.M{"$gte": quantity},
        }

        update := bson.M{
                "$inc": bson.M{"reserved_stock": -quantity},
                "$set": bson.M{"updated_at": time.Now().UTC()},
        }

        opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

        var updated domain.StockItem
        err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.New("BAD_REQUEST", "cannot release more than reserved", 400)
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to release stock", 500, err)
        }
        return &updated, nil
}

// ConsumeStock atomically decrements both total_stock and reserved_stock.
func (r *stockRepository) ConsumeStock(ctx context.Context, sku, warehouseID string, quantity int) (*domain.StockItem, error) {
        filter := bson.M{
                "sku":            sku,
                "warehouse_id":    warehouseID,
                "total_stock":     bson.M{"$gte": quantity},
                "reserved_stock":  bson.M{"$gte": quantity},
        }

        update := bson.M{
                "$inc": bson.M{
                        "total_stock":    -quantity,
                        "reserved_stock": -quantity,
                },
                "$set": bson.M{"updated_at": time.Now().UTC()},
        }

        opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

        var updated domain.StockItem
        err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.New("BAD_REQUEST", "insufficient stock to consume", 400)
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to consume stock", 500, err)
        }
        return &updated, nil
}

func (r *stockRepository) ListLowStock(ctx context.Context) ([]*domain.StockItem, error) {
        filter := bson.M{
                "low_stock_threshold": bson.M{"$gt": 0},
                "$expr": bson.M{
                        "$lte": bson.A{"$total_stock", "$low_stock_threshold"},
                },
        }

        cursor, err := r.coll.Find(ctx, filter)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to list low stock items", 500, err)
        }
        defer cursor.Close(ctx)

        var items []*domain.StockItem
        if err := cursor.All(ctx, &items); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode low stock items", 500, err)
        }
        return items, nil
}

// --- Reservation Repository ---

type reservationRepository struct {
        coll *mongo.Collection
}

// NewReservationRepository creates a MongoDB-backed ReservationRepository.
func NewReservationRepository(db *mongo.Database) domain.ReservationRepository {
        return &reservationRepository{coll: db.Collection("inventory_reservations")}
}

func (r *reservationRepository) CreateReservation(ctx context.Context, res *domain.Reservation) error {
        _, err := r.coll.InsertOne(ctx, res)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to create reservation", 500, err)
        }
        return nil
}

func (r *reservationRepository) GetReservation(ctx context.Context, id string) (*domain.Reservation, error) {
        var res domain.Reservation
        filter := bson.M{"_id": id}
        if err := r.coll.FindOne(ctx, filter).Decode(&res); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to get reservation", 500, err)
        }
        return &res, nil
}

func (r *reservationRepository) GetActiveReservation(ctx context.Context, sku, reference string) (*domain.Reservation, error) {
        var res domain.Reservation
        filter := bson.M{
                "sku":       sku,
                "reference":  reference,
                "status":    domain.ReservationStatusActive,
                "expires_at": bson.M{"$gt": time.Now().UTC()},
        }
        if err := r.coll.FindOne(ctx, filter).Decode(&res); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, nil
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to get active reservation", 500, err)
        }
        return &res, nil
}

func (r *reservationRepository) FindActiveReservationsByReference(ctx context.Context, reference string) ([]*domain.Reservation, error) {
        filter := bson.M{
                "reference":  reference,
                "status":    domain.ReservationStatusActive,
                "expires_at": bson.M{"$gt": time.Now().UTC()},
        }
        cursor, err := r.coll.Find(ctx, filter)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find active reservations by reference", 500, err)
        }
        defer cursor.Close(ctx)

        var reservations []*domain.Reservation
        if err := cursor.All(ctx, &reservations); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode reservations", 500, err)
        }
        return reservations, nil
}

func (r *reservationRepository) ReleaseReservation(ctx context.Context, id string) error {
        now := time.Now().UTC()
        result, err := r.coll.UpdateOne(ctx,
                bson.M{"_id": id, "status": domain.ReservationStatusActive},
                bson.M{"$set": bson.M{"status": domain.ReservationStatusReleased, "updated_at": now}},
        )
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to release reservation", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.New("BAD_REQUEST", "reservation not found or not active", 400)
        }
        return nil
}

func (r *reservationRepository) ConsumeReservation(ctx context.Context, id string) error {
        now := time.Now().UTC()
        result, err := r.coll.UpdateOne(ctx,
                bson.M{"_id": id, "status": domain.ReservationStatusActive},
                bson.M{"$set": bson.M{"status": domain.ReservationStatusConsumed, "updated_at": now}},
        )
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to consume reservation", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.New("BAD_REQUEST", "reservation not found or not active", 400)
        }
        return nil
}

func (r *reservationRepository) ExpireReservations(ctx context.Context, sku string) ([]*domain.Reservation, error) {
        now := time.Now().UTC()
        filter := bson.M{
                "sku":       sku,
                "status":    domain.ReservationStatusActive,
                "expires_at": bson.M{"$lte": now},
        }
        update := bson.M{"$set": bson.M{"status": domain.ReservationStatusExpired, "updated_at": now}}

        // Find matching docs first, then update
        cursor, err := r.coll.Find(ctx, filter)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find expired reservations", 500, err)
        }
        var expired []*domain.Reservation
        if err := cursor.All(ctx, &expired); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode expired reservations", 500, err)
        }
        cursor.Close(ctx)

        if len(expired) > 0 {
                _, err = r.coll.UpdateMany(ctx, filter, update)
                if err != nil {
                        return nil, errors.Wrap("INTERNAL_ERROR", "failed to expire reservations", 500, err)
                }
        }
        return expired, nil
}

func (r *reservationRepository) ListReservations(ctx context.Context, sku, status string, p pagination.Params) ([]*domain.Reservation, int64, error) {
        filter := bson.M{}
        if sku != "" {
                filter["sku"] = sku
        }
        if status != "" {
                filter["status"] = status
        }

        total, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count reservations", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        cursor, err := r.coll.Find(ctx, filter, opts)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list reservations", 500, err)
        }
        defer cursor.Close(ctx)

        var reservations []*domain.Reservation
        if err := cursor.All(ctx, &reservations); err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode reservations", 500, err)
        }
        return reservations, total, nil
}

// --- Movement Repository ---

type movementRepository struct {
        coll *mongo.Collection
}

// NewMovementRepository creates a MongoDB-backed MovementRepository.
func NewMovementRepository(db *mongo.Database) domain.MovementRepository {
        return &movementRepository{coll: db.Collection("inventory_movements")}
}

func (r *movementRepository) CreateMovement(ctx context.Context, m *domain.Movement) error {
        _, err := r.coll.InsertOne(ctx, m)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to create movement record", 500, err)
        }
        return nil
}

func (r *movementRepository) ListMovements(ctx context.Context, sku string, p pagination.Params) ([]*domain.Movement, int64, error) {
        filter := bson.M{}
        if sku != "" {
                filter["sku"] = sku
        }

        total, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count movements", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        cursor, err := r.coll.Find(ctx, filter, opts)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list movements", 500, err)
        }
        defer cursor.Close(ctx)

        var movements []*domain.Movement
        if err := cursor.All(ctx, &movements); err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode movements", 500, err)
        }
        return movements, total, nil
}

// --- Warehouse Repository ---

type warehouseRepository struct {
        coll *mongo.Collection
}

// NewWarehouseRepository creates a MongoDB-backed WarehouseRepository.
func NewWarehouseRepository(db *mongo.Database) domain.WarehouseRepository {
        return &warehouseRepository{coll: db.Collection("warehouses")}
}

func (r *warehouseRepository) Create(ctx context.Context, w *domain.Warehouse) error {
        _, err := r.coll.InsertOne(ctx, w)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        if strings.Contains(err.Error(), "code") {
                                return errors.Wrap("CONFLICT", "warehouse code already exists", 409, err)
                        }
                        return errors.Wrap("CONFLICT", "warehouse already exists", 409, err)
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to create warehouse", 500, err)
        }
        return nil
}

func (r *warehouseRepository) FindByID(ctx context.Context, id string) (*domain.Warehouse, error) {
        var w domain.Warehouse
        filter := bson.M{"_id": id}
        if err := r.coll.FindOne(ctx, filter).Decode(&w); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find warehouse", 500, err)
        }
        return &w, nil
}

func (r *warehouseRepository) FindByCode(ctx context.Context, code string) (*domain.Warehouse, error) {
        var w domain.Warehouse
        filter := bson.M{"code": code}
        if err := r.coll.FindOne(ctx, filter).Decode(&w); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find warehouse by code", 500, err)
        }
        return &w, nil
}

func (r *warehouseRepository) Update(ctx context.Context, w *domain.Warehouse) error {
        w.UpdatedAt = time.Now().UTC()
        filter := bson.M{"_id": w.ID}
        update := bson.M{"$set": w}
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to update warehouse", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *warehouseRepository) List(ctx context.Context, p pagination.Params) ([]*domain.Warehouse, int64, error) {
        total, err := r.coll.CountDocuments(ctx, bson.M{})
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count warehouses", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "name", Value: 1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        cursor, err := r.coll.Find(ctx, bson.M{}, opts)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list warehouses", 500, err)
        }
        defer cursor.Close(ctx)

        var warehouses []*domain.Warehouse
        if err := cursor.All(ctx, &warehouses); err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode warehouses", 500, err)
        }
        return warehouses, total, nil
}

// EnsureIndexes creates all required indexes for inventory collections.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        // inventory collection
        invColl := db.Collection("inventory")
        invIndexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "sku", Value: 1}, {Key: "warehouse_id", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_inventory_sku_warehouse_unique"),
                },
                {
                        Keys:    bson.D{{Key: "sku", Value: 1}},
                        Options: options.Index().SetName("idx_inventory_sku"),
                },
                {
                        Keys:    bson.D{{Key: "warehouse_id", Value: 1}},
                        Options: options.Index().SetName("idx_inventory_warehouse"),
                },
        }
        if _, err := invColl.Indexes().CreateMany(ctx, invIndexes); err != nil {
                return fmt.Errorf("failed to create inventory indexes: %w", err)
        }

        // inventory_reservations collection
        resColl := db.Collection("inventory_reservations")
        resIndexes := []mongo.IndexModel{
                {
                        Keys: bson.D{{Key: "sku", Value: 1}, {Key: "reference", Value: 1}, {Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_reservation_sku_ref_status"),
                },
                {
                        Keys:    bson.D{{Key: "sku", Value: 1}, {Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_reservation_sku_status"),
                },
                {
                        Keys:    bson.D{{Key: "expires_at", Value: 1}, {Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_reservation_expires_status"),
                },
        }
        if _, err := resColl.Indexes().CreateMany(ctx, resIndexes); err != nil {
                return fmt.Errorf("failed to create reservation indexes: %w", err)
        }

        // inventory_movements collection
        movColl := db.Collection("inventory_movements")
        movIndexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "sku", Value: 1}, {Key: "created_at", Value: -1}},
                        Options: options.Index().SetName("idx_movement_sku_created"),
                },
        }
        if _, err := movColl.Indexes().CreateMany(ctx, movIndexes); err != nil {
                return fmt.Errorf("failed to create movement indexes: %w", err)
        }

        // warehouses collection
        whColl := db.Collection("warehouses")
        whIndexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "code", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_warehouse_code_unique"),
                },
        }
        if _, err := whColl.Indexes().CreateMany(ctx, whIndexes); err != nil {
                return fmt.Errorf("failed to create warehouse indexes: %w", err)
        }

        return nil
}
