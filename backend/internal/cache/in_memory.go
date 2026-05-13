package cache

import (
	"context"
	"sync"
	"time"
)

type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string]cacheItem
}

type cacheItem struct {
	value      []byte
	expiration time.Time // zero value means no expiration
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{store: make(map[string]cacheItem)}
}

func (c *InMemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	item, ok := c.store[key]
	c.mu.RUnlock()
	if !ok {
		return nil, ErrCacheMiss
	}
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		c.mu.Lock()
		delete(c.store, key)
		c.mu.Unlock()
		return nil, ErrCacheMiss
	}
	// Defensive copy so callers cannot mutate the stored slice.
	out := make([]byte, len(item.value))
	copy(out, item.value)
	return out, nil
}

func (c *InMemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	stored := make([]byte, len(value))
	copy(stored, value)
	item := cacheItem{value: stored}
	if ttl > 0 {
		item.expiration = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.store[key] = item
	c.mu.Unlock()
	return nil
}

func (c *InMemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.store, key)
	c.mu.Unlock()
	return nil
}

func (c *InMemoryCache) Close() error { return nil }
