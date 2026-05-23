package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/event"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/google/uuid"
)

// fakeBus records published events and satisfies event.EventBus.
type fakeBus struct{ published []any }

func (f *fakeBus) Publish(_ context.Context, ev any) error {
	f.published = append(f.published, ev)
	return nil
}
func (f *fakeBus) Subscribe(_ reflect.Type, _ event.HandlerFunc, _ event.DispatchMode) {}

const testBaseURL = "https://short.ly"

// ── helpers ───────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }

func newService(
	urlRepo repository.URLRepository,
	metaRepo repository.URLMetadataRepository,
	userPlanRepo repository.UserPlanRepository,
	planRepo repository.PlanRepository,
	bus event.EventBus,
) URLService {
	return NewURLService(urlRepo, metaRepo, userPlanRepo, planRepo, bus)
}

func freePlan() *model.Plan {
	return &model.Plan{Code: "free", MaxURLs: 5}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestURLService_Create_HappyPath(t *testing.T) {
	userID := uuid.New()
	urlRepo := &stubURLRepo{}
	metaRepo := &stubMetaRepo{}
	bus := &fakeBus{}

	svc := newService(urlRepo, metaRepo,
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		bus,
	)

	resp, err := svc.Create(context.Background(), userID, dto.CreateURLRequest{OriginalURL: "https://example.com"}, testBaseURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ShortCode == "" {
		t.Error("expected a non-empty short code")
	}
	if resp.ShortURL != testBaseURL+"/r/"+resp.ShortCode {
		t.Errorf("short_url: got %q", resp.ShortURL)
	}
	if resp.Metadata != nil {
		t.Error("Create response should not include metadata (nil expected)")
	}
	if len(bus.published) == 0 {
		t.Error("expected URLCreated event to be published after Create")
	}
	if _, ok := bus.published[0].(event.URLCreated); !ok {
		t.Errorf("expected URLCreated event, got %T", bus.published[0])
	}
}

func TestURLService_Create_PlanLimitReached(t *testing.T) {
	userID := uuid.New()
	urlRepo := &stubURLRepo{count: 5}

	svc := newService(urlRepo,
		&stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	_, err := svc.Create(context.Background(), userID, dto.CreateURLRequest{OriginalURL: "https://example.com"}, testBaseURL)
	if !errors.Is(err, ErrPlanLimitReached) {
		t.Errorf("expected ErrPlanLimitReached, got %v", err)
	}
}

func TestURLService_Create_InvalidURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/html,<h1>x</h1>"},
		{"file scheme", "file:///etc/passwd"},
		{"no scheme", "example.com/path"},
		{"ftp scheme", "ftp://files.example.com"},
		{"empty", ""},
	}

	svc := newService(
		&stubURLRepo{},
		&stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), uuid.New(), dto.CreateURLRequest{OriginalURL: tc.url}, testBaseURL)
			if !errors.Is(err, ErrInvalidURL) {
				t.Errorf("expected ErrInvalidURL for %q, got %v", tc.url, err)
			}
		})
	}
}

