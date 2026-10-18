package application

import (
        "context"
        "testing"

        "github.com/ecommerce/backend/internal/admin/domain"
        custDomain "github.com/ecommerce/backend/internal/customers/domain"
        revDomain "github.com/ecommerce/backend/internal/reviews/domain"
        userDomain "github.com/ecommerce/backend/internal/users/domain"
        "github.com/ecommerce/backend/pkg/audit"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
)

// --- Mocks ---

type mockUserRepo struct {
        users        []*userDomain.User
        findByID     func(ctx context.Context, id string) (*userDomain.User, error)
        updateRole   func(ctx context.Context, userID string, role userDomain.Role) error
        updateStatus func(ctx context.Context, userID string, status userDomain.AccountStatus) error
}

func (m *mockUserRepo) Create(_ context.Context, _ *userDomain.User) error { return nil }
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*userDomain.User, error) {
        if m.findByID != nil {
                return m.findByID(ctx, id)
        }
        for _, u := range m.users {
                if u.ID == id {
                        return u, nil
                }
        }
        return nil, errors.ErrNotFound
}
func (m *mockUserRepo) FindByEmail(_ context.Context, _ string) (*userDomain.User, error) { return nil, errors.ErrNotFound }
func (m *mockUserRepo) Update(_ context.Context, _ *userDomain.User) error            { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error)  { return false, nil }
func (m *mockUserRepo) List(_ context.Context, _ *userDomain.UserFilter, p pagination.Params) ([]*userDomain.User, int64, error) {
        return m.users, int64(len(m.users)), nil
}
func (m *mockUserRepo) Count(_ context.Context, _ *userDomain.UserFilter) (int64, error) {
        if len(m.users) == 0 {
                return 2, nil
        }
        return int64(len(m.users)), nil
}
func (m *mockUserRepo) UpdateRole(ctx context.Context, userID string, role userDomain.Role) error {
        if m.updateRole != nil {
                return m.updateRole(ctx, userID, role)
        }
        return nil
}
func (m *mockUserRepo) UpdateStatus(ctx context.Context, userID string, status userDomain.AccountStatus) error {
        if m.updateStatus != nil {
                return m.updateStatus(ctx, userID, status)
        }
        return nil
}

type mockCustomerRepo struct {
        customers []*custDomain.Customer
}

func (m *mockCustomerRepo) Create(_ context.Context, _ *custDomain.Customer) error { return nil }
func (m *mockCustomerRepo) FindByUserID(_ context.Context, _ string) (*custDomain.Customer, error) {
        return nil, errors.ErrNotFound
}
func (m *mockCustomerRepo) FindByID(_ context.Context, id string) (*custDomain.Customer, error) {
        for _, c := range m.customers {
                if c.ID == id {
                        return c, nil
                }
        }
        return nil, errors.ErrNotFound
}
func (m *mockCustomerRepo) Update(_ context.Context, _ *custDomain.Customer) error { return nil }
func (m *mockCustomerRepo) List(_ context.Context, p pagination.Params) ([]*custDomain.Customer, int64, error) {
        return m.customers, int64(len(m.customers)), nil
}
func (m *mockCustomerRepo) Count(_ context.Context) (int64, error) {
        return int64(len(m.customers)), nil
}

type mockReviewRepo struct {
        listAllResult *revDomain.ReviewListResult
}

func (m *mockReviewRepo) Create(_ context.Context, _ *revDomain.Review) error { return nil }
func (m *mockReviewRepo) FindByID(_ context.Context, _ string) (*revDomain.Review, error) {
        return nil, errors.ErrNotFound
}
func (m *mockReviewRepo) Update(_ context.Context, _ *revDomain.Review) error { return nil }
func (m *mockReviewRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockReviewRepo) ListByProduct(_ context.Context, _ string, _ pagination.Params) (*revDomain.ReviewListResult, error) {
        return &revDomain.ReviewListResult{}, nil
}
func (m *mockReviewRepo) ListByProductAdmin(_ context.Context, _ string, _ pagination.Params) (*revDomain.ReviewListResult, error) {
        return &revDomain.ReviewListResult{}, nil
}
func (m *mockReviewRepo) ListAll(_ context.Context, _ revDomain.ModerationStatus, _ pagination.Params) (*revDomain.ReviewListResult, error) {
        if m.listAllResult != nil {
                return m.listAllResult, nil
        }
        return &revDomain.ReviewListResult{Total: 3}, nil
}
func (m *mockReviewRepo) ExistsByProductAndCustomer(_ context.Context, _, _ string) (bool, error) {
        return false, nil
}

// mockAuditListRepo implements audit.Repository for testing.
type mockAuditListRepo struct {
        entries []*audit.Entry
}

func (m *mockAuditListRepo) Insert(_ context.Context, _ *audit.Entry) error { return nil }
func (m *mockAuditListRepo) List(_ context.Context, _ *audit.Filter, _ pagination.Params) ([]*audit.Entry, int64, error) {
        result := make([]*audit.Entry, len(m.entries))
        for i := range m.entries {
                result[i] = m.entries[i]
        }
        return result, int64(len(m.entries)), nil
}

