package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/internal/sse"
	"github.com/google/uuid"
)

// ── parse helpers ────────────────────────────────────────────────────────────

func TestParseMetadata_Title(t *testing.T) {
	body := []byte(`<html><head><title>Hello World</title></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Title == nil || *r.Title != "Hello World" {
		t.Errorf("title: got %v want %q", r.Title, "Hello World")
	}
}

func TestParseMetadata_Description(t *testing.T) {
	body := []byte(`<html><head><meta name="description" content="Great page"></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Description == nil || *r.Description != "Great page" {
		t.Errorf("description: got %v want %q", r.Description, "Great page")
	}
}

func TestParseMetadata_OgImage(t *testing.T) {
	body := []byte(`<html><head><meta property="og:image" content="https://example.com/img.png"></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.OgImage == nil || *r.OgImage != "https://example.com/img.png" {
		t.Errorf("og:image: got %v want %q", r.OgImage, "https://example.com/img.png")
	}
}

func TestParseMetadata_Favicon(t *testing.T) {
	body := []byte(`<html><head><link rel="icon" href="/favicon.ico"></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.FaviconURL == nil || *r.FaviconURL != "/favicon.ico" {
		t.Errorf("favicon: got %v want %q", r.FaviconURL, "/favicon.ico")
	}
}

func TestParseMetadata_AllFields(t *testing.T) {
	body := []byte(`<!DOCTYPE html><html><head>
		<title>  My Page  </title>
		<meta name="description" content="A description">
		<meta property="og:image" content="https://img.example.com/og.png">
		<link rel="shortcut icon" href="/static/favicon.ico">
	</head><body></body></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Title == nil || *r.Title != "My Page" {
		t.Errorf("title: got %v", r.Title)
	}
	if r.Description == nil || *r.Description != "A description" {
		t.Errorf("description: got %v", r.Description)
	}
	if r.OgImage == nil || *r.OgImage != "https://img.example.com/og.png" {
		t.Errorf("og:image: got %v", r.OgImage)
	}
	if r.FaviconURL == nil || *r.FaviconURL != "/static/favicon.ico" {
		t.Errorf("favicon: got %v", r.FaviconURL)
	}
}

func TestParseMetadata_EmptyBody(t *testing.T) {
	var r model.FetchResult
	parseMetadata([]byte(""), &r)
	if r.Title != nil || r.Description != nil || r.OgImage != nil || r.FaviconURL != nil {
		t.Error("expected all fields nil for empty body")
	}
}

func TestParseMetadata_MissingFields(t *testing.T) {
	body := []byte(`<html><head></head><body>Just text</body></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Title != nil {
		t.Errorf("expected nil title, got %q", *r.Title)
	}
}

// ── fetch() via httptest server ──────────────────────────────────────────────

func newTestWorker(t *testing.T) *metadataWorker {
	t.Helper()
	log, _ := zap.NewDevelopment()
	return &metadataWorker{
		jobs:     make(chan MetadataJob, 1),
		metaRepo: &fakeMetaRepo{},
		urlRepo:  &fakeURLRepo{},
		cache:    cache.NewInMemoryCache(),
		notifier: &fakeNotifier{},
		log:      log,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func TestFetch_200_ParsesMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Test Page</title></head></html>`))
	}))
	defer srv.Close()

	w := newTestWorker(t)
	result := w.fetch(srv.URL)

	if result.FetchStatus != model.FetchStatusOK {
		t.Errorf("fetch_status: got %q want %q", result.FetchStatus, model.FetchStatusOK)
	}
	if result.HTTPStatus != 200 {
		t.Errorf("http_status: got %d want 200", result.HTTPStatus)
	}
	if result.Title == nil || *result.Title != "Test Page" {
		t.Errorf("title: got %v want %q", result.Title, "Test Page")
	}
}

func TestFetch_404_ReturnsFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	w := newTestWorker(t)
	result := w.fetch(srv.URL)

	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
	if result.HTTPStatus != 404 {
		t.Errorf("http_status: got %d want 404", result.HTTPStatus)
	}
}

func TestFetch_500_ReturnsFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	w := newTestWorker(t)
	result := w.fetch(srv.URL)

	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
	if result.HTTPStatus != 500 {
		t.Errorf("http_status: got %d want 500", result.HTTPStatus)
	}
}

func TestFetch_NetworkError_ReturnsFailed(t *testing.T) {
	w := newTestWorker(t)
	// Use an address that refuses connections immediately.
	result := w.fetch("http://127.0.0.1:1")

	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
}

func TestFetch_InvalidURL_ReturnsFailed(t *testing.T) {
	w := newTestWorker(t)
	result := w.fetch("://not-a-url")

	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
}

func TestFetch_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := newTestWorker(t)
	w.fetch(srv.URL)

	if gotUA != "urlshortener-bot/1.0" {
		t.Errorf("user-agent: got %q want %q", gotUA, "urlshortener-bot/1.0")
	}
}

// ── process() — end-to-end with fakes ────────────────────────────────────────

func TestProcess_200_StoresOK_NoDelete_NoNotify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>OK</title></head></html>`))
	}))
	defer srv.Close()

	metaRepo := &fakeMetaRepo{}
	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}
	log, _ := zap.NewDevelopment()

	w := &metadataWorker{
		jobs:     make(chan MetadataJob, 1),
		metaRepo: metaRepo,
		urlRepo:  urlRepo,
		cache:    cache.NewInMemoryCache(),
		notifier: notifier,
		log:      log,
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	job := MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "abc123",
		UserID:     uuid.New(),
		TargetURL:  srv.URL,
	}
	w.process(context.Background(), job)

	if metaRepo.updateCalled == 0 {
		t.Error("expected UpdateFetched to be called")
	}
	if metaRepo.lastStatus != string(model.FetchStatusOK) {
		t.Errorf("fetch_status: got %q want ok", metaRepo.lastStatus)
	}
	if urlRepo.softDeleteByIDCalled {
		t.Error("should NOT soft-delete on 200")
	}
	if notifier.notifyCalled {
		t.Error("should NOT notify on 200")
	}
}

func TestProcess_404_SoftDeletes_Notifies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	metaRepo := &fakeMetaRepo{}
	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}
	log, _ := zap.NewDevelopment()
	appCache := cache.NewInMemoryCache()

	// Pre-seed cache to verify it gets invalidated.
	appCache.Set(context.Background(), "url:abc123", []byte("cached"), time.Minute)

	w := &metadataWorker{
		jobs:     make(chan MetadataJob, 1),
		metaRepo: metaRepo,
		urlRepo:  urlRepo,
		cache:    appCache,
		notifier: notifier,
		log:      log,
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	userID := uuid.New()
	job := MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "abc123",
		UserID:     userID,
		TargetURL:  srv.URL,
	}
	w.process(context.Background(), job)

	if !urlRepo.softDeleteByIDCalled {
		t.Error("expected SoftDeleteByID to be called on 404")
	}
	if !notifier.notifyCalled {
		t.Error("expected Notify to be called on 404")
	}
	if notifier.lastUserID != userID {
		t.Errorf("notify userID: got %v want %v", notifier.lastUserID, userID)
	}
	if notifier.lastEvent.Type != "url_deleted" {
		t.Errorf("event type: got %q want url_deleted", notifier.lastEvent.Type)
	}
	// Cache entry should be gone.
	_, err := appCache.Get(context.Background(), "url:abc123")
	if err == nil {
		t.Error("expected cache key to be deleted after 404")
	}
}

func TestProcess_500_NoDelete_NoNotify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}
	log, _ := zap.NewDevelopment()

	w := &metadataWorker{
		jobs:     make(chan MetadataJob, 1),
		metaRepo: &fakeMetaRepo{},
		urlRepo:  urlRepo,
		cache:    cache.NewInMemoryCache(),
		notifier: notifier,
		log:      log,
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	w.process(context.Background(), MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "xyz",
		UserID:     uuid.New(),
		TargetURL:  srv.URL,
	})

	if urlRepo.softDeleteByIDCalled {
		t.Error("should NOT soft-delete on 5xx (transient)")
	}
	if notifier.notifyCalled {
		t.Error("should NOT notify on 5xx")
	}
}

