package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	TokenPurposeVerifyEmail   = "verify_email"
	TokenPurposePasswordReset = "password_reset"
)

type Token struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Purpose    string
	TokenHash  string
	ExpiresAt  time.Time
	UsedAt     *time.Time
	CreatedAt  time.Time
}
