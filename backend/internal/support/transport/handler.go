package transport

import (
	"strings"
	"time"

	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	"github.com/ecommerce/backend/internal/support/application"
	"github.com/ecommerce/backend/internal/support/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// Handler serves the customer and staff support endpoints.
type Handler struct{ service *application.Service }

// NewHandler creates a support handler.
func NewHandler(s *application.Service) *Handler { return &Handler{service: s} }

// --- request / response shapes ---

// CreateTicketRequest is the body for POST /api/v1/support/tickets.
type CreateTicketRequest struct {
	Subject     string `json:"subject"`
	Category    string `json:"category"`
	OrderNumber string `json:"order_number"`
	Message     string `json:"message"`
}

// MessageRequest is the body for adding a reply.
type MessageRequest struct {
	Message string `json:"message"`
}

// StatusRequest is the body for PATCH /api/v1/admin/support/tickets/:id/status.
type StatusRequest struct {
	Status string `json:"status"`
}

// LastMessage is a short preview shown in lists.
type LastMessage struct {
	SenderRole string `json:"sender_role"`
	Body       string `json:"body"`
}

// TicketSummary is one row in a list.
type TicketSummary struct {
	ID             string       `json:"id"`
	Number         string       `json:"number"`
	Subject        string       `json:"subject"`
	Category       string       `json:"category"`
	OrderNumber    string       `json:"order_number,omitempty"`
	Status         string       `json:"status"`
	CustomerName   string       `json:"customer_name,omitempty"`
	CustomerEmail  string       `json:"customer_email,omitempty"`
	CustomerUnread bool         `json:"customer_unread"`
	StaffUnread    bool         `json:"staff_unread"`
	LastMessage    *LastMessage `json:"last_message,omitempty"`
	LastMessageAt  time.Time    `json:"last_message_at"`
	CreatedAt      time.Time    `json:"created_at"`
}

func preview(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > 140 {
		return string(r[:140]) + "…"
	}
	return string(r)
}

func toSummary(t *domain.Ticket, staff bool) TicketSummary {
	s := TicketSummary{
		ID: t.ID, Number: t.Number, Subject: t.Subject, Category: string(t.Category), OrderNumber: t.OrderNumber,
		Status: string(t.Status), CustomerUnread: t.CustomerUnread, StaffUnread: t.StaffUnread,
		LastMessageAt: t.LastMessageAt, CreatedAt: t.CreatedAt,
	}
	if staff {
		s.CustomerName, s.CustomerEmail = t.CustomerName, t.CustomerEmail
	}
	if n := len(t.Messages); n > 0 {
		last := t.Messages[n-1]
		s.LastMessage = &LastMessage{SenderRole: last.SenderRole, Body: preview(last.Body)}
	}
	return s
}

func summaries(list []*domain.Ticket, staff bool) []TicketSummary {
	out := make([]TicketSummary, len(list))
	for i, t := range list {
		out[i] = toSummary(t, staff)
	}
	return out
}

func authOf(c *fiber.Ctx) (*authTransport.AuthContext, error) {
	a, ok := c.Locals(authTransport.AuthContextKey).(*authTransport.AuthContext)
	if !ok || a == nil {
		return nil, errors.ErrUnauthorized
	}
	return a, nil
}

func parseBody(c *fiber.Ctx, v any) error {
	if err := c.BodyParser(v); err != nil {
		return errors.Wrap("VALIDATION_ERROR", "invalid request body", 422, err)
	}
	return nil
}

// --- customer endpoints ---

// CreateTicket handles POST /api/v1/support/tickets.
func (h *Handler) CreateTicket(c *fiber.Ctx) error {
	a, err := authOf(c)
	if err != nil {
		return response.Error(c, err)
	}
	var req CreateTicketRequest
	if err := parseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	t, err := h.service.Create(c.Context(), application.CreateInput{
		CustomerID: a.UserID, Subject: req.Subject, Category: req.Category, OrderNumber: req.OrderNumber, Message: req.Message,
	})
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, t)
}

// ListMine handles GET /api/v1/support/tickets.
func (h *Handler) ListMine(c *fiber.Ctx) error {
	a, err := authOf(c)
	if err != nil {
		return response.Error(c, err)
	}
	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}
	list, total, err := h.service.CustomerList(c.Context(), a.UserID, strings.ToUpper(c.Query("status")), p)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OKWithMeta(c, summaries(list, false), response.PaginatedMeta(p.Page, p.PerPage, total))
}

