package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/google/uuid"
)

// ── ListUsers ─────────────────────────────────────────────────────────────────

func TestAdminService_ListUsers_AppliesDefaultLimit(t *testing.T) {
	userRepo := &fakeAdminUserRepo{
		users: []model.UserWithPlan{
			{User: model.User{ID: uuid.New(), Email: "a@e.com"}, PlanCode: "free"},
		},
		count: 1,
	}
	svc := NewAdminService(userRepo, &fakeUserPlanRepo{}, &stubPlanRepo{}, &fakeAuditRepo{})

	resp, err := svc.ListUsers(context.Background(), 0, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userRepo.lastLimit != 50 {
		t.Errorf("default limit not applied: got %d want 50", userRepo.lastLimit)
	}
	if userRepo.lastOffset != 0 {
		t.Errorf("negative offset not clamped: got %d", userRepo.lastOffset)
	}
	if resp.Total != 1 || len(resp.Users) != 1 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// ── SetUserDisabled ───────────────────────────────────────────────────────────

func TestAdminService_SetUserDisabled_HappyPath_WritesAudit(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	userRepo := &fakeAdminUserRepo{
		byID: map[uuid.UUID]*model.User{target: {ID: target, Email: "t@e.com"}},
	}
	audit := &fakeAuditRepo{}
	svc := NewAdminService(userRepo, &fakeUserPlanRepo{}, &stubPlanRepo{}, audit)

	resp, err := svc.SetUserDisabled(context.Background(), actor, target, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Disabled {
		t.Error("response should report disabled=true")
	}
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	got := audit.entries[0]
	if got.Action != "user_disabled" {
		t.Errorf("action: got %q", got.Action)
	}
	if got.TargetID != target.String() {
		t.Errorf("target id: got %q", got.TargetID)
	}
	if got.ActorID == nil || *got.ActorID != actor {
		t.Errorf("actor id: got %v want %v", got.ActorID, actor)
	}
}

func TestAdminService_SetUserDisabled_BlocksSelf(t *testing.T) {
	self := uuid.New()
	audit := &fakeAuditRepo{}
	svc := NewAdminService(
		&fakeAdminUserRepo{byID: map[uuid.UUID]*model.User{self: {ID: self}}},
		&fakeUserPlanRepo{},
		&stubPlanRepo{},
		audit,
	)
	_, err := svc.SetUserDisabled(context.Background(), self, self, true)
	if !errors.Is(err, ErrCannotDisableSelf) {
		t.Errorf("expected ErrCannotDisableSelf, got %v", err)
	}
	if len(audit.entries) != 0 {
		t.Error("no audit entry should be written when self-disable is blocked")
	}
}

func TestAdminService_SetUserDisabled_TargetNotFound(t *testing.T) {
	svc := NewAdminService(&fakeAdminUserRepo{}, &fakeUserPlanRepo{}, &stubPlanRepo{}, &fakeAuditRepo{})
	_, err := svc.SetUserDisabled(context.Background(), uuid.New(), uuid.New(), true)
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// ── UpdatePlanFeatures ────────────────────────────────────────────────────────

func TestAdminService_UpdatePlanFeatures_HappyPath_WritesAudit(t *testing.T) {
	actor := uuid.New()
	plan := &model.Plan{Code: "pro", Name: "Pro", Features: map[string]bool{"webhooks": false}}
	audit := &fakeAuditRepo{}
	svc := NewAdminService(
		&fakeAdminUserRepo{},
		&fakeUserPlanRepo{},
		&stubPlanRepo{plan: plan},
		audit,
	)

	updated, err := svc.UpdatePlanFeatures(context.Background(), actor, "pro", map[string]bool{"webhooks": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.Features["webhooks"] {
		t.Error("expected webhooks=true after update")
	}
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "plan_features_updated" {
		t.Errorf("action: got %q", audit.entries[0].Action)
	}
}

func TestAdminService_UpdatePlanFeatures_PlanNotFound(t *testing.T) {
	svc := NewAdminService(
		&fakeAdminUserRepo{},
		&fakeUserPlanRepo{},
		&stubPlanRepo{}, // empty
		&fakeAuditRepo{},
	)
	_, err := svc.UpdatePlanFeatures(context.Background(), uuid.New(), "missing", map[string]bool{})
	if !errors.Is(err, repository.ErrPlanNotFound) {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// ── ListAudit ─────────────────────────────────────────────────────────────────

func TestAdminService_ListAudit_HappyPath(t *testing.T) {
	actorID := uuid.New()
	audit := &fakeAuditRepo{
		entries: []repository.AdminAuditEntry{
			{
				ID:        uuid.New(),
				ActorID:   &actorID,
				Action:    "plan_features_updated",
				TargetID:  "pro",
				CreatedAt: time.Now(),
			},
		},
	}
	svc := NewAdminService(&fakeAdminUserRepo{}, &fakeUserPlanRepo{}, &stubPlanRepo{}, audit)
	resp, err := svc.ListAudit(context.Background(), 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].ActorID == nil || *resp.Entries[0].ActorID != actorID.String() {
		t.Errorf("actor id mismatch: %+v", resp.Entries[0].ActorID)
	}
}

// ── fakes ─────────────────────────────────────────────────────────────────────

// fakeAdminUserRepo satisfies repository.UserRepository. We use a dedicated
// fake (instead of reusing fakeUserRepo from auth tests) because the admin
// service exercises ListForAdmin / CountForAdmin / SetDisabled paths the
// auth fake doesn't bother tracking.
type fakeAdminUserRepo struct {
	users          []model.UserWithPlan
	byID           map[uuid.UUID]*model.User
	count          int64
	lastLimit      int32
	lastOffset     int32
	disabledCalls  map[uuid.UUID]*time.Time
}

func (r *fakeAdminUserRepo) Create(_ context.Context, u *model.User) (*model.User, error) { return u, nil }
func (r *fakeAdminUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, repository.ErrUserNotFound
}
func (r *fakeAdminUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}
func (r *fakeAdminUserRepo) MarkEmailVerified(_ context.Context, _ uuid.UUID) error { return nil }
func (r *fakeAdminUserRepo) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (r *fakeAdminUserRepo) ListForAdmin(_ context.Context, limit, offset int32) ([]model.UserWithPlan, error) {
	r.lastLimit = limit
	r.lastOffset = offset
	return r.users, nil
}
func (r *fakeAdminUserRepo) CountForAdmin(_ context.Context) (int64, error) { return r.count, nil }
func (r *fakeAdminUserRepo) SetDisabled(_ context.Context, id uuid.UUID, t *time.Time) error {
	if r.disabledCalls == nil {
		r.disabledCalls = map[uuid.UUID]*time.Time{}
	}
	r.disabledCalls[id] = t
	if u, ok := r.byID[id]; ok {
		u.DisabledAt = t
	}
	return nil
}

// fakeAuditRepo satisfies repository.AdminAuditRepository.
type fakeAuditRepo struct {
	entries []repository.AdminAuditEntry
}

func (r *fakeAuditRepo) Create(_ context.Context, e *repository.AdminAuditEntry) error {
	r.entries = append(r.entries, *e)
	return nil
}

func (r *fakeAuditRepo) List(_ context.Context, _, _ int32) ([]repository.AdminAuditEntry, error) {
	return r.entries, nil
}

func (r *fakeAuditRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.entries)), nil
}
