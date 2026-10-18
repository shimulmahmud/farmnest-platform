package application

import (
        "context"
        "errors"

        mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
        "github.com/ecommerce/backend/pkg/audit"
        pkgerrors "github.com/ecommerce/backend/pkg/errors"
)

// Service implements marketing use cases (campaign, segment, promotion management).
type Service struct {
        campaignRepo  mktDomain.CampaignRepository
        segmentRepo   mktDomain.SegmentRepository
        promotionRepo mktDomain.PromotionRepository
        auditor       audit.Auditor
}

// NewService creates a marketing application service.
func NewService(campaignRepo mktDomain.CampaignRepository, segmentRepo mktDomain.SegmentRepository, promotionRepo mktDomain.PromotionRepository) *Service {
        return &Service{
                campaignRepo:  campaignRepo,
                segmentRepo:   segmentRepo,
                promotionRepo: promotionRepo,
                auditor:       audit.NopAuditor,
        }
}

// SetAuditor injects an async auditor for cross-cutting audit logging.
func (s *Service) SetAuditor(a audit.Auditor) {
        s.auditor = a
}

// --- Campaigns ---

// CreateCampaignInput holds the fields for creating a campaign.
type CreateCampaignInput struct {
        Name        string
        Type        mktDomain.CampaignType
        Description string
        Rules       mktDomain.CampaignRules
}

// CreateCampaign creates a new campaign in DRAFT status.
func (s *Service) CreateCampaign(ctx context.Context, input CreateCampaignInput) (*mktDomain.Campaign, error) {
        campaign := mktDomain.NewCampaign(input.Name, input.Type, input.Rules)
        campaign.Description = input.Description

        if err := s.campaignRepo.Create(ctx, campaign); err != nil {
                return nil, err
        }
        return campaign, nil
}

// GetCampaign retrieves a campaign by ID.
func (s *Service) GetCampaign(ctx context.Context, id string) (*mktDomain.Campaign, error) {
        campaign, err := s.campaignRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrCampaignNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "campaign not found", 404)
                }
                return nil, err
        }
        return campaign, nil
}

// UpdateCampaignInput holds the fields for updating a campaign.
type UpdateCampaignInput struct {
        Name        string
        Description string
        Rules       mktDomain.CampaignRules
}

// UpdateCampaign updates an existing campaign.
func (s *Service) UpdateCampaign(ctx context.Context, id string, input UpdateCampaignInput) (*mktDomain.Campaign, error) {
        campaign, err := s.campaignRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrCampaignNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "campaign not found", 404)
                }
                return nil, err
        }

        campaign.Name = input.Name
        campaign.Description = input.Description
        campaign.Rules = input.Rules

        if err := s.campaignRepo.Update(ctx, campaign); err != nil {
                return nil, err
        }
        return campaign, nil
}

// TransitionCampaign changes a campaign's status.
func (s *Service) TransitionCampaign(ctx context.Context, id string, targetStatus mktDomain.CampaignStatus) (*mktDomain.Campaign, error) {
        campaign, err := s.campaignRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrCampaignNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "campaign not found", 404)
                }
                return nil, err
        }

        if err := campaign.TransitionTo(targetStatus); err != nil {
                if err == mktDomain.ErrInvalidCampaignTransition {
                        return nil, pkgerrors.New("VALIDATION_ERROR", err.Error(), 422)
                }
                return nil, err
        }

        if err := s.campaignRepo.Update(ctx, campaign); err != nil {
                return nil, err
        }
        return campaign, nil
}

// ListCampaignsInput holds the filters for listing campaigns.
type ListCampaignsInput struct {
        Status string
        Page   int
        Limit  int
}

// ListCampaignsResult holds the result of listing campaigns.
type ListCampaignsResult struct {
        Campaigns []*mktDomain.Campaign
        Total     int64
}

// ListCampaigns returns a paginated list of campaigns.
func (s *Service) ListCampaigns(ctx context.Context, input ListCampaignsInput) (*ListCampaignsResult, error) {
        if input.Page < 1 {
                input.Page = 1
        }
        if input.Limit < 1 || input.Limit > 100 {
                input.Limit = 20
        }
        offset := (input.Page - 1) * input.Limit

        campaigns, total, err := s.campaignRepo.List(ctx, input.Status, offset, input.Limit)
        if err != nil {
                return nil, err
        }
        return &ListCampaignsResult{Campaigns: campaigns, Total: total}, nil
}

