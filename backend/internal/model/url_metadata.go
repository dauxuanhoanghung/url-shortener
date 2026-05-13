package model

import (
	"time"

	"github.com/google/uuid"
)

type FetchStatus string

const (
	FetchStatusPending FetchStatus = "pending"
	FetchStatusOK      FetchStatus = "ok"
	FetchStatusFailed  FetchStatus = "failed"
)

type URLMetadata struct {
	ID          uuid.UUID
	URLID       uuid.UUID
	Title       *string
	Description *string
	OgImage     *string
	FaviconURL  *string
	HTTPStatus  *int32
	FetchStatus FetchStatus
	FetchedAt   *time.Time
	CreatedAt   time.Time
}

// FetchResult carries the outcome of a background origin fetch.
type FetchResult struct {
	Title       *string
	Description *string
	OgImage     *string
	FaviconURL  *string
	HTTPStatus  int32
	FetchStatus FetchStatus
}
