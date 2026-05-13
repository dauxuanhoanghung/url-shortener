package model

import (
	"time"

	"github.com/google/uuid"
)

type ShortURL struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	ShortCode      string
	OriginalURL    string
	ClickCount     int64
	LastAccessedAt *time.Time
	CreatedAt      time.Time
	DeletedAt      *time.Time
}
