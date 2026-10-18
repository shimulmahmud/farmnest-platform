package application

import (
        "context"
        "testing"

        "github.com/ecommerce/backend/internal/reviews/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
)

// mockReviewRepo implements domain.ReviewRepository for testing.
type mockReviewRepo struct {
        reviews     map[string]*domain.Review
        duplicateErr error
        createErr    error
        findErr      error
        updateErr    error
        deleteErr    error
        listErr      error
}

func newMockReviewRepo() *mockReviewRepo {
        return &mockReviewRepo{reviews: make(map[string]*domain.Review)}
}

func (m *mockReviewRepo) Create(_ context.Context, r *domain.Review) error {
        if m.createErr != nil {
                return m.createErr
        }
        m.reviews[r.ID] = r
        return nil
}

func (m *mockReviewRepo) FindByID(_ context.Context, id string) (*domain.Review, error) {
        if m.findErr != nil {
                return nil, m.findErr
        }
        r, ok := m.reviews[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return r, nil
}

func (m *mockReviewRepo) Update(_ context.Context, r *domain.Review) error {
        if m.updateErr != nil {
                return m.updateErr
        }
        if _, ok := m.reviews[r.ID]; !ok {
                return errors.ErrNotFound
        }
        m.reviews[r.ID] = r
        return nil
}

func (m *mockReviewRepo) Delete(_ context.Context, id string) error {
        if m.deleteErr != nil {
                return m.deleteErr
        }
        if _, ok := m.reviews[id]; !ok {
                return errors.ErrNotFound
        }
        delete(m.reviews, id)
        return nil
}

func (m *mockReviewRepo) ListByProduct(_ context.Context, productID string, p pagination.Params) (*domain.ReviewListResult, error) {
        if m.listErr != nil {
                return nil, m.listErr
        }
        var filtered []*domain.Review
        for _, r := range m.reviews {
                if r.ProductID == productID && r.Status == domain.ModerationApproved {
                        filtered = append(filtered, r)
                }
        }
        return &domain.ReviewListResult{
                Reviews: filtered,
                Total:   int64(len(filtered)),
                Page:    p.Page,
                PerPage: p.PerPage,
        }, nil
}

func (m *mockReviewRepo) ListByProductAdmin(_ context.Context, productID string, p pagination.Params) (*domain.ReviewListResult, error) {
        if m.listErr != nil {
                return nil, m.listErr
        }
        var filtered []*domain.Review
        for _, r := range m.reviews {
                if r.ProductID == productID {
                        filtered = append(filtered, r)
                }
        }
        return &domain.ReviewListResult{
                Reviews: filtered,
                Total:   int64(len(filtered)),
                Page:    p.Page,
                PerPage: p.PerPage,
        }, nil
}

func (m *mockReviewRepo) ExistsByProductAndCustomer(_ context.Context, productID, customerID string) (bool, error) {
        if m.duplicateErr != nil {
                return false, m.duplicateErr
        }
        for _, r := range m.reviews {
                if r.ProductID == productID && r.CustomerID == customerID {
                        return true, nil
                }
        }
        return false, nil
}

func (m *mockReviewRepo) ListAll(_ context.Context, _ domain.ModerationStatus, p pagination.Params) (*domain.ReviewListResult, error) {
        if m.listErr != nil {
                return nil, m.listErr
        }
        var filtered []*domain.Review
        for _, r := range m.reviews {
                filtered = append(filtered, r)
        }
        return &domain.ReviewListResult{
                Reviews: filtered,
                Total:   int64(len(filtered)),
                Page:    p.Page,
                PerPage: p.PerPage,
        }, nil
}

// mockVPChecker implements domain.VerifiedPurchaseChecker for testing.
type mockVPChecker struct {
        result bool
        err    error
}

func (m *mockVPChecker) IsVerifiedPurchase(_ context.Context, _, _ string) (bool, error) {
        return m.result, m.err
}

func TestCreateReview(t *testing.T) {
        ctx := context.Background()

        t.Run("successful creation with verified purchase", func(t *testing.T) {
                repo := newMockReviewRepo()
                vp := &mockVPChecker{result: true}
                svc := NewService(repo, vp)

                review, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-1",
                        Rating:     5,
                        Title:      "Great product",
                        Comment:    "Really loved it",
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if review.Rating != 5 {
                        t.Errorf("expected rating 5, got %d", review.Rating)
                }
                if !review.VerifiedPurchase {
                        t.Error("expected verified_purchase=true")
                }
                if review.Status != domain.ModerationPending {
                        t.Errorf("expected PENDING status, got %s", review.Status)
                }
                if review.ID == "" {
                        t.Error("expected non-empty ID")
                }
        })

        t.Run("creation without verified purchase", func(t *testing.T) {
                repo := newMockReviewRepo()
                vp := &mockVPChecker{result: false}
                svc := NewService(repo, vp)

                review, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-2",
                        Rating:     3,
                        Title:      "Okay",
                        Comment:    "It was alright",
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if review.VerifiedPurchase {
                        t.Error("expected verified_purchase=false")
                }
        })

        t.Run("creation with nil vp checker", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                review, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-3",
                        Rating:     4,
                        Title:      "Good",
                        Comment:    "Nice product",
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if review.VerifiedPurchase {
                        t.Error("expected verified_purchase=false when no checker")
                }
        })

        t.Run("vp checker error does not block creation", func(t *testing.T) {
                repo := newMockReviewRepo()
                vp := &mockVPChecker{err: errors.New("INTERNAL_ERROR", "db down", 500)}
                svc := NewService(repo, vp)

                review, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-4",
                        Rating:     4,
                        Title:      "Good",
                        Comment:    "Nice",
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if review.VerifiedPurchase {
                        t.Error("expected verified_purchase=false on checker error")
                }
        })

        t.Run("duplicate review rejected", func(t *testing.T) {
                repo := newMockReviewRepo()
                vp := &mockVPChecker{result: true}
                svc := NewService(repo, vp)

                input := CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-5",
                        Rating:     5,
                        Title:      "Amazing",
                        Comment:    "Love it",
                }
                _, err := svc.CreateReview(ctx, input)
                if err != nil {
                        t.Fatalf("first create failed: %v", err)
                }

                _, err = svc.CreateReview(ctx, input)
                if err == nil {
                        t.Fatal("expected duplicate error")
                }
                appErr, ok := errors.IsAppError(err)
                if !ok || appErr.Code != "CONFLICT" {
                        t.Errorf("expected CONFLICT error, got %v", err)
                }
        })

        t.Run("invalid rating rejected", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                _, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-6",
                        Rating:     0,
                        Title:      "Bad",
                        Comment:    "No",
                })
                if err == nil {
                        t.Fatal("expected validation error")
                }
        })

        t.Run("empty title rejected", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                _, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-7",
                        Rating:     3,
                        Title:      "",
                        Comment:    "No title",
                })
                if err == nil {
                        t.Fatal("expected validation error")
                }
        })

        t.Run("rating above 5 rejected", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                _, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-8",
                        Rating:     6,
                        Title:      "Too high",
                        Comment:    "No",
                })
                if err == nil {
                        t.Fatal("expected validation error")
                }
        })

        t.Run("with images", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                images := []domain.ReviewImage{
                        {URL: "https://example.com/img1.jpg", Alt: "photo 1", SortOrder: 0},
                        {URL: "https://example.com/img2.jpg", Alt: "photo 2", SortOrder: 1},
                }
                review, err := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-9",
                        Rating:     4,
                        Title:      "With photos",
                        Comment:    "See images",
                        Images:     images,
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if len(review.Images) != 2 {
                        t.Errorf("expected 2 images, got %d", len(review.Images))
                }
        })
}

