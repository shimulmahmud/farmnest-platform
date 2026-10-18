package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	user := NewUser("test@example.com", "$2a$12$hash", "John", "Doe", RoleCustomer)

	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "$2a$12$hash", user.PasswordHash)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, RoleCustomer, user.Role)
	assert.Equal(t, AccountStatusPendingVerification, user.Status)
	assert.False(t, user.EmailVerified)
	assert.Nil(t, user.LastLoginAt)
	assert.False(t, user.CreatedAt.IsZero())
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status AccountStatus
		locked *time.Time
		active bool
	}{
		{"active user", AccountStatusActive, nil, true},
		{"inactive user", AccountStatusInactive, nil, false},
		{"suspended user", AccountStatusSuspended, nil, false},
		{"pending user", AccountStatusPendingVerification, nil, true},
		{"locked user - future", AccountStatusLocked, ptrTime(time.Now().Add(1 * time.Hour)), false},
		{"locked user - past", AccountStatusLocked, ptrTime(time.Now().Add(-1 * time.Hour)), false}, // Status still LOCKED
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Status: tt.status, LockedUntil: tt.locked}
			assert.Equal(t, tt.active, u.IsActive())
		})
	}
}

func TestUser_RecordFailedAttempt(t *testing.T) {
	u := &User{Status: AccountStatusActive, FailedAttempts: 0}

	u.RecordFailedAttempt(5, 15*time.Minute)
	assert.Equal(t, 1, u.FailedAttempts)
	assert.Nil(t, u.LockedUntil)

	// Simulate reaching max
	for i := 0; i < 4; i++ {
		u.RecordFailedAttempt(5, 15*time.Minute)
	}
	assert.Equal(t, 5, u.FailedAttempts)
	assert.NotNil(t, u.LockedUntil)
	assert.Equal(t, AccountStatusLocked, u.Status)
}

func TestUser_ResetFailedAttempts(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	u := &User{Status: AccountStatusLocked, FailedAttempts: 5, LockedUntil: &future}

	u.ResetFailedAttempts()
	assert.Equal(t, 0, u.FailedAttempts)
	assert.Nil(t, u.LockedUntil)
	assert.Equal(t, AccountStatusActive, u.Status)
}

func TestUser_VerifyEmail(t *testing.T) {
	u := &User{Status: AccountStatusPendingVerification, EmailVerified: false}

	u.VerifyEmail()
	assert.True(t, u.EmailVerified)
	assert.Equal(t, AccountStatusActive, u.Status)
}

func TestUser_VerifyEmail_AlreadyActive(t *testing.T) {
	u := &User{Status: AccountStatusActive, EmailVerified: false}

	u.VerifyEmail()
	assert.True(t, u.EmailVerified)
	assert.Equal(t, AccountStatusActive, u.Status)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
