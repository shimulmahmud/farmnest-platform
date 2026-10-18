package infrastructure

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ecommerce/backend/internal/support/domain"
	"github.com/ecommerce/backend/pkg/emailtemplate"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ---- user names (users collection) ----

type userLookup struct{ users *mongo.Collection }

// NewUserLookup returns a domain.UserLookup reading first/last name and email from "users".
func NewUserLookup(db *mongo.Database) domain.UserLookup {
	return &userLookup{users: db.Collection("users")}
}

func (u *userLookup) NameAndEmail(ctx context.Context, userID string) (string, string, error) {
	var d struct {
		First string `bson:"first_name"`
		Last  string `bson:"last_name"`
		Email string `bson:"email"`
	}
	if err := u.users.FindOne(ctx, bson.M{"_id": userID}).Decode(&d); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(d.First + " " + d.Last), d.Email, nil
}

// ---- orders (orders collection) ----

type orderChecker struct{ orders *mongo.Collection }

// NewOrderChecker returns a domain.OrderChecker reading from "orders".
func NewOrderChecker(db *mongo.Database) domain.OrderChecker {
	return &orderChecker{orders: db.Collection("orders")}
}

func (o *orderChecker) CustomerOrderNumber(ctx context.Context, customerID, orderNumber string) (string, error) {
	var d struct {
		Number string `bson:"order_number"`
	}
	re := bson.Regex{Pattern: "^" + regexp.QuoteMeta(orderNumber) + "$", Options: "i"}
	err := o.orders.FindOne(ctx, bson.M{"order_number": re, "customer_id": customerID}).Decode(&d)
	if err == mongo.ErrNoDocuments {
		return "", nil
	}
	return d.Number, err
}

// ---- email notifications ----

// HTMLSender sends an email with an HTML version (the SMTP mailer implements it).
type HTMLSender interface {
	SendHTML(ctx context.Context, to, subject, text, html string) error
}

type emailNotifier struct {
	sender     HTMLSender
	brand      emailtemplate.Brand
	staffEmail string
}

// NewEmailNotifier emails the customer when the shop replies, and the support inbox
// (staffEmail, optional) when a customer writes.
func NewEmailNotifier(sender HTMLSender, brand emailtemplate.Brand, staffEmail string) domain.Notifier {
	return &emailNotifier{sender: sender, brand: brand.Normalize(), staffEmail: strings.TrimSpace(staffEmail)}
}

func excerpt(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}

// async delivers in the background so a slow mail server never delays the reply, and
// failures never fail the conversation.
func (n *emailNotifier) async(to string, r emailtemplate.Rendered) {
	if to == "" || r.HTML == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = n.sender.SendHTML(ctx, to, r.Subject, r.Text, r.HTML)
	}()
}

func (n *emailNotifier) CustomerWrote(_ context.Context, t *domain.Ticket, m domain.Message, isNew bool) {
	if n.staffEmail == "" {
		return
	}
	title := "New support message"
	if !isNew {
		title = "Customer replied"
	}
	subject := fmt.Sprintf("[%s] %s: %s", t.Number, title, t.Subject)
	paras := []string{
		fmt.Sprintf("%s (%s) wrote:", t.CustomerName, t.CustomerEmail),
		excerpt(m.Body, 600),
	}
	if t.OrderNumber != "" {
		paras = append(paras, "Order: "+t.OrderNumber)
	}
	n.async(n.staffEmail, n.brand.Notice(subject, title, "", paras, "Open the conversation", n.brand.AdminURL+"/admin/support?ticket="+t.ID))
}

func (n *emailNotifier) StaffReplied(_ context.Context, t *domain.Ticket, m domain.Message) {
	subject := fmt.Sprintf("%s replied to your message: %s", n.brand.Name, t.Subject)
	paras := []string{
		fmt.Sprintf("We replied to your message “%s”:", t.Subject),
		excerpt(m.Body, 600),
	}
	n.async(t.CustomerEmail, n.brand.Notice(subject, "We replied to your message", t.CustomerName, paras, "View and reply", n.brand.AppURL+"/support/"+t.ID))
}
