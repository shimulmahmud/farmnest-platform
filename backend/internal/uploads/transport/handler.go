package transport

import (
	"io"

	"github.com/ecommerce/backend/internal/uploads/application"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// MaxRequestBytes is the biggest upload request the API accepts (the picture limit is checked separately).
const MaxRequestBytes = 10 << 20

// Handler serves picture uploads for the admin panel.
type Handler struct{ service *application.Service }

// NewHandler creates an uploads handler.
func NewHandler(s *application.Service) *Handler { return &Handler{service: s} }

// UploadImage handles POST /api/v1/admin/uploads/images (multipart form, field "file").
func (h *Handler) UploadImage(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, errors.New("VALIDATION_ERROR", "choose a picture to upload (form field \"file\")", 422))
	}
	if fh.Size > MaxRequestBytes {
		return response.Error(c, errors.New("VALIDATION_ERROR", "the picture is too big", 422))
	}
	f, err := fh.Open()
	if err != nil {
		return response.Error(c, errors.Wrap("INTERNAL_ERROR", "could not read the upload", 500, err))
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, MaxRequestBytes+1))
	if err != nil {
		return response.Error(c, errors.Wrap("INTERNAL_ERROR", "could not read the upload", 500, err))
	}

	res, err := h.service.UploadImage(c.Context(), data)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, res)
}

// RegisterRoutes mounts the admin-only upload route.
func RegisterRoutes(router fiber.Router, h *Handler, authMW, adminMW fiber.Handler) {
	g := router.Group("/api/v1/admin/uploads", authMW, adminMW)
	g.Post("/images", h.UploadImage)
}
