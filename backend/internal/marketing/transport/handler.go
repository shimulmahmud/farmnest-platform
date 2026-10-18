package transport

import (
        mktApp "github.com/ecommerce/backend/internal/marketing/application"
        mktDomain "github.com/ecommerce/backend/internal/marketing/domain"
        authTransport "github.com/ecommerce/backend/internal/auth/transport"
        userDomain "github.com/ecommerce/backend/internal/users/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/ecommerce/backend/pkg/response"
        "github.com/ecommerce/backend/pkg/validator"
        "github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for marketing admin endpoints.
type Handler struct {
        service   *mktApp.Service
        validator *validator.Validator
}

// NewHandler creates a marketing transport handler.
func NewHandler(service *mktApp.Service) *Handler {
        return &Handler{
                service:   service,
                validator: validator.New(),
        }
}

// getAuth extracts the auth context from Fiber locals.
func getAuth(c *fiber.Ctx) (*authTransport.AuthContext, error) {
        auth, ok := c.Locals("auth").(*authTransport.AuthContext)
        if !ok || auth == nil {
                return nil, errors.ErrUnauthorized
        }
        return auth, nil
}

// isAdmin checks if the role is an admin-level role.
func isAdmin(role userDomain.Role) bool {
        return role == userDomain.RoleSuperAdmin || role == userDomain.RoleAdmin || role == userDomain.RoleManager
}

// --- Campaign Handlers ---

// CreateCampaign handles POST /api/v1/admin/marketing/campaigns.
func (h *Handler) CreateCampaign(c *fiber.Ctx) error {
        var req CreateCampaignRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "name":        req.Name,
                "type":        req.Type,
        }); err != nil {
                return response.Error(c, err)
        }

        campaign, err := h.service.CreateCampaign(c.Context(), mktApp.CreateCampaignInput{
                Name:        req.Name,
                Type:        mktDomain.CampaignType(req.Type),
                Description: req.Description,
                Rules: mktDomain.CampaignRules{
                        SegmentIDs:          req.Rules.SegmentIDs,
                        EventTypes:          req.Rules.EventTypes,
                        ProductIDs:          req.Rules.ProductIDs,
                        CategoryIDs:         req.Rules.CategoryIDs,
                        MinAbandonedMinutes: req.Rules.MinAbandonedMinutes,
                        MaxEmailsPerUser:    req.Rules.MaxEmailsPerUser,
                },
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.Created(c, toCampaignResponse(campaign))
}

// GetCampaign handles GET /api/v1/admin/marketing/campaigns/:id.
func (h *Handler) GetCampaign(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "campaign id is required", 422))
        }

        campaign, err := h.service.GetCampaign(c.Context(), id)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toCampaignResponse(campaign))
}

// UpdateCampaign handles PUT /api/v1/admin/marketing/campaigns/:id.
func (h *Handler) UpdateCampaign(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "campaign id is required", 422))
        }

        var req UpdateCampaignRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        campaign, err := h.service.UpdateCampaign(c.Context(), id, mktApp.UpdateCampaignInput{
                Name:        req.Name,
                Description: req.Description,
                Rules: mktDomain.CampaignRules{
                        SegmentIDs:          req.Rules.SegmentIDs,
                        EventTypes:          req.Rules.EventTypes,
                        ProductIDs:          req.Rules.ProductIDs,
                        CategoryIDs:         req.Rules.CategoryIDs,
                        MinAbandonedMinutes: req.Rules.MinAbandonedMinutes,
                        MaxEmailsPerUser:    req.Rules.MaxEmailsPerUser,
                },
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toCampaignResponse(campaign))
}

// TransitionCampaign handles PATCH /api/v1/admin/marketing/campaigns/:id/status.
func (h *Handler) TransitionCampaign(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "campaign id is required", 422))
        }

        var req TransitionCampaignRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        campaign, err := h.service.TransitionCampaign(c.Context(), id, mktDomain.CampaignStatus(req.Status))
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toCampaignResponse(campaign))
}

// ListCampaigns handles GET /api/v1/admin/marketing/campaigns.
func (h *Handler) ListCampaigns(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        result, err := h.service.ListCampaigns(c.Context(), mktApp.ListCampaignsInput{
                Status: c.Query("status"),
                Page:   p.Page,
                Limit:  p.PerPage,
        })
        if err != nil {
                return response.Error(c, err)
        }

        campaigns := make([]CampaignResponse, len(result.Campaigns))
        for i, c := range result.Campaigns {
                campaigns[i] = toCampaignResponse(c)
        }

        meta := response.PaginatedMeta(p.Page, p.PerPage, result.Total)
        return response.OKWithMeta(c, campaigns, meta)
}

// --- Segment Handlers ---

// CreateSegment handles POST /api/v1/admin/marketing/segments.
func (h *Handler) CreateSegment(c *fiber.Ctx) error {
        var req CreateSegmentRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "name": req.Name,
                "type": req.Type,
        }); err != nil {
                return response.Error(c, err)
        }

        segment, err := h.service.CreateSegment(c.Context(), mktApp.CreateSegmentInput{
                Name:        req.Name,
                Type:        mktDomain.SegmentType(req.Type),
                Description: req.Description,
                Rules:       toDomainSegmentRules(req.Rules),
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.Created(c, toSegmentResponse(segment))
}

// GetSegment handles GET /api/v1/admin/marketing/segments/:id.
func (h *Handler) GetSegment(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "segment id is required", 422))
        }

        segment, err := h.service.GetSegment(c.Context(), id)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toSegmentResponse(segment))
}

