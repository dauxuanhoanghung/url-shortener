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

const verifyEmailTokenTTL = 24 * time.Hour

// SendVerificationEmail returns an event.HandlerFunc that issues a new
// verify-email token and sends the confirmation link to the user.
// Handles both UserRegistered (new account) and ResendVerification flows.
func SendVerificationEmail(
	tokenRepo repository.TokenRepository,
	m mailer.Mailer,
	frontendBaseURL string,
) event.HandlerFunc {
	return func(ctx context.Context, ev any) error {
		e := ev.(event.UserRegistered)
		return issueVerificationEmail(ctx, e.User, tokenRepo, m, frontendBaseURL)
	}
}

func issueVerificationEmail(
	ctx context.Context,
	user *model.User,
	tokenRepo repository.TokenRepository,
	m mailer.Mailer,
	frontendBaseURL string,
) error {
	if err := tokenRepo.InvalidateByPurpose(ctx, user.ID, model.TokenPurposeVerifyEmail); err != nil {
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
		Purpose:   model.TokenPurposeVerifyEmail,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(verifyEmailTokenTTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}
	link := frontendBaseURL + "/verify-email?token=" + raw
	return m.Send(ctx, mailer.Message{
		To:      user.Email,
		Subject: "Verify your email",
		Body:    fmt.Sprintf("Welcome! Please verify your email (valid 24h):\n\n%s", link),
	})
}
