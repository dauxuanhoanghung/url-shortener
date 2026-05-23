package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/event"
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
	defaultPlanCode = "free"
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
	userRepo     repository.UserRepository
	userPlanRepo repository.UserPlanRepository
	tokenRepo    repository.TokenRepository
	bus          event.EventBus
	jwtSecret    string
}

func NewAuthService(
	userRepo repository.UserRepository,
	userPlanRepo repository.UserPlanRepository,
	tokenRepo repository.TokenRepository,
	bus event.EventBus,
	jwtSecret string,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		userPlanRepo: userPlanRepo,
		tokenRepo:    tokenRepo,
		bus:          bus,
		jwtSecret:    jwtSecret,
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
		Role:         model.RoleUser,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, err
	}

	userPlan, err := s.userPlanRepo.Create(ctx, user.ID, defaultPlanCode)
	if err != nil {
		return nil, err
	}

	// Side-effects run asynchronously via the bus; email failure does not block registration.
	_ = s.bus.Publish(ctx, event.UserRegistered{User: user, UserPlan: userPlan})

	return s.generateAuthResponse(user, userPlan)
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

	userPlan, err := s.userPlanRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return s.generateAuthResponse(user, userPlan)
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
	// Reuse the UserRegistered event — the handler only needs the User field.
	return s.bus.Publish(ctx, event.UserRegistered{User: user})
}

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	// Uniform response regardless of whether the user exists (anti-enumeration).
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}
	return s.bus.Publish(ctx, event.PasswordResetRequested{User: user})
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
	return s.tokenRepo.MarkUsed(ctx, tok.ID)
}

// --- internal helpers ---------------------------------------------------

func (s *authService) lookupToken(ctx context.Context, rawToken, purpose string) (*model.Token, error) {
	tok, err := s.tokenRepo.GetUsableByHash(ctx, sha256Hex(rawToken), purpose)
	if errors.Is(err, repository.ErrTokenNotFound) {
		return nil, ErrTokenInvalid
	}
	return tok, err
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *authService) generateAuthResponse(user *model.User, userPlan *model.UserPlan) (*dto.AuthResponse, error) {
	userID := user.ID.String()

	tokenIn := utils.TokenInput{UserID: userID, Email: user.Email, Role: user.Role}
	accessToken, err := utils.GenerateAccessToken(tokenIn, s.jwtSecret)
	if err != nil {
		return nil, err
	}
	refreshToken, err := utils.GenerateRefreshToken(tokenIn, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:            userID,
			Email:         user.Email,
			PlanCode:      userPlan.PlanCode,
			EmailVerified: user.IsEmailVerified(),
			Role:          user.Role,
		},
	}, nil
}
