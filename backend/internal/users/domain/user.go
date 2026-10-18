package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents user authorization levels.
type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleManager    Role = "MANAGER"
	RoleSupport    Role = "SUPPORT"
	RoleCustomer   Role = "CUSTOMER"
	RoleVendor     Role = "VENDOR"
)

// AccountStatus represents the state of a user account.
type AccountStatus string

const (
	AccountStatusActive              AccountStatus = "ACTIVE"
	AccountStatusInactive            AccountStatus = "INACTIVE"
	AccountStatusSuspended           AccountStatus = "SUSPENDED"
	AccountStatusPendingVerification AccountStatus = "PENDING_VERIFICATION"
	AccountStatusLocked              AccountStatus = "LOCKED"
)

// User represents a system user.
type User struct {
	ID             string        `bson:"_id" json:"id"`
	Email          string        `bson:"email" json:"email"`
	PasswordHash   string        `bson:"password_hash" json:"-"`
	FirstName      string        `bson:"first_name" json:"first_name"`
	LastName       string        `bson:"last_name" json:"last_name"`
	Role           Role          `bson:"role" json:"role"`
	Status         AccountStatus `bson:"status" json:"status"`
	EmailVerified  bool          `bson:"email_verified" json:"email_verified"`
	FailedAttempts int           `bson:"failed_attempts" json:"-"`
	LockedUntil    *time.Time    `bson:"locked_until" json:"-"`
	LastLoginAt    *time.Time    `bson:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt      time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time     `bson:"updated_at" json:"updated_at"`
}

// NewUser creates a new User with generated ID and timestamps.
func NewUser(email, passwordHash, firstName, lastName string, role Role) *User {
	now := time.Now().UTC()
	return &User{
		ID:            uuid.New().String(),
		Email:         email,
		PasswordHash:  passwordHash,
		FirstName:     firstName,
		LastName:      lastName,
		Role:          role,
		Status:        AccountStatusPendingVerification,
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// IsActive returns true if the account can be used for authentication.
func (u *User) IsActive() bool {
	switch u.Status {
	case AccountStatusActive, AccountStatusPendingVerification:
		// Both active and pending-verification users can authenticate
	default:
		return false
	}
	if u.LockedUntil != nil && u.LockedUntil.After(time.Now().UTC()) {
		return false
	}
	return true
}

// RecordFailedAttempt increments the failed login counter.
func (u *User) RecordFailedAttempt(maxAttempts int, lockDuration time.Duration) {
	u.FailedAttempts++
	u.UpdatedAt = time.Now().UTC()
	if u.FailedAttempts >= maxAttempts {
		t := time.Now().UTC().Add(lockDuration)
		u.LockedUntil = &t
		u.Status = AccountStatusLocked
	}
}

// ResetFailedAttempts clears the failed login counter after successful auth.
func (u *User) ResetFailedAttempts() {
	u.FailedAttempts = 0
	u.LockedUntil = nil
	if u.Status == AccountStatusLocked {
		u.Status = AccountStatusActive
	}
	u.UpdatedAt = time.Now().UTC()
}

// UpdateLastLogin sets the last login timestamp.
func (u *User) UpdateLastLogin() {
	now := time.Now().UTC()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// VerifyEmail marks the email as verified and activates the account if pending.
func (u *User) VerifyEmail() {
	u.EmailVerified = true
	if u.Status == AccountStatusPendingVerification {
		u.Status = AccountStatusActive
	}
	u.UpdatedAt = time.Now().UTC()
}
