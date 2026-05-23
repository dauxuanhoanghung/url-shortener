package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/event"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const authTestSecret = "auth-test-secret"

// ── helpers ───────────────────────────────────────────────────────────────────

func newAuthService(
	userRepo repository.UserRepository,
	userPlanRepo repository.UserPlanRepository,
	tokenRepo repository.TokenRepository,
	bus event.EventBus,
) AuthService {
	return NewAuthService(userRepo, userPlanRepo, tokenRepo, bus, authTestSecret)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestAuthService_Register_HappyPath(t *testing.T) {
	userRepo := &fakeUserRepo{}
	planRepo := &fakeUserPlanRepo{}
	tokenRepo := &fakeTokenRepo{}
	bus := &fakeBus{}

	svc := newAuthService(userRepo, planRepo, tokenRepo, bus)
	resp, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:    "new@example.com",
		Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Error("expected non-empty tokens on register")
	}
	if resp.User.Email != "new@example.com" {
		t.Errorf("user email: got %q", resp.User.Email)
	}
	if resp.User.PlanCode != "free" {
		t.Errorf("default plan: got %q want free", resp.User.PlanCode)
	}
	if userRepo.createCalls != 1 {
		t.Errorf("expected 1 user create, got %d", userRepo.createCalls)
	}
	if planRepo.createCalls != 1 {
		t.Errorf("expected 1 user_plan create, got %d", planRepo.createCalls)
	}
	if len(bus.published) == 0 {
		t.Error("expected UserRegistered event to be published")
	}
	if _, ok := bus.published[0].(event.UserRegistered); !ok {
		t.Errorf("expected event.UserRegistered, got %T", bus.published[0])
	}
}

