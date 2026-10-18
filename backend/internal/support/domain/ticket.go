package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ecommerce/backend/pkg/pagination"
)

// Status of a support conversation.
type Status string

const (
	StatusOpen     Status = "OPEN"     // waiting for the shop to answer
	StatusAnswered Status = "ANSWERED" // the shop answered; waiting for the customer
	StatusClosed   Status = "CLOSED"
)

// IsValid reports whether s is a known status.
func (s Status) IsValid() bool {
	return s == StatusOpen || s == StatusAnswered || s == StatusClosed
}

// Category helps the shop route and prioritise messages.
type Category string

const (
	CategoryOrder    Category = "ORDER"
	CategoryPayment  Category = "PAYMENT"
	CategoryDelivery Category = "DELIVERY"
	CategoryProduct  Category = "PRODUCT"
	CategoryAccount  Category = "ACCOUNT"
	CategoryOther    Category = "OTHER"
)

// IsValid reports whether c is a known category.
func (c Category) IsValid() bool {
	switch c {
	case CategoryOrder, CategoryPayment, CategoryDelivery, CategoryProduct, CategoryAccount, CategoryOther:
		return true
	}
	return false
}

// Sender roles inside a conversation.
const (
	SenderCustomer = "CUSTOMER"
	SenderStaff    = "STAFF"
)

// Limits (kept here so the rules are in one place).
const (
	MaxOpenTicketsPerCustomer = 5
	MaxMessageLength          = 2000
	MaxSubjectLength          = 120
	MaxMessagesPerTicket      = 200
)

// Message is one entry in a conversation.
type Message struct {
	ID         string    `bson:"id" json:"id"`
	SenderID   string    `bson:"sender_id" json:"-"`
	SenderRole string    `bson:"sender_role" json:"sender_role"` // CUSTOMER or STAFF
	SenderName string    `bson:"sender_name" json:"sender_name"`
	Body       string    `bson:"body" json:"body"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}

// Ticket is a conversation between one customer and the shop's support team.
type Ticket struct {
	ID            string    `bson:"_id" json:"id"`
	Number        string    `bson:"number" json:"number"`
	CustomerID    string    `bson:"customer_id" json:"customer_id"`
	CustomerName  string    `bson:"customer_name" json:"customer_name"`
	CustomerEmail string    `bson:"customer_email" json:"customer_email"`
	Subject       string    `bson:"subject" json:"subject"`
	Category      Category  `bson:"category" json:"category"`
	OrderNumber   string    `bson:"order_number,omitempty" json:"order_number,omitempty"`
	Status        Status    `bson:"status" json:"status"`
	Messages      []Message `bson:"messages" json:"messages"`
	// CustomerUnread: the shop wrote and the customer has not opened it yet.
	CustomerUnread bool `bson:"customer_unread" json:"customer_unread"`
	// StaffUnread: the customer wrote and nobody on the team has opened it yet.
	StaffUnread   bool      `bson:"staff_unread" json:"staff_unread"`
	LastMessageAt time.Time `bson:"last_message_at" json:"last_message_at"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
}

// NewMessage creates a message stamped now.
func NewMessage(senderID, role, name, body string) Message {
	return Message{ID: uuid.New().String(), SenderID: senderID, SenderRole: role, SenderName: name, Body: body, CreatedAt: time.Now().UTC()}
}

// NewTicket starts a conversation with the customer's first message.
func NewTicket(customerID, customerName, customerEmail, subject string, category Category, orderNumber string, first Message) *Ticket {
	now := time.Now().UTC()
	id := uuid.New().String()
	return &Ticket{
		ID:             id,
		Number:         fmt.Sprintf("SUP-%s-%s", now.Format("060102"), id[:4]),
		CustomerID:     customerID,
		CustomerName:   customerName,
		CustomerEmail:  customerEmail,
		Subject:        subject,
		Category:       category,
		OrderNumber:    orderNumber,
		Status:         StatusOpen,
		Messages:       []Message{first},
		StaffUnread:    true,
		LastMessageAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Summary counts tickets for the admin badge and filters.
type Summary struct {
	Open     int64 `json:"open"`
	Answered int64 `json:"answered"`
	Closed   int64 `json:"closed"`
	Unread   int64 `json:"unread"` // tickets with a customer message nobody on the team has opened
}

// Repository persists tickets.
type Repository interface {
	Create(ctx context.Context, t *Ticket) error
	FindByID(ctx context.Context, id string) (*Ticket, error)
	// ListByCustomer returns one customer's tickets (messages left out), newest activity first.
	ListByCustomer(ctx context.Context, customerID string, status string, p pagination.Params) ([]*Ticket, int64, error)
	// ListAll returns every ticket for the team (messages left out), newest activity first.
	ListAll(ctx context.Context, status, search string, p pagination.Params) ([]*Ticket, int64, error)
	// AddMessage appends a message and sets the status and unread flags in one atomic update.
	AddMessage(ctx context.Context, ticketID string, m Message, status Status, customerUnread, staffUnread bool) error
	SetStatus(ctx context.Context, ticketID string, status Status) error
	MarkRead(ctx context.Context, ticketID string, byStaff bool) error
	CountOpenByCustomer(ctx context.Context, customerID string) (int64, error)
	CustomerUnreadCount(ctx context.Context, customerID string) (int64, error)
	Summary(ctx context.Context) (*Summary, error)
}

// OrderChecker finds one of the customer's orders by its number, ignoring letter case.
type OrderChecker interface {
	// CustomerOrderNumber returns the order's exact stored number, or "" when the customer has no such order.
	CustomerOrderNumber(ctx context.Context, customerID, orderNumber string) (string, error)
}

// UserLookup returns a user's display name and email.
type UserLookup interface {
	NameAndEmail(ctx context.Context, userID string) (name, email string, err error)
}

// Notifier tells people about new messages (email). Failures never block a reply.
type Notifier interface {
	// CustomerWrote tells the support team about a new ticket or a customer reply.
	CustomerWrote(ctx context.Context, t *Ticket, m Message, isNew bool)
	// StaffReplied tells the customer the shop answered.
	StaffReplied(ctx context.Context, t *Ticket, m Message)
}
