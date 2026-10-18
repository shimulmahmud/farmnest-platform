package application

import (
	"context"
	"testing"

	"github.com/ecommerce/backend/internal/wishlist/domain"
	"github.com/ecommerce/backend/pkg/errors"
)

// mockWishlistRepo implements domain.WishlistRepository for testing.
type mockWishlistRepo struct {
	wishlists   map[string]*domain.Wishlist
	createErr   error
	findErr     error
	updateErr   error
	addItemErr  error
	removeErr   error
	existsResult bool
	existsErr   error
}

func newMockWishlistRepo() *mockWishlistRepo {
	return &mockWishlistRepo{wishlists: make(map[string]*domain.Wishlist)}
}

func (m *mockWishlistRepo) FindByCustomer(_ context.Context, customerID string) (*domain.Wishlist, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	w, ok := m.wishlists[customerID]
	if !ok {
		return nil, nil
	}
	return w, nil
}

func (m *mockWishlistRepo) Create(_ context.Context, w *domain.Wishlist) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.wishlists[w.CustomerID] = w
	return nil
}

func (m *mockWishlistRepo) Update(_ context.Context, w *domain.Wishlist) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.wishlists[w.CustomerID]; !ok {
		return errors.ErrNotFound
	}
	m.wishlists[w.CustomerID] = w
	return nil
}

func (m *mockWishlistRepo) AddItem(_ context.Context, customerID string, item domain.WishlistItem) error {
	if m.addItemErr != nil {
		return m.addItemErr
	}
	w, ok := m.wishlists[customerID]
	if !ok {
		w = domain.NewWishlist(customerID)
		m.wishlists[customerID] = w
	}
	added, ok := w.AddItem(item.ProductID, item.VariantID)
	if !ok {
		return domain.ErrDuplicateWishlistItem
	}
	_ = added
	return nil
}

func (m *mockWishlistRepo) RemoveItem(_ context.Context, customerID, itemID string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	w, ok := m.wishlists[customerID]
	if !ok {
		return domain.ErrWishlistItemNotFound
	}
	if !w.RemoveItem(itemID) {
		return domain.ErrWishlistItemNotFound
	}
	return nil
}

func (m *mockWishlistRepo) ExistsByCustomerAndProduct(_ context.Context, customerID, productID, variantID string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	w, ok := m.wishlists[customerID]
	if !ok {
		return false, nil
	}
	return w.HasItem(productID, variantID), nil
}

// mockProductValidator implements domain.ProductValidator.
type mockProductValidator struct {
	productExists  bool
	variantExists  bool
	productErr     error
	variantErr     error
}

func (m *mockProductValidator) ProductExists(_ context.Context, _ string) (bool, error) {
	return m.productExists, m.productErr
}

func (m *mockProductValidator) VariantExists(_ context.Context, _ string) (bool, error) {
	return m.variantExists, m.variantErr
}

// mockEventPublisher implements domain.EventPublisher.
type mockEventPublisher struct {
	addedEvents   []*domain.WishlistEvent
	removedEvents []*domain.WishlistEvent
	err           error
}

func (m *mockEventPublisher) PublishAdded(_ context.Context, e *domain.WishlistEvent) error {
	m.addedEvents = append(m.addedEvents, e)
	return m.err
}

func (m *mockEventPublisher) PublishRemoved(_ context.Context, e *domain.WishlistEvent) error {
	m.removedEvents = append(m.removedEvents, e)
	return m.err
}