func TestAuthService_Register_EmailExists(t *testing.T) {
	existing := &model.User{ID: uuid.New(), Email: "taken@example.com"}
	svc := newAuthService(
		&fakeUserRepo{byEmail: map[string]*model.User{"taken@example.com": existing}},
		&fakeUserPlanRepo{},
		&fakeTokenRepo{},
		&fakeBus{},
	)
	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:    "taken@example.com",
		Password: "anotherone",
	})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestAuthService_Login_HappyPath(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpw"), bcrypt.MinCost)
	user := &model.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Role:         model.RoleUser,
	}
	svc := newAuthService(
		&fakeUserRepo{byEmail: map[string]*model.User{user.Email: user}},
		&fakeUserPlanRepo{plans: map[uuid.UUID]*model.UserPlan{user.ID: {UserID: user.ID, PlanCode: "pro"}}},
		&fakeTokenRepo{},
		&fakeBus{},
	)
	resp, err := svc.Login(context.Background(), dto.LoginRequest{Email: user.Email, Password: "correctpw"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.User.PlanCode != "pro" {
		t.Errorf("plan: got %q want pro", resp.User.PlanCode)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpw"), bcrypt.MinCost)
	user := &model.User{ID: uuid.New(), Email: "u@e.com", PasswordHash: string(hash), Role: model.RoleUser}
	svc := newAuthService(
		&fakeUserRepo{byEmail: map[string]*model.User{user.Email: user}},
		&fakeUserPlanRepo{plans: map[uuid.UUID]*model.UserPlan{user.ID: {UserID: user.ID, PlanCode: "free"}}},
		&fakeTokenRepo{},
		&fakeBus{},
	)
	_, err := svc.Login(context.Background(), dto.LoginRequest{Email: user.Email, Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	// Unknown email and wrong password must return the SAME error code to
	// avoid leaking which accounts exist (anti-enumeration).
	svc := newAuthService(&fakeUserRepo{}, &fakeUserPlanRepo{}, &fakeTokenRepo{}, &fakeBus{})
	_, err := svc.Login(context.Background(), dto.LoginRequest{Email: "nobody@example.com", Password: "anything"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for unknown email, got %v", err)
	}
}

func TestAuthService_Login_DisabledUser(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpw"), bcrypt.MinCost)
	now := time.Now()
	user := &model.User{
		ID:           uuid.New(),
		Email:        "disabled@example.com",
		PasswordHash: string(hash),
		Role:         model.RoleUser,
		DisabledAt:   &now,
	}
	svc := newAuthService(
		&fakeUserRepo{byEmail: map[string]*model.User{user.Email: user}},
		&fakeUserPlanRepo{},
		&fakeTokenRepo{},
		&fakeBus{},
	)
	_, err := svc.Login(context.Background(), dto.LoginRequest{Email: user.Email, Password: "correctpw"})
	if !errors.Is(err, ErrUserDisabled) {
		t.Errorf("expected ErrUserDisabled, got %v", err)
	}
}

// ── VerifyEmail ───────────────────────────────────────────────────────────────

func TestAuthService_VerifyEmail_HappyPath(t *testing.T) {
	raw := "raw-token-bytes"
	tokenID := uuid.New()
	userID := uuid.New()
	tokenRepo := &fakeTokenRepo{
		usableByHash: map[string]*model.Token{
			hashToken(raw): {ID: tokenID, UserID: userID, Purpose: model.TokenPurposeVerifyEmail},
		},
	}
	userRepo := &fakeUserRepo{}
	svc := newAuthService(userRepo, &fakeUserPlanRepo{}, tokenRepo, &fakeBus{})

	if err := svc.VerifyEmail(context.Background(), raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !userRepo.markedVerified[userID] {
		t.Error("expected user to be marked verified")
	}
	if !tokenRepo.usedTokens[tokenID] {
		t.Error("expected token to be marked used")
	}
}

func TestAuthService_VerifyEmail_InvalidToken(t *testing.T) {
	svc := newAuthService(&fakeUserRepo{}, &fakeUserPlanRepo{}, &fakeTokenRepo{}, &fakeBus{})
	if err := svc.VerifyEmail(context.Background(), "wrong"); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

// ── ForgotPassword ────────────────────────────────────────────────────────────

func TestAuthService_ForgotPassword_KnownEmail(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "known@example.com"}
	bus := &fakeBus{}
	svc := newAuthService(
		&fakeUserRepo{byEmail: map[string]*model.User{user.Email: user}},
		&fakeUserPlanRepo{},
		&fakeTokenRepo{},
		bus,
	)
	if err := svc.ForgotPassword(context.Background(), user.Email); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bus.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(bus.published))
	}
	if _, ok := bus.published[0].(event.PasswordResetRequested); !ok {
		t.Errorf("expected PasswordResetRequested, got %T", bus.published[0])
	}
}

func TestAuthService_ForgotPassword_UnknownEmail_SilentNoop(t *testing.T) {
	// Must return nil for unknown email to avoid enumeration; no event published.
	bus := &fakeBus{}
	svc := newAuthService(&fakeUserRepo{}, &fakeUserPlanRepo{}, &fakeTokenRepo{}, bus)
	if err := svc.ForgotPassword(context.Background(), "nobody@example.com"); err != nil {
		t.Errorf("expected nil error for unknown email, got %v", err)
	}
	if len(bus.published) != 0 {
		t.Errorf("expected 0 events, got %d", len(bus.published))
	}
}

// ── ResetPassword ─────────────────────────────────────────────────────────────

func TestAuthService_ResetPassword_HappyPath(t *testing.T) {
	raw := "reset-raw-token"
	tokenID := uuid.New()
	userID := uuid.New()
	userRepo := &fakeUserRepo{
		byID: map[uuid.UUID]*model.User{userID: {ID: userID, Email: "u@e.com"}},
	}
	tokenRepo := &fakeTokenRepo{
		usableByHash: map[string]*model.Token{
			hashToken(raw): {ID: tokenID, UserID: userID, Purpose: model.TokenPurposePasswordReset},
		},
	}
	svc := newAuthService(userRepo, &fakeUserPlanRepo{}, tokenRepo, &fakeBus{})

	if err := svc.ResetPassword(context.Background(), raw, "freshpassword"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userRepo.updatedPasswords[userID] == "" {
		t.Error("expected password to be updated")
	}
	if userRepo.updatedPasswords[userID] == "freshpassword" {
		t.Error("password must be stored as bcrypt hash, not plaintext")
	}
	if !tokenRepo.usedTokens[tokenID] {
		t.Error("expected token to be marked used")
	}
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	svc := newAuthService(&fakeUserRepo{}, &fakeUserPlanRepo{}, &fakeTokenRepo{}, &fakeBus{})
	if err := svc.ResetPassword(context.Background(), "bad", "newpassword"); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

// ── ResendVerification ────────────────────────────────────────────────────────

func TestAuthService_ResendVerification_AlreadyVerified(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	svc := newAuthService(
		&fakeUserRepo{byID: map[uuid.UUID]*model.User{
			userID: {ID: userID, EmailVerifiedAt: &now},
		}},
		&fakeUserPlanRepo{},
		&fakeTokenRepo{},
		&fakeBus{},
	)
	if err := svc.ResendVerification(context.Background(), userID); !errors.Is(err, ErrEmailAlreadyVerified) {
		t.Errorf("expected ErrEmailAlreadyVerified, got %v", err)
	}
}

// ── fakes ─────────────────────────────────────────────────────────────────────

// fakeUserRepo satisfies repository.UserRepository.
type fakeUserRepo struct {
	byEmail          map[string]*model.User
	byID             map[uuid.UUID]*model.User
	createCalls      int
	markedVerified   map[uuid.UUID]bool
	updatedPasswords map[uuid.UUID]string
}

func (r *fakeUserRepo) Create(_ context.Context, u *model.User) (*model.User, error) {
	r.createCalls++
	if r.byEmail == nil {
		r.byEmail = map[string]*model.User{}
	}
	if r.byID == nil {
		r.byID = map[uuid.UUID]*model.User{}
	}
	out := *u
	out.CreatedAt = time.Now()
	out.UpdatedAt = out.CreatedAt
	r.byEmail[u.Email] = &out
	r.byID[u.ID] = &out
	return &out, nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if u, ok := r.byEmail[email]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) MarkEmailVerified(_ context.Context, id uuid.UUID) error {
	if r.markedVerified == nil {
		r.markedVerified = map[uuid.UUID]bool{}
	}
	r.markedVerified[id] = true
	return nil
}

func (r *fakeUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, passwordHash string) error {
	if r.updatedPasswords == nil {
		r.updatedPasswords = map[uuid.UUID]string{}
	}
	r.updatedPasswords[id] = passwordHash
	return nil
}

func (r *fakeUserRepo) ListForAdmin(_ context.Context, _, _ int32) ([]model.UserWithPlan, error) {
	return nil, nil
}

func (r *fakeUserRepo) CountForAdmin(_ context.Context) (int64, error) { return 0, nil }

func (r *fakeUserRepo) SetDisabled(_ context.Context, _ uuid.UUID, _ *time.Time) error { return nil }

// fakeUserPlanRepo satisfies repository.UserPlanRepository.
type fakeUserPlanRepo struct {
	plans       map[uuid.UUID]*model.UserPlan
	createCalls int
}

func (r *fakeUserPlanRepo) Create(_ context.Context, userID uuid.UUID, code string) (*model.UserPlan, error) {
	r.createCalls++
	if r.plans == nil {
		r.plans = map[uuid.UUID]*model.UserPlan{}
	}
	up := &model.UserPlan{UserID: userID, PlanCode: code}
	r.plans[userID] = up
	return up, nil
}

func (r *fakeUserPlanRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*model.UserPlan, error) {
	if up, ok := r.plans[userID]; ok {
		return up, nil
	}
	return nil, repository.ErrUserPlanNotFound
}

func (r *fakeUserPlanRepo) Update(_ context.Context, userID uuid.UUID, code string) (*model.UserPlan, error) {
	if r.plans == nil {
		r.plans = map[uuid.UUID]*model.UserPlan{}
	}
	up := &model.UserPlan{UserID: userID, PlanCode: code}
	r.plans[userID] = up
	return up, nil
}

// fakeTokenRepo satisfies repository.TokenRepository.
type fakeTokenRepo struct {
	usableByHash map[string]*model.Token
	usedTokens   map[uuid.UUID]bool
	createCalls  int
}

func (r *fakeTokenRepo) Create(_ context.Context, _ *model.Token) error {
	r.createCalls++
	return nil
}

func (r *fakeTokenRepo) GetUsableByHash(_ context.Context, hash, _ string) (*model.Token, error) {
	if tok, ok := r.usableByHash[hash]; ok {
		return tok, nil
	}
	return nil, repository.ErrTokenNotFound
}

func (r *fakeTokenRepo) MarkUsed(_ context.Context, id uuid.UUID) error {
	if r.usedTokens == nil {
		r.usedTokens = map[uuid.UUID]bool{}
	}
	r.usedTokens[id] = true
	return nil
}

func (r *fakeTokenRepo) InvalidateByPurpose(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