func TestGetReview(t *testing.T) {
        ctx := context.Background()

        t.Run("found", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID:  "prod-1",
                        CustomerID: "cust-1",
                        Rating:     5,
                        Title:      "Great",
                        Comment:    "Awesome",
                })

                found, err := svc.GetReview(ctx, created.ID)
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if found.ID != created.ID {
                        t.Errorf("expected ID %s, got %s", created.ID, found.ID)
                }
        })

        t.Run("not found", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                _, err := svc.GetReview(ctx, "nonexistent")
                if err == nil {
                        t.Fatal("expected not found error")
                }
        })
}

func TestListReviewsByProduct(t *testing.T) {
        ctx := context.Background()

        t.Run("only approved reviews returned", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                // Create 3 reviews, approve only 1
                r1, _ := svc.CreateReview(ctx, CreateReviewInput{ProductID: "prod-1", CustomerID: "c1", Rating: 5, Title: "R1", Comment: "C1"})
                r2, _ := svc.CreateReview(ctx, CreateReviewInput{ProductID: "prod-1", CustomerID: "c2", Rating: 4, Title: "R2", Comment: "C2"})
                _, _ = svc.CreateReview(ctx, CreateReviewInput{ProductID: "prod-2", CustomerID: "c3", Rating: 3, Title: "R3", Comment: "C3"})

                r1.Approve()
                repo.Update(ctx, r1)

                result, err := svc.ListReviewsByProduct(ctx, "prod-1", pagination.Params{Page: 1, PerPage: 10})
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if result.Total != 1 {
                        t.Errorf("expected 1 approved review, got %d", result.Total)
                }
                // r2 should not appear since it's still PENDING
                for _, r := range result.Reviews {
                        if r.ID == r2.ID {
                                t.Error("pending review should not appear in public listing")
                        }
                }
        })

        t.Run("empty result", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                result, err := svc.ListReviewsByProduct(ctx, "nonexistent-prod", pagination.Params{Page: 1, PerPage: 10})
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if result.Total != 0 {
                        t.Errorf("expected 0 reviews, got %d", result.Total)
                }
        })
}