func testDeps() Dependencies {
        return Dependencies{
                UserRepo:      &mockUserRepo{},
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: &mockAuditListRepo{},
                Auditor:       audit.NopAuditor,
        }
}

var testAC = func() *AuditContext {
        return &AuditContext{ActorID: "admin-1", ActorRole: "SUPER_ADMIN", IPAddress: "10.0.0.1", UserAgent: "test", RequestID: "req-1"}
}

// --- Tests ---

func TestListUsers(t *testing.T) {
        users := []*userDomain.User{
                {ID: "u1", Email: "a@test.com", Role: userDomain.RoleCustomer, Status: userDomain.AccountStatusActive},
                {ID: "u2", Email: "b@test.com", Role: userDomain.RoleAdmin, Status: userDomain.AccountStatusActive},
        }
        svc := NewService(Dependencies{
                UserRepo:      &mockUserRepo{users: users},
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: &mockAuditListRepo{},
                Auditor:       audit.NopAuditor,
        })

        result, total, err := svc.ListUsers(context.Background(), nil, pagination.Params{Page: 1, PerPage: 10})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if len(result) != 2 {
                t.Errorf("expected 2 users, got %d", len(result))
        }
        if total != 2 {
                t.Errorf("expected total 2, got %d", total)
        }
}

func TestUpdateUserRole_PreventsSelfChange(t *testing.T) {
        svc := NewService(testDeps())

        err := svc.UpdateUserRole(context.Background(), testAC(), "admin-1", userDomain.RoleAdmin)
        if err == nil {
                t.Fatal("expected error for self role change")
        }
        appErr, ok := errors.IsAppError(err)
        if !ok || appErr.Status != 403 {
                t.Errorf("expected 403, got %v", err)
        }
}

func TestUpdateUserRole_InvalidRole(t *testing.T) {
        repo := &mockUserRepo{users: []*userDomain.User{{ID: "u1", Role: userDomain.RoleCustomer}}}
        svc := NewService(Dependencies{
                UserRepo:      repo,
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: &mockAuditListRepo{},
                Auditor:       audit.NopAuditor,
        })

        err := svc.UpdateUserRole(context.Background(), testAC(), "u1", userDomain.Role("INVALID"))
        if err == nil {
                t.Fatal("expected error for invalid role")
        }
        appErr, ok := errors.IsAppError(err)
        if !ok || appErr.Status != 422 {
                t.Errorf("expected 422, got %v", err)
        }
}

func TestUpdateUserStatus_PreventsSelfChange(t *testing.T) {
        svc := NewService(testDeps())

        err := svc.UpdateUserStatus(context.Background(), testAC(), "admin-1", userDomain.AccountStatusSuspended)
        if err == nil {
                t.Fatal("expected error for self status change")
        }
}

func TestUpdateUserStatus_InvalidStatus(t *testing.T) {
        repo := &mockUserRepo{users: []*userDomain.User{{ID: "u1", Role: userDomain.RoleCustomer}}}
        svc := NewService(Dependencies{
                UserRepo:      repo,
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: &mockAuditListRepo{},
                Auditor:       audit.NopAuditor,
        })

        err := svc.UpdateUserStatus(context.Background(), testAC(), "u1", userDomain.AccountStatus("LOCKED"))
        if err == nil {
                t.Fatal("expected error for invalid status")
        }
        appErr, ok := errors.IsAppError(err)
        if !ok || appErr.Status != 422 {
                t.Errorf("expected 422, got %v", err)
        }
}

func TestModerateReview_InvalidAction(t *testing.T) {
        repo := &mockReviewRepo{
                listAllResult: &revDomain.ReviewListResult{Total: 0},
        }
        svc := NewService(Dependencies{
                UserRepo:      &mockUserRepo{},
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    repo,
                AuditListRepo: &mockAuditListRepo{},
                Auditor:       audit.NopAuditor,
        })

        _, err := svc.ModerateReview(context.Background(), testAC(), "rev-1", &domain.ReviewModerationAction{Action: "delete"})
        if err == nil {
                t.Fatal("expected error for invalid action")
        }
}

func TestGetDashboardCounts(t *testing.T) {
        svc := NewService(Dependencies{
                UserRepo:      &mockUserRepo{users: []*userDomain.User{{ID: "u1"}}},
                CustomerRepo:  &mockCustomerRepo{customers: []*custDomain.Customer{{ID: "c1"}}},
                ReviewRepo:    &mockReviewRepo{listAllResult: &revDomain.ReviewListResult{Total: 5}},
                AuditListRepo: &mockAuditListRepo{},
                Auditor:       audit.NopAuditor,
        })

        counts, err := svc.GetDashboardCounts(context.Background())
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if counts.TotalUsers != 1 {
                t.Errorf("expected 1 user, got %d", counts.TotalUsers)
        }
        if counts.TotalCustomers != 1 {
                t.Errorf("expected 1 customer, got %d", counts.TotalCustomers)
        }
        if counts.PendingReviews != 5 {
                t.Errorf("expected 5 pending reviews, got %d", counts.PendingReviews)
        }
}

