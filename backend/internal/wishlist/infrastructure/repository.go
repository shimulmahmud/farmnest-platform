package infrastructure

import (
        "context"
        "fmt"
        "time"

        "github.com/ecommerce/backend/internal/wishlist/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type wishlistRepository struct {
        coll *mongo.Collection
}

// NewWishlistRepository creates a MongoDB-backed WishlistRepository.
func NewWishlistRepository(db *mongo.Database) domain.WishlistRepository {
        return &wishlistRepository{coll: db.Collection("wishlists")}
}

func (r *wishlistRepository) FindByCustomer(ctx context.Context, customerID string) (*domain.Wishlist, error) {
        var wishlist domain.Wishlist
        if err := r.coll.FindOne(ctx, bson.M{"customer_id": customerID}).Decode(&wishlist); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, nil
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find wishlist", 500, err)
        }
        return &wishlist, nil
}

func (r *wishlistRepository) Create(ctx context.Context, wishlist *domain.Wishlist) error {
        _, err := r.coll.InsertOne(ctx, wishlist)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to create wishlist", 500, err)
        }
        return nil
}

func (r *wishlistRepository) Update(ctx context.Context, wishlist *domain.Wishlist) error {
        filter := bson.M{"_id": wishlist.ID}
        update := bson.M{"$set": wishlist}
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to update wishlist", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

// AddItem atomically pushes a new item, preventing duplicates.
// Uses findOneAndUpdate to ensure atomicity.
func (r *wishlistRepository) AddItem(ctx context.Context, customerID string, item domain.WishlistItem) error {
        // Check for duplicate first
        exists, err := r.ExistsByCustomerAndProduct(ctx, customerID, item.ProductID, item.VariantID)
        if err != nil {
                return err
        }
        if exists {
                return domain.ErrDuplicateWishlistItem
        }

        // Upsert: create wishlist if it doesn't exist, then push the item.
        filter := bson.M{"customer_id": customerID}
        push := bson.M{
                "$push": bson.M{"items": item},
                "$set": bson.M{"updated_at": item.AddedAt},
        }
        opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
        var result bson.M
        err = r.coll.FindOneAndUpdate(ctx, filter, push, opts).Decode(&result)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        return domain.ErrDuplicateWishlistItem
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to add wishlist item", 500, err)
        }
        return nil
}

// RemoveItem atomically pulls an item from the wishlist.
func (r *wishlistRepository) RemoveItem(ctx context.Context, customerID, itemID string) error {
        filter := bson.M{
                "customer_id": customerID,
                "items._id":      itemID,
        }
        pull := bson.M{
                "$pull": bson.M{"items": bson.M{"_id": itemID}},
                "$set": bson.M{"updated_at": timeNow().UTC()},
        }

        result, err := r.coll.UpdateOne(ctx, filter, pull)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to remove wishlist item", 500, err)
        }
        if result.MatchedCount == 0 {
                return domain.ErrWishlistItemNotFound
        }
        return nil
}

func (r *wishlistRepository) ExistsByCustomerAndProduct(ctx context.Context, customerID, productID, variantID string) (bool, error) {
        // Build query: customer_id matches and items array contains matching product_id + variant_id
        filter := bson.M{
                "customer_id":          customerID,
                "items.product_id":     productID,
                "items.variant_id":     variantID,
        }
        count, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return false, errors.Wrap("INTERNAL_ERROR", "failed to check wishlist item existence", 500, err)
        }
        return count > 0, nil
}

// EnsureIndexes creates required indexes on the wishlists collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("wishlists")
        indexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "customer_id", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_wishlist_customer_unique"),
                },
                {
                        Keys: bson.D{
                                {Key: "customer_id", Value: 1},
                                {Key: "items.product_id", Value: 1},
                                {Key: "items.variant_id", Value: 1},
                        },
                        Options: options.Index().SetName("idx_wishlist_customer_product_variant"),
                },
                {
                        Keys:    bson.D{{Key: "updated_at", Value: -1}},
                        Options: options.Index().SetName("idx_wishlist_updated_at"),
                },
        }
        _, err := coll.Indexes().CreateMany(ctx, indexes)
        if err != nil {
                return fmt.Errorf("failed to create wishlist indexes: %w", err)
        }
        return nil
}

// timeNow is a variable for testability.
var timeNow = func() time.Time { return time.Now() }