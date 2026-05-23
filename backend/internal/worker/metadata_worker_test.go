package worker

import (
	"context"
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

// ── process() — end-to-end with fakes ────────────────────────────────────────

func newWorkerWith(fetcher *fakeFetcher, metaRepo *fakeMetaRepo, urlRepo *fakeURLRepo, notifier *fakeNotifier) *metadataWorker {
	log, _ := zap.NewDevelopment()
	return &metadataWorker{
		jobs:     make(chan MetadataJob, 4),
		metaRepo: metaRepo,
		urlRepo:  urlRepo,
		cache:    cache.NewInMemoryCache(),
		notifier: notifier,
		fetcher:  fetcher,
		log:      log,
	}
}

func TestProcess_200_StoresOK_NotifiesMetadataUpdated(t *testing.T) {
	title := "Test Page"
	fetcher := &fakeFetcher{result: model.FetchResult{
		FetchStatus: model.FetchStatusOK,
		HTTPStatus:  200,
		Title:       &title,
	}}
	metaRepo := &fakeMetaRepo{}
	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}

	w := newWorkerWith(fetcher, metaRepo, urlRepo, notifier)

	userID := uuid.New()
	job := MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "abc123",
		UserID:     userID,
		TargetURL:  "https://example.com",
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
	if !notifier.notifyCalled {
		t.Error("expected metadata_updated SSE event on 200")
	}
	if notifier.lastEvent.Type != "metadata_updated" {
		t.Errorf("event type: got %q want metadata_updated", notifier.lastEvent.Type)
	}
	if notifier.lastUserID != userID {
		t.Errorf("notify userID: got %v want %v", notifier.lastUserID, userID)
	}
}

func TestProcess_404_SoftDeletes_NotifiesUrlDeleted(t *testing.T) {
	fetcher := &fakeFetcher{result: model.FetchResult{
		FetchStatus: model.FetchStatusFailed,
		HTTPStatus:  404,
	}}
	metaRepo := &fakeMetaRepo{}
	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}
	appCache := cache.NewInMemoryCache()

	// Pre-seed cache to verify it gets invalidated.
	appCache.Set(context.Background(), "url:abc123", []byte("cached"), time.Minute)

	log, _ := zap.NewDevelopment()
	w := &metadataWorker{
		jobs:     make(chan MetadataJob, 1),
		metaRepo: metaRepo,
		urlRepo:  urlRepo,
		cache:    appCache,
		notifier: notifier,
		fetcher:  fetcher,
		log:      log,
	}

	userID := uuid.New()
	job := MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "abc123",
		UserID:     userID,
		TargetURL:  "https://example.com",
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
	fetcher := &fakeFetcher{result: model.FetchResult{
		FetchStatus: model.FetchStatusFailed,
		HTTPStatus:  500,
	}}
	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}

	w := newWorkerWith(fetcher, &fakeMetaRepo{}, urlRepo, notifier)
	w.process(context.Background(), MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "xyz",
		UserID:     uuid.New(),
		TargetURL:  "https://example.com",
	})

	if urlRepo.softDeleteByIDCalled {
		t.Error("should NOT soft-delete on 5xx (transient)")
	}
	if notifier.notifyCalled {
		t.Error("should NOT notify on 5xx")
	}
}

func TestProcess_NetworkError_SoftDeletes_Notifies(t *testing.T) {
	fetcher := &fakeFetcher{result: model.FetchResult{
		FetchStatus: model.FetchStatusFailed,
		HTTPStatus:  0,
	}}
	urlRepo := &fakeURLRepo{}
	notifier := &fakeNotifier{}

	w := newWorkerWith(fetcher, &fakeMetaRepo{}, urlRepo, notifier)
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
	title := "Goroutine Page"
	fetcher := &fakeFetcher{result: model.FetchResult{
		FetchStatus: model.FetchStatusOK,
		HTTPStatus:  200,
		Title:       &title,
	}}
	metaRepo := &fakeMetaRepo{}
	log, _ := zap.NewDevelopment()

	wkr := NewMetadataWorker(MetadataWorkerConfig{
		MetaRepo: metaRepo,
		URLRepo:  &fakeURLRepo{},
		Cache:    cache.NewInMemoryCache(),
		Notifier: &fakeNotifier{},
		Fetcher:  fetcher,
		Logger:   log,
		Workers:  1,
	})
	wkr.Start(context.Background())

	wkr.Submit(MetadataJob{
		MetadataID: uuid.New(),
		URLID:      uuid.New(),
		ShortCode:  "abc",
		UserID:     uuid.New(),
		TargetURL:  "https://example.com",
	})

	wkr.Stop()

	if metaRepo.updateCalled == 0 {
		t.Error("expected UpdateFetched to be called by goroutine worker")
	}
}

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeFetcher struct {
	result model.FetchResult
}

func (f *fakeFetcher) Fetch(_ string) model.FetchResult { return f.result }

type fakeMetaRepo struct {
	mu           sync.Mutex
	updateCalled int
	lastStatus   string
}

func (f *fakeMetaRepo) Create(_ context.Context, urlID uuid.UUID) (*model.URLMetadata, error) {
	return &model.URLMetadata{ID: uuid.New(), URLID: urlID, FetchStatus: model.FetchStatusPending}, nil
}
func (f *fakeMetaRepo) GetByURLID(_ context.Context, _ uuid.UUID) (*model.URLMetadata, error) {
	return nil, nil
}
func (f *fakeMetaRepo) UpdateFetched(_ context.Context, _ uuid.UUID, result model.FetchResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateCalled++
	f.lastStatus = string(result.FetchStatus)
	return nil
}
func (f *fakeMetaRepo) ListPending(_ context.Context, _ int) ([]*model.URLMetadata, error) {
	return nil, nil
}

type fakeURLRepo struct {
	mu                   sync.Mutex
	softDeleteByIDCalled bool
}

func (f *fakeURLRepo) Create(_ context.Context, url *model.ShortURL) (*model.ShortURL, error) {
	return url, nil
}
func (f *fakeURLRepo) ListByUser(_ context.Context, _ uuid.UUID, _, _ int) ([]model.ShortURL, error) {
	return nil, nil
}
func (f *fakeURLRepo) ListByUserWithMeta(_ context.Context, _ uuid.UUID, _, _ int) ([]repository.ShortURLWithMeta, error) {
	return nil, nil
}
func (f *fakeURLRepo) CountByUser(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (f *fakeURLRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.ShortURL, error) {
	return nil, nil
}
func (f *fakeURLRepo) GetByShortCode(_ context.Context, _ string) (*model.ShortURL, error) {
	return nil, nil
}
func (f *fakeURLRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (f *fakeURLRepo) SoftDeleteByID(_ context.Context, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.softDeleteByIDCalled = true
	return nil
}
func (f *fakeURLRepo) IncrementClick(_ context.Context, _ string) error { return nil }

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
