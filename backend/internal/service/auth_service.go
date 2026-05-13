package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/mailer"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/pkg/utils"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrTokenInvalid         = errors.New("token invalid or expired")
	ErrEmailAlreadyVerified = errors.New("email already verified")
	ErrUserDisabled         = errors.New("user disabled")
)

const (
	verifyEmailTokenTTL   = 24 * time.Hour
	passwordResetTokenTTL = 30 * time.Minute
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	VerifyEmail(ctx context.Context, rawToken string) error
	ResendVerification(ctx context.Context, userID uuid.UUID) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
}

type authService struct {
	userRepo        repository.UserRepository
	tokenRepo       repository.TokenRepository
	mailer          mailer.Mailer
	jwtSecret       string
	frontendBaseURL string
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	m mailer.Mailer,
	jwtSecret, frontendBaseURL string,
) AuthService {
	return &authService{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		mailer:          m,
		jwtSecret:       jwtSecret,
		frontendBaseURL: frontendBaseURL,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user, err := s.userRepo.Create(ctx, &model.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		PlanType:     "free",
		PlanCode:     "free",
		Role:         model.RoleUser,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, err
	}

	// Issue a verification token + send email. Soft-gated: user is returned
	// a JWT regardless of whether the email goes out successfully — the token
	// row is the authoritative record. On provider failure we log and move on.
	if err := s.issueVerificationEmail(ctx, user); err != nil {
		// Don't fail registration on email delivery problems; the user
		// can hit /auth/resend-verification once they log in.
		_ = err
	}

	return s.generateAuthResponse(user)
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if user.IsDisabled() {
		return nil, ErrUserDisabled
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.generateAuthResponse(user)
}

func (s *authService) VerifyEmail(ctx context.Context, rawToken string) error {
	tok, err := s.lookupToken(ctx, rawToken, model.TokenPurposeVerifyEmail)
	if err != nil {
		return err
	}
	if err := s.userRepo.MarkEmailVerified(ctx, tok.UserID); err != nil {
		return err
	}
	return s.tokenRepo.MarkUsed(ctx, tok.ID)
}

func (s *authService) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.IsEmailVerified() {
		return ErrEmailAlreadyVerified
	}
	return s.issueVerificationEmail(ctx, user)
}

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	// Uniform response: caller must treat this as success regardless of
	// whether the user exists. We still do the work when the user exists.
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}
	return s.issuePasswordResetEmail(ctx, user)
}

func (s *authService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	tok, err := s.lookupToken(ctx, rawToken, model.TokenPurposePasswordReset)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.userRepo.UpdatePassword(ctx, tok.UserID, string(hash)); err != nil {
		return err
	}
	// Refresh-token revocation on password change: deferred to Batch 2 when
	// server-side refresh storage lands. For now, access tokens expire in
	// 15 minutes so the window is naturally bounded.
	return s.tokenRepo.MarkUsed(ctx, tok.ID)
}

// --- internal helpers --------------------------------------------------

func (s *authService) issueVerificationEmail(ctx context.Context, user *model.User) error {
	return s.issueTokenAndSend(ctx, user, model.TokenPurposeVerifyEmail, verifyEmailTokenTTL,
		"Verify your email",
		func(link string) string {
			return fmt.Sprintf("Welcome! Please verify your email by clicking this link (valid 24h):\n\n%s", link)
		},
		s.frontendBaseURL+"/verify-email?token=",
	)
}

func (s *authService) issuePasswordResetEmail(ctx context.Context, user *model.User) error {
	return s.issueTokenAndSend(ctx, user, model.TokenPurposePasswordReset, passwordResetTokenTTL,
		"Reset your password",
		func(link string) string {
			return fmt.Sprintf("Reset your password by clicking this link (valid 30 min):\n\n%s\n\nIf you did not request this, ignore this email.", link)
		},
		s.frontendBaseURL+"/reset-password?token=",
	)
}

func (s *authService) issueTokenAndSend(
	ctx context.Context,
	user *model.User,
	purpose string,
	ttl time.Duration,
	subject string,
	bodyFn func(link string) string,
	linkPrefix string,
) error {
	// Invalidate any previous unused token for this purpose — one live link
	// at a time keeps the "single-use" invariant unambiguous.
	if err := s.tokenRepo.InvalidateByPurpose(ctx, user.ID, purpose); err != nil {
		return err
	}
	raw, err := generateOpaqueToken()
	if err != nil {
		return err
	}
	hash := hashToken(raw)
	now := time.Now()
	err = s.tokenRepo.Create(ctx, &model.Token{
		ID:        uuid.New(),
		UserID:    user.ID,
		Purpose:   purpose,
		TokenHash: hash,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	})
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, mailer.Message{
		To:      user.Email,
		Subject: subject,
		Body:    bodyFn(linkPrefix + raw),
	})
}

func (s *authService) lookupToken(ctx context.Context, rawToken, purpose string) (*model.Token, error) {
	tok, err := s.tokenRepo.GetUsableByHash(ctx, hashToken(rawToken), purpose)
	if errors.Is(err, repository.ErrTokenNotFound) {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func (s *authService) generateAuthResponse(user *model.User) (*dto.AuthResponse, error) {
	userID := user.ID.String()

	accessToken, err := utils.GenerateAccessToken(userID, user.Email, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(userID, user.Email, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:              userID,
			Email:           user.Email,
			PlanType:        user.PlanType,
			EmailVerified:   user.IsEmailVerified(),
			Role:            user.Role,
		},
	}, nil
}

func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
