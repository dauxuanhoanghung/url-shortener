package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/event"
	"github.com/dauxuanhoanghung/url-shortener/internal/mailer"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/google/uuid"
)

const passwordResetTokenTTL = 30 * time.Minute

// SendPasswordResetEmail returns an event.HandlerFunc that issues a new
// password-reset token and emails the reset link to the user.
func SendPasswordResetEmail(
	tokenRepo repository.TokenRepository,
	m mailer.Mailer,
	frontendBaseURL string,
) event.HandlerFunc {
	return func(ctx context.Context, ev any) error {
		e := ev.(event.PasswordResetRequested)
		return issuePasswordResetEmail(ctx, e.User, tokenRepo, m, frontendBaseURL)
	}
}

func issuePasswordResetEmail(
	ctx context.Context,
	user *model.User,
	tokenRepo repository.TokenRepository,
	m mailer.Mailer,
	frontendBaseURL string,
) error {
	if err := tokenRepo.InvalidateByPurpose(ctx, user.ID, model.TokenPurposePasswordReset); err != nil {
		return err
	}
	raw, err := generateOpaqueToken()
	if err != nil {
		return err
	}
	now := time.Now()
	if err := tokenRepo.Create(ctx, &model.Token{
		ID:        uuid.New(),
		UserID:    user.ID,
		Purpose:   model.TokenPurposePasswordReset,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(passwordResetTokenTTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}
	link := frontendBaseURL + "/reset-password?token=" + raw
	return m.Send(ctx, mailer.Message{
		To:      user.Email,
		Subject: "Reset your password",
		Body:    fmt.Sprintf("Reset your password (valid 30 min):\n\n%s\n\nIf you did not request this, ignore this email.", link),
	})
}