func TestGetWishlist(t *testing.T) {
	ctx := context.Background()

	t.Run("creates empty wishlist on first access", func(t *testing.T) {
		repo := newMockWishlistRepo()
		svc := NewService(repo, nil, nil)

		w, err := svc.GetWishlist(ctx, "cust-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.CustomerID != "cust-1" {
			t.Errorf("expected customer_id cust-1, got %s", w.CustomerID)
		}
		if len(w.Items) != 0 {
			t.Errorf("expected 0 items, got %d", len(w.Items))
		}
	})

	t.Run("returns existing wishlist", func(t *testing.T) {
		repo := newMockWishlistRepo()
		existing := domain.NewWishlist("cust-2")
		existing.AddItem("prod-1", "")
		repo.wishlists["cust-2"] = existing

		svc := NewService(repo, nil, nil)
		w, err := svc.GetWishlist(ctx, "cust-2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ItemCount() != 1 {
			t.Errorf("expected 1 item, got %d", w.ItemCount())
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := newMockWishlistRepo()
		repo.findErr = errors.New("INTERNAL_ERROR", "db down", 500)
		svc := NewService(repo, nil, nil)

		_, err := svc.GetWishlist(ctx, "cust-3")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAddItem(t *testing.T) {
	ctx := context.Background()

	t.Run("successful add with product validation", func(t *testing.T) {
		repo := newMockWishlistRepo()
		validator := &mockProductValidator{productExists: true}
		pub := &mockEventPublisher{}
		svc := NewService(repo, validator, pub)

		item, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ProductID != "prod-1" {
			t.Errorf("expected product_id prod-1, got %s", item.ProductID)
		}
		if item.ID == "" {
			t.Error("expected non-empty item ID")
		}
		if len(pub.addedEvents) != 1 {
			t.Errorf("expected 1 event, got %d", len(pub.addedEvents))
		}
		if pub.addedEvents[0].EventType != "wishlist.added" {
			t.Errorf("expected wishlist.added event, got %s", pub.addedEvents[0].EventType)
		}
	})

	t.Run("successful add with variant", func(t *testing.T) {
		repo := newMockWishlistRepo()
		validator := &mockProductValidator{productExists: true, variantExists: true}
		svc := NewService(repo, validator, nil)

		item, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1", VariantID: "var-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.VariantID != "var-1" {
			t.Errorf("expected variant_id var-1, got %s", item.VariantID)
		}
	})

	t.Run("duplicate item rejected", func(t *testing.T) {
		repo := newMockWishlistRepo()
		validator := &mockProductValidator{productExists: true}
		svc := NewService(repo, validator, nil)

		input := AddItemInput{ProductID: "prod-1"}
		_, err := svc.AddItem(ctx, "cust-1", input)
		if err != nil {
			t.Fatalf("first add failed: %v", err)
		}

		_, err = svc.AddItem(ctx, "cust-1", input)
		if err == nil {
			t.Fatal("expected duplicate error")
		}
		appErr, ok := errors.IsAppError(err)
		if !ok || appErr.Code != "CONFLICT" {
			t.Errorf("expected CONFLICT, got %v", err)
		}
	})

	t.Run("duplicate via DB rejected", func(t *testing.T) {
		repo := newMockWishlistRepo()
		repo.addItemErr = domain.ErrDuplicateWishlistItem
		validator := &mockProductValidator{productExists: true}
		svc := NewService(repo, validator, nil)

		_, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})
		if err == nil {
			t.Fatal("expected duplicate error from DB")
		}
		appErr, ok := errors.IsAppError(err)
		if !ok || appErr.Code != "CONFLICT" {
			t.Errorf("expected CONFLICT, got %v", err)
		}
	})

	t.Run("product not found rejected", func(t *testing.T) {
		repo := newMockWishlistRepo()
		validator := &mockProductValidator{productExists: false}
		svc := NewService(repo, validator, nil)

		_, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "nonexistent"})
		if err == nil {
			t.Fatal("expected validation error")
		}
		appErr, ok := errors.IsAppError(err)
		if !ok || appErr.Code != "VALIDATION_ERROR" {
			t.Errorf("expected VALIDATION_ERROR, got %v", err)
		}
	})

	t.Run("variant not found rejected", func(t *testing.T) {
		repo := newMockWishlistRepo()
		validator := &mockProductValidator{productExists: true, variantExists: false}
		svc := NewService(repo, validator, nil)

		_, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1", VariantID: "bad-var"})
		if err == nil {
			t.Fatal("expected validation error")
		}
		appErr, ok := errors.IsAppError(err)
		if !ok || appErr.Code != "VALIDATION_ERROR" {
			t.Errorf("expected VALIDATION_ERROR, got %v", err)
		}
	})

	t.Run("works without validator", func(t *testing.T) {
		repo := newMockWishlistRepo()
		svc := NewService(repo, nil, nil)

		item, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ProductID != "prod-1" {
			t.Errorf("expected product_id prod-1, got %s", item.ProductID)
		}
	})

	t.Run("works without publisher", func(t *testing.T) {
		repo := newMockWishlistRepo()
		svc := NewService(repo, nil, nil)

		_, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("publisher error does not block", func(t *testing.T) {
		repo := newMockWishlistRepo()
		pub := &mockEventPublisher{err: errors.New("INTERNAL_ERROR", "nats down", 500)}
		svc := NewService(repo, nil, pub)

		_, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})
		if err != nil {
			t.Fatalf("publisher error should not block: %v", err)
		}
	})

	t.Run("validator error propagated", func(t *testing.T) {
		repo := newMockWishlistRepo()
		validator := &mockProductValidator{productErr: errors.New("INTERNAL_ERROR", "db down", 500)}
		svc := NewService(repo, validator, nil)

		_, err := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRemoveItem(t *testing.T) {
	ctx := context.Background()

	t.Run("successful remove", func(t *testing.T) {
		repo := newMockWishlistRepo()
		pub := &mockEventPublisher{}
		svc := NewService(repo, nil, pub)

		// Add item first
		item, _ := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})

		err := svc.RemoveItem(ctx, "cust-1", item.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pub.removedEvents) != 1 {
			t.Errorf("expected 1 removed event, got %d", len(pub.removedEvents))
		}
		if pub.removedEvents[0].EventType != "wishlist.removed" {
			t.Errorf("expected wishlist.removed event, got %s", pub.removedEvents[0].EventType)
		}
		if pub.removedEvents[0].ProductID != "prod-1" {
			t.Errorf("expected product_id prod-1 in event, got %s", pub.removedEvents[0].ProductID)
		}
	})

	t.Run("item not found", func(t *testing.T) {
		repo := newMockWishlistRepo()
		svc := NewService(repo, nil, nil)

		err := svc.RemoveItem(ctx, "cust-1", "nonexistent")
		if err == nil {
			t.Fatal("expected error")
		}
		if err != domain.ErrWishlistItemNotFound {
			t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
		}
	})

	t.Run("wishlist not found", func(t *testing.T) {
		repo := newMockWishlistRepo()
		svc := NewService(repo, nil, nil)

		err := svc.RemoveItem(ctx, "no-wishlist", "item-1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err != domain.ErrWishlistNotFound {
			t.Errorf("expected ErrWishlistNotFound, got %v", err)
		}
	})

	t.Run("publisher error does not block", func(t *testing.T) {
		repo := newMockWishlistRepo()
		pub := &mockEventPublisher{err: errors.New("INTERNAL_ERROR", "nats down", 500)}
		svc := NewService(repo, nil, pub)

		item, _ := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})

		err := svc.RemoveItem(ctx, "cust-1", item.ID)
		if err != nil {
			t.Fatalf("publisher error should not block: %v", err)
		}
	})

	t.Run("remove then verify empty", func(t *testing.T) {
		repo := newMockWishlistRepo()
		svc := NewService(repo, nil, nil)

		item, _ := svc.AddItem(ctx, "cust-1", AddItemInput{ProductID: "prod-1"})
		_ = svc.RemoveItem(ctx, "cust-1", item.ID)

		w, _ := svc.GetWishlist(ctx, "cust-1")
		if w.ItemCount() != 0 {
			t.Errorf("expected 0 items after remove, got %d", w.ItemCount())
		}
	})
}

