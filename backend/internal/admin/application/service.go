package application

import (
        "context"

        "fmt"

        "github.com/ecommerce/backend/internal/admin/domain"
        custDomain "github.com/ecommerce/backend/internal/customers/domain"
        revDomain "github.com/ecommerce/backend/internal/reviews/domain"
        userDomain "github.com/ecommerce/backend/internal/users/domain"
        "github.com/ecommerce/backend/pkg/audit"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
)

// Dependencies holds all repository references the admin service needs.
type Dependencies struct {
        UserRepo       userDomain.UserRepository
        CustomerRepo   custDomain.CustomerRepository
        ReviewRepo     revDomain.ReviewRepository
        AuditListRepo  audit.Repository
        Auditor        audit.Auditor
}

// AuditContext carries request-scoped metadata for audit log entries.
type AuditContext struct {
        ActorID   string
        ActorRole string
        IPAddress string
        UserAgent string
        RequestID string
}

// Service provides admin operations: user management, customer management,
// review moderation, audit logging, and dashboard statistics.
type Service struct {
        deps Dependencies
}

// NewService creates an admin service.
func NewService(deps Dependencies) *Service {
        return &Service{deps: deps}
}

// recordAudit creates and persists an audit log entry via the shared auditor.
func (s *Service) recordAudit(ac *AuditContext, action audit.Action, resourceType, resourceID, oldValue, newValue, details string) {
        if s.deps.Auditor == nil {
                return
        }
        meta := &audit.RequestMetadata{
                IPAddress: ac.IPAddress,
                RequestID: ac.RequestID,
        }
        s.deps.Auditor.RecordWithMetadata(meta, ac.ActorID, action, resourceType, resourceID, oldValue, newValue, details)
}

// --- User Management ---

// ListUsers returns a paginated list of users with optional filters.
func (s *Service) ListUsers(ctx context.Context, filter *userDomain.UserFilter, p pagination.Params) ([]*userDomain.User, int64, error) {
        return s.deps.UserRepo.List(ctx, filter, p)
}

// GetUser retrieves a single user by ID.
func (s *Service) GetUser(ctx context.Context, id string) (*userDomain.User, error) {
        return s.deps.UserRepo.FindByID(ctx, id)
}

// UpdateUserRole changes a user's role.
// Prevents self-demotion of super admins and creates an audit record.
func (s *Service) UpdateUserRole(ctx context.Context, ac *AuditContext, targetID string, newRole userDomain.Role) error {
        user, err := s.deps.UserRepo.FindByID(ctx, targetID)
        if err != nil {
                return err
        }

        if ac.ActorID == targetID {
                return errors.New("FORBIDDEN", "cannot change your own role", 403)
        }

        if user.Role == userDomain.RoleSuperAdmin && newRole != userDomain.RoleSuperAdmin {
                count, err := s.deps.UserRepo.Count(ctx, &userDomain.UserFilter{Role: userDomain.RoleSuperAdmin})
                if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to count super admins", 500, err)
                }
                if count <= 1 {
                        return errors.New("FORBIDDEN", "cannot remove the last super admin", 403)
                }
        }

        switch newRole {
        case userDomain.RoleSuperAdmin, userDomain.RoleAdmin, userDomain.RoleManager,
                userDomain.RoleSupport, userDomain.RoleCustomer, userDomain.RoleVendor:
        default:
                return errors.New("VALIDATION_ERROR", "invalid role: "+string(newRole), 422)
        }

        if err := s.deps.UserRepo.UpdateRole(ctx, targetID, newRole); err != nil {
                return err
        }

        oldRole := string(user.Role)
        newRoleStr := string(newRole)
        s.recordAudit(ac, audit.ActionUserRoleUpdate, "user", targetID, oldRole, newRoleStr, "role changed")

        // Audit: record permission changes that result from the role change
        // Since permissions derive from roles, a role change effectively changes permissions.
        oldPerms := domain.RolePermissions[user.Role]
        newPerms := domain.RolePermissions[newRole]
        s.recordAudit(ac, audit.ActionPermissionChange, "user", targetID, formatPerms(oldPerms), formatPerms(newPerms), fmt.Sprintf("permissions changed via role %s -> %s", oldRole, newRoleStr))

        return nil
}

// UpdateUserStatus changes a user's account status.
// Prevents self-deactivation and protects the last super admin. Creates an audit record.
func (s *Service) UpdateUserStatus(ctx context.Context, ac *AuditContext, targetID string, newStatus userDomain.AccountStatus) error {
        user, err := s.deps.UserRepo.FindByID(ctx, targetID)
        if err != nil {
                return err
        }

        if ac.ActorID == targetID {
                return errors.New("FORBIDDEN", "cannot change your own account status", 403)
        }

        if user.Role == userDomain.RoleSuperAdmin && newStatus != userDomain.AccountStatusActive {
                count, err := s.deps.UserRepo.Count(ctx, &userDomain.UserFilter{Role: userDomain.RoleSuperAdmin})
                if err != nil {
                return errors.Wrap("INTERNAL_ERROR", "failed to count super admins", 500, err)
                }
                if count <= 1 {
                        return errors.New("FORBIDDEN", "cannot deactivate the last super admin", 403)
                }
        }

        switch newStatus {
        case userDomain.AccountStatusActive, userDomain.AccountStatusInactive,
                userDomain.AccountStatusSuspended, userDomain.AccountStatusPendingVerification:
        default:
                return errors.New("VALIDATION_ERROR", "invalid status: "+string(newStatus), 422)
        }

        if err := s.deps.UserRepo.UpdateStatus(ctx, targetID, newStatus); err != nil {
                return err
        }

        oldStatus := string(user.Status)
        newStatusStr := string(newStatus)
        s.recordAudit(ac, audit.ActionUserStatusUpdate, "user", targetID, oldStatus, newStatusStr, "status changed")

        return nil
}

