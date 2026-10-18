package transport

import (
        "github.com/ecommerce/backend/internal/categories/application"
        categoryDomain "github.com/ecommerce/backend/internal/categories/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/ecommerce/backend/pkg/response"
        "github.com/ecommerce/backend/pkg/validator"
        "github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for category endpoints.
type Handler struct {
        service   *application.Service
        validator *validator.Validator
}

// NewHandler creates a category transport handler.
func NewHandler(service *application.Service) *Handler {
        return &Handler{
                service:   service,
                validator: validator.New(),
        }
}

// ListCategories handles GET /api/v1/categories.
func (h *Handler) ListCategories(c *fiber.Ctx) error {
        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        status := c.Query("status")

        result, err := h.service.List(c.Context(), status, p)
        if err != nil {
                return response.Error(c, err)
        }

        categories := make([]CategoryResponse, len(result.Categories))
        for i, cat := range result.Categories {
                categories[i] = toCategoryResponse(cat)
        }

        meta := response.PaginatedMeta(result.Page, result.PerPage, result.Total)
        return response.OKWithMeta(c, categories, meta)
}

// GetCategory handles GET /api/v1/categories/:slug.
func (h *Handler) GetCategory(c *fiber.Ctx) error {
        slug := c.Params("slug")
        if slug == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "slug is required", 422))
        }

        category, err := h.service.GetBySlug(c.Context(), slug)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toCategoryResponse(category))
}

// CreateCategory handles POST /api/v1/admin/categories.
func (h *Handler) CreateCategory(c *fiber.Ctx) error {
        var req CreateCategoryRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "name": req.Name,
                "slug": req.Slug,
        }); err != nil {
                return response.Error(c, err)
        }

        input := application.CreateCategoryInput{
                Name:        req.Name,
                Slug:        req.Slug,
                Description: req.Description,
                ParentID:    req.ParentID,
                Image:       req.Image,
                SortOrder:   req.SortOrder,
                SEO:         toSEOInput(req.SEO),
        }

        category, err := h.service.Create(c.Context(), input)
        if err != nil {
                return response.Error(c, err)
        }

        return response.Created(c, toCategoryResponse(category))
}

// UpdateCategory handles PATCH /api/v1/admin/categories/:id.
func (h *Handler) UpdateCategory(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "id is required", 422))
        }

        var req UpdateCategoryRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        input := application.UpdateCategoryInput{
                Name:        req.Name,
                Slug:        req.Slug,
                Description: req.Description,
                ParentID:    req.ParentID,
                Image:       req.Image,
                SortOrder:   req.SortOrder,
                Status:      req.Status,
        }

        if req.SEO != nil {
                seo := toSEOInput(*req.SEO)
                input.SEO = &seo
        }

        category, err := h.service.Update(c.Context(), id, input)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toCategoryResponse(category))
}

// DeleteCategory handles DELETE /api/v1/admin/categories/:id.
func (h *Handler) DeleteCategory(c *fiber.Ctx) error {
        id := c.Params("id")
        if id == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "id is required", 422))
        }

        if err := h.service.Delete(c.Context(), id); err != nil {
                return response.Error(c, err)
        }

        return response.NoContent(c)
}

func toCategoryResponse(c *categoryDomain.Category) CategoryResponse {
        return CategoryResponse{
                ID:          c.ID,
                Name:        c.Name,
                Slug:        c.Slug,
                Description: c.Description,
                ParentID:    c.ParentID,
                Image:       c.Image,
                SortOrder:   c.SortOrder,
                Status:      string(c.Status),
                SEO:         toSEOOutput(c.SEO),
                CreatedAt:   c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
                UpdatedAt:   c.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
        }
}
