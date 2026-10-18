package application

import (
	"context"
	"testing"
)

type fakeNames struct{}

func (fakeNames) DisplayName(_ context.Context, _ string) string { return "Rahim U." }

func TestCreateReview_RequireVerifiedPurchase(t *testing.T) {
	ctx := context.Background()
	in := CreateReviewInput{ProductID: "p1", CustomerID: "c1", Rating: 4, Title: "Good"}

	t.Run("rejects a customer who has not received the product", func(t *testing.T) {
		svc := NewService(newMockReviewRepo(), &mockVPChecker{result: false})
		svc.RequireVerifiedPurchase(true)
		if _, err := svc.CreateReview(ctx, in); err == nil {
			t.Fatal("expected a FORBIDDEN error for an unverified buyer")
		}
	})

	t.Run("accepts a customer with a delivered order, and stores the author name", func(t *testing.T) {
		svc := NewService(newMockReviewRepo(), &mockVPChecker{result: true})
		svc.RequireVerifiedPurchase(true)
		svc.SetAuthorNames(fakeNames{})
		r, err := svc.CreateReview(ctx, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.AuthorName != "Rahim U." {
			t.Errorf("expected author name Rahim U., got %q", r.AuthorName)
		}
		if !r.VerifiedPurchase {
			t.Error("expected verified_purchase=true")
		}
	})

	t.Run("gate is off by default (existing behaviour)", func(t *testing.T) {
		svc := NewService(newMockReviewRepo(), &mockVPChecker{result: false})
		if _, err := svc.CreateReview(ctx, in); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
