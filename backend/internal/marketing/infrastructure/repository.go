package infrastructure

import (
        "context"
        "fmt"
        "time"

        mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"
        "go.mongodb.org/mongo-driver/v2/mongo/options"
)

// --- Analytics Event Repository ---

type analyticsEventRepository struct {
        coll *mongo.Collection
}

// NewAnalyticsEventRepository creates a MongoDB-backed AnalyticsEventRepository.
func NewAnalyticsEventRepository(db *mongo.Database) mktDomain.AnalyticsEventRepository {
        return &analyticsEventRepository{coll: db.Collection("analytics_events")}
}

func (r *analyticsEventRepository) Insert(ctx context.Context, event *mktDomain.AnalyticsEvent) error {
        _, err := r.coll.InsertOne(ctx, event)
        if err != nil {
                if mongo.IsDuplicateKeyError(err) {
                        return nil // idempotent
                }
                return fmt.Errorf("failed to insert analytics event: %w", err)
        }
        return nil
}

func (r *analyticsEventRepository) ExistsByEventID(ctx context.Context, eventID string) (bool, error) {
        filter := bson.M{"event_id": eventID}
        count, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return false, fmt.Errorf("failed to check analytics event existence: %w", err)
        }
        return count > 0, nil
}

func (r *analyticsEventRepository) CountByCustomer(ctx context.Context, customerID string, eventType mktDomain.AnalyticsEventType, since time.Time) (int64, error) {
        filter := bson.M{
                "customer_id": customerID,
                "created_at": bson.M{"$gte": since},
        }
        if eventType != "" {
                filter["event_type"] = eventType
        }
        count, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return 0, fmt.Errorf("failed to count analytics events: %w", err)
        }
        return count, nil
}

func (r *analyticsEventRepository) ListByCustomer(ctx context.Context, customerID string, limit int) ([]*mktDomain.AnalyticsEvent, error) {
        filter := bson.M{"customer_id": customerID}
        opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit))
        cursor, err := r.coll.Find(ctx, filter, opts)
        if err != nil {
                return nil, fmt.Errorf("failed to list analytics events: %w", err)
        }
        defer cursor.Close(ctx)

        var events []*mktDomain.AnalyticsEvent
        if err := cursor.All(ctx, &events); err != nil {
                return nil, fmt.Errorf("failed to decode analytics events: %w", err)
        }
        return events, nil
}

// --- Campaign Repository ---

type campaignRepository struct {
        coll *mongo.Collection
}

// NewCampaignRepository creates a MongoDB-backed CampaignRepository.
func NewCampaignRepository(db *mongo.Database) mktDomain.CampaignRepository {
        return &campaignRepository{coll: db.Collection("campaigns")}
}

func (r *campaignRepository) Create(ctx context.Context, campaign *mktDomain.Campaign) error {
        _, err := r.coll.InsertOne(ctx, campaign)
        if err != nil {
                return fmt.Errorf("failed to create campaign: %w", err)
        }
        return nil
}

func (r *campaignRepository) FindByID(ctx context.Context, id string) (*mktDomain.Campaign, error) {
        filter := bson.M{"_id": id}
        var campaign mktDomain.Campaign
        err := r.coll.FindOne(ctx, filter).Decode(&campaign)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, mktDomain.ErrCampaignNotFound
                }
                return nil, fmt.Errorf("failed to find campaign: %w", err)
        }
        return &campaign, nil
}

func (r *campaignRepository) Update(ctx context.Context, campaign *mktDomain.Campaign) error {
        filter := bson.M{"_id": campaign.ID}
        update := bson.M{"$set": bson.M{
                "name":        campaign.Name,
                "type":        campaign.Type,
                "status":      campaign.Status,
                "description": campaign.Description,
                "rules":       campaign.Rules,
                "metrics":     campaign.Metrics,
                "start_at":    campaign.StartAt,
                "end_at":      campaign.EndAt,
                "updated_at":  campaign.UpdatedAt,
        }}
        _, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return fmt.Errorf("failed to update campaign: %w", err)
        }
        return nil
}

func (r *campaignRepository) FindActiveByEventType(ctx context.Context, eventType string) ([]*mktDomain.Campaign, error) {
        filter := bson.M{
                "status":       mktDomain.CampaignActive,
                "rules.event_types": eventType,
        }
        cursor, err := r.coll.Find(ctx, filter)
        if err != nil {
                return nil, fmt.Errorf("failed to find active campaigns by event type: %w", err)
        }
        defer cursor.Close(ctx)

        var campaigns []*mktDomain.Campaign
        if err := cursor.All(ctx, &campaigns); err != nil {
                return nil, fmt.Errorf("failed to decode campaigns: %w", err)
        }
        return campaigns, nil
}

