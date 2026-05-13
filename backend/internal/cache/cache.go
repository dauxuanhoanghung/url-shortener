package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned when the requested key does not exist.
// Drivers must translate their own "not found" sentinel into this error
// so callers can branch on it portably.
var ErrCacheMiss = errors.New("cache: miss")

// Cache is the driver-agnostic interface used by services.
// Values are raw bytes — serialization is the caller's responsibility
// (keeps the cache layer free of domain types).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
}
