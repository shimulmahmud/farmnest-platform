package transport

import (
	"strings"
	"time"

	"github.com/ecommerce/backend/internal/reports/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// Handler serves the admin profit reports.
type Handler struct{ repo domain.Repository }

// NewHandler creates a reports handler.
func NewHandler(repo domain.Repository) *Handler { return &Handler{repo: repo} }

// bst is Bangladesh time (UTC+6). Dates in requests are Bangladesh dates.
var bst = time.FixedZone("BST", 6*60*60)

var validStatuses = map[string]bool{
	"PAID": true, "CONFIRMED": true, "PROCESSING": true, "SHIPPED": true, "DELIVERED": true,
	"PAYMENT_PENDING": true, "CANCELLED": true, "REFUND_PENDING": true, "REFUNDED": true,
}

// parseRange reads from, to (YYYY-MM-DD, Bangladesh time, both inclusive) and
// statuses (comma list). Defaults: this month so far, SHIPPED and DELIVERED orders.
func parseRange(c *fiber.Ctx, now time.Time) (domain.Range, error) {
	local := now.In(bst)
	from := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, bst)
	to := now

	if v := c.Query("from"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, bst)
		if err != nil {
			return domain.Range{}, errors.New("VALIDATION_ERROR", "from must be YYYY-MM-DD", 422)
		}
		from = t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, bst)
		if err != nil {
			return domain.Range{}, errors.New("VALIDATION_ERROR", "to must be YYYY-MM-DD", 422)
		}
		to = t.Add(24*time.Hour - time.Nanosecond)
	}
	if to.Before(from) {
		return domain.Range{}, errors.New("VALIDATION_ERROR", "to must not be before from", 422)
	}

	statuses := []string{"SHIPPED", "DELIVERED"}
	if v := c.Query("statuses"); v != "" {
		statuses = nil
		for _, s := range strings.Split(v, ",") {
			s = strings.ToUpper(strings.TrimSpace(s))
			if !validStatuses[s] {
				return domain.Range{}, errors.New("VALIDATION_ERROR", "unknown status: "+s, 422)
			}
			statuses = append(statuses, s)
		}
	}
	return domain.Range{From: from.UTC(), To: to.UTC(), Statuses: statuses}, nil
}

// Summary handles GET /api/v1/admin/reports/profit.
// Query: from, to (YYYY-MM-DD), statuses (default SHIPPED,DELIVERED), group_by=day|month (default day).
func (h *Handler) Summary(c *fiber.Ctx) error {
	rg, err := parseRange(c, time.Now())
	if err != nil {
		return response.Error(c, err)
	}
	summary, err := h.repo.Summary(c.Context(), rg)
	if err != nil {
		return response.Error(c, err)
	}
	trend, err := h.repo.Trend(c.Context(), rg, c.Query("group_by") == "month")
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, fiber.Map{
		"from":     rg.From.In(bst).Format("2006-01-02"),
		"to":       rg.To.In(bst).Format("2006-01-02"),
		"statuses": rg.Statuses,
		"summary":  summary,
		"trend":    trend,
	})
}

// Products handles GET /api/v1/admin/reports/profit/products.
// Query: from, to, statuses, limit (default 20, max 100), sort=best|worst (default best).
func (h *Handler) Products(c *fiber.Ctx) error {
	rg, err := parseRange(c, time.Now())
	if err != nil {
		return response.Error(c, err)
	}
	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := h.repo.Products(c.Context(), rg, limit, c.Query("sort") == "worst")
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, rows)
}

// Orders handles GET /api/v1/admin/reports/profit/orders (profit of each order).
func (h *Handler) Orders(c *fiber.Ctx) error {
	rg, err := parseRange(c, time.Now())
	if err != nil {
		return response.Error(c, err)
	}
	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}
	rows, total, err := h.repo.Orders(c.Context(), rg, p.Skip, p.Limit)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OKWithMeta(c, rows, response.PaginatedMeta(p.Page, p.PerPage, total))
}

// RegisterRoutes mounts the admin-only report routes.
func RegisterRoutes(router fiber.Router, h *Handler, authMW, adminMW fiber.Handler) {
	g := router.Group("/api/v1/admin/reports/profit", authMW, adminMW)
	g.Get("/", h.Summary)
	g.Get("/products", h.Products)
	g.Get("/orders", h.Orders)
}