func TestUpdateReview(t *testing.T) {
        ctx := context.Background()

        t.Run("author can update pending review", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 3, Title: "Old", Comment: "Old comment",
                })

                newTitle := "Updated title"
                newRating := 5
                updated, err := svc.UpdateReview(ctx, created.ID, "cust-1", UpdateReviewInput{
                        Rating: &newRating,
                        Title:  &newTitle,
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if updated.Title != newTitle {
                        t.Errorf("expected title %q, got %q", newTitle, updated.Title)
                }
                if updated.Rating != 5 {
                        t.Errorf("expected rating 5, got %d", updated.Rating)
                }
                // PENDING stays PENDING
                if updated.Status != domain.ModerationPending {
                        t.Errorf("expected PENDING, got %s", updated.Status)
                }
        })

        t.Run("approved review resets to pending after edit", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 3, Title: "Old", Comment: "C",
                })
                created.Approve()
                repo.Update(ctx, created)

                newTitle := "Edited"
                updated, err := svc.UpdateReview(ctx, created.ID, "cust-1", UpdateReviewInput{
                        Title: &newTitle,
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if updated.Status != domain.ModerationPending {
                        t.Errorf("expected PENDING after edit, got %s", updated.Status)
                }
        })

        t.Run("non-author cannot update", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 5, Title: "Mine", Comment: "C",
                })

                _, err := svc.UpdateReview(ctx, created.ID, "cust-other", UpdateReviewInput{
                        Title: ptrStr("Hack"),
                })
                if err == nil {
                        t.Fatal("expected forbidden error")
                }
                appErr, ok := errors.IsAppError(err)
                if !ok || appErr.Code != "FORBIDDEN" {
                        t.Errorf("expected FORBIDDEN, got %v", err)
                }
        })

        t.Run("rejected review cannot be updated", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 3, Title: "R", Comment: "C",
                })
                created.Reject()
                repo.Update(ctx, created)

                _, err := svc.UpdateReview(ctx, created.ID, "cust-1", UpdateReviewInput{
                        Title: ptrStr("Updated"),
                })
                if err == nil {
                        t.Fatal("expected validation error for rejected review")
                }
        })

        t.Run("hidden review cannot be updated", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 3, Title: "R", Comment: "C",
                })
                created.Hide()
                repo.Update(ctx, created)

                _, err := svc.UpdateReview(ctx, created.ID, "cust-1", UpdateReviewInput{
                        Title: ptrStr("Updated"),
                })
                if err == nil {
                        t.Fatal("expected validation error for hidden review")
                }
        })

        t.Run("invalid rating on update rejected", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 3, Title: "R", Comment: "C",
                })

                badRating := 0
                _, err := svc.UpdateReview(ctx, created.ID, "cust-1", UpdateReviewInput{
                        Rating: &badRating,
                })
                if err == nil {
                        t.Fatal("expected validation error")
                }
        })

        t.Run("empty title on update rejected", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 3, Title: "R", Comment: "C",
                })

                empty := ""
                _, err := svc.UpdateReview(ctx, created.ID, "cust-1", UpdateReviewInput{
                        Title: &empty,
                })
                if err == nil {
                        t.Fatal("expected validation error")
                }
        })

        t.Run("update images", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 4, Title: "R", Comment: "C",
                })

                images := []domain.ReviewImage{{URL: "https://img.com/1.jpg", SortOrder: 0}}
                updated, err := svc.UpdateReview(ctx, created.ID, "cust-1", UpdateReviewInput{
                        Images: &images,
                })
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
                if len(updated.Images) != 1 {
                        t.Errorf("expected 1 image, got %d", len(updated.Images))
                }
        })

        t.Run("nonexistent review returns not found", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                _, err := svc.UpdateReview(ctx, "nonexistent", "cust-1", UpdateReviewInput{
                        Title: ptrStr("X"),
                })
                if err == nil {
                        t.Fatal("expected not found error")
                }
        })
}

