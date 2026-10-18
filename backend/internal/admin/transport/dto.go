package transport

import (
        "time"

        "github.com/ecommerce/backend/pkg/audit"
        custDomain "github.com/ecommerce/backend/internal/customers/domain"
        revDomain "github.com/ecommerce/backend/internal/reviews/domain"
        userDomain "github.com/ecommerce/backend/internal/users/domain"
)

const timestampFormat = "2006-01-02T15:04:05Z"

// --- Request DTOs ---

type UpdateRoleRequest struct {
        Role string `json:"role" validate:"required"`
}

type UpdateStatusRequest struct {
        Status string `json:"status" validate:"required"`
}

type ModerateReviewRequest struct {
        Action string `json:"action" validate:"required,oneof=approve reject hide"`
        Reason string `json:"reason,omitempty"`
}

// --- Response DTOs ---

type UserResponse struct {
        ID            string `json:"id"`
        Email         string `json:"email"`
        FirstName     string `json:"first_name"`
        LastName      string `json:"last_name"`
        Role          string `json:"role"`
        Status        string `json:"status"`
        EmailVerified bool   `json:"email_verified"`
        LastLoginAt   string `json:"last_login_at,omitempty"`
        CreatedAt     string `json:"created_at"`
        UpdatedAt     string `json:"updated_at"`
}

type CustomerResponse struct {
        ID           string `json:"id"`
        UserID       string `json:"user_id"`
        Phone        string `json:"phone,omitempty"`
        AvatarURL    string `json:"avatar_url,omitempty"`
        DateOfBirth  string `json:"date_of_birth,omitempty"`
        AddressCount int    `json:"address_count"`
        CreatedAt    string `json:"created_at"`
        UpdatedAt    string `json:"updated_at"`
}

type ReviewModerationResponse struct {
        ID              string `json:"id"`
        ProductID       string `json:"product_id"`
        CustomerID      string `json:"customer_id"`
        AuthorName      string `json:"author_name"`
        Rating          int    `json:"rating"`
        Title           string `json:"title"`
        Comment         string `json:"comment"`
        Status          string `json:"status"`
        VerifiedPurchase bool   `json:"verified_purchase"`
        CreatedAt       string `json:"created_at"`
        UpdatedAt       string `json:"updated_at"`
}

type DashboardCountsResponse struct {
        TotalUsers     int64 `json:"total_users"`
        TotalCustomers int64 `json:"total_customers"`
        PendingReviews int64 `json:"pending_reviews"`
}

// AuditEntryResponse is the API response for an audit log entry.
// It maps from pkg/audit.Entry (the append-only audit record).
type AuditEntryResponse struct {
        ID           string `json:"id"`
        ActorID      string `json:"actor_id"`
        Action       string `json:"action"`
        ResourceType string `json:"resource_type"`
        ResourceID   string `json:"resource_id"`
        OldValue     string `json:"old_value,omitempty"`
        NewValue     string `json:"new_value,omitempty"`
        Details      string `json:"details,omitempty"`
        IPAddress    string `json:"ip_address"`
        RequestID    string `json:"request_id,omitempty"`
        CreatedAt    string `json:"created_at"`
}

// --- Mappers ---

func toUserResponse(u *userDomain.User) UserResponse {
        r := UserResponse{
                ID:            u.ID,
                Email:         u.Email,
                FirstName:     u.FirstName,
                LastName:      u.LastName,
                Role:          string(u.Role),
                Status:        string(u.Status),
                EmailVerified: u.EmailVerified,
                CreatedAt:     u.CreatedAt.Format(timestampFormat),
                UpdatedAt:     u.UpdatedAt.Format(timestampFormat),
        }
        if u.LastLoginAt != nil {
                r.LastLoginAt = u.LastLoginAt.Format(timestampFormat)
        }
        return r
}

func toUserResponses(users []*userDomain.User) []UserResponse {
        if len(users) == 0 {
                return []UserResponse{}
        }
        result := make([]UserResponse, len(users))
        for i, u := range users {
                result[i] = toUserResponse(u)
        }
        return result
}

func toCustomerResponse(c *custDomain.Customer) CustomerResponse {
        r := CustomerResponse{
                ID:           c.ID,
                UserID:       c.UserID,
                Phone:        c.Phone,
                AvatarURL:    c.AvatarURL,
                AddressCount: len(c.Addresses),
                CreatedAt:    c.CreatedAt.Format(timestampFormat),
                UpdatedAt:    c.UpdatedAt.Format(timestampFormat),
        }
        if c.DateOfBirth != nil {
                r.DateOfBirth = *c.DateOfBirth
        }
        return r
}

func toCustomerResponses(customers []*custDomain.Customer) []CustomerResponse {
        if len(customers) == 0 {
                return []CustomerResponse{}
        }
        result := make([]CustomerResponse, len(customers))
        for i, c := range customers {
                result[i] = toCustomerResponse(c)
        }
        return result
}

func toReviewModerationResponse(r *revDomain.Review) ReviewModerationResponse {
        return ReviewModerationResponse{
                ID:              r.ID,
                ProductID:       r.ProductID,
                CustomerID:      r.CustomerID,
                AuthorName:      r.AuthorName,
                Rating:          r.Rating,
                Title:           r.Title,
                Comment:         r.Comment,
                Status:          string(r.Status),
                VerifiedPurchase: r.VerifiedPurchase,
                CreatedAt:       r.CreatedAt.Format(timestampFormat),
                UpdatedAt:       r.UpdatedAt.Format(timestampFormat),
        }
}

func toReviewModerationResponses(reviews []*revDomain.Review) []ReviewModerationResponse {
        if len(reviews) == 0 {
                return []ReviewModerationResponse{}
        }
        result := make([]ReviewModerationResponse, len(reviews))
        for i, r := range reviews {
                result[i] = toReviewModerationResponse(r)
        }
        return result
}

func toAuditEntryResponse(e *audit.Entry) AuditEntryResponse {
        return AuditEntryResponse{
                ID:           e.ID,
                ActorID:      e.ActorID,
                Action:       string(e.Action),
                ResourceType: e.ResourceType,
                ResourceID:   e.ResourceID,
                OldValue:     e.OldValue,
                NewValue:     e.NewValue,
                Details:      e.Details,
                IPAddress:    e.IPAddress,
                RequestID:    e.RequestID,
                CreatedAt:    time.Unix(e.CreatedAt, 0).UTC().Format(timestampFormat),
        }
}

func toAuditEntryResponses(entries []*audit.Entry) []AuditEntryResponse {
        if len(entries) == 0 {
                return []AuditEntryResponse{}
        }
        result := make([]AuditEntryResponse, len(entries))
        for i, e := range entries {
                result[i] = toAuditEntryResponse(e)
        }
        return result
}
