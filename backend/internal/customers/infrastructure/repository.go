package infrastructure

import (
        "context"
        "fmt"
        "time"

        "github.com/ecommerce/backend/internal/customers/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type customerRepository struct {
        coll *mongo.Collection
}

// NewCustomerRepository creates a MongoDB-backed CustomerRepository.
func NewCustomerRepository(db *mongo.Database) domain.CustomerRepository {
        return &customerRepository{coll: db.Collection("customers")}
}

func (r *customerRepository) Create(ctx context.Context, customer *domain.Customer) error {
        _, err := r.coll.InsertOne(ctx, customer)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to create customer", 500, err)
        }
        return nil
}

func (r *customerRepository) FindByUserID(ctx context.Context, userID string) (*domain.Customer, error) {
        var customer domain.Customer
        filter := bson.M{"user_id": userID}
        if err := r.coll.FindOne(ctx, filter).Decode(&customer); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find customer", 500, err)
        }
        return &customer, nil
}

func (r *customerRepository) Update(ctx context.Context, customer *domain.Customer) error {
        customer.UpdatedAt = time.Now().UTC()
        filter := bson.M{"_id": customer.ID}
        update := bson.M{"$set": customer}
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to update customer", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *customerRepository) FindByID(ctx context.Context, id string) (*domain.Customer, error) {
        var customer domain.Customer
        filter := bson.M{"_id": id}
        if err := r.coll.FindOne(ctx, filter).Decode(&customer); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find customer", 500, err)
        }
        return &customer, nil
}

func (r *customerRepository) List(ctx context.Context, p pagination.Params) ([]*domain.Customer, int64, error) {
        total, err := r.coll.CountDocuments(ctx, bson.M{})
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count customers", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        cursor, err := r.coll.Find(ctx, bson.M{}, opts)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list customers", 500, err)
        }
        defer cursor.Close(ctx)

        var customers []*domain.Customer
        if err := cursor.All(ctx, &customers); err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode customers", 500, err)
        }
        return customers, total, nil
}

func (r *customerRepository) Count(ctx context.Context) (int64, error) {
        count, err := r.coll.CountDocuments(ctx, bson.M{})
        if err != nil {
                return 0, errors.Wrap("INTERNAL_ERROR", "failed to count customers", 500, err)
        }
        return count, nil
}

// EnsureIndexes creates required indexes on the customers collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("customers")
        indexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "user_id", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_customer_user_id_unique"),
                },
        }
        _, err := coll.Indexes().CreateMany(ctx, indexes)
        if err != nil {
                return fmt.Errorf("failed to create customer indexes: %w", err)
        }
        return nil
}
