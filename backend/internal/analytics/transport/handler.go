package transport

import (
	"strings"
	"github.com/ecommerce/backend/internal/analytics/application"
	domain "github.com/ecommerce/backend/internal/analytics/domain"
	appErrors "github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// GetOverview returns a high-level analytics summary for a time range.
// GET /api/v1/analytics/overview?start=...&end=...
func (h *Handler) GetOverview(c *fiber.Ctx) error {
	var query GetAnalyticsQuery
	query.Start = c.Query("start")
	query.StartAt = c.Query("start_at")
	query.EndAt = c.Query("end_at")
	query.End = c.Query("end")

	tr, err := query.ParseTimeRange()
	if err != nil {
		return response.Error(c, appErrors.New("VALIDATION_ERROR", err.Error(), 422))
	}

	result, err := h.service.GetOverview(c.Context(), tr)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toOverviewResponse(result))
}

// GetTopProducts returns the most-viewed products within a time range.
// GET /api/v1/analytics/top-products?start=...&end=...&limit=...
func (h *Handler) GetTopProducts(c *fiber.Ctx) error {
	var query GetAnalyticsQuery
	query.Start = c.Query("start")
	query.StartAt = c.Query("start_at")
	query.EndAt = c.Query("end_at")
	query.End = c.Query("end")
	query.Limit = c.Query("limit")

	tr, err := query.ParseTimeRange()
	if err != nil {
		return response.Error(c, appErrors.New("VALIDATION_ERROR", err.Error(), 422))
	}
	limit := query.ParseLimit()

	stats, err := h.service.GetTopProducts(c.Context(), tr, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toTopProductResponses(stats))
}

// GetDailyTrend returns per-day event counts for a specific event type.
// GET /api/v1/analytics/trends/:event_type?start=...&end=...
func (h *Handler) GetDailyTrend(c *fiber.Ctx) error {
	// The admin sends order_completed; the domain uses order.completed.
	eventType := domain.EventType(strings.ReplaceAll(c.Params("event_type"), "_", "."))

	var query GetAnalyticsQuery
	query.Start = c.Query("start")
	query.StartAt = c.Query("start_at")
	query.EndAt = c.Query("end_at")
	query.End = c.Query("end")

	tr, err := query.ParseTimeRange()
	if err != nil {
		return response.Error(c, appErrors.New("VALIDATION_ERROR", err.Error(), 422))
	}

	counts, err := h.service.GetDailyTrend(c.Context(), eventType, tr)
	if err != nil {
		return response.Error(c, err)
	}

	return response.OK(c, toDailyCountResponses(counts))
}