// UpdateSegment handles PUT /api/v1/admin/marketing/segments/:id.
func (h *Handler) UpdateSegment(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "segment id is required", 422))
        }

        var req UpdateSegmentRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        segment, err := h.service.UpdateSegment(c.Context(), id, mktApp.UpdateSegmentInput{
                Name:        req.Name,
                Description: req.Description,
                Rules:       toDomainSegmentRules(req.Rules),
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toSegmentResponse(segment))
}

// DeleteSegment handles DELETE /api/v1/admin/marketing/segments/:id.
func (h *Handler) DeleteSegment(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "segment id is required", 422))
        }

        if err := h.service.DeleteSegment(c.Context(), id); err != nil {
                return response.Error(c, err)
        }

        return response.NoContent(c)
}

// ListSegments handles GET /api/v1/admin/marketing/segments.
func (h *Handler) ListSegments(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        result, err := h.service.ListSegments(c.Context(), p.Page, p.PerPage)
        if err != nil {
                return response.Error(c, err)
        }

        segments := make([]SegmentResponse, len(result.Segments))
        for i, s := range result.Segments {
                segments[i] = toSegmentResponse(s)
        }

        meta := response.PaginatedMeta(p.Page, p.PerPage, result.Total)
        return response.OKWithMeta(c, segments, meta)
}

// --- Promotion Handlers ---

// CreatePromotion handles POST /api/v1/admin/marketing/promotions.
func (h *Handler) CreatePromotion(c *fiber.Ctx) error {
        var req CreatePromotionRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "name":          req.Name,
                "code":          req.Code,
                "discount_type": req.DiscountType,
        }); err != nil {
                return response.Error(c, err)
        }

        promotion, err := h.service.CreatePromotion(c.Context(), mktApp.CreatePromotionInput{
                Name:             req.Name,
                Code:             req.Code,
                Description:      req.Description,
                DiscountType:     mktDomain.DiscountType(req.DiscountType),
                DiscountValue:    req.DiscountValue,
                MinOrderCents:    req.MinOrderCents,
                MaxDiscountCents: req.MaxDiscountCents,
                UsageLimit:       req.UsageLimit,
                ProductIDs:       req.ProductIDs,
                CategoryIDs:      req.CategoryIDs,
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.Created(c, toPromotionResponse(promotion))
}

// GetPromotion handles GET /api/v1/admin/marketing/promotions/:id.
func (h *Handler) GetPromotion(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "promotion id is required", 422))
        }

        promotion, err := h.service.GetPromotion(c.Context(), id)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toPromotionResponse(promotion))
}

// GetPromotionByCode handles GET /api/v1/admin/marketing/promotions/code/:code.
func (h *Handler) GetPromotionByCode(c *fiber.Ctx) error {
        code := c.Params("code")
        if code == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "promotion code is required", 422))
        }

        promotion, err := h.service.GetPromotionByCode(c.Context(), code)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toPromotionResponse(promotion))
}

// UpdatePromotion handles PUT /api/v1/admin/marketing/promotions/:id.
func (h *Handler) UpdatePromotion(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "promotion id is required", 422))
        }

        var req UpdatePromotionRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        promotion, err := h.service.UpdatePromotion(c.Context(), id, mktApp.UpdatePromotionInput{
                Name:             req.Name,
                Code:             req.Code,
                Description:      req.Description,
                DiscountType:     mktDomain.DiscountType(req.DiscountType),
                DiscountValue:    req.DiscountValue,
                MinOrderCents:    req.MinOrderCents,
                MaxDiscountCents: req.MaxDiscountCents,
                UsageLimit:       req.UsageLimit,
                ProductIDs:       req.ProductIDs,
                CategoryIDs:      req.CategoryIDs,
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toPromotionResponse(promotion))
}

// TransitionPromotion handles PATCH /api/v1/admin/marketing/promotions/:id/status.
func (h *Handler) TransitionPromotion(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "promotion id is required", 422))
        }

        var req TransitionPromotionRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        promotion, err := h.service.TransitionPromotion(c.Context(), id, mktDomain.PromotionStatus(req.Status))
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toPromotionResponse(promotion))
}

// ListPromotions handles GET /api/v1/admin/marketing/promotions.
func (h *Handler) ListPromotions(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        result, err := h.service.ListPromotions(c.Context(), c.Query("status"), p.Page, p.PerPage)
        if err != nil {
                return response.Error(c, err)
        }

        promotions := make([]PromotionResponse, len(result.Promotions))
        for i, p := range result.Promotions {
                promotions[i] = toPromotionResponse(p)
        }

        meta := response.PaginatedMeta(p.Page, p.PerPage, result.Total)
        return response.OKWithMeta(c, promotions, meta)
}
