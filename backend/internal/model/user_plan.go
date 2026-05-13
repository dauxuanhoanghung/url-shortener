package model

import (
	"time"

	"github.com/google/uuid"
)

type UserPlan struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	PlanCode  string
	StartedAt time.Time
	UpdatedAt time.Time
}
