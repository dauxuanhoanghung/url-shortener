package repository

import (
	"context"
	"errors"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	sqlc "github.com/dauxuanhoanghung/url-shortener/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserPlanNotFound = errors.New("user plan not found")

type UserPlanRepository interface {
	Create(ctx context.Context, userID uuid.UUID, planCode string) (*model.UserPlan, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.UserPlan, error)
	Update(ctx context.Context, userID uuid.UUID, planCode string) (*model.UserPlan, error)
}

type userPlanRepository struct {
	q *sqlc.Queries
}

func NewUserPlanRepository(db *pgxpool.Pool) UserPlanRepository {
	return &userPlanRepository{q: sqlc.New(db)}
}

func (r *userPlanRepository) Create(ctx context.Context, userID uuid.UUID, planCode string) (*model.UserPlan, error) {
	now := time.Now()
	row, err := r.q.CreateUserPlan(ctx, sqlc.CreateUserPlanParams{
		UserID:    userID,
		PlanCode:  planCode,
		StartedAt: pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: now, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return userPlanFromRow(row), nil
}

func (r *userPlanRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.UserPlan, error) {
	row, err := r.q.GetUserPlan(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return userPlanFromRow(row), nil
}

func (r *userPlanRepository) Update(ctx context.Context, userID uuid.UUID, planCode string) (*model.UserPlan, error) {
	row, err := r.q.UpdateUserPlan(ctx, sqlc.UpdateUserPlanParams{
		UserID:   userID,
		PlanCode: planCode,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return userPlanFromRow(row), nil
}

func userPlanFromRow(row sqlc.UserPlan) *model.UserPlan {
	return &model.UserPlan{
		ID:        row.ID,
		UserID:    row.UserID,
		PlanCode:  row.PlanCode,
		StartedAt: row.StartedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
