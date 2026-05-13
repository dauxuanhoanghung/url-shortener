package cache

import (
	"context"
	"fmt"
)

type Config struct {
	Driver   string // "redis", "valkey", "memory"
	Addr     string
	Fallback bool // if true, wrap primary with an in-memory fallback
}

// New builds a Cache from Config. When Fallback is enabled and the primary
// is a remote driver, the result is a chain: primary first, in-memory second.
// See fallback.go for the semantics.
func New(ctx context.Context, cfg Config) (Cache, error) {
	primary, err := buildPrimary(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Fallback && cfg.Driver != "memory" {
		return NewChain(primary, NewInMemoryCache()), nil
	}
	return primary, nil
}

func buildPrimary(ctx context.Context, cfg Config) (Cache, error) {
	switch cfg.Driver {
	case "redis", "valkey":
		return NewRedisCache(ctx, cfg.Addr)
	case "memory":
		return NewInMemoryCache(), nil
	default:
		return nil, fmt.Errorf("unsupported cache driver: %q", cfg.Driver)
	}
}
