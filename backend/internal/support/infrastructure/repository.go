package infrastructure

import (
	"context"
	"regexp"
	"time"

	"github.com/ecommerce/backend/internal/support/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "support_tickets"

type repository struct{ coll *mongo.Collection }

// NewRepository creates a MongoDB-backed ticket repository.
func NewRepository(db *mongo.Database) domain.Repository {
	return &repository{coll: db.Collection(collectionName)}
}

func dbErr(msg string, err error) error { return errors.Wrap("INTERNAL_ERROR", msg, 500, err) }

func (r *repository) Create(ctx context.Context, t *domain.Ticket) error {
	if _, err := r.coll.InsertOne(ctx, t); err != nil {
		return dbErr("failed to create support ticket", err)
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	var t domain.Ticket
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&t); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrNotFound
		}
		return nil, dbErr("failed to load support ticket", err)
	}
	return &t, nil
}

// list returns tickets with only their LAST message (enough for a preview line).
func (r *repository) list(ctx context.Context, filter bson.M, p pagination.Params) ([]*domain.Ticket, int64, error) {
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, dbErr("failed to count tickets", err)
	}
	cur, err := r.coll.Find(ctx, filter, options.Find().
		SetProjection(bson.M{"messages": bson.M{"$slice": -1}}).
		SetSort(bson.D{{Key: "last_message_at", Value: -1}}).
		SetSkip(int64(p.Skip)).SetLimit(int64(p.Limit)))
	if err != nil {
		return nil, 0, dbErr("failed to list tickets", err)
	}
	defer cur.Close(ctx)
	var out []*domain.Ticket
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, dbErr("failed to decode tickets", err)
	}
	return out, total, nil
}

func (r *repository) ListByCustomer(ctx context.Context, customerID, status string, p pagination.Params) ([]*domain.Ticket, int64, error) {
	f := bson.M{"customer_id": customerID}
	if status != "" {
		f["status"] = status
	}
	return r.list(ctx, f, p)
}

func (r *repository) ListAll(ctx context.Context, status, search string, p pagination.Params) ([]*domain.Ticket, int64, error) {
	f := bson.M{}
	if status != "" {
		f["status"] = status
	}
	if search != "" {
		re := bson.Regex{Pattern: regexp.QuoteMeta(search), Options: "i"}
		f["$or"] = bson.A{
			bson.M{"number": re}, bson.M{"subject": re}, bson.M{"customer_email": re},
			bson.M{"customer_name": re}, bson.M{"order_number": re},
		}
	}
	return r.list(ctx, f, p)
}

func (r *repository) AddMessage(ctx context.Context, id string, m domain.Message, status domain.Status, customerUnread, staffUnread bool) error {
	now := time.Now().UTC()
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$push": bson.M{"messages": m},
		"$set": bson.M{
			"status": status, "customer_unread": customerUnread, "staff_unread": staffUnread,
			"last_message_at": now, "updated_at": now,
		},
	})
	if err != nil {
		return dbErr("failed to add message", err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *repository) SetStatus(ctx context.Context, id string, status domain.Status) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": status, "updated_at": time.Now().UTC()}})
	if err != nil {
		return dbErr("failed to update ticket status", err)
	}
	if res.MatchedCount == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *repository) MarkRead(ctx context.Context, id string, byStaff bool) error {
	field := "customer_unread"
	if byStaff {
		field = "staff_unread"
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{field: false}})
	if err != nil {
		return dbErr("failed to mark ticket read", err)
	}
	return nil
}

func (r *repository) CountOpenByCustomer(ctx context.Context, customerID string) (int64, error) {
	n, err := r.coll.CountDocuments(ctx, bson.M{"customer_id": customerID, "status": bson.M{"$ne": domain.StatusClosed}})
	if err != nil {
		return 0, dbErr("failed to count open tickets", err)
	}
	return n, nil
}

func (r *repository) CustomerUnreadCount(ctx context.Context, customerID string) (int64, error) {
	n, err := r.coll.CountDocuments(ctx, bson.M{"customer_id": customerID, "customer_unread": true})
	if err != nil {
		return 0, dbErr("failed to count unread tickets", err)
	}
	return n, nil
}

func (r *repository) Summary(ctx context.Context) (*domain.Summary, error) {
	count := func(f bson.M) (int64, error) {
		n, err := r.coll.CountDocuments(ctx, f)
		if err != nil {
			return 0, dbErr("failed to count tickets", err)
		}
		return n, nil
	}
	var s domain.Summary
	var err error
	if s.Open, err = count(bson.M{"status": domain.StatusOpen}); err != nil {
		return nil, err
	}
	if s.Answered, err = count(bson.M{"status": domain.StatusAnswered}); err != nil {
		return nil, err
	}
	if s.Closed, err = count(bson.M{"status": domain.StatusClosed}); err != nil {
		return nil, err
	}
	if s.Unread, err = count(bson.M{"staff_unread": true, "status": bson.M{"$ne": domain.StatusClosed}}); err != nil {
		return nil, err
	}
	return &s, nil
}

// EnsureIndexes creates the indexes used by the lists.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection(collectionName).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "number", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_support_number")},
		{Keys: bson.D{{Key: "customer_id", Value: 1}, {Key: "last_message_at", Value: -1}}, Options: options.Index().SetName("idx_support_customer")},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "last_message_at", Value: -1}}, Options: options.Index().SetName("idx_support_status")},
	})
	return err
}