func (r *campaignRepository) List(ctx context.Context, status string, offset, limit int) ([]*mktDomain.Campaign, int64, error) {
        filter := bson.M{}
        if status != "" {
                filter["status"] = status
        }
        total, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return nil, 0, fmt.Errorf("failed to count campaigns: %w", err)
        }
        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(offset)).
                SetLimit(int64(limit))
        cursor, err := r.coll.Find(ctx, filter, opts)
        if err != nil {
                return nil, 0, fmt.Errorf("failed to list campaigns: %w", err)
        }
        defer cursor.Close(ctx)

        var campaigns []*mktDomain.Campaign
        if err := cursor.All(ctx, &campaigns); err != nil {
                return nil, 0, fmt.Errorf("failed to decode campaigns: %w", err)
        }
        return campaigns, total, nil
}

func (r *campaignRepository) IncrementMetrics(ctx context.Context, id string, field string, value int64) error {
        filter := bson.M{"_id": id}
        update := bson.M{
                "$inc":       bson.M{"metrics." + field: value},
                "$set":       bson.M{"updated_at": time.Now().UTC()},
        }
        _, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return fmt.Errorf("failed to increment campaign metrics: %w", err)
        }
        return nil
}

// --- Segment Repository ---

type segmentRepository struct {
        coll *mongo.Collection
}

// NewSegmentRepository creates a MongoDB-backed SegmentRepository.
func NewSegmentRepository(db *mongo.Database) mktDomain.SegmentRepository {
        return &segmentRepository{coll: db.Collection("segments")}
}

func (r *segmentRepository) Create(ctx context.Context, segment *mktDomain.Segment) error {
        _, err := r.coll.InsertOne(ctx, segment)
        if err != nil {
                return fmt.Errorf("failed to create segment: %w", err)
        }
        return nil
}

func (r *segmentRepository) FindByID(ctx context.Context, id string) (*mktDomain.Segment, error) {
        filter := bson.M{"_id": id}
        var segment mktDomain.Segment
        err := r.coll.FindOne(ctx, filter).Decode(&segment)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, mktDomain.ErrSegmentNotFound
                }
                return nil, fmt.Errorf("failed to find segment: %w", err)
        }
        return &segment, nil
}

func (r *segmentRepository) Update(ctx context.Context, segment *mktDomain.Segment) error {
        filter := bson.M{"_id": segment.ID}
        update := bson.M{"$set": bson.M{
                "name":           segment.Name,
                "type":           segment.Type,
                "description":    segment.Description,
                "rules":          segment.Rules,
                "customer_ids":   segment.CustomerIDs,
                "customer_count": segment.CustomerCount,
                "updated_at":     segment.UpdatedAt,
        }}
        _, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return fmt.Errorf("failed to update segment: %w", err)
        }
        return nil
}

func (r *segmentRepository) Delete(ctx context.Context, id string) error {
        filter := bson.M{"_id": id}
        _, err := r.coll.DeleteOne(ctx, filter)
        if err != nil {
                return fmt.Errorf("failed to delete segment: %w", err)
        }
        return nil
}

func (r *segmentRepository) List(ctx context.Context, offset, limit int) ([]*mktDomain.Segment, int64, error) {
        total, err := r.coll.CountDocuments(ctx, bson.M{})
        if err != nil {
                return nil, 0, fmt.Errorf("failed to count segments: %w", err)
        }
        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(offset)).
                SetLimit(int64(limit))
        cursor, err := r.coll.Find(ctx, bson.M{}, opts)
        if err != nil {
                return nil, 0, fmt.Errorf("failed to list segments: %w", err)
        }
        defer cursor.Close(ctx)

        var segments []*mktDomain.Segment
        if err := cursor.All(ctx, &segments); err != nil {
                return nil, 0, fmt.Errorf("failed to decode segments: %w", err)
        }
        return segments, total, nil
}

