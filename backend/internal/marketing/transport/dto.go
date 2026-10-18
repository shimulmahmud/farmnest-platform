package transport

import (
	mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
)

// --- Campaign DTOs ---

// CreateCampaignRequest is the request body for creating a campaign.
type CreateCampaignRequest struct {
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Description string                   `json:"description,omitempty"`
	Rules       CampaignRulesRequest      `json:"rules"`
}

// CampaignRulesRequest is the rules section of a campaign request.
type CampaignRulesRequest struct {
	SegmentIDs          []string `json:"segment_ids"`
	EventTypes          []string `json:"event_types"`
	ProductIDs          []string `json:"product_ids,omitempty"`
	CategoryIDs         []string `json:"category_ids,omitempty"`
	MinAbandonedMinutes int     `json:"min_abandoned_minutes,omitempty"`
	MaxEmailsPerUser    int     `json:"max_emails_per_user,omitempty"`
}

// UpdateCampaignRequest is the request body for updating a campaign.
type UpdateCampaignRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Rules       CampaignRulesRequest `json:"rules"`
}

// TransitionCampaignRequest is the request body for transitioning campaign status.
type TransitionCampaignRequest struct {
	Status string `json:"status"`
}

// CampaignResponse is the response DTO for a campaign.
type CampaignResponse struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Type        string                  `json:"type"`
	Status      string                  `json:"status"`
	Description string                  `json:"description,omitempty"`
	Rules       CampaignRulesResponse     `json:"rules"`
	Metrics     CampaignMetricsResponse  `json:"metrics"`
	StartAt     *string                 `json:"start_at,omitempty"`
	EndAt       *string                 `json:"end_at,omitempty"`
	CreatedAt   string                  `json:"created_at"`
	UpdatedAt   string                  `json:"updated_at"`
}

// CampaignRulesResponse is the rules section of a campaign response.
type CampaignRulesResponse struct {
	SegmentIDs          []string `json:"segment_ids"`
	EventTypes          []string `json:"event_types"`
	ProductIDs          []string `json:"product_ids,omitempty"`
	CategoryIDs         []string `json:"category_ids,omitempty"`
	MinAbandonedMinutes int     `json:"min_abandoned_minutes,omitempty"`
	MaxEmailsPerUser    int     `json:"max_emails_per_user,omitempty"`
}

// CampaignMetricsResponse is the metrics section of a campaign response.
type CampaignMetricsResponse struct {
	EventsReceived  int64 `json:"events_received"`
	CustomersReached int64 `json:"customers_reached"`
	MessagesSent    int64 `json:"messages_sent"`
	Conversions     int64 `json:"conversions"`
	RevenueCents    int64 `json:"revenue_cents"`
}

// --- Segment DTOs ---

// CreateSegmentRequest is the request body for creating a segment.
type CreateSegmentRequest struct {
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Rules       []SegmentRuleRequest `json:"rules,omitempty"`
}

// SegmentRuleRequest is a single rule in a segment request.
type SegmentRuleRequest struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// UpdateSegmentRequest is the request body for updating a segment.
type UpdateSegmentRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Rules       []SegmentRuleRequest `json:"rules,omitempty"`
}

// SegmentResponse is the response DTO for a segment.
type SegmentResponse struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	Description   string              `json:"description,omitempty"`
	Rules         []SegmentRuleRequest `json:"rules,omitempty"`
	CustomerIDs   []string            `json:"customer_ids,omitempty"`
	CustomerCount int64               `json:"customer_count"`
	CreatedAt     string              `json:"created_at"`
	UpdatedAt     string              `json:"updated_at"`
}

// --- Promotion DTOs ---

// CreatePromotionRequest is the request body for creating a promotion.
type CreatePromotionRequest struct {
	Name             string   `json:"name"`
	Code             string   `json:"code"`
	Description      string   `json:"description,omitempty"`
	DiscountType     string   `json:"discount_type"`
	DiscountValue    int      `json:"discount_value"`
	MinOrderCents    int      `json:"min_order_cents,omitempty"`
	MaxDiscountCents int      `json:"max_discount_cents,omitempty"`
	UsageLimit       int      `json:"usage_limit,omitempty"`
	ProductIDs       []string `json:"product_ids,omitempty"`
	CategoryIDs      []string `json:"category_ids,omitempty"`
}

// UpdatePromotionRequest is the request body for updating a promotion.
type UpdatePromotionRequest struct {
	Name             string   `json:"name"`
	Code             string   `json:"code"`
	Description      string   `json:"description,omitempty"`
	DiscountType     string   `json:"discount_type"`
	DiscountValue    int      `json:"discount_value"`
	MinOrderCents    int      `json:"min_order_cents,omitempty"`
	MaxDiscountCents int      `json:"max_discount_cents,omitempty"`
	UsageLimit       int      `json:"usage_limit,omitempty"`
	ProductIDs       []string `json:"product_ids,omitempty"`
	CategoryIDs      []string `json:"category_ids,omitempty"`
}

