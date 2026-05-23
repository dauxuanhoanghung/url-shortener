package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Role            string     `json:"role"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	DisabledAt      *time.Time `json:"disabled_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (u *User) IsEmailVerified() bool { return u.EmailVerifiedAt != nil }
func (u *User) IsAdmin() bool         { return u.Role == RoleAdmin }
func (u *User) IsDisabled() bool      { return u.DisabledAt != nil }

// UserWithPlan is the row shape returned by admin listing queries: the user
// record joined to user_plans so the admin UI can show plan_code without a
// second round-trip. PlanCode is empty when the user has no user_plans row.
type UserWithPlan struct {
	User
	PlanCode string
}
