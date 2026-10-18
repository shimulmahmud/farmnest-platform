package infrastructure

import (
        "context"
        "fmt"
        "regexp"
        "time"

        "github.com/ecommerce/backend/internal/users/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type userRepository struct {
        coll *mongo.Collection
}

// NewUserRepository creates a MongoDB-backed UserRepository.
func NewUserRepository(db *mongo.Database) domain.UserRepository {
        coll := db.Collection("users")
        return &userRepository{coll: coll}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
        _, err := r.coll.InsertOne(ctx, user)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        return errors.Wrap("CONFLICT", "email already registered", 409, err)
                }
                return errors.Wrap("INTERNAL_ERROR", "failed to create user", 500, err)
        }
        return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
        var user domain.User
        filter := bson.M{"_id": id}
        if err := r.coll.FindOne(ctx, filter).Decode(&user); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find user", 500, err)
        }
        return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
        var user domain.User
        filter := bson.M{"email": email}
        if err := r.coll.FindOne(ctx, filter).Decode(&user); err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, errors.ErrNotFound
                }
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to find user by email", 500, err)
        }
        return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
        user.UpdatedAt = time.Now().UTC()
        filter := bson.M{"_id": user.ID}
        update := bson.M{"$set": user}
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to update user", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
        filter := bson.M{"email": email}
        count, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return false, errors.Wrap("INTERNAL_ERROR", "failed to check user existence", 500, err)
        }
        return count > 0, nil
}

// buildUserFilter constructs a MongoDB filter from a UserFilter.
// If filter is nil, returns an empty filter (match all).
func buildUserFilter(filter *domain.UserFilter) bson.M {
        if filter == nil {
                return bson.M{}
        }
        f := bson.M{}
        if filter.Role != "" {
                f["role"] = filter.Role
        }
        if filter.Status != "" {
                f["status"] = filter.Status
        }
        if filter.Search != "" {
                // Sanitize: strip regex special characters to prevent ReDoS.
                safe := regexp.QuoteMeta(filter.Search)
                f["$or"] = []bson.M{
                        {"email": bson.M{"$regex": safe, "$options": "i"}},
                        {"first_name": bson.M{"$regex": safe, "$options": "i"}},
                        {"last_name": bson.M{"$regex": safe, "$options": "i"}},
                }
        }
        return f
}

func (r *userRepository) List(ctx context.Context, filter *domain.UserFilter, p pagination.Params) ([]*domain.User, int64, error) {
        mf := buildUserFilter(filter)

        total, err := r.coll.CountDocuments(ctx, mf)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to count users", 500, err)
        }

        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(p.Skip)).
                SetLimit(int64(p.Limit))

        // Exclude password_hash from results.
        opts.SetProjection(bson.M{"password_hash": 0, "failed_attempts": 0, "locked_until": 0})

        cursor, err := r.coll.Find(ctx, mf, opts)
        if err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to list users", 500, err)
        }
        defer cursor.Close(ctx)

        var users []*domain.User
        if err := cursor.All(ctx, &users); err != nil {
                return nil, 0, errors.Wrap("INTERNAL_ERROR", "failed to decode users", 500, err)
        }
        return users, total, nil
}

func (r *userRepository) Count(ctx context.Context, filter *domain.UserFilter) (int64, error) {
        mf := buildUserFilter(filter)
        count, err := r.coll.CountDocuments(ctx, mf)
        if err != nil {
                return 0, errors.Wrap("INTERNAL_ERROR", "failed to count users", 500, err)
        }
        return count, nil
}

func (r *userRepository) UpdateRole(ctx context.Context, userID string, role domain.Role) error {
        now := time.Now().UTC()
        filter := bson.M{"_id": userID}
        update := bson.M{"$set": bson.M{"role": role, "updated_at": now}}
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to update user role", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

func (r *userRepository) UpdateStatus(ctx context.Context, userID string, status domain.AccountStatus) error {
        now := time.Now().UTC()
        filter := bson.M{"_id": userID}
        update := bson.M{
                "$set": bson.M{
                        "status":       status,
                        "updated_at":   now,
                        "failed_attempts": 0,
                        "locked_until":  nil,
                },
        }
        result, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to update user status", 500, err)
        }
        if result.MatchedCount == 0 {
                return errors.ErrNotFound
        }
        return nil
}

// EnsureIndexes creates required indexes on the users collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        coll := db.Collection("users")
        indexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "email", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_email_unique"),
                },
                {
                        Keys:    bson.D{{Key: "role", Value: 1}},
                        Options: options.Index().SetName("idx_role"),
                },
                {
                        Keys:    bson.D{{Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_status"),
                },
        }
        _, err := coll.Indexes().CreateMany(ctx, indexes)
        if err != nil {
                return fmt.Errorf("failed to create user indexes: %w", err)
        }
        return nil
}