func (r *segmentRepository) FindMatchingSegments(ctx context.Context, customerID string, eventType string) ([]*mktDomain.Segment, error) {
        // Dynamic segments that reference the event type in their rules.
        filter := bson.M{
                "type": mktDomain.SegmentTypeDynamic,
                "rules.field": "event_type",
                "rules.value": eventType,
        }
        cursor, err := r.coll.Find(ctx, filter)
        if err != nil {
                return nil, fmt.Errorf("failed to find matching segments: %w", err)
        }
        defer cursor.Close(ctx)

        var segments []*mktDomain.Segment
        if err := cursor.All(ctx, &segments); err != nil {
                return nil, fmt.Errorf("failed to decode matching segments: %w", err)
        }
        return segments, nil
}

// --- Promotion Repository ---

type promotionRepository struct {
        coll *mongo.Collection
}

// NewPromotionRepository creates a MongoDB-backed PromotionRepository.
func NewPromotionRepository(db *mongo.Database) mktDomain.PromotionRepository {
        return &promotionRepository{coll: db.Collection("promotions")}
}

func (r *promotionRepository) Create(ctx context.Context, promotion *mktDomain.Promotion) error {
        _, err := r.coll.InsertOne(ctx, promotion)
        if err != nil {
                return fmt.Errorf("failed to create promotion: %w", err)
        }
        return nil
}

func (r *promotionRepository) FindByID(ctx context.Context, id string) (*mktDomain.Promotion, error) {
        filter := bson.M{"_id": id}
        var promotion mktDomain.Promotion
        err := r.coll.FindOne(ctx, filter).Decode(&promotion)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, mktDomain.ErrPromotionNotFound
                }
                return nil, fmt.Errorf("failed to find promotion: %w", err)
        }
        return &promotion, nil
}

func (r *promotionRepository) FindByCode(ctx context.Context, code string) (*mktDomain.Promotion, error) {
        filter := bson.M{"code": code}
        var promotion mktDomain.Promotion
        err := r.coll.FindOne(ctx, filter).Decode(&promotion)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, mktDomain.ErrPromotionNotFound
                }
                return nil, fmt.Errorf("failed to find promotion by code: %w", err)
        }
        return &promotion, nil
}

func (r *promotionRepository) Update(ctx context.Context, promotion *mktDomain.Promotion) error {
        filter := bson.M{"_id": promotion.ID}
        update := bson.M{"$set": bson.M{
                "name":                  promotion.Name,
                "code":                  promotion.Code,
                "description":           promotion.Description,
                "status":                promotion.Status,
                "discount_type":         promotion.DiscountType,
                "discount_value":        promotion.DiscountValue,
                "min_order_cents":       promotion.MinOrderCents,
                "max_discount_cents":    promotion.MaxDiscountCents,
                "usage_limit":           promotion.UsageLimit,
                "usage_count":           promotion.UsageCount,
                "applies_to_product_ids":  promotion.AppliesToProductIDs,
                "applies_to_category_ids": promotion.AppliesToCategoryIDs,
                "campaign_id":           promotion.CampaignID,
                "start_at":              promotion.StartAt,
                "end_at":                promotion.EndAt,
                "updated_at":            promotion.UpdatedAt,
        }}
        _, err := r.coll.UpdateOne(ctx, filter, update)
        if err != nil {
                return fmt.Errorf("failed to update promotion: %w", err)
        }
        return nil
}

func (r *promotionRepository) List(ctx context.Context, status string, offset, limit int) ([]*mktDomain.Promotion, int64, error) {
        filter := bson.M{}
        if status != "" {
                filter["status"] = status
        }
        total, err := r.coll.CountDocuments(ctx, filter)
        if err != nil {
                return nil, 0, fmt.Errorf("failed to count promotions: %w", err)
        }
        opts := options.Find().
                SetSort(bson.D{{Key: "created_at", Value: -1}}).
                SetSkip(int64(offset)).
                SetLimit(int64(limit))
        cursor, err := r.coll.Find(ctx, filter, opts)
        if err != nil {
                return nil, 0, fmt.Errorf("failed to list promotions: %w", err)
        }
        defer cursor.Close(ctx)

        var promotions []*mktDomain.Promotion
        if err := cursor.All(ctx, &promotions); err != nil {
                return nil, 0, fmt.Errorf("failed to decode promotions: %w", err)
        }
        return promotions, total, nil
}

