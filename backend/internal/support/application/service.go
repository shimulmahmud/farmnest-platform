package application

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/ecommerce/backend/internal/support/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
)

// Service implements the support-messaging use cases.
type Service struct {
	repo     domain.Repository
	users    domain.UserLookup
	orders   domain.OrderChecker
	notifier domain.Notifier
}

// NewService creates a support service. orders and notifier are optional.
func NewService(repo domain.Repository, users domain.UserLookup) *Service {
	return &Service{repo: repo, users: users}
}

// SetOrderChecker enables checking that a referenced order belongs to the customer.
func (s *Service) SetOrderChecker(o domain.OrderChecker) { s.orders = o }

// SetNotifier enables email notifications.
func (s *Service) SetNotifier(n domain.Notifier) { s.notifier = n }

func invalid(msg string) error { return errors.New("VALIDATION_ERROR", msg, 422) }

func cleanText(s string) string { return strings.TrimSpace(s) }

func checkBody(body string) (string, error) {
	body = cleanText(body)
	if body == "" {
		return "", invalid("message cannot be empty")
	}
	if utf8.RuneCountInString(body) > domain.MaxMessageLength {
		return "", invalid("message is too long (maximum 2000 characters)")
	}
	return body, nil
}

// CreateInput starts a conversation.
type CreateInput struct {
	CustomerID  string
	Subject     string
	Category    string
	OrderNumber string
	Message     string
}

