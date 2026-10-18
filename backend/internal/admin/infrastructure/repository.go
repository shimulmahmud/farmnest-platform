package infrastructure

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	adminDomain "github.com/ecommerce/backend/internal/admin/domain"
	"github.com/ecommerce/backend/pkg/pagination"
)

const adminAuditCollection = "audit_logs"

type auditRepository struct {
	col *mongo.Collection
}

// NewAuditRepository creates a MongoDB-backed audit log repository for admin queries.
// This reads from the same audit_logs collection that pkg/audit writes to.
func NewAuditRepository(db *mongo.Database) adminDomain.AuditRepository {
	return &auditRepository{col: db.Collection(adminAuditCollection)}
}

// Insert persists a new audit log entry.
func (r *auditRepository) Insert(ctx context.Context, log *adminDomain.AuditLog) error {
	_, err := r.col.InsertOne(ctx, log)
	return err
}

// List returns paginated audit logs with optional filters.
func (r *auditRepository) List(ctx context.Context, filter *adminDomain.AuditFilter, p pagination.Params) ([]*adminDomain.AuditLog, int64, error) {
	q := bson.M{}
	if filter != nil {
		if filter.ActorID != "" {
			q["actor_id"] = filter.ActorID
		}
		if filter.Action != "" {
			q["action"] = filter.Action
		}
		if filter.ResourceType != "" {
			q["resource_type"] = filter.ResourceType
		}
		if filter.ResourceID != "" {
			q["resource_id"] = filter.ResourceID
		}
		if filter.RequestID != "" {
			q["request_id"] = filter.RequestID
		}
	}

	total, err := r.col.CountDocuments(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(p.Skip)).
		SetLimit(int64(p.Limit))

	cursor, err := r.col.Find(ctx, q, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []*adminDomain.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// EnsureAuditIndexes creates the indexes for the audit_logs collection.
// This is called from main.go; pkg/audit.EnsureIndexes handles the shared indexes.
func EnsureAuditIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(adminAuditCollection)
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "actor_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_actor_created"),
		},
		{
			Keys:    bson.D{{Key: "action", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_action_created"),
		},
		{
			Keys:    bson.D{{Key: "resource_type", Value: 1}, {Key: "resource_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_resource_created"),
		},
		{
			Keys:    bson.D{{Key: "request_id", Value: 1}},
			Options: options.Index().SetName("idx_audit_request_id"),
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_created").
				SetExpireAfterSeconds(90 * 24 * 60 * 60),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}
