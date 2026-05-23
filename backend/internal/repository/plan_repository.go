package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	sqlc "github.com/dauxuanhoanghung/url-shortener/internal/repository/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPlanNotFound = errors.New("plan not found")

type PlanRepository interface {
	GetByCode(ctx context.Context, code string) (*model.Plan, error)
	List(ctx context.Context) ([]model.Plan, error)
	UpdateFeatures(ctx context.Context, code string, features map[string]bool) (*model.Plan, error)
}

type planRepository struct {
	q *sqlc.Queries
}

func NewPlanRepository(db *pgxpool.Pool) PlanRepository {
	return &planRepository{q: sqlc.New(db)}
}

func (r *planRepository) GetByCode(ctx context.Context, code string) (*model.Plan, error) {
	row, err := r.q.GetPlanByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return planFromRow(row)
}

func (r *planRepository) List(ctx context.Context) ([]model.Plan, error) {
	rows, err := r.q.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	plans := make([]model.Plan, 0, len(rows))
	for _, row := range rows {
		p, err := planFromRow(row)
		if err != nil {
			return nil, err
		}
		plans = append(plans, *p)
	}
	return plans, nil
}

func (r *planRepository) UpdateFeatures(ctx context.Context, code string, features map[string]bool) (*model.Plan, error) {
	payload, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdatePlanFeatures(ctx, sqlc.UpdatePlanFeaturesParams{
		Code:     code,
		Features: payload,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return planFromRow(row)
}

func planFromRow(row sqlc.Plan) (*model.Plan, error) {
	features := map[string]bool{}
	if len(row.Features) > 0 {
		if err := json.Unmarshal(row.Features, &features); err != nil {
			return nil, err
		}
	}
	var apiLimit *int32
	if row.ApiRateLimitPerMin.Valid {
		v := row.ApiRateLimitPerMin.Int32
		apiLimit = &v
	}
	return &model.Plan{
		Code:                   row.Code,
		Name:                   row.Name,
		PriceCents:             row.PriceCents,
		MaxURLs:                row.MaxUrls,
		MaxDomains:             row.MaxDomains,
		MaxTeamMembers:         row.MaxTeamMembers,
		AnalyticsRetentionDays: row.AnalyticsRetentionDays,
		APIRateLimitPerMin:     apiLimit,
		Features:               features,
		CreatedAt:              row.CreatedAt.Time,
	}, nil
}
