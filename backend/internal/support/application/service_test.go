package application

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/support/domain"
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/ecommerce/backend/pkg/pagination"
)

type memRepo struct {
	mu      sync.Mutex
	tickets map[string]*domain.Ticket
}

func newMemRepo() *memRepo { return &memRepo{tickets: map[string]*domain.Ticket{}} }

func (r *memRepo) Create(_ context.Context, t *domain.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	cp.Messages = append([]domain.Message(nil), t.Messages...)
	r.tickets[t.ID] = &cp
	return nil
}
func (r *memRepo) FindByID(_ context.Context, id string) (*domain.Ticket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tickets[id]
	if !ok {
		return nil, errors.ErrNotFound
	}
	cp := *t
	cp.Messages = append([]domain.Message(nil), t.Messages...)
	return &cp, nil
}
func (r *memRepo) ListByCustomer(_ context.Context, customerID, _ string, _ pagination.Params) ([]*domain.Ticket, int64, error) {
	var out []*domain.Ticket
	for _, t := range r.tickets {
		if t.CustomerID == customerID {
			out = append(out, t)
		}
	}
	return out, int64(len(out)), nil
}
func (r *memRepo) ListAll(_ context.Context, _, _ string, _ pagination.Params) ([]*domain.Ticket, int64, error) {
	var out []*domain.Ticket
	for _, t := range r.tickets {
		out = append(out, t)
	}
	return out, int64(len(out)), nil
}
func (r *memRepo) AddMessage(_ context.Context, id string, m domain.Message, st domain.Status, cu, su bool) error {
	t, ok := r.tickets[id]
	if !ok {
		return errors.ErrNotFound
	}
	t.Messages = append(t.Messages, m)
	t.Status, t.CustomerUnread, t.StaffUnread = st, cu, su
	return nil
}
func (r *memRepo) SetStatus(_ context.Context, id string, st domain.Status) error {
	t, ok := r.tickets[id]
	if !ok {
		return errors.ErrNotFound
	}
	t.Status = st
	return nil
}
func (r *memRepo) MarkRead(_ context.Context, id string, byStaff bool) error {
	t := r.tickets[id]
	if byStaff {
		t.StaffUnread = false
	} else {
		t.CustomerUnread = false
	}
	return nil
}
func (r *memRepo) CountOpenByCustomer(_ context.Context, customerID string) (int64, error) {
	var n int64
	for _, t := range r.tickets {
		if t.CustomerID == customerID && t.Status != domain.StatusClosed {
			n++
		}
	}
	return n, nil
}
func (r *memRepo) CustomerUnreadCount(_ context.Context, customerID string) (int64, error) {
	var n int64
	for _, t := range r.tickets {
		if t.CustomerID == customerID && t.CustomerUnread {
			n++
		}
	}
	return n, nil
}
func (r *memRepo) Summary(_ context.Context) (*domain.Summary, error) { return &domain.Summary{}, nil }

type fakeUsers struct{}

func (fakeUsers) NameAndEmail(_ context.Context, id string) (string, string, error) {
	return "Name " + id, id + "@example.com", nil
}

// fakeOrders knows each customer's real order numbers and matches them ignoring letter case.
type fakeOrders struct{ owned map[string]bool }

func (f fakeOrders) CustomerOrderNumber(_ context.Context, c, o string) (string, error) {
	for key := range f.owned {
		if strings.EqualFold(key, c+"/"+o) {
			return key[len(c)+1:], nil
		}
	}
	return "", nil
}

type recorder struct{ staffNotified, customerNotified int }

func (r *recorder) CustomerWrote(_ context.Context, _ *domain.Ticket, _ domain.Message, _ bool) { r.staffNotified++ }
func (r *recorder) StaffReplied(_ context.Context, _ *domain.Ticket, _ domain.Message)         { r.customerNotified++ }

func newSvc() (*Service, *memRepo, *recorder) {
	repo := newMemRepo()
	rec := &recorder{}
	s := NewService(repo, fakeUsers{})
	s.SetNotifier(rec)
	s.SetOrderChecker(fakeOrders{owned: map[string]bool{"alice/ORD-20261014-fba3f974": true}})
	return s, repo, rec
}

func create(t *testing.T, s *Service, customer string) *domain.Ticket {
	t.Helper()
	tk, err := s.Create(context.Background(), CreateInput{CustomerID: customer, Subject: "Where is my order?", Category: "order", Message: "Hello, any news?"})
	require.NoError(t, err)
	return tk
}