func TestDomainWishlist(t *testing.T) {
	t.Run("NewWishlist", func(t *testing.T) {
		w := domain.NewWishlist("cust-1")
		if w.ID == "" {
			t.Error("expected non-empty ID")
		}
		if w.CustomerID != "cust-1" {
			t.Errorf("expected customer_id cust-1, got %s", w.CustomerID)
		}
		if len(w.Items) != 0 {
			t.Errorf("expected 0 items, got %d", len(w.Items))
		}
	})

	t.Run("AddItem and HasItem", func(t *testing.T) {
		w := domain.NewWishlist("cust-1")
		item, ok := w.AddItem("prod-1", "")
		if !ok {
			t.Fatal("expected add to succeed")
		}
		if item.ProductID != "prod-1" {
			t.Errorf("expected product_id prod-1")
		}
		if !w.HasItem("prod-1", "") {
			t.Error("expected HasItem to return true")
		}
	})

	t.Run("duplicate AddItem returns false", func(t *testing.T) {
		w := domain.NewWishlist("cust-1")
		w.AddItem("prod-1", "")
		_, ok := w.AddItem("prod-1", "")
		if ok {
			t.Error("expected duplicate add to fail")
		}
	})

	t.Run("variant differentiation", func(t *testing.T) {
		w := domain.NewWishlist("cust-1")
		w.AddItem("prod-1", "var-1")
		ok := w.HasItem("prod-1", "var-1")
		if !ok {
			t.Error("expected to find prod-1/var-1")
		}
		ok = w.HasItem("prod-1", "var-2")
		if ok {
			t.Error("expected not to find prod-1/var-2")
		}
		// Same product, different variant — should succeed
		_, added := w.AddItem("prod-1", "var-2")
		if !added {
			t.Error("expected adding different variant to succeed")
		}
	})

	t.Run("RemoveItem", func(t *testing.T) {
		w := domain.NewWishlist("cust-1")
		item, _ := w.AddItem("prod-1", "")
		ok := w.RemoveItem(item.ID)
		if !ok {
			t.Fatal("expected remove to succeed")
		}
		if w.ItemCount() != 0 {
			t.Errorf("expected 0 items, got %d", w.ItemCount())
		}
		// Remove non-existent
		ok = w.RemoveItem("nonexistent")
		if ok {
			t.Error("expected remove of nonexistent to fail")
		}
	})

	t.Run("FindItemByID", func(t *testing.T) {
		w := domain.NewWishlist("cust-1")
		item, _ := w.AddItem("prod-1", "var-1")
		found := w.FindItemByID(item.ID)
		if found == nil {
			t.Fatal("expected to find item")
		}
		if found.ProductID != "prod-1" {
			t.Errorf("expected product_id prod-1, got %s", found.ProductID)
		}
		nilResult := w.FindItemByID("nonexistent")
		if nilResult != nil {
			t.Error("expected nil for nonexistent item")
		}
	})

	t.Run("domain errors", func(t *testing.T) {
		err := domain.ErrDuplicateWishlistItem
		if err.Error() != "DUPLICATE_ITEM: item already in wishlist" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
		err = domain.ErrWishlistNotFound
		if err.Error() != "WISHLIST_NOT_FOUND: wishlist not found" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})
}
