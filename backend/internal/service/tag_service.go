package service

import (
	"context"
	"errors"
	"strings"

	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrTagLimitExceeded = errors.New("tag limit exceeded")
	ErrInvalidTag       = errors.New("invalid tag")
)

const (
	maxTagsPerURL = 20
	maxTagLength  = 50
)

// TagService owns all tag business rules: normalization, limits and
// ownership checks. URL code never touches the tag repository directly.
type TagService interface {
	// SetTags normalizes and persists tags for a URL the caller already owns
	// (e.g. right after creating it). Returns the normalized list.
	SetTags(ctx context.Context, urlID uuid.UUID, tags []string) ([]string, error)
	// UpdateTags verifies ownership, then replaces the URL's tag list.
	UpdateTags(ctx context.Context, urlID, userID uuid.UUID, tags []string) ([]string, error)
	// TagsForURLs batch-fetches tags for a set of URLs (avoids N+1 in list).
	TagsForURLs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID][]string, error)
}

type tagService struct {
	tagRepo repository.TagRepository
	urlRepo repository.URLRepository
}

func NewTagService(tagRepo repository.TagRepository, urlRepo repository.URLRepository) TagService {
	return &tagService{tagRepo: tagRepo, urlRepo: urlRepo}
}

func (s *tagService) SetTags(ctx context.Context, urlID uuid.UUID, tags []string) ([]string, error) {
	normalized, err := normalizeTags(tags)
	if err != nil {
		return nil, err
	}
	if err := s.tagRepo.SetTagsForURL(ctx, urlID, normalized); err != nil {
		if errors.Is(err, repository.ErrTagLimitExceeded) {
			return nil, ErrTagLimitExceeded
		}
		return nil, err
	}
	return normalized, nil
}

func (s *tagService) UpdateTags(ctx context.Context, urlID, userID uuid.UUID, tags []string) ([]string, error) {
	existing, err := s.urlRepo.GetByID(ctx, urlID)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}
	if existing.UserID != userID {
		return nil, ErrURLForbidden
	}
	return s.SetTags(ctx, urlID, tags)
}

func (s *tagService) TagsForURLs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(urlIDs) == 0 {
		return map[uuid.UUID][]string{}, nil
	}
	return s.tagRepo.ListForURLs(ctx, urlIDs)
}

// normalizeTags trims whitespace, drops empties, dedupes case-insensitively
// (first casing wins) and enforces per-tag and per-URL limits.
func normalizeTags(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len(t) > maxTagLength {
			return nil, ErrInvalidTag
		}
		lower := strings.ToLower(t)
		if _, dup := seen[lower]; dup {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, t)
	}
	if len(out) > maxTagsPerURL {
		return nil, ErrTagLimitExceeded
	}
	return out, nil
}