// RedeemPromotion atomically increments usage_count if the promotion is redeemable.
// A promotion is redeemable when:
//   - status is ACTIVE
//   - usage_limit is 0 (unlimited) OR usage_count < usage_limit
//
// Returns the updated promotion on success, or an error if the promotion
// cannot be redeemed (not found, not active, or usage limit reached).
func (r *promotionRepository) RedeemPromotion(ctx context.Context, id string) (*mktDomain.Promotion, error) {
        now := time.Now().UTC()

        // Filter: must be ACTIVE, and either unlimited (usage_limit=0) or under the limit
        filter := bson.M{
                "_id":         id,
                "status":      mktDomain.PromotionActive,
                "$or": []bson.M{
                        {"usage_limit": 0},
                        {"$expr": bson.M{"$lt": bson.A{"$usage_count", "$usage_limit"}}},
                },
        }

        update := bson.M{
                "$inc": bson.M{"usage_count": 1},
                "$set": bson.M{"updated_at": now},
        }

        opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

        var updated mktDomain.Promotion
        err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        return nil, fmt.Errorf("promotion not found, not active, or usage limit reached: %w", mktDomain.ErrPromotionNotActive)
                }
                return nil, fmt.Errorf("failed to redeem promotion: %w", err)
        }
        return &updated, nil
}

// --- Indexes ---

// EnsureIndexes creates required indexes for all marketing collections.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
        // Analytics events
        eventColl := db.Collection("analytics_events")
        eventIndexes := []mongo.IndexModel{
                {
                        Keys: bson.D{
                                {Key: "event_id", Value: 1},
                        },
                        Options: options.Index().
                                SetUnique(true).
                                SetName("idx_analytics_event_id_unique"),
                },
                {
                        Keys: bson.D{
                                {Key: "customer_id", Value: 1},
                                {Key: "event_type", Value: 1},
                                {Key: "created_at", Value: -1},
                        },
                        Options: options.Index().
                                SetName("idx_analytics_customer_type_time"),
                },
                {
                        Keys: bson.D{
                                {Key: "event_type", Value: 1},
                                {Key: "created_at", Value: -1},
                        },
                        Options: options.Index().
                                SetName("idx_analytics_event_type_time"),
                },
                {
                        Keys:    bson.D{{Key: "created_at", Value: 1}},
                        Options: options.Index().
                                SetExpireAfterSeconds(int32((30 * 24 * time.Hour).Seconds())).
                                SetName("idx_analytics_created_at_ttl"),
                },
        }
        if _, err := eventColl.Indexes().CreateMany(ctx, eventIndexes); err != nil {
                return fmt.Errorf("failed to create analytics event indexes: %w", err)
        }

        // Campaigns
        campaignColl := db.Collection("campaigns")
        campaignIndexes := []mongo.IndexModel{
                {
                        Keys: bson.D{
                                {Key: "status", Value: 1},
                                {Key: "rules.event_types", Value: 1},
                        },
                        Options: options.Index().
                                SetName("idx_campaign_status_event_types"),
                },
                {
                        Keys:    bson.D{{Key: "created_at", Value: -1}},
                        Options: options.Index().SetName("idx_campaign_created_at"),
                },
        }
        if _, err := campaignColl.Indexes().CreateMany(ctx, campaignIndexes); err != nil {
                return fmt.Errorf("failed to create campaign indexes: %w", err)
        }

        // Segments
        segmentColl := db.Collection("segments")
        segmentIndexes := []mongo.IndexModel{
                {
                        Keys: bson.D{
                                {Key: "type", Value: 1},
                                {Key: "rules.field", Value: 1},
                                {Key: "rules.value", Value: 1},
                        },
                        Options: options.Index().
                                SetName("idx_segment_type_rules"),
                },
                {
                        Keys:    bson.D{{Key: "created_at", Value: -1}},
                        Options: options.Index().SetName("idx_segment_created_at"),
                },
        }
        if _, err := segmentColl.Indexes().CreateMany(ctx, segmentIndexes); err != nil {
                return fmt.Errorf("failed to create segment indexes: %w", err)
        }

        // Promotions
        promoColl := db.Collection("promotions")
        promoIndexes := []mongo.IndexModel{
                {
                        Keys:    bson.D{{Key: "code", Value: 1}},
                        Options: options.Index().SetUnique(true).SetName("idx_promotion_code_unique"),
                },
                {
                        Keys:    bson.D{{Key: "status", Value: 1}},
                        Options: options.Index().SetName("idx_promotion_status"),
                },
                {
                        Keys:    bson.D{{Key: "created_at", Value: -1}},
                        Options: options.Index().SetName("idx_promotion_created_at"),
                },
        }
        if _, err := promoColl.Indexes().CreateMany(ctx, promoIndexes); err != nil {
                return fmt.Errorf("failed to create promotion indexes: %w", err)
        }

        return nil
}
