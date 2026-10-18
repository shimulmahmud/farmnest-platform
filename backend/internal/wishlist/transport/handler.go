package transport

import (
	"github.com/ecommerce/backend/internal/wishlist/application"
	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/ecommerce/backend/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler implements HTTP handlers for wishlist endpoints.
type Handler struct {
	service   *application.Service
	validator *validator.Validator
}

// NewHandler creates a wishlist transport handler.
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

// GetWishlist handles GET /api/v1/wishlist.
func (h *Handler) GetWishlist(c *fiber.Ctx) error {
	auth, err := getAuth(c)
	if err != nil {
		return response.Error(c, err)
	}

	wishlist, err := h.service.GetWishlist(c.Context(), auth.UserID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toWishlistResponse(wishlist))
}

// AddItem handles POST /api/v1/wishlist/items.
func (h *Handler) AddItem(c *fiber.Ctx) error {
	auth, err := getAuth(c)
	if err != nil {
		return response.Error(c, err)
	}

	var req AddWishlistItemRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err))
	}

	if err := h.validator.ValidateRequired(map[string]string{
		"product_id": req.ProductID,
	}); err != nil {
		return response.Error(c, err)
	}

	item, err := h.service.AddItem(c.Context(), auth.UserID, application.AddItemInput{
		ProductID: req.ProductID,
		VariantID: req.VariantID,
	})
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, toItemAddedResponse(item))
}

// RemoveItem handles DELETE /api/v1/wishlist/items/:id.
func (h *Handler) RemoveItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
		return response.Error(c, errors.New("VALIDATION_ERROR", "item id is required", 422))
	}

	auth, err := getAuth(c)
	if err != nil {
		return response.Error(c, err)
	}

	if err := h.service.RemoveItem(c.Context(), auth.UserID, itemID); err != nil {
		return response.Error(c, err)
	}

	return response.NoContent(c)
}
