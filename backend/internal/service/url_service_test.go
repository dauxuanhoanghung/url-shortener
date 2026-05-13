package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/internal/worker"
	"github.com/google/uuid"
)

const testBaseURL = "https://short.ly"

// ── helpers ───────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }

func newService(
	urlRepo repository.URLRepository,
	metaRepo repository.URLMetadataRepository,
	userPlanRepo repository.UserPlanRepository,
	planRepo repository.PlanRepository,
	w worker.MetadataWorker,
) URLService {
	return NewURLService(urlRepo, metaRepo, userPlanRepo, planRepo, w)
}

func freePlan() *model.Plan {
	return &model.Plan{Code: "free", MaxURLs: 5}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestURLService_Create_HappyPath(t *testing.T) {
	userID := uuid.New()
	urlRepo := &stubURLRepo{}
	metaRepo := &stubMetaRepo{}
	w := &captureWorker{}

	svc := newService(urlRepo,
		metaRepo,
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		w,
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
	if !w.submitted {
		t.Error("expected MetadataWorker.Submit to be called after Create")
	}
}

func TestURLService_Create_PlanLimitReached(t *testing.T) {
	userID := uuid.New()
	// Already at the limit (5 URLs, max 5).
	urlRepo := &stubURLRepo{count: 5}

	svc := newService(urlRepo,
		&stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&captureWorker{},
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
		&captureWorker{},
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
	// Fail with conflict on the first 3 attempts, succeed on the 4th.
	urlRepo := &stubURLRepo{conflictUntil: 3}

	svc := newService(urlRepo,
		&stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&captureWorker{},
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
	urlRepo := &stubURLRepo{conflictUntil: 99} // always conflict

	svc := newService(urlRepo,
		&stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&captureWorker{},
	)

	_, err := svc.Create(context.Background(), uuid.New(), dto.CreateURLRequest{OriginalURL: "https://example.com"}, testBaseURL)
	if !errors.Is(err, ErrShortCodeRetry) {
		t.Errorf("expected ErrShortCodeRetry, got %v", err)
	}
}

func TestURLService_Create_MetadataFailure_StillReturnsURL(t *testing.T) {
	// If the metadata row creation fails, Create should still succeed —
	// the worker just won't be submitted (best-effort).
	urlRepo := &stubURLRepo{}
	metaRepo := &stubMetaRepo{createErr: errors.New("db down")}
	w := &captureWorker{}

	svc := newService(urlRepo, metaRepo,
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		w,
	)

	resp, err := svc.Create(context.Background(), uuid.New(), dto.CreateURLRequest{OriginalURL: "https://example.com"}, testBaseURL)
	if err != nil {
		t.Fatalf("Create should succeed even when meta repo fails: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if w.submitted {
		t.Error("worker should NOT be submitted when metadata creation fails")
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
			Metadata: nil, // still pending
		},
	}

	urlRepo := &stubURLRepo{withMetaRows: rows, count: len(rows)}
	svc := newService(urlRepo, &stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&captureWorker{},
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
		&captureWorker{},
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
		&captureWorker{},
	)

	if err := svc.Delete(context.Background(), urlID, userID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestURLService_Delete_NotFound(t *testing.T) {
	svc := newService(&stubURLRepo{}, &stubMetaRepo{},
		&stubUserPlanRepo{planCode: "free"},
		&stubPlanRepo{plan: freePlan()},
		&captureWorker{},
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
		&captureWorker{},
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

func (r *stubURLRepo) GetByShortCode(_ context.Context, _ string) (*model.ShortURL, error) {
	return nil, repository.ErrURLNotFound
}

func (r *stubURLRepo) SoftDelete(_ context.Context, id, userID uuid.UUID) error {
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

// captureWorker satisfies worker.MetadataWorker.
type captureWorker struct {
	submitted bool
}

func (w *captureWorker) Submit(_ worker.MetadataJob)    { w.submitted = true }
func (w *captureWorker) Start(_ context.Context)        {}
func (w *captureWorker) Stop()                          {}