func TestCreate_StartsOpenAndUnreadForStaff_AndNotifiesTeam(t *testing.T) {
	s, _, rec := newSvc()
	tk := create(t, s, "alice")
	assert.Equal(t, domain.StatusOpen, tk.Status)
	assert.True(t, tk.StaffUnread)
	assert.Len(t, tk.Messages, 1)
	assert.Equal(t, "Name alice", tk.CustomerName)
	assert.Equal(t, 1, rec.staffNotified)
}

func TestCreate_Validation(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()
	_, err := s.Create(ctx, CreateInput{CustomerID: "alice", Subject: "x", Message: "hi"})
	require.Error(t, err, "subject too short")
	_, err = s.Create(ctx, CreateInput{CustomerID: "alice", Subject: "Help", Message: "   "})
	require.Error(t, err, "empty message")
	_, err = s.Create(ctx, CreateInput{CustomerID: "alice", Subject: "Help", Category: "WEIRD", Message: "hi"})
	require.Error(t, err, "unknown category")
	_, err = s.Create(ctx, CreateInput{CustomerID: "alice", Subject: "Help", OrderNumber: "ORD-999", Message: "hi"})
	require.Error(t, err, "order not in this customer's account")
	tk, err := s.Create(ctx, CreateInput{CustomerID: "alice", Subject: "Help", OrderNumber: "ord-20261014-FBA3F974", Message: "hi"})
	require.NoError(t, err, "own order accepted, in any letter case")
	assert.Equal(t, "ORD-20261014-fba3f974", tk.OrderNumber, "the stored number is the real one")
}

func TestCreate_LimitsOpenConversations(t *testing.T) {
	s, _, _ := newSvc()
	for i := 0; i < domain.MaxOpenTicketsPerCustomer; i++ {
		create(t, s, "alice")
	}
	_, err := s.Create(context.Background(), CreateInput{CustomerID: "alice", Subject: "One more", Message: "hi"})
	require.Error(t, err)
}

func TestCustomerCannotSeeOthersTickets(t *testing.T) {
	s, _, _ := newSvc()
	tk := create(t, s, "alice")
	_, err := s.CustomerGet(context.Background(), tk.ID, "bob")
	require.ErrorIs(t, err, errors.ErrNotFound)
	_, err = s.CustomerReply(context.Background(), tk.ID, "bob", "hi")
	require.ErrorIs(t, err, errors.ErrNotFound)
}

func TestStaffReply_AnswersAndFlagsCustomerUnread_ThenCustomerReadClearsIt(t *testing.T) {
	s, _, rec := newSvc()
	ctx := context.Background()
	tk := create(t, s, "alice")

	got, err := s.StaffGet(ctx, tk.ID)
	require.NoError(t, err)
	assert.False(t, got.StaffUnread, "opening it marks it read for the team")

	got, err = s.StaffReply(ctx, tk.ID, "agent1", "It ships today.")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusAnswered, got.Status)
	assert.True(t, got.CustomerUnread)
	assert.Equal(t, domain.SenderStaff, got.Messages[1].SenderRole)
	assert.Equal(t, 1, rec.customerNotified)

	n, _ := s.CustomerUnread(ctx, "alice")
	assert.Equal(t, int64(1), n)
	seen, err := s.CustomerGet(ctx, tk.ID, "alice")
	require.NoError(t, err)
	assert.False(t, seen.CustomerUnread)
}

func TestCustomerReply_ReopensClosedConversation(t *testing.T) {
	s, _, _ := newSvc()
	ctx := context.Background()
	tk := create(t, s, "alice")
	_, err := s.CustomerClose(ctx, tk.ID, "alice")
	require.NoError(t, err)

	got, err := s.CustomerReply(ctx, tk.ID, "alice", "Actually one more question")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusOpen, got.Status)
	assert.True(t, got.StaffUnread)
}

func TestMessageTooLong(t *testing.T) {
	s, _, _ := newSvc()
	tk := create(t, s, "alice")
	long := make([]rune, domain.MaxMessageLength+1)
	for i := range long {
		long[i] = 'a'
	}
	_, err := s.CustomerReply(context.Background(), tk.ID, "alice", string(long))
	require.Error(t, err)
}

func TestStaffSetStatus_RejectsUnknown(t *testing.T) {
	s, _, _ := newSvc()
	tk := create(t, s, "alice")
	_, err := s.StaffSetStatus(context.Background(), tk.ID, "WEIRD")
	require.Error(t, err)
	got, err := s.StaffSetStatus(context.Background(), tk.ID, domain.StatusClosed)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusClosed, got.Status)
}
