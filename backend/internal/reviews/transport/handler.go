package transport

import (
        "github.com/ecommerce/backend/internal/reviews/application"
        authTransport "github.com/ecommerce/backend/internal/auth/transport"
        userDomain "github.com/ecommerce/backend/internal/users/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/ecommerce/backend/pkg/response"
        "github.com/ecommerce/backend/pkg/validator"
        "github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for review endpoints.
type Handler struct {
        service   *application.Service
        validator *validator.Validator
}

// NewHandler creates a review transport handler.
func NewHandler(service *application.Service) *Handler {
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

// CreateReview handles POST /api/v1/products/:id/reviews.
func (h *Handler) CreateReview(c *fiber.Ctx) error {
        productID := c.Params("id")
        if productID == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "product id is required", 422))
        }

        auth, err := getAuth(c)
        if err != nil {
                return response.Error(c, err)
        }

        var req CreateReviewRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        if err := h.validator.ValidateRequired(map[string]string{
                "title": req.Title,
        }); err != nil {
                return response.Error(c, err)
        }
        if req.Rating < 1 || req.Rating > 5 {
                return response.Error(c, errors.New("VALIDATION_ERROR", "rating must be between 1 and 5", 422))
        }

        // Use the authenticated user's ID as customer_id for the review.
        // The orders module also uses auth.UserID as the customer identifier.
        customerID := auth.UserID

        review, err := h.service.CreateReview(c.Context(), application.CreateReviewInput{
                ProductID:  productID,
                CustomerID: customerID,
                Rating:     req.Rating,
                Title:      req.Title,
                Comment:    req.Comment,
                Images:     toDomainImages(req.Images),
        })
        if err != nil {
                return response.Error(c, err)
        }

        return response.Created(c, toReviewResponse(review))
}

// ListReviews handles GET /api/v1/products/:id/reviews.
func (h *Handler) ListReviews(c *fiber.Ctx) error {
        productID := c.Params("id")
        if productID == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "product id is required", 422))
        }

        p, err := pagination.Parse(c)
        if err != nil {
                return response.Error(c, err)
        }

        result, err := h.service.ListReviewsByProduct(c.Context(), productID, p)
        if err != nil {
                return response.Error(c, err)
        }

        reviews := make([]ReviewResponse, len(result.Reviews))
        for i, r := range result.Reviews {
                reviews[i] = toReviewResponse(r)
                reviews[i].CustomerID = "" // public list: never expose account ids
        }

        meta := response.PaginatedMeta(result.Page, result.PerPage, result.Total)
        return response.OKWithMeta(c, reviews, meta)
}

// UpdateReview handles PATCH /api/v1/reviews/:id.
func (h *Handler) UpdateReview(c *fiber.Ctx) error {
        reviewID := c.Params("id")
        if reviewID == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "review id is required", 422))
        }

        auth, err := getAuth(c)
        if err != nil {
                return response.Error(c, err)
        }

        var req UpdateReviewRequest
        if err := c.BodyParser(&req); err != nil {
                return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
        }

        input := application.UpdateReviewInput{
                Rating:  req.Rating,
                Title:   req.Title,
                Comment: req.Comment,
        }
        if req.Images != nil {
                imgs := toDomainImages(*req.Images)
                input.Images = &imgs
        }

        review, err := h.service.UpdateReview(c.Context(), reviewID, auth.UserID, input)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toReviewResponse(review))
}

// DeleteReview handles DELETE /api/v1/reviews/:id.
func (h *Handler) DeleteReview(c *fiber.Ctx) error {
        reviewID := c.Params("id")
        if reviewID == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "review id is required", 422))
        }

        auth, err := getAuth(c)
        if err != nil {
                return response.Error(c, err)
        }

        if err := h.service.DeleteReview(c.Context(), reviewID, auth.UserID, isAdmin(auth.Role)); err != nil {
                return response.Error(c, err)
        }

        return response.NoContent(c)
}

// GetReview handles GET /api/v1/reviews/:id.
func (h *Handler) GetReview(c *fiber.Ctx) error {
        reviewID := c.Params("id")
        if reviewID == "" {
                return response.Error(c, errors.New("VALIDATION_ERROR", "review id is required", 422))
        }

        review, err := h.service.GetReview(c.Context(), reviewID)
        if err != nil {
                return response.Error(c, err)
        }

        return response.OK(c, toReviewResponse(review))
}
