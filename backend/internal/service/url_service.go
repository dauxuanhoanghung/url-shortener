package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/pkg/utils"
	"github.com/google/uuid"
)

var (
	ErrInvalidURL       = errors.New("invalid url")
	ErrPlanLimitReached = errors.New("plan limit reached")
	ErrURLForbidden     = errors.New("forbidden")
	ErrURLNotFound      = errors.New("url not found")
	ErrShortCodeRetry   = errors.New("failed to generate unique short code")
)

const (
	shortCodeLength  = 6
	shortCodeRetries = 5
	defaultListLimit = 50
)

var blockedSchemes = []string{"javascript:", "data:", "file:"}

type URLService interface {
	Create(ctx context.Context, userID uuid.UUID, req dto.CreateURLRequest, baseURL string) (*dto.URLResponse, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int, baseURL string) (*dto.ListURLResponse, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type urlService struct {
	repo         repository.URLRepository
	userPlanRepo repository.UserPlanRepository
	planRepo     repository.PlanRepository
}

func NewURLService(
	repo repository.URLRepository,
	userPlanRepo repository.UserPlanRepository,
	planRepo repository.PlanRepository,
) URLService {
	return &urlService{
		repo:         repo,
		userPlanRepo: userPlanRepo,
		planRepo:     planRepo,
	}
}

func (s *urlService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateURLRequest, baseURL string) (*dto.URLResponse, error) {
	if err := validateURL(req.OriginalURL); err != nil {
		return nil, err
	}

	// Fetch the user's plan to get the authoritative URL limit.
	userPlan, err := s.userPlanRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	plan, err := s.planRepo.GetByCode(ctx, userPlan.PlanCode)
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= int(plan.MaxURLs) {
		return nil, ErrPlanLimitReached
	}

	var created *model.ShortURL
	for attempt := 0; attempt < shortCodeRetries; attempt++ {
		code, err := utils.GenerateShortCode(shortCodeLength)
		if err != nil {
			return nil, err
		}
		persisted, err := s.repo.Create(ctx, &model.ShortURL{
			ID:          uuid.New(),
			UserID:      userID,
			ShortCode:   code,
			OriginalURL: req.OriginalURL,
			ClickCount:  0,
			CreatedAt:   time.Now(),
		})
		if err == nil {
			created = persisted
			break
		}
		if !errors.Is(err, repository.ErrShortCodeConflict) {
			return nil, err
		}
	}
	if created == nil {
		return nil, ErrShortCodeRetry
	}

	return toURLResponse(created, baseURL), nil
}

func (s *urlService) List(ctx context.Context, userID uuid.UUID, limit, offset int, baseURL string) (*dto.ListURLResponse, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}

	urls, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.URLResponse, 0, len(urls))
	for i := range urls {
		items = append(items, *toURLResponse(&urls[i], baseURL))
	}
	return &dto.ListURLResponse{URLs: items, Total: total}, nil
}

func (s *urlService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return ErrURLNotFound
		}
		return err
	}
	if existing.UserID != userID {
		return ErrURLForbidden
	}
	if err := s.repo.SoftDelete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return ErrURLNotFound
		}
		return err
	}
	return nil
}

func validateURL(raw string) error {
	lower := strings.ToLower(strings.TrimSpace(raw))
	for _, scheme := range blockedSchemes {
		if strings.HasPrefix(lower, scheme) {
			return ErrInvalidURL
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	if parsed.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func toURLResponse(u *model.ShortURL, baseURL string) *dto.URLResponse {
	return &dto.URLResponse{
		ID:          u.ID.String(),
		ShortCode:   u.ShortCode,
		ShortURL:    strings.TrimRight(baseURL, "/") + "/r/" + u.ShortCode,
		OriginalURL: u.OriginalURL,
		ClickCount:  u.ClickCount,
		CreatedAt:   u.CreatedAt,
		LastAccess:  u.LastAccessedAt,
	}
}
