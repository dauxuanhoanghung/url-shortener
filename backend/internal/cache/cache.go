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

// ErrNotSupported is returned by drivers that do not implement an optional
// operation (e.g. Increment on the in-memory fallback).
var ErrNotSupported = errors.New("cache: operation not supported by this driver")

// Cache is the driver-agnostic interface used by services.
// Values are raw bytes — serialization is the caller's responsibility
// (keeps the cache layer free of domain types).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	// Increment atomically increments key by 1 and sets ttl on first creation.
	// Returns the new value after increment.
	// Drivers that cannot safely implement distributed counters must return
	// ErrNotSupported — callers (e.g. rate-limit middleware) must treat that as
	// a hard error rather than silently allowing the request through.
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Close() error
}
