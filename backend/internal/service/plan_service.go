package service

import (
	"context"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
)

type PlanService interface {
	List(ctx context.Context) ([]dto.PlanResponse, error)
}

type planService struct {
	repo repository.PlanRepository
}

func NewPlanService(repo repository.PlanRepository) PlanService {
	return &planService{repo: repo}
}

func (s *planService) List(ctx context.Context) ([]dto.PlanResponse, error) {
	plans, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.PlanResponse, 0, len(plans))
	for _, p := range plans {
		out = append(out, dto.PlanResponse{
			Code:                   p.Code,
			Name:                   p.Name,
			PriceCents:             p.PriceCents,
			MaxURLs:                p.MaxURLs,
			MaxDomains:             p.MaxDomains,
			MaxTeamMembers:         p.MaxTeamMembers,
			AnalyticsRetentionDays: p.AnalyticsRetentionDays,
			APIRateLimitPerMin:     p.APIRateLimitPerMin,
			Features:               p.Features,
		})
	}
	return out, nil
}