func TestProcess_NetworkError_SoftDeletes_Notifies(t *testing.T) {
	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}
	log, _ := zap.NewDevelopment()

	w := &metadataWorker{
		jobs:     make(chan MetadataJob, 1),
		metaRepo: &fakeMetaRepo{},
		urlRepo:  urlRepo,
		cache:    cache.NewInMemoryCache(),
		notifier: notifier,
		log:      log,
		client:   &http.Client{Timeout: 100 * time.Millisecond},
	}

	w.process(context.Background(), MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "gone",
		UserID:     uuid.New(),
		TargetURL:  "http://127.0.0.1:1",
	})

	if !urlRepo.softDeleteByIDCalled {
		t.Error("expected SoftDeleteByID on network error")
	}
	if !notifier.notifyCalled {
		t.Error("expected Notify on network error")
	}
}

// ── Submit: full goroutine pipeline ──────────────────────────────────────────

func TestWorker_Submit_ProcessesJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Goroutine Page</title></head></html>`))
	}))
	defer srv.Close()

	metaRepo := &fakeMetaRepo{}
	log, _ := zap.NewDevelopment()

	wkr := NewMetadataWorker(MetadataWorkerConfig{
		MetaRepo: metaRepo,
		URLRepo:  &fakeURLRepo{},
		Cache:    cache.NewInMemoryCache(),
		Notifier: &fakeNotifier{},
		Logger:   log,
		Workers:  1,
	})
	wkr.Start(context.Background())

	wkr.Submit(MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "abc",
		UserID:     uuid.New(),
		TargetURL:  srv.URL,
	})

	wkr.Stop()

	if metaRepo.updateCalled == 0 {
		t.Error("expected UpdateFetched to be called by goroutine worker")
	}
}

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeMetaRepo struct {
	mu           sync.Mutex
	updateCalled int
	lastStatus   string
}

func (f *fakeMetaRepo) Create(ctx context.Context, urlID uuid.UUID) (*model.URLMetadata, error) {
	return &model.URLMetadata{ID: uuid.New(), URLID: urlID, FetchStatus: model.FetchStatusPending}, nil
}
func (f *fakeMetaRepo) GetByURLID(ctx context.Context, urlID uuid.UUID) (*model.URLMetadata, error) {
	return nil, nil
}
func (f *fakeMetaRepo) UpdateFetched(ctx context.Context, id uuid.UUID, result model.FetchResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateCalled++
	f.lastStatus = string(result.FetchStatus)
	return nil
}
func (f *fakeMetaRepo) ListPending(ctx context.Context, limit int) ([]*model.URLMetadata, error) {
	return nil, nil
}

type fakeURLRepo struct {
	mu                   sync.Mutex
	softDeleteByIDCalled bool
}

func (f *fakeURLRepo) Create(ctx context.Context, url *model.ShortURL) (*model.ShortURL, error) {
	return url, nil
}
func (f *fakeURLRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.ShortURL, error) {
	return nil, nil
}
func (f *fakeURLRepo) ListByUserWithMeta(ctx context.Context, userID uuid.UUID, limit, offset int) ([]repository.ShortURLWithMeta, error) {
	return nil, nil
}
func (f *fakeURLRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (f *fakeURLRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.ShortURL, error) {
	return nil, nil
}
func (f *fakeURLRepo) GetByShortCode(ctx context.Context, shortCode string) (*model.ShortURL, error) {
	return nil, nil
}
func (f *fakeURLRepo) SoftDelete(ctx context.Context, id, userID uuid.UUID) error { return nil }
func (f *fakeURLRepo) SoftDeleteByID(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.softDeleteByIDCalled = true
	return nil
}
func (f *fakeURLRepo) IncrementClick(ctx context.Context, shortCode string) error { return nil }

type fakeNotifier struct {
	mu           sync.Mutex
	notifyCalled bool
	lastUserID   uuid.UUID
	lastEvent    sse.Event
}

func (f *fakeNotifier) Notify(userID uuid.UUID, event sse.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifyCalled = true
	f.lastUserID = userID
	f.lastEvent = event
}
