package domain

import (
	"testing"
)

func TestModerationStatusConstants(t *testing.T) {
	statuses := []ModerationStatus{ModerationPending, ModerationApproved, ModerationRejected, ModerationHidden}
	expected := []string{"PENDING", "APPROVED", "REJECTED", "HIDDEN"}
	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], s)
		}
	}
}

func TestNewReview(t *testing.T) {
	t.Run("valid review", func(t *testing.T) {
		r, err := NewReview("prod-1", "cust-1", "Great", "Love it", 5, nil, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.ID == "" {
			t.Error("expected non-empty ID")
		}
		if r.ProductID != "prod-1" {
			t.Errorf("expected product_id prod-1, got %s", r.ProductID)
		}
		if r.CustomerID != "cust-1" {
			t.Errorf("expected customer_id cust-1, got %s", r.CustomerID)
		}
		if r.Rating != 5 {
			t.Errorf("expected rating 5, got %d", r.Rating)
		}
		if r.Status != ModerationPending {
			t.Errorf("expected PENDING, got %s", r.Status)
		}
		if !r.VerifiedPurchase {
			t.Error("expected verified_purchase=true")
		}
		if r.CreatedAt.IsZero() {
			t.Error("expected non-zero created_at")
		}
		if r.UpdatedAt.IsZero() {
			t.Error("expected non-zero updated_at")
		}
	})

	t.Run("rating too low", func(t *testing.T) {
		_, err := NewReview("p", "c", "T", "C", 0, nil, false)
		if err != ErrInvalidRating {
			t.Errorf("expected ErrInvalidRating, got %v", err)
		}
	})

	t.Run("rating too high", func(t *testing.T) {
		_, err := NewReview("p", "c", "T", "C", 6, nil, false)
		if err != ErrInvalidRating {
			t.Errorf("expected ErrInvalidRating, got %v", err)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		_, err := NewReview("p", "c", "", "C", 3, nil, false)
		if err != ErrTitleRequired {
			t.Errorf("expected ErrTitleRequired, got %v", err)
		}
	})

	t.Run("with images", func(t *testing.T) {
		images := []ReviewImage{
			{URL: "https://img.com/1.jpg", Alt: "photo", SortOrder: 0},
		}
		r, err := NewReview("p", "c", "T", "C", 4, images, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(r.Images) != 1 {
			t.Errorf("expected 1 image, got %d", len(r.Images))
		}
	})

	t.Run("boundary rating 1", func(t *testing.T) {
		r, err := NewReview("p", "c", "T", "C", 1, nil, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Rating != 1 {
			t.Errorf("expected rating 1, got %d", r.Rating)
		}
	})
}

func TestReviewLifecycle(t *testing.T) {
	r, _ := NewReview("p", "c", "T", "C", 4, nil, false)

	t.Run("approve", func(t *testing.T) {
		r.Approve()
		if r.Status != ModerationApproved {
			t.Errorf("expected APPROVED, got %s", r.Status)
		}
		if !r.IsVisible() {
			t.Error("approved review should be visible")
		}
	})

	t.Run("reject", func(t *testing.T) {
		r.Reject()
		if r.Status != ModerationRejected {
			t.Errorf("expected REJECTED, got %s", r.Status)
		}
		if r.IsVisible() {
			t.Error("rejected review should not be visible")
		}
	})

	t.Run("hide", func(t *testing.T) {
		r.Hide()
		if r.Status != ModerationHidden {
			t.Errorf("expected HIDDEN, got %s", r.Status)
		}
		if r.IsVisible() {
			t.Error("hidden review should not be visible")
		}
	})

	t.Run("pending not visible", func(t *testing.T) {
		r2, _ := NewReview("p", "c", "T", "C", 3, nil, false)
		if r2.IsVisible() {
			t.Error("pending review should not be visible")
		}
	})
}

func TestReviewError(t *testing.T) {
	err := ErrInvalidRating
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
	if err.Error() != "INVALID_RATING: rating must be between 1 and 5" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}
