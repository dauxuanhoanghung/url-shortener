package cache

import (
	"context"
	"errors"
	"time"
)

// Chain is a facade that composes caches in priority order.
//
// Read path:  try caches[0], caches[1], ... until one returns a value or all
//             return ErrCacheMiss. Driver errors (not misses) skip to the next
//             cache — the remote may be down, but a local fallback still works.
// Write path: Set/Delete fan out to every cache. Writes to the primary are
//             authoritative; secondary write errors are swallowed (logged by
//             the caller if desired) because a failed fallback write must not
//             break the request path.
//
// Typical use: Chain{redis, in-memory}. If Redis is reachable every entry is
// also mirrored into the process for cheap re-reads; if Redis is down, reads
// transparently fall through to the local copy.
type Chain struct {
	caches []Cache
}

func NewChain(caches ...Cache) *Chain {
	return &Chain{caches: caches}
}

func (c *Chain) Get(ctx context.Context, key string) ([]byte, error) {
	var lastErr error
	for _, inner := range c.caches {
		v, err := inner.Get(ctx, key)
		if err == nil {
			return v, nil
		}
		if !errors.Is(err, ErrCacheMiss) {
			lastErr = err
			continue
		}
		lastErr = ErrCacheMiss
	}
	return nil, lastErr
}

func (c *Chain) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	var primaryErr error
	for i, inner := range c.caches {
		err := inner.Set(ctx, key, value, ttl)
		if i == 0 {
			primaryErr = err
		}
	}
	return primaryErr
}

func (c *Chain) Delete(ctx context.Context, key string) error {
	var primaryErr error
	for i, inner := range c.caches {
		err := inner.Delete(ctx, key)
		if i == 0 {
			primaryErr = err
		}
	}
	return primaryErr
}

// Increment delegates only to the primary cache (caches[0]).
// Rate-limit counters must not fan-out to in-memory replicas — each replica
// would hold an independent counter, breaking the global limit.
func (c *Chain) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if len(c.caches) == 0 {
		return 0, ErrNotSupported
	}
	return c.caches[0].Increment(ctx, key, ttl)
}

func (c *Chain) Close() error {
	var firstErr error
	for _, inner := range c.caches {
		if err := inner.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