// --- Segments ---

// CreateSegmentInput holds the fields for creating a segment.
type CreateSegmentInput struct {
        Name        string
        Type        mktDomain.SegmentType
        Description string
        Rules       []mktDomain.SegmentRule
}

// CreateSegment creates a new customer segment.
func (s *Service) CreateSegment(ctx context.Context, input CreateSegmentInput) (*mktDomain.Segment, error) {
        segment := mktDomain.NewSegment(input.Name, input.Type, input.Rules)
        segment.Description = input.Description

        if err := s.segmentRepo.Create(ctx, segment); err != nil {
                return nil, err
        }
        return segment, nil
}

// GetSegment retrieves a segment by ID.
func (s *Service) GetSegment(ctx context.Context, id string) (*mktDomain.Segment, error) {
        segment, err := s.segmentRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrSegmentNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "segment not found", 404)
                }
                return nil, err
        }
        return segment, nil
}

// UpdateSegmentInput holds the fields for updating a segment.
type UpdateSegmentInput struct {
        Name        string
        Description string
        Rules       []mktDomain.SegmentRule
}

// UpdateSegment updates an existing segment.
func (s *Service) UpdateSegment(ctx context.Context, id string, input UpdateSegmentInput) (*mktDomain.Segment, error) {
        segment, err := s.segmentRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrSegmentNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "segment not found", 404)
                }
                return nil, err
        }

        segment.Name = input.Name
        segment.Description = input.Description
        segment.Rules = input.Rules

        if err := s.segmentRepo.Update(ctx, segment); err != nil {
                return nil, err
        }
        return segment, nil
}

// DeleteSegment removes a segment.
func (s *Service) DeleteSegment(ctx context.Context, id string) error {
        if err := s.segmentRepo.Delete(ctx, id); err != nil {
                return err
        }
        return nil
}

// ListSegmentsResult holds the result of listing segments.
type ListSegmentsResult struct {
        Segments []*mktDomain.Segment
        Total    int64
}

// ListSegments returns a paginated list of segments.
func (s *Service) ListSegments(ctx context.Context, page, limit int) (*ListSegmentsResult, error) {
        if page < 1 {
                page = 1
        }
        if limit < 1 || limit > 100 {
                limit = 20
        }
        offset := (page - 1) * limit

        segments, total, err := s.segmentRepo.List(ctx, offset, limit)
        if err != nil {
                return nil, err
        }
        return &ListSegmentsResult{Segments: segments, Total: total}, nil
}

// --- Promotions ---

// CreatePromotionInput holds the fields for creating a promotion.
type CreatePromotionInput struct {
        Name            string
        Code            string
        Description     string
        DiscountType    mktDomain.DiscountType
        DiscountValue   int
        MinOrderCents   int
        MaxDiscountCents int
        UsageLimit      int
        ProductIDs      []string
        CategoryIDs     []string
}

// CreatePromotion creates a new promotion in DRAFT status.
func (s *Service) CreatePromotion(ctx context.Context, input CreatePromotionInput) (*mktDomain.Promotion, error) {
        promotion := mktDomain.NewPromotion(input.Name, input.Code, input.DiscountType, input.DiscountValue)
        promotion.Description = input.Description
        promotion.MinOrderCents = input.MinOrderCents
        promotion.MaxDiscountCents = input.MaxDiscountCents
        promotion.UsageLimit = input.UsageLimit
        promotion.AppliesToProductIDs = input.ProductIDs
        promotion.AppliesToCategoryIDs = input.CategoryIDs

        if err := s.promotionRepo.Create(ctx, promotion); err != nil {
                return nil, err
        }
        s.auditor.Record("", audit.ActionPromotionCreate, "promotion", promotion.ID, "", promotion.Code, "promotion created")
        return promotion, nil
}

// GetPromotion retrieves a promotion by ID.
func (s *Service) GetPromotion(ctx context.Context, id string) (*mktDomain.Promotion, error) {
        promotion, err := s.promotionRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrPromotionNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "promotion not found", 404)
                }
                return nil, err
        }
        return promotion, nil
}