func TestDeleteReview(t *testing.T) {
        ctx := context.Background()

        t.Run("author can delete own review", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 5, Title: "R", Comment: "C",
                })

                err := svc.DeleteReview(ctx, created.ID, "cust-1", false)
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }

                // Verify deleted
                _, err = svc.GetReview(ctx, created.ID)
                if err == nil {
                        t.Fatal("expected not found after delete")
                }
        })

        t.Run("admin can delete any review", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 5, Title: "R", Comment: "C",
                })

                err := svc.DeleteReview(ctx, created.ID, "admin-user", true)
                if err != nil {
                        t.Fatalf("unexpected error: %v", err)
                }
        })

        t.Run("non-author non-admin cannot delete", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                created, _ := svc.CreateReview(ctx, CreateReviewInput{
                        ProductID: "prod-1", CustomerID: "cust-1", Rating: 5, Title: "R", Comment: "C",
                })

                err := svc.DeleteReview(ctx, created.ID, "other-user", false)
                if err == nil {
                        t.Fatal("expected forbidden error")
                }
                appErr, ok := errors.IsAppError(err)
                if !ok || appErr.Code != "FORBIDDEN" {
                        t.Errorf("expected FORBIDDEN, got %v", err)
                }
        })

        t.Run("delete nonexistent review", func(t *testing.T) {
                repo := newMockReviewRepo()
                svc := NewService(repo, nil)

                err := svc.DeleteReview(ctx, "nonexistent", "cust-1", true)
                if err == nil {
                        t.Fatal("expected not found error")
                }
        })
}

func ptrStr(s string) *string { return &s }
func ptrInt(i int) *int    { return &i }

// Ensure unused helpers don't cause compile errors.
var _ = ptrInt
