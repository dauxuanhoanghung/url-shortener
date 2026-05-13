package service

import (
	"context"
	"errors"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
)

const (
	redirectCacheTTL    = 24 * time.Hour
	redirectCachePrefix = "url:"
)

type RedirectService interface {
	Resolve(ctx context.Context, shortCode string) (string, error)
}

type redirectService struct {
	repo  repository.URLRepository
	cache cache.Cache
}

func NewRedirectService(repo repository.URLRepository, c cache.Cache) RedirectService {
	return &redirectService{repo: repo, cache: c}
}

func (s *redirectService) Resolve(ctx context.Context, shortCode string) (string, error) {
	key := redirectCachePrefix + shortCode

	if v, err := s.cache.Get(ctx, key); err == nil {
		s.recordClick(shortCode)
		return string(v), nil
	}

	u, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return "", ErrURLNotFound
		}
		return "", err
	}

	_ = s.cache.Set(ctx, key, []byte(u.OriginalURL), redirectCacheTTL)
	s.recordClick(shortCode)

	return u.OriginalURL, nil
}

func (s *redirectService) recordClick(shortCode string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.repo.IncrementClick(ctx, shortCode)
	}()
}
