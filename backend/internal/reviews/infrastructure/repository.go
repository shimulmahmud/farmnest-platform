package infrastructure

import (
        "context"
        "fmt"

        "github.com/ecommerce/backend/internal/reviews/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type reviewRepository struct {
        coll *mongo.Collection
}

// NewReviewRepository creates a MongoDB-backed ReviewRepository.
func NewReviewRepository(db *mongo.Database) domain.ReviewRepository {
        return &reviewRepository{coll: db.Collection("reviews")}
}

func (r *reviewRepository) Create(ctx context.Context, review *domain.Review) error {
        _, err := r.coll.InsertOne(ctx, review)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        return errors.Wrap("CONFLICT", "you have already reviewed this product", 409, err)
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to create review", 500, err)
        }
        return nil
}

func (r *reviewRepository) FindByID(ctx context.Context, id string) (*domain.Review, error) {
        var review domain.Review
        if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&review); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find review", 500, err)
        }
        return &review, nil
}

func (r *reviewRepository) Update(ctx context.Context, review *domain.Review) error {
        filter := bson.M{"_id": review.ID}
        update := bson.M{"$set": review}
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to update review", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *reviewRepository) Delete(ctx context.Context, id string) error {
        result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to delete review", 500, err)
        }
        if result.DeletedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *reviewRepository) ListByProduct(ctx context.Context, productID string, p pagination.Params) (*domain.ReviewListResult, error) {
        return r.listByProductWithFilter(ctx, productID, bson.M{
                "product_id": productID,
                "status":    domain.ModerationApproved,
        }, p)
}

func (r *reviewRepository) ListByProductAdmin(ctx context.Context, productID string, p pagination.Params) (*domain.ReviewListResult, error) {
        return r.listByProductWithFilter(ctx, productID, bson.M{
                "product_id": productID,
        }, p)
}

func (r *reviewRepository) listByProductWithFilter(ctx context.Context, productID string, filter bson.M, p pagination.Params) (*domain.ReviewListResult, error) {
        total, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to count reviews", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        cursor, err := r.coll.Find(ctx, filter, opts)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to list reviews", 500, err)
        }
        defer cursor.Close(ctx)

        var reviews []*domain.Review
        if err := cursor.All(ctx, &reviews); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode reviews", 500, err)
        }

        return &domain.ReviewListResult{
                Reviews: reviews,
                Total:   total,
                Page:    p.Page,
                PerPage: p.PerPage,
        }, nil
}

func (r *reviewRepository) ExistsByProductAndCustomer(ctx context.Context, productID, customerID string) (bool, error) {
        count, err := r.coll.CountDocuments(ctx, bson.M{
                "product_id":  productID,
                "customer_id": customerID,
        })
        if err != nil {
                return false, errors.Wrap("INTERNAL_ERROR", "failed to check review existence", 500, err)
        }
        return count > 0, nil
}

func (r *reviewRepository) ListAll(ctx context.Context, status domain.ModerationStatus, p pagination.Params) (*domain.ReviewListResult, error) {
        filter := bson.M{}
        if status != "" {
                filter["status"] = status
        }

        total, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to count reviews", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        cursor, err := r.coll.Find(ctx, filter, opts)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to list reviews", 500, err)
        }
        defer cursor.Close(ctx)

        var reviews []*domain.Review
        if err := cursor.All(ctx, &reviews); err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to decode reviews", 500, err)
        }

        return &domain.ReviewListResult{
                Reviews: reviews,
                Total:   total,
                Page:    p.Page,
                PerPage: p.PerPage,
        }, nil
}

// OrderVerifiedPurchaseChecker checks if a customer has purchased a product
// by querying the orders collection for delivered orders containing the product.
type OrderVerifiedPurchaseChecker struct {
        ordersColl *mongo.Collection
}

// NewOrderVerifiedPurchaseChecker creates a VerifiedPurchaseChecker backed by the orders collection.
func NewOrderVerifiedPurchaseChecker(db *mongo.Database) domain.VerifiedPurchaseChecker {
        return &OrderVerifiedPurchaseChecker{ordersColl: db.Collection("orders")}
}

func (c *OrderVerifiedPurchaseChecker) IsVerifiedPurchase(ctx context.Context, customerID, productID string) (bool, error) {
        // A verified purchase means the customer has a delivered order
        // that contains an item with this product_id.
        filter := bson.M{
                "customer_id": customerID,
                "status":      "DELIVERED",
                "items.product_id": productID,
        }
        count, err := c.ordersColl.CountDocuments(ctx, filter)
        if err != nil {
                return false, errors.Wrap("INTERNAL_ERROR", "failed to check verified purchase", 500, err)
        }
        return count > 0, nil
}

// userNameResolver reads first/last name from the users collection.
type userNameResolver struct{ users *mongo.Collection }

// NewUserNameResolver returns a resolver giving short public names like "Rahim U.".
func NewUserNameResolver(db *mongo.Database) domain.AuthorNameResolver {
	return &userNameResolver{users: db.Collection("users")}
}

func (r *userNameResolver) DisplayName(ctx context.Context, userID string) string {
	var u struct {
		First string `bson:"first_name"`
		Last  string `bson:"last_name"`
	}
	if err := r.users.FindOne(ctx, bson.M{"_id": userID}).Decode(&u); err != nil || u.First == "" {
		return "Customer"
	}
	if u.Last == "" {
		return u.First
	}
	return u.First + " " + string([]rune(u.Last)[:1]) + "."
}

// EnsureIndexes creates required indexes on the reviews collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("reviews")
        indexes := []mongo.IndexModel{
                {
                        Keys: bson.D{
                                {Key: "product_id", Value: 1},
                                {Key: "customer_id", Value: 1},
                        },
                        Options: options.Index().
                                SetUnique(true).
                                SetName("idx_review_product_customer_unique"),
                },
                {
                        Keys:    bson.D{{Key: "product_id", Value: 1}, {Key: "status", Value: 1}, {Key: "created_at", Value: -1}},
                        Options: options.Index().SetName("idx_review_product_status_created"),
                },
                {
                        Keys:    bson.D{{Key: "customer_id", Value: 1}, {Key: "created_at", Value: -1}},
                        Options: options.Index().SetName("idx_review_customer_created"),
                },
                {
                        Keys:    bson.D{{Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_review_status"),
                },
                {
                        Keys:    bson.D{{Key: "rating", Value: 1}},
                        Options: options.Index().SetName("idx_review_rating"),
                },
        }
        _, err := coll.Indexes().CreateMany(ctx, indexes)
        if err != nil {
                return fmt.Errorf("failed to create review indexes: %w", err)
        }
        return nil
}