// GetMine handles GET /api/v1/support/tickets/:id.
func (h *Handler) GetMine(c *fiber.Ctx) error {
	a, err := authOf(c)
	if err != nil {
		return response.Error(c, err)
	}
	t, err := h.service.CustomerGet(c.Context(), c.Params("id"), a.UserID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, t)
}

// ReplyMine handles POST /api/v1/support/tickets/:id/messages.
func (h *Handler) ReplyMine(c *fiber.Ctx) error {
	a, err := authOf(c)
	if err != nil {
		return response.Error(c, err)
	}
	var req MessageRequest
	if err := parseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	t, err := h.service.CustomerReply(c.Context(), c.Params("id"), a.UserID, req.Message)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, t)
}

// CloseMine handles POST /api/v1/support/tickets/:id/close.
func (h *Handler) CloseMine(c *fiber.Ctx) error {
	a, err := authOf(c)
	if err != nil {
		return response.Error(c, err)
	}
	t, err := h.service.CustomerClose(c.Context(), c.Params("id"), a.UserID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, t)
}

// UnreadMine handles GET /api/v1/support/unread (for the badge in the storefront menu).
func (h *Handler) UnreadMine(c *fiber.Ctx) error {
	a, err := authOf(c)
	if err != nil {
		return response.Error(c, err)
	}
	n, err := h.service.CustomerUnread(c.Context(), a.UserID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, fiber.Map{"count": n})
}

// --- staff endpoints ---

// StaffList handles GET /api/v1/admin/support/tickets.
func (h *Handler) StaffList(c *fiber.Ctx) error {
	p, err := pagination.Parse(c)
	if err != nil {
		return response.Error(c, err)
	}
	list, total, err := h.service.StaffList(c.Context(), strings.ToUpper(c.Query("status")), strings.TrimSpace(c.Query("q")), p)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OKWithMeta(c, summaries(list, true), response.PaginatedMeta(p.Page, p.PerPage, total))
}

// StaffGet handles GET /api/v1/admin/support/tickets/:id.
func (h *Handler) StaffGet(c *fiber.Ctx) error {
	t, err := h.service.StaffGet(c.Context(), c.Params("id"))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, t)
}

// StaffReply handles POST /api/v1/admin/support/tickets/:id/messages.
func (h *Handler) StaffReply(c *fiber.Ctx) error {
	a, err := authOf(c)
	if err != nil {
		return response.Error(c, err)
	}
	var req MessageRequest
	if err := parseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	t, err := h.service.StaffReply(c.Context(), c.Params("id"), a.UserID, req.Message)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, t)
}

// StaffSetStatus handles PATCH /api/v1/admin/support/tickets/:id/status.
func (h *Handler) StaffSetStatus(c *fiber.Ctx) error {
	var req StatusRequest
	if err := parseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	t, err := h.service.StaffSetStatus(c.Context(), c.Params("id"), domain.Status(strings.ToUpper(req.Status)))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, t)
}

// Summary handles GET /api/v1/admin/support/summary.
func (h *Handler) Summary(c *fiber.Ctx) error {
	s, err := h.service.Summary(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, s)
}

// RegisterRoutes mounts customer routes (any signed-in user) and staff routes (staffMW).
func RegisterRoutes(router fiber.Router, h *Handler, authMW, staffMW fiber.Handler) {
	me := router.Group("/api/v1/support", authMW)
	me.Post("/tickets", h.CreateTicket)
	me.Get("/tickets", h.ListMine)
	me.Get("/tickets/:id", h.GetMine)
	me.Post("/tickets/:id/messages", h.ReplyMine)
	me.Post("/tickets/:id/close", h.CloseMine)
	me.Get("/unread", h.UnreadMine)

	staff := router.Group("/api/v1/admin/support", authMW, staffMW)
	staff.Get("/summary", h.Summary)
	staff.Get("/tickets", h.StaffList)
	staff.Get("/tickets/:id", h.StaffGet)
	staff.Post("/tickets/:id/messages", h.StaffReply)
	staff.Patch("/tickets/:id/status", h.StaffSetStatus)
}