// --- Customer Management ---

// ListCustomers returns a paginated list of all customers.
func (s *Service) ListCustomers(ctx context.Context, p pagination.Params) ([]*custDomain.Customer, int64, error) {
        return s.deps.CustomerRepo.List(ctx, p)
}

// GetCustomer retrieves a customer by ID.
func (s *Service) GetCustomer(ctx context.Context, id string) (*custDomain.Customer, error) {
        return s.deps.CustomerRepo.FindByID(ctx, id)
}

// --- Review Moderation ---

// ListReviews returns a paginated list of all reviews with optional status filter.
func (s *Service) ListReviews(ctx context.Context, status revDomain.ModerationStatus, p pagination.Params) (*revDomain.ReviewListResult, error) {
        return s.deps.ReviewRepo.ListAll(ctx, status, p)
}

// ModerateReview applies an admin moderation action (approve/reject/hide) to a review.
func (s *Service) ModerateReview(ctx context.Context, ac *AuditContext, reviewID string, action *domain.ReviewModerationAction) (*revDomain.Review, error) {
        review, err := s.deps.ReviewRepo.FindByID(ctx, reviewID)
        if err != nil {
                return nil, err
        }

        oldStatus := string(review.Status)

        switch action.Action {
        case "approve":
                review.Approve()
        case "reject":
                review.Reject()
        case "hide":
                review.Hide()
        default:
                return nil, errors.New("VALIDATION_ERROR", "invalid moderation action: "+action.Action, 422)
        }

        if err := s.deps.ReviewRepo.Update(ctx, review); err != nil {
                return nil, err
        }

        auditAction := audit.ActionReviewApprove
        switch action.Action {
        case "reject":
                auditAction = audit.ActionReviewReject
        case "hide":
                auditAction = audit.ActionReviewHide
        }
        newStatus := string(review.Status)
        reason := action.Action + " " + action.Reason
        s.recordAudit(ac, auditAction, "review", reviewID, oldStatus, newStatus, reason)

        return review, nil
}

// --- Audit Log ---

// AuditFilter holds query parameters for listing audit entries.
type AuditFilter struct {
        ActorID      string
        Action       string
        ResourceType string
        ResourceID   string
        RequestID    string
}

// ListAuditLogs returns paginated audit log entries using the shared pkg/audit repository.
// This reads from the same audit_logs collection that the Auditor writes to.
func (s *Service) ListAuditLogs(ctx context.Context, filter *AuditFilter, p pagination.Params) ([]*audit.Entry, int64, error) {
        if s.deps.AuditListRepo == nil {
                return nil, 0, errors.New("INTERNAL_ERROR", "audit repository not configured", 500)
        }
        af := &audit.Filter{
                ActorID:      filter.ActorID,
                Action:       audit.Action(filter.Action),
                ResourceType: filter.ResourceType,
                ResourceID:   filter.ResourceID,
                RequestID:    filter.RequestID,
        }
        return s.deps.AuditListRepo.List(ctx, af, p)
}

// formatPerms returns a count-based summary of a permission set for audit logging.
// Full permission lists are too verbose; the admin can query the RBAC table for details.
func formatPerms(perms map[domain.Permission]bool) string {
        if perms == nil {
                return "(none)"
        }
        return fmt.Sprintf("%d permissions", len(perms))
}

// --- Dashboard ---

// DashboardCounts holds count-only stats for the admin dashboard.
type DashboardCounts struct {
        TotalUsers     int64 `json:"total_users"`
        TotalCustomers int64 `json:"total_customers"`
        PendingReviews int64 `json:"pending_reviews"`
}

// GetDashboardCounts returns high-level platform counts.
func (s *Service) GetDashboardCounts(ctx context.Context) (*DashboardCounts, error) {
        userCount, err := s.deps.UserRepo.Count(ctx, nil)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to count users", 500, err)
        }

        customerCount, err := s.deps.CustomerRepo.Count(ctx)
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to count customers", 500, err)
        }

        pendingResult, err := s.deps.ReviewRepo.ListAll(ctx, revDomain.ModerationPending, pagination.Params{Page: 1, PerPage: 1})
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "failed to count pending reviews", 500, err)
        }

        return &DashboardCounts{
                TotalUsers:     userCount,
                TotalCustomers: customerCount,
                PendingReviews: pendingResult.Total,
        }, nil
}