// Create opens a ticket with the customer's first message.
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Ticket, error) {
	subject := cleanText(in.Subject)
	if utf8.RuneCountInString(subject) < 3 {
		return nil, invalid("please add a short subject (at least 3 characters)")
	}
	if utf8.RuneCountInString(subject) > domain.MaxSubjectLength {
		return nil, invalid("subject is too long (maximum 120 characters)")
	}
	category := domain.Category(strings.ToUpper(cleanText(in.Category)))
	if category == "" {
		category = domain.CategoryOther
	}
	if !category.IsValid() {
		return nil, invalid("unknown category")
	}
	body, err := checkBody(in.Message)
	if err != nil {
		return nil, err
	}

	open, err := s.repo.CountOpenByCustomer(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	if open >= domain.MaxOpenTicketsPerCustomer {
		return nil, errors.New("TOO_MANY_REQUESTS", "you already have several open conversations; please continue one of them or close it first", 429)
	}

	orderNumber := cleanText(in.OrderNumber)
	if orderNumber != "" && s.orders != nil {
		exact, err := s.orders.CustomerOrderNumber(ctx, in.CustomerID, orderNumber)
		if err != nil {
			return nil, err
		}
		if exact == "" {
			return nil, invalid("we could not find that order in your account")
		}
		orderNumber = exact // store the real number, not what was typed
	}

	name, email, err := s.users.NameAndEmail(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	first := domain.NewMessage(in.CustomerID, domain.SenderCustomer, name, body)
	ticket := domain.NewTicket(in.CustomerID, name, email, subject, category, orderNumber, first)
	if err := s.repo.Create(ctx, ticket); err != nil {
		return nil, err
	}
	if s.notifier != nil {
		s.notifier.CustomerWrote(ctx, ticket, first, true)
	}
	return ticket, nil
}

// ownTicket loads a ticket and hides other customers' tickets (NOT_FOUND, no existence leak).
func (s *Service) ownTicket(ctx context.Context, id, customerID string) (*domain.Ticket, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.CustomerID != customerID {
		return nil, errors.ErrNotFound
	}
	return t, nil
}

// CustomerGet returns the customer's own ticket and marks the shop's replies as read.
func (s *Service) CustomerGet(ctx context.Context, id, customerID string) (*domain.Ticket, error) {
	t, err := s.ownTicket(ctx, id, customerID)
	if err != nil {
		return nil, err
	}
	if t.CustomerUnread {
		if err := s.repo.MarkRead(ctx, id, false); err == nil {
			t.CustomerUnread = false
		}
	}
	return t, nil
}

// CustomerReply adds a customer message. Writing to a closed conversation reopens it.
func (s *Service) CustomerReply(ctx context.Context, id, customerID, message string) (*domain.Ticket, error) {
	t, err := s.ownTicket(ctx, id, customerID)
	if err != nil {
		return nil, err
	}
	body, err := checkBody(message)
	if err != nil {
		return nil, err
	}
	if len(t.Messages) >= domain.MaxMessagesPerTicket {
		return nil, invalid("this conversation is full; please start a new one")
	}
	m := domain.NewMessage(customerID, domain.SenderCustomer, t.CustomerName, body)
	if err := s.repo.AddMessage(ctx, id, m, domain.StatusOpen, false, true); err != nil {
		return nil, err
	}
	if s.notifier != nil {
		t.Status = domain.StatusOpen
		s.notifier.CustomerWrote(ctx, t, m, false)
	}
	return s.repo.FindByID(ctx, id)
}

// CustomerClose lets the customer end the conversation.
func (s *Service) CustomerClose(ctx context.Context, id, customerID string) (*domain.Ticket, error) {
	if _, err := s.ownTicket(ctx, id, customerID); err != nil {
		return nil, err
	}
	if err := s.repo.SetStatus(ctx, id, domain.StatusClosed); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

// CustomerList returns the customer's conversations (without messages).
func (s *Service) CustomerList(ctx context.Context, customerID, status string, p pagination.Params) ([]*domain.Ticket, int64, error) {
	return s.repo.ListByCustomer(ctx, customerID, status, p)
}

// CustomerUnread is the number of conversations with a reply the customer has not read.
func (s *Service) CustomerUnread(ctx context.Context, customerID string) (int64, error) {
	return s.repo.CustomerUnreadCount(ctx, customerID)
}

// --- Staff side ---

// StaffList returns all conversations for the team.
func (s *Service) StaffList(ctx context.Context, status, search string, p pagination.Params) ([]*domain.Ticket, int64, error) {
	return s.repo.ListAll(ctx, status, search, p)
}

// StaffGet returns a conversation and marks it read for the team.
func (s *Service) StaffGet(ctx context.Context, id string) (*domain.Ticket, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.StaffUnread {
		if err := s.repo.MarkRead(ctx, id, true); err == nil {
			t.StaffUnread = false
		}
	}
	return t, nil
}

// StaffReply adds the shop's answer and tells the customer.
func (s *Service) StaffReply(ctx context.Context, id, staffID, message string) (*domain.Ticket, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	body, err := checkBody(message)
	if err != nil {
		return nil, err
	}
	if len(t.Messages) >= domain.MaxMessagesPerTicket {
		return nil, invalid("this conversation is full")
	}
	name := "Support"
	if n, _, err := s.users.NameAndEmail(ctx, staffID); err == nil && n != "" {
		name = n + " (Support)"
	}
	m := domain.NewMessage(staffID, domain.SenderStaff, name, body)
	// A reply to a closed conversation reopens it as ANSWERED so the customer sees it.
	if err := s.repo.AddMessage(ctx, id, m, domain.StatusAnswered, true, false); err != nil {
		return nil, err
	}
	if s.notifier != nil {
		s.notifier.StaffReplied(ctx, t, m)
	}
	return s.repo.FindByID(ctx, id)
}

// StaffSetStatus closes or reopens a conversation.
func (s *Service) StaffSetStatus(ctx context.Context, id string, status domain.Status) (*domain.Ticket, error) {
	if !status.IsValid() {
		return nil, invalid("status must be OPEN, ANSWERED or CLOSED")
	}
	if err := s.repo.SetStatus(ctx, id, status); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

// Summary returns counts for the admin badge.
func (s *Service) Summary(ctx context.Context) (*domain.Summary, error) {
	return s.repo.Summary(ctx)
}