func TestURLService_Create_ShortCodeConflictRetries(t *testing.T) {
	userID := uuid.New()
	urlRepo := &stubURLRepo{conflictUntil: 3}

	svc := newService(urlRepo,
		&stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	resp, err := svc.Create(context.Background(), userID, dto.CreateURLRequest{OriginalURL: "https://example.com"}, testBaseURL)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if resp.ShortCode == "" {
		t.Error("expected a short code on success")
	}
	if urlRepo.createCalls != 4 {
		t.Errorf("expected 4 Create calls (3 conflicts + 1 success), got %d", urlRepo.createCalls)
	}
}

func TestURLService_Create_ExhaustsRetries(t *testing.T) {
	urlRepo := &stubURLRepo{conflictUntil: 99}

	svc := newService(urlRepo,
		&stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	_, err := svc.Create(context.Background(), uuid.New(), dto.CreateURLRequest{OriginalURL: "https://example.com"}, testBaseURL)
	if !errors.Is(err, ErrShortCodeRetry) {
		t.Errorf("expected ErrShortCodeRetry, got %v", err)
	}
}

func TestURLService_Create_MetadataFailure_StillReturnsURL(t *testing.T) {
	// If the metadata row creation fails, Create should still succeed —
	// the URLCreated event won't be published (best-effort).
	urlRepo := &stubURLRepo{}
	metaRepo := &stubMetaRepo{createErr: errors.New("db down")}
	bus := &fakeBus{}

	svc := newService(urlRepo, metaRepo,
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		bus,
	)

	resp, err := svc.Create(context.Background(), uuid.New(), dto.CreateURLRequest{OriginalURL: "https://example.com"}, testBaseURL)
	if err != nil {
		t.Fatalf("Create should succeed even when meta repo fails: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(bus.published) != 0 {
		t.Error("URLCreated event should NOT be published when metadata creation fails")
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestURLService_List_ReturnsItemsWithMetadata(t *testing.T) {
	userID := uuid.New()
	title := "Example Domain"
	rows := []repository.ShortURLWithMeta{
		{
			URL: model.ShortURL{
				ID:          uuid.New(),
				UserID:      userID,
				ShortCode:   "abc123",
				OriginalURL: "https://example.com",
				CreatedAt:   time.Now(),
			},
			Metadata: &model.URLMetadata{
				FetchStatus: model.FetchStatusOK,
				Title:       &title,
			},
		},
		{
			URL: model.ShortURL{
				ID:          uuid.New(),
				UserID:      userID,
				ShortCode:   "def456",
				OriginalURL: "https://other.com",
				CreatedAt:   time.Now(),
			},
			Metadata: nil,
		},
	}

	urlRepo := &stubURLRepo{withMetaRows: rows, count: len(rows)}
	svc := newService(urlRepo, &stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	resp, err := svc.List(context.Background(), userID, 50, 0, testBaseURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.URLs) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(resp.URLs))
	}

	first := resp.URLs[0]
	if first.Metadata == nil {
		t.Fatal("first URL should have metadata")
	}
	if first.Metadata.FetchStatus != "ok" {
		t.Errorf("fetch_status: got %q want ok", first.Metadata.FetchStatus)
	}
	if first.Metadata.Title == nil || *first.Metadata.Title != title {
		t.Errorf("title: got %v want %q", first.Metadata.Title, title)
	}

	second := resp.URLs[1]
	if second.Metadata != nil {
		t.Error("second URL metadata should be nil (no row yet)")
	}
}

func TestURLService_List_DefaultLimit(t *testing.T) {
	urlRepo := &stubURLRepo{}
	svc := newService(urlRepo, &stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	_, err := svc.List(context.Background(), uuid.New(), 0, 0, testBaseURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if urlRepo.lastLimit != defaultListLimit {
		t.Errorf("limit: got %d want %d", urlRepo.lastLimit, defaultListLimit)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestURLService_Delete_HappyPath(t *testing.T) {
	userID := uuid.New()
	urlID := uuid.New()
	urlRepo := &stubURLRepo{storedURL: &model.ShortURL{ID: urlID, UserID: userID}}

	svc := newService(urlRepo, &stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	if err := svc.Delete(context.Background(), urlID, userID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestURLService_Delete_NotFound(t *testing.T) {
	svc := newService(&stubURLRepo{}, &stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrURLNotFound) {
		t.Errorf("expected ErrURLNotFound, got %v", err)
	}
}

func TestURLService_Delete_Forbidden(t *testing.T) {
	urlID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()

	urlRepo := &stubURLRepo{storedURL: &model.ShortURL{ID: urlID, UserID: ownerID}}
	svc := newService(urlRepo, &stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&fakeBus{},
	)

	err := svc.Delete(context.Background(), urlID, otherID)
	if !errors.Is(err, ErrURLForbidden) {
		t.Errorf("expected ErrURLForbidden, got %v", err)
	}
}

// ── fakes ─────────────────────────────────────────────────────────────────────

// stubURLRepo satisfies repository.URLRepository.
type stubURLRepo struct {
	storedURL     *model.ShortURL
	shortCodeURL  *model.ShortURL // set this for GetByShortCode to return a hit
	count         int
	conflictUntil int
	createCalls   int
	lastLimit     int
	withMetaRows  []repository.ShortURLWithMeta
}

func (r *stubURLRepo) Create(_ context.Context, u *model.ShortURL) (*model.ShortURL, error) {
	r.createCalls++
	if r.createCalls <= r.conflictUntil {
		return nil, repository.ErrShortCodeConflict
	}
	out := *u
	out.CreatedAt = time.Now()
	return &out, nil
}

func (r *stubURLRepo) ListByUser(_ context.Context, _ uuid.UUID, limit, _ int) ([]model.ShortURL, error) {
	r.lastLimit = limit
	return nil, nil
}

func (r *stubURLRepo) ListByUserWithMeta(_ context.Context, _ uuid.UUID, limit, _ int) ([]repository.ShortURLWithMeta, error) {
	r.lastLimit = limit
	return r.withMetaRows, nil
}

func (r *stubURLRepo) CountByUser(_ context.Context, _ uuid.UUID) (int, error) {
	return r.count, nil
}

func (r *stubURLRepo) GetByID(_ context.Context, id uuid.UUID) (*model.ShortURL, error) {
	if r.storedURL != nil && r.storedURL.ID == id {
		return r.storedURL, nil
	}
	return nil, repository.ErrURLNotFound
}

func (r *stubURLRepo) GetByShortCode(_ context.Context, code string) (*model.ShortURL, error) {
	if r.shortCodeURL != nil && r.shortCodeURL.ShortCode == code {
		return r.shortCodeURL, nil
	}
	return nil, repository.ErrURLNotFound
}

func (r *stubURLRepo) SoftDelete(_ context.Context, id, _ uuid.UUID) error {
	if r.storedURL != nil && r.storedURL.ID == id {
		return nil
	}
	return repository.ErrURLNotFound
}

func (r *stubURLRepo) SoftDeleteByID(_ context.Context, _ uuid.UUID) error { return nil }
func (r *stubURLRepo) IncrementClick(_ context.Context, _ string) error     { return nil }

// stubMetaRepo satisfies repository.URLMetadataRepository.
type stubMetaRepo struct {
	createErr error
}

func (r *stubMetaRepo) Create(_ context.Context, urlID uuid.UUID) (*model.URLMetadata, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	return &model.URLMetadata{ID: uuid.New(), URLID: urlID, FetchStatus: model.FetchStatusPending}, nil
}

func (r *stubMetaRepo) GetByURLID(_ context.Context, _ uuid.UUID) (*model.URLMetadata, error) {
	return nil, repository.ErrURLMetadataNotFound
}

func (r *stubMetaRepo) UpdateFetched(_ context.Context, _ uuid.UUID, _ model.FetchResult) error {
	return nil
}

func (r *stubMetaRepo) ListPending(_ context.Context, _ int) ([]*model.URLMetadata, error) {
	return nil, nil
}

// stubUserPlanRepo satisfies repository.UserPlanRepository.
type stubUserPlanRepo struct {
	planCode string
}

func (r *stubUserPlanRepo) Create(_ context.Context, userID uuid.UUID, planCode string) (*model.UserPlan, error) {
	return &model.UserPlan{UserID: userID, PlanCode: planCode}, nil
}

func (r *stubUserPlanRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*model.UserPlan, error) {
	return &model.UserPlan{UserID: userID, PlanCode: r.planCode}, nil
}

func (r *stubUserPlanRepo) Update(_ context.Context, userID uuid.UUID, planCode string) (*model.UserPlan, error) {
	return &model.UserPlan{UserID: userID, PlanCode: planCode}, nil
}

// stubPlanRepo satisfies repository.PlanRepository.
type stubPlanRepo struct {
	plan *model.Plan
}

func (r *stubPlanRepo) GetByCode(_ context.Context, _ string) (*model.Plan, error) {
	if r.plan == nil {
		return nil, repository.ErrPlanNotFound
	}
	return r.plan, nil
}

func (r *stubPlanRepo) List(_ context.Context) ([]model.Plan, error) {
	if r.plan != nil {
		return []model.Plan{*r.plan}, nil
	}
	return nil, nil
}

func (r *stubPlanRepo) UpdateFeatures(_ context.Context, _ string, features map[string]bool) (*model.Plan, error) {
	if r.plan == nil {
		return nil, repository.ErrPlanNotFound
	}
	out := *r.plan
	out.Features = features
	return &out, nil
}
