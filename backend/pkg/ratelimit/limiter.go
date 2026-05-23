package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
)

// Limiter is a generic distributed rate limiter backed by an atomic cache
// counter (Redis INCR). It is safe for concurrent use and can be called from
// middleware, services, or any other layer.
type Limiter struct {
	cache cache.Cache
}

func New(c cache.Cache) *Limiter {
	return &Limiter{cache: c}
}

// Allow reports whether the caller identified by key is within the limit for
// the current window. It increments the counter on every call, so each call
// consumes one unit of quota regardless of the return value.
//
// Returns (true, nil) when the request is allowed.
// Returns (false, nil) when the limit is exceeded.
// Returns (false, err) when the underlying cache is unavailable or does not
// support atomic increments (e.g. in-memory fallback).
func (l *Limiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	count, err := l.cache.Increment(ctx, key, window)
	if err != nil {
		return false, err
	}
	return count <= limit, nil
}

// Key returns a canonical rate-limit key scoped to a namespace and identity.
// Use this to keep key formatting consistent across callers.
//
//	ratelimit.Key("login", "192.168.1.1")      → "rate_limit:login:192.168.1.1"
//	ratelimit.Key("url_create", "user:abc-123") → "rate_limit:url_create:user:abc-123"
func Key(namespace, identity string) string {
	return fmt.Sprintf("rate_limit:%s:%s", namespace, identity)
}
