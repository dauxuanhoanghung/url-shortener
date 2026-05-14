package event

import (
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/google/uuid"
)

type UserRegistered struct {
	User     *model.User
	UserPlan *model.UserPlan
}

type EmailVerified struct {
	UserID uuid.UUID
}

type PasswordResetRequested struct {
	User *model.User
}

type PasswordReset struct {
	UserID uuid.UUID
}

type URLCreated struct {
	URL        *model.ShortURL
	MetadataID uuid.UUID
	UserID     uuid.UUID
}

type URLDeleted struct {
	URLID  uuid.UUID
	UserID uuid.UUID
}