// TransitionPromotionRequest is the request body for transitioning promotion status.
type TransitionPromotionRequest struct {
	Status string `json:"status"`
}

// PromotionResponse is the response DTO for a promotion.
type PromotionResponse struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Code                string   `json:"code"`
	Description         string   `json:"description,omitempty"`
	Status              string   `json:"status"`
	DiscountType        string   `json:"discount_type"`
	DiscountValue       int      `json:"discount_value"`
	MinOrderCents       int      `json:"min_order_cents,omitempty"`
	MaxDiscountCents    int      `json:"max_discount_cents,omitempty"`
	UsageLimit          int      `json:"usage_limit,omitempty"`
	UsageCount          int      `json:"usage_count"`
	AppliesToProductIDs  []string `json:"applies_to_product_ids,omitempty"`
	AppliesToCategoryIDs []string `json:"applies_to_category_ids,omitempty"`
	CampaignID          string   `json:"campaign_id,omitempty"`
	StartAt             *string  `json:"start_at,omitempty"`
	EndAt               *string  `json:"end_at,omitempty"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

// --- Mappers ---

func toCampaignResponse(c *mktDomain.Campaign) CampaignResponse {
	resp := CampaignResponse{
		ID:          c.ID,
		Name:        c.Name,
		Type:        string(c.Type),
		Status:      string(c.Status),
		Description: c.Description,
		Rules: CampaignRulesResponse{
			SegmentIDs:          c.Rules.SegmentIDs,
			EventTypes:          c.Rules.EventTypes,
			ProductIDs:          c.Rules.ProductIDs,
			CategoryIDs:         c.Rules.CategoryIDs,
			MinAbandonedMinutes: c.Rules.MinAbandonedMinutes,
			MaxEmailsPerUser:    c.Rules.MaxEmailsPerUser,
		},
		Metrics: CampaignMetricsResponse{
			EventsReceived:   c.Metrics.EventsReceived,
			CustomersReached: c.Metrics.CustomersReached,
			MessagesSent:     c.Metrics.MessagesSent,
			Conversions:      c.Metrics.Conversions,
			RevenueCents:     c.Metrics.RevenueCents,
		},
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if c.StartAt != nil {
		s := c.StartAt.Format("2006-01-02T15:04:05Z07:00")
		resp.StartAt = &s
	}
	if c.EndAt != nil {
		s := c.EndAt.Format("2006-01-02T15:04:05Z07:00")
		resp.EndAt = &s
	}
	return resp
}

func toSegmentResponse(s *mktDomain.Segment) SegmentResponse {
	return SegmentResponse{
		ID:            s.ID,
		Name:          s.Name,
		Type:          string(s.Type),
		Description:   s.Description,
		Rules:         toSegmentRuleRequests(s.Rules),
		CustomerIDs:   s.CustomerIDs,
		CustomerCount: s.CustomerCount,
		CreatedAt:     s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toSegmentRuleRequests(rules []mktDomain.SegmentRule) []SegmentRuleRequest {
	result := make([]SegmentRuleRequest, len(rules))
	for i, r := range rules {
		result[i] = SegmentRuleRequest{
			Field:    r.Field,
			Operator: r.Operator,
			Value:    r.Value,
		}
	}
	return result
}

func toDomainSegmentRules(rules []SegmentRuleRequest) []mktDomain.SegmentRule {
	result := make([]mktDomain.SegmentRule, len(rules))
	for i, r := range rules {
		result[i] = mktDomain.SegmentRule{
			Field:    r.Field,
			Operator: r.Operator,
			Value:    r.Value,
		}
	}
	return result
}

func toPromotionResponse(p *mktDomain.Promotion) PromotionResponse {
	resp := PromotionResponse{
		ID:                  p.ID,
		Name:                p.Name,
		Code:                p.Code,
		Description:         p.Description,
		Status:              string(p.Status),
		DiscountType:        string(p.DiscountType),
		DiscountValue:       p.DiscountValue,
		MinOrderCents:       p.MinOrderCents,
		MaxDiscountCents:    p.MaxDiscountCents,
		UsageLimit:          p.UsageLimit,
		UsageCount:          p.UsageCount,
		AppliesToProductIDs:  p.AppliesToProductIDs,
		AppliesToCategoryIDs: p.AppliesToCategoryIDs,
		CampaignID:          p.CampaignID,
		CreatedAt:           p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if p.StartAt != nil {
		s := p.StartAt.Format("2006-01-02T15:04:05Z07:00")
		resp.StartAt = &s
	}
	if p.EndAt != nil {
		s := p.EndAt.Format("2006-01-02T15:04:05Z07:00")
		resp.EndAt = &s
	}
	return resp
}