func TestListAuditLogs_NilRepo(t *testing.T) {
        svc := NewService(Dependencies{
                UserRepo:      &mockUserRepo{},
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: nil,
                Auditor:       audit.NopAuditor,
        })

        _, _, err := svc.ListAuditLogs(context.Background(), nil, pagination.Params{Page: 1, PerPage: 10})
        if err == nil {
                t.Fatal("expected error when audit repo is nil")
        }
}

func TestListAuditLogs_Success(t *testing.T) {
        alRepo := &mockAuditListRepo{
                entries: []*audit.Entry{
                        {ID: "a1", Action: audit.ActionUserRoleUpdate, ActorID: "admin-1", OldValue: "CUSTOMER", NewValue: "ADMIN"},
                },
        }
        svc := NewService(Dependencies{
                UserRepo:      &mockUserRepo{},
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: alRepo,
                Auditor:       audit.NopAuditor,
        })

        entries, total, err := svc.ListAuditLogs(context.Background(), &AuditFilter{Action: "user.role.update"}, pagination.Params{Page: 1, PerPage: 10})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if total != 1 {
                t.Errorf("expected total 1, got %d", total)
        }
        if len(entries) != 1 || entries[0].ID != "a1" {
                t.Errorf("expected audit entry a1, got %v", entries)
        }
        if entries[0].OldValue != "CUSTOMER" {
                t.Errorf("expected old_value CUSTOMER, got %s", entries[0].OldValue)
        }
        if entries[0].NewValue != "ADMIN" {
                t.Errorf("expected new_value ADMIN, got %s", entries[0].NewValue)
        }
}

func TestListAuditLogs_FilterByActor(t *testing.T) {
        alRepo := &mockAuditListRepo{
                entries: []*audit.Entry{
                        {ID: "a1", ActorID: "admin-1", Action: audit.ActionProductCreate},
                        {ID: "a2", ActorID: "admin-2", Action: audit.ActionCategoryCreate},
                },
        }
        svc := NewService(Dependencies{
                UserRepo:      &mockUserRepo{},
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: alRepo,
                Auditor:       audit.NopAuditor,
        })

        entries, _, err := svc.ListAuditLogs(context.Background(), &AuditFilter{ActorID: "admin-1"}, pagination.Params{Page: 1, PerPage: 10})
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        // Note: our mock doesn't actually filter; this tests the plumbing works
        if len(entries) != 2 {
                t.Errorf("expected 2 entries from mock, got %d", len(entries))
        }
}

func TestFormatPerms_NonNil(t *testing.T) {
        perms := domain.RolePermissions[userDomain.RoleSuperAdmin]
        got := formatPerms(perms)
        if got == "" {
                t.Error("formatPerms returned empty string for non-nil map")
        }
        if got == "(none)" {
                t.Error("formatPerms should not return (none) for populated map")
        }
}

func TestFormatPerms_Nil(t *testing.T) {
        got := formatPerms(nil)
        if got != "(none)" {
                t.Errorf("expected (none) for nil map, got %s", got)
        }
}

func TestUpdateUserRole_RecordsPermissionChangeAudit(t *testing.T) {
        var recordedActions []audit.Action
        mockAuditor := &mockRecorderAuditor{onRecord: func(a audit.Action) {
                recordedActions = append(recordedActions, a)
        }}

        repo := &mockUserRepo{
                users: []*userDomain.User{{ID: "u1", Role: userDomain.RoleSupport}},
        }
        svc := NewService(Dependencies{
                UserRepo:      repo,
                CustomerRepo:  &mockCustomerRepo{},
                ReviewRepo:    &mockReviewRepo{},
                AuditListRepo: &mockAuditListRepo{},
                Auditor:       mockAuditor,
        })

        err := svc.UpdateUserRole(context.Background(), testAC(), "u1", userDomain.RoleAdmin)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if len(recordedActions) != 2 {
                t.Fatalf("expected 2 audit actions (role + permission), got %d", len(recordedActions))
        }
        if recordedActions[0] != audit.ActionUserRoleUpdate {
                t.Errorf("expected first action user.role.update, got %s", recordedActions[0])
        }
        if recordedActions[1] != audit.ActionPermissionChange {
                t.Errorf("expected second action permission.change, got %s", recordedActions[1])
        }
}

// mockRecorderAuditor captures action types for test assertions.
type mockRecorderAuditor struct {
        onRecord func(a audit.Action)
}

func (m *mockRecorderAuditor) Record(_, _ string, _ audit.Action, _, _, _, _ string) {
        // not used in permission change test (uses RecordWithMetadata via recordAudit)
}
func (m *mockRecorderAuditor) RecordWithMetadata(_ *audit.RequestMetadata, _ string, action audit.Action, _, _, _, _, _ string) {
        if m.onRecord != nil {
                m.onRecord(action)
        }
}
func (m *mockRecorderAuditor) Stop() {}