// GetPromotionByCode retrieves a promotion by its code.
func (s *Service) GetPromotionByCode(ctx context.Context, code string) (*mktDomain.Promotion, error) {
        promotion, err := s.promotionRepo.FindByCode(ctx, code)
        if err != nil {
                if err == mktDomain.ErrPromotionNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "promotion not found", 404)
                }
                return nil, err
        }
        return promotion, nil
}

// UpdatePromotionInput holds the fields for updating a promotion.
type UpdatePromotionInput struct {
        Name            string
        Code            string
        Description     string
        DiscountType    mktDomain.DiscountType
        DiscountValue   int
        MinOrderCents   int
        MaxDiscountCents int
        UsageLimit      int
        ProductIDs      []string
        CategoryIDs     []string
}

// UpdatePromotion updates an existing promotion.
func (s *Service) UpdatePromotion(ctx context.Context, id string, input UpdatePromotionInput) (*mktDomain.Promotion, error) {
        promotion, err := s.promotionRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrPromotionNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "promotion not found", 404)
                }
                return nil, err
        }

        promotion.Name = input.Name
        promotion.Code = input.Code
        promotion.Description = input.Description
        promotion.DiscountType = input.DiscountType
        promotion.DiscountValue = input.DiscountValue
        promotion.MinOrderCents = input.MinOrderCents
        promotion.MaxDiscountCents = input.MaxDiscountCents
        promotion.UsageLimit = input.UsageLimit
        promotion.AppliesToProductIDs = input.ProductIDs
        promotion.AppliesToCategoryIDs = input.CategoryIDs

        if err := s.promotionRepo.Update(ctx, promotion); err != nil {
                return nil, err
        }
        s.auditor.Record("", audit.ActionPromotionUpdate, "promotion", id, "", promotion.Code, "promotion updated")
        return promotion, nil
}

// TransitionPromotion changes a promotion's status.
func (s *Service) TransitionPromotion(ctx context.Context, id string, targetStatus mktDomain.PromotionStatus) (*mktDomain.Promotion, error) {
        promotion, err := s.promotionRepo.FindByID(ctx, id)
        if err != nil {
                if err == mktDomain.ErrPromotionNotFound {
                        return nil, pkgerrors.New("NOT_FOUND", "promotion not found", 404)
                }
                return nil, err
        }

        oldStatus := promotion.Status

        if err := promotion.TransitionTo(targetStatus); err != nil {
                if err == mktDomain.ErrInvalidPromotionTransition {
                        return nil, pkgerrors.New("VALIDATION_ERROR", err.Error(), 422)
                }
                return nil, err
        }

        if err := s.promotionRepo.Update(ctx, promotion); err != nil {
                return nil, err
        }
        s.auditor.Record("", audit.ActionPromotionUpdate, "promotion", id, string(oldStatus), string(targetStatus), "promotion status transitioned")
        return promotion, nil
}

// ListPromotionsResult holds the result of listing promotions.
type ListPromotionsResult struct {
        Promotions []*mktDomain.Promotion
        Total      int64
}

// ListPromotions returns a paginated list of promotions.
func (s *Service) ListPromotions(ctx context.Context, status string, page, limit int) (*ListPromotionsResult, error) {
        if page < 1 {
                page = 1
        }
        if limit < 1 || limit > 100 {
                limit = 20
        }
        offset := (page - 1) * limit

        promotions, total, err := s.promotionRepo.List(ctx, status, offset, limit)
        if err != nil {
                return nil, err
        }
        return &ListPromotionsResult{Promotions: promotions, Total: total}, nil
}

// RedeemPromotion atomically increments the usage count of an active promotion.
// Returns the updated promotion or an error if the promotion cannot be redeemed.
func (s *Service) RedeemPromotion(ctx context.Context, id string) (*mktDomain.Promotion, error) {
        promotion, err := s.promotionRepo.RedeemPromotion(ctx, id)
        if err != nil {
                if errors.Is(err, mktDomain.ErrPromotionNotActive) {
                        return nil, pkgerrors.New("VALIDATION_ERROR", "promotion cannot be redeemed: not active or usage limit reached", 422)
                }
                return nil, err
        }
        return promotion, nil
}
