package domain

// DashboardStats holds aggregated platform statistics for the admin dashboard.
type DashboardStats struct {
	TotalUsers      int64 `json:"total_users"`
	TotalCustomers  int64 `json:"total_customers"`
	TotalProducts   int64 `json:"total_products"`
	TotalOrders     int64 `json:"total_orders"`
	TotalRevenue    int64 `json:"total_revenue_cents"`
	PendingReviews  int64 `json:"pending_reviews"`
}

// UserRoleUpdate represents a request to change a user's role.
type UserRoleUpdate struct {
	Role string `json:"role" validate:"required"`
}

// UserStatusUpdate represents a request to change a user's account status.
type UserStatusUpdate struct {
	Status string `json:"status" validate:"required"`
}

// ReviewModerationAction represents a moderation action on a review.
type ReviewModerationAction struct {
	Action  string `json:"action" validate:"required,oneof=approve reject hide"`
	Reason string `json:"reason,omitempty"`
}
