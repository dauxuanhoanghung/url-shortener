package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
)

func TestRedirectService_Resolve_CacheHit(t *testing.T) {
	c := newFakeCache()
	_ = c.Set(context.Background(), "url:abc123", []byte("https://target.example.com"), time.Hour)

	// repo stays empty — a cache hit must not consult the database.
	svc := NewRedirectService(&stubURLRepo{}, c)

	got, err := svc.Resolve(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://target.example.com" {
		t.Errorf("got %q want target", got)
	}
}

func TestRedirectService_Resolve_CacheMiss_FillsCache(t *testing.T) {
	repo := &stubURLRepo{
		shortCodeURL: &model.ShortURL{
			ShortCode:   "miss01",
			OriginalURL: "https://from-db.example.com",
		},
	}
	c := newFakeCache()
	svc := NewRedirectService(repo, c)

	got, err := svc.Resolve(context.Background(), "miss01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://from-db.example.com" {
		t.Errorf("got %q", got)
	}
	// Verify the cache was warmed for the next request.
	v, err := c.Get(context.Background(), "url:miss01")
	if err != nil {
		t.Fatalf("expected cache to be filled after miss, got: %v", err)
	}
	if string(v) != "https://from-db.example.com" {
		t.Errorf("cache value: got %q", v)
	}
}

func TestRedirectService_Resolve_NotFound(t *testing.T) {
	svc := NewRedirectService(&stubURLRepo{}, newFakeCache())
	_, err := svc.Resolve(context.Background(), "nope")
	if !errors.Is(err, ErrURLNotFound) {
		t.Errorf("expected ErrURLNotFound, got %v", err)
	}
}

func TestRedirectService_Resolve_CacheErrorFallsBackToDB(t *testing.T) {
	// Cache errors (non-miss) must not block the redirect — degraded
	// operation is the goal, per docs/23-backend-architecture.md §4.7.
	repo := &stubURLRepo{
		shortCodeURL: &model.ShortURL{
			ShortCode:   "deg01",
			OriginalURL: "https://degraded.example.com",
		},
	}
	svc := NewRedirectService(repo, &errCache{err: errors.New("redis down")})

	got, err := svc.Resolve(context.Background(), "deg01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://degraded.example.com" {
		t.Errorf("got %q", got)
	}
}

// ── fakes ─────────────────────────────────────────────────────────────────────

// fakeCache is a minimal cache.Cache for tests. We don't reuse
// cache.InMemoryCache to keep this test free of cross-package coupling.
type fakeCache struct {
	mu    sync.Mutex
	store map[string][]byte
}

func newFakeCache() *fakeCache {
	return &fakeCache{store: map[string][]byte{}}
}

func (c *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.store[key]; ok {
		return v, nil
	}
	return nil, cache.ErrCacheMiss
}

func (c *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = append([]byte(nil), value...)
	return nil
}

func (c *fakeCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
	return nil
}

func (c *fakeCache) Increment(_ context.Context, _ string, _ time.Duration) (int64, error) {
	return 0, cache.ErrNotSupported
}

func (c *fakeCache) Close() error { return nil }

// errCache returns the configured error on every operation. Used to assert
// graceful fallback when Redis is unhealthy.
type errCache struct{ err error }

func (c *errCache) Get(_ context.Context, _ string) ([]byte, error) { return nil, c.err }
func (c *errCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return c.err
}
func (c *errCache) Delete(_ context.Context, _ string) error { return c.err }
func (c *errCache) Increment(_ context.Context, _ string, _ time.Duration) (int64, error) {
	return 0, c.err
}
func (c *errCache) Close() error { return nil }
