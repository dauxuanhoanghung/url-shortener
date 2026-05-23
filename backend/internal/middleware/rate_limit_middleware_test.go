package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

func newRateLimitRouter(limiter *ratelimit.Limiter, limit int64, setUserID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setUserID != "" {
			c.Set("userID", setUserID)
		}
		c.Next()
	})
	r.GET("/ping", RateLimit(limiter, limit, time.Minute), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimit_UnderLimit_AllowsAllCalls(t *testing.T) {
	limiter := ratelimit.New(newCountingCache())
	r := newRateLimitRouter(limiter, 5, "")

	for i := range 5 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("call %d: got %d want 200", i+1, w.Code)
		}
	}
}

func TestRateLimit_OverLimit_Returns429(t *testing.T) {
	limiter := ratelimit.New(newCountingCache())
	r := newRateLimitRouter(limiter, 2, "")

	for i := range 2 {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("call %d should pass: %d", i+1, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd call: got %d want 429", w.Code)
	}
	if !strings.Contains(w.Body.String(), "RATE_LIMIT_EXCEEDED") {
		t.Errorf("expected RATE_LIMIT_EXCEEDED, body=%s", w.Body.String())
	}
}

func TestRateLimit_KeyedByUserID_WhenAuthenticated(t *testing.T) {
	// When userID is set (i.e. AuthRequired has run), the counter must be
	// scoped to the user rather than the IP. Two different users sharing an
	// IP should each get their own quota.
	c := newCountingCache()
	limiter := ratelimit.New(c)

	// User A — fill the bucket.
	rA := newRateLimitRouter(limiter, 1, "user-a")
	rA.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
	// Second hit from A should fail.
	wA2 := httptest.NewRecorder()
	rA.ServeHTTP(wA2, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if wA2.Code != http.StatusTooManyRequests {
		t.Errorf("user A second call should be 429, got %d", wA2.Code)
	}

	// User B's first hit should succeed despite same IP.
	rB := newRateLimitRouter(limiter, 1, "user-b")
	wB := httptest.NewRecorder()
	rB.ServeHTTP(wB, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if wB.Code != http.StatusOK {
		t.Errorf("user B first call should pass: got %d", wB.Code)
	}
}

func TestRateLimit_CacheError_Returns500(t *testing.T) {
	// If the cache cannot increment (e.g. in-memory primary returns
	// ErrNotSupported, or Redis is down), we must fail closed — never let
	// the request through silently.
	limiter := ratelimit.New(&unsupportedCache{})
	r := newRateLimitRouter(limiter, 5, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d want 500", w.Code)
	}
}

// ── fakes ─────────────────────────────────────────────────────────────────────

// countingCache is a minimal Cache that supports atomic Increment so we can
// drive the rate-limit logic without touching Redis.
type countingCache struct {
	mu       sync.Mutex
	counters map[string]int64
}

func newCountingCache() *countingCache {
	return &countingCache{counters: map[string]int64{}}
}

func (c *countingCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, cache.ErrCacheMiss
}
func (c *countingCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (c *countingCache) Delete(_ context.Context, _ string) error { return nil }
func (c *countingCache) Increment(_ context.Context, key string, _ time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[key]++
	return c.counters[key], nil
}
func (c *countingCache) Close() error { return nil }

// unsupportedCache mirrors what the in-memory fallback driver returns:
// rate limiting must fail closed in this state, never allow.
type unsupportedCache struct{}

func (unsupportedCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, cache.ErrCacheMiss
}
func (unsupportedCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (unsupportedCache) Delete(_ context.Context, _ string) error { return nil }
func (unsupportedCache) Increment(_ context.Context, _ string, _ time.Duration) (int64, error) {
	return 0, cache.ErrNotSupported
}
func (unsupportedCache) Close() error { return nil }
