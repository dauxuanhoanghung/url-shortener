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

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	MarkEmailVerified(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
}

type userRepository struct {
	q *sqlc.Queries
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{q: sqlc.New(db)}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:              user.ID,
		Email:           user.Email,
		PasswordHash:    user.PasswordHash,
		Role:            user.Role,
		CreatedAt:       pgtype.Timestamp{Time: user.CreatedAt, Valid: true},
		UpdatedAt:       pgtype.Timestamp{Time: user.UpdatedAt, Valid: true},
		EmailVerifiedAt: nullableTime(user.EmailVerifiedAt),
	})
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID:              row.ID,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            row.Role,
		EmailVerifiedAt: timeFromPg(row.EmailVerifiedAt),
		DisabledAt:      timeFromPg(row.DisabledAt),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID:              row.ID,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            row.Role,
		EmailVerifiedAt: timeFromPg(row.EmailVerifiedAt),
		DisabledAt:      timeFromPg(row.DisabledAt),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID:              row.ID,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            row.Role,
		EmailVerifiedAt: timeFromPg(row.EmailVerifiedAt),
		DisabledAt:      timeFromPg(row.DisabledAt),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}

func (r *userRepository) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkUserEmailVerified(ctx, id)
}

func (r *userRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return r.q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	})
}

func nullableTime(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

func timeFromPg(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
