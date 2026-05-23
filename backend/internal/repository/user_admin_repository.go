package repository

import (
	"context"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	sqlc "github.com/dauxuanhoanghung/url-shortener/internal/repository/sqlc"
	"github.com/google/uuid"
)

// Admin-only queries against the users table. Kept in a separate file so
// user_repository.go stays focused on the per-user request path. Methods
// still belong to UserRepository (declared in user_repository.go) — splitting
// the file is purely for readability, not a layering boundary.

func (r *userRepository) ListForAdmin(ctx context.Context, limit, offset int32) ([]model.UserWithPlan, error) {
	rows, err := r.q.ListUsersForAdmin(ctx, sqlc.ListUsersForAdminParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]model.UserWithPlan, 0, len(rows))
	for _, row := range rows {
		out = append(out, userWithPlanFromListRow(row))
	}
	return out, nil
}

func (r *userRepository) CountForAdmin(ctx context.Context) (int64, error) {
	return r.q.CountUsersForAdmin(ctx)
}

func (r *userRepository) SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time) error {
	return r.q.SetUserDisabled(ctx, sqlc.SetUserDisabledParams{
		ID:         id,
		DisabledAt: nullableTime(disabledAt),
	})
}

func userWithPlanFromListRow(row sqlc.ListUsersForAdminRow) model.UserWithPlan {
	planCode := ""
	if row.PlanCode.Valid {
		planCode = row.PlanCode.String
	}
	return model.UserWithPlan{
		User: model.User{
			ID:              row.ID,
			Email:           row.Email,
			Role:            row.Role,
			EmailVerifiedAt: timeFromPg(row.EmailVerifiedAt),
			DisabledAt:      timeFromPg(row.DisabledAt),
			CreatedAt:       row.CreatedAt.Time,
			UpdatedAt:       row.UpdatedAt.Time,
		},
		PlanCode: planCode,
	}
}
