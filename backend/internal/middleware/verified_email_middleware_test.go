package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newVerifiedRouter(repo repository.UserRepository, setUserID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setUserID != "" {
			c.Set("userID", setUserID)
		}
		c.Next()
	})
	r.GET("/protected", VerifiedEmailRequired(repo), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestVerifiedEmailRequired_MissingUserID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	newVerifiedRouter(&stubUserRepo{}, "").ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", w.Code)
	}
}

func TestVerifiedEmailRequired_InvalidUserIDFormat(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	newVerifiedRouter(&stubUserRepo{}, "not-a-uuid").ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", w.Code)
	}
}

func TestVerifiedEmailRequired_UserNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	newVerifiedRouter(&stubUserRepo{}, uuid.New().String()).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", w.Code)
	}
}

func TestVerifiedEmailRequired_WithinGracePeriod_Allowed(t *testing.T) {
	id := uuid.New()
	repo := &stubUserRepo{users: map[uuid.UUID]*model.User{
		id: {ID: id, CreatedAt: time.Now().Add(-24 * time.Hour)}, // 1 day old, unverified
	}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	newVerifiedRouter(repo, id.String()).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestVerifiedEmailRequired_PastGracePeriod_Unverified_Forbidden(t *testing.T) {
	id := uuid.New()
	repo := &stubUserRepo{users: map[uuid.UUID]*model.User{
		id: {ID: id, CreatedAt: time.Now().Add(-(VerificationGracePeriod + time.Hour))},
	}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	newVerifiedRouter(repo, id.String()).ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "EMAIL_VERIFICATION_REQUIRED") {
		t.Errorf("expected EMAIL_VERIFICATION_REQUIRED, body=%s", w.Body.String())
	}
}

func TestVerifiedEmailRequired_VerifiedUser_AlwaysAllowed(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	repo := &stubUserRepo{users: map[uuid.UUID]*model.User{
		// 1 year old, verified — past grace, but verification flag overrides.
		id: {
			ID:              id,
			CreatedAt:       time.Now().Add(-365 * 24 * time.Hour),
			EmailVerifiedAt: &now,
		},
	}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	newVerifiedRouter(repo, id.String()).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("verified user should be allowed: got %d", w.Code)
	}
}

// ── fakes ─────────────────────────────────────────────────────────────────────

// stubUserRepo satisfies repository.UserRepository, but the middleware only
// touches GetByID — the other methods are intentional no-ops.
type stubUserRepo struct {
	users map[uuid.UUID]*model.User
}

func (r *stubUserRepo) Create(_ context.Context, u *model.User) (*model.User, error) { return u, nil }
func (r *stubUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, repository.ErrUserNotFound
}
func (r *stubUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}
func (r *stubUserRepo) MarkEmailVerified(_ context.Context, _ uuid.UUID) error { return nil }
func (r *stubUserRepo) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (r *stubUserRepo) ListForAdmin(_ context.Context, _, _ int32) ([]model.UserWithPlan, error) {
	return nil, nil
}
func (r *stubUserRepo) CountForAdmin(_ context.Context) (int64, error) { return 0, nil }
func (r *stubUserRepo) SetDisabled(_ context.Context, _ uuid.UUID, _ *time.Time) error {
	return nil
}
