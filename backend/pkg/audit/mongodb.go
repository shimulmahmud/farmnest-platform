package audit

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/ecommerce/backend/pkg/logger"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/google/uuid"
)

// sensitivePatterns matches fields that must never appear in audit old_value/new_value/details.
// This prevents secrets, payment credentials, and PII from leaking into the audit trail.
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|secret|token|api_key|private_key|card_number|cvv|cvc)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`),
}

// sanitize strips sensitive information from audit entry text fields.
// It replaces matched patterns with "[REDACTED]".
func sanitize(s string) string {
	for _, re := range sensitivePatterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return strings.TrimSpace(s)
}

const collectionName = "audit_logs"

// --- Repository Implementation ---

type mongoRepository struct {
	col *mongo.Collection
}

// NewRepository creates a MongoDB-backed audit entry repository.
func NewRepository(db *mongo.Database) Repository {
	return &mongoRepository{col: db.Collection(collectionName)}
}

func (r *mongoRepository) Insert(ctx context.Context, entry *Entry) error {
	_, err := r.col.InsertOne(ctx, entry)
	return err
}

func (r *mongoRepository) List(ctx context.Context, filter *Filter, p pagination.Params) ([]*Entry, int64, error) {
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

	var entries []*Entry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// --- Async Auditor Implementation ---

// RequestMetadata carries per-request context for the auditor.
type RequestMetadata struct {
	IPAddress string
	RequestID string
}

// mongoAuditor is an async, failure-safe auditor backed by MongoDB.
type mongoAuditor struct {
	repo Repository
	log  *logger.Logger
	ch   chan *Entry
	done chan struct{}
	once sync.Once
}

// NewAuditor creates an async auditor that persists entries via the repository.
// Call Stop() to flush and shut down. If repo is nil, returns NopAuditor.
func NewAuditor(repo Repository, log *logger.Logger) Auditor {
	if repo == nil {
		return NopAuditor
	}
	a := &mongoAuditor{
		repo: repo,
		log:  log,
		ch:   make(chan *Entry, 256),
		done: make(chan struct{}),
	}
	go a.run()
	return a
}

// Record enqueues an audit entry for async persistence.
// This method never blocks the caller and never returns an error.
// Sensitive fields (passwords, tokens, payment credentials) are sanitized before storage.
func (a *mongoAuditor) Record(actorID string, action Action, resourceType, resourceID, oldValue, newValue, details string) {
	entry := &Entry{
		ID:           uuid.New().String(),
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValue:     sanitize(oldValue),
		NewValue:     sanitize(newValue),
		Details:      sanitize(details),
		CreatedAt:    time.Now().UTC().Unix(),
	}

	// Non-blocking send; drop if buffer is full to avoid blocking the caller.
	select {
	case a.ch <- entry:
	default:
		if a.log != nil {
			a.log.Warnw("audit buffer full, dropping entry", "action", action, "resource_id", resourceID)
		}
	}
}

// RecordWithMetadata enqueues an audit entry with request-scoped IP and request_id.
// Sensitive fields are sanitized before storage.
func (a *mongoAuditor) RecordWithMetadata(meta *RequestMetadata, actorID string, action Action, resourceType, resourceID, oldValue, newValue, details string) {
	entry := &Entry{
		ID:           uuid.New().String(),
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValue:     sanitize(oldValue),
		NewValue:     sanitize(newValue),
		Details:      sanitize(details),
		IPAddress:    meta.IPAddress,
		RequestID:    meta.RequestID,
		CreatedAt:    time.Now().UTC().Unix(),
	}

	select {
	case a.ch <- entry:
	default:
		if a.log != nil {
			a.log.Warnw("audit buffer full, dropping entry", "action", action, "resource_id", resourceID)
		}
	}
}

// run is the background goroutine that drains the channel and persists entries.
func (a *mongoAuditor) run() {
	for entry := range a.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := a.repo.Insert(ctx, entry); err != nil && a.log != nil {
			a.log.Errorw("audit insert failed", "error", err, "action", entry.Action, "resource_id", entry.ResourceID)
		}
		cancel()
	}
	close(a.done)
}

// Stop drains the buffer and shuts down the background goroutine.
func (a *mongoAuditor) Stop() {
	a.once.Do(func() {
		close(a.ch)
	})
	<-a.done
}

// --- Index Management ---

// EnsureIndexes creates the indexes for the audit_logs collection.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(collectionName)
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "actor_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_actor_created"),
		},
		{
			Keys: bson.D{{Key: "action", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_action_created"),
		},
		{
			Keys: bson.D{{Key: "resource_type", Value: 1}, {Key: "resource_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_resource_created"),
		},
		{
			Keys:    bson.D{{Key: "request_id", Value: 1}},
			Options: options.Index().SetName("idx_audit_request_id"),
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_audit_created").
				SetExpireAfterSeconds(90 * 24 * 60 * 60), // 90-day TTL
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}
