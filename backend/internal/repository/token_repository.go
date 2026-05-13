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

var ErrTokenNotFound = errors.New("token not found or expired")

type TokenRepository interface {
	Create(ctx context.Context, tok *model.Token) error
	GetUsableByHash(ctx context.Context, hash, purpose string) (*model.Token, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	InvalidateByPurpose(ctx context.Context, userID uuid.UUID, purpose string) error
}

type tokenRepository struct {
	q *sqlc.Queries
}

func NewTokenRepository(db *pgxpool.Pool) TokenRepository {
	return &tokenRepository{q: sqlc.New(db)}
}

func (r *tokenRepository) Create(ctx context.Context, tok *model.Token) error {
	return r.q.CreateToken(ctx, sqlc.CreateTokenParams{
		ID:        tok.ID,
		UserID:    tok.UserID,
		Purpose:   sqlc.TokenPurpose(tok.Purpose),
		TokenHash: tok.TokenHash,
		ExpiresAt: pgtype.Timestamp{Time: tok.ExpiresAt, Valid: true},
		CreatedAt: pgtype.Timestamp{Time: tok.CreatedAt, Valid: true},
	})
}

func (r *tokenRepository) GetUsableByHash(ctx context.Context, hash, purpose string) (*model.Token, error) {
	row, err := r.q.GetTokenByHash(ctx, sqlc.GetTokenByHashParams{
		TokenHash: hash,
		Purpose:   sqlc.TokenPurpose(purpose),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	var usedAt *time.Time
	if row.UsedAt.Valid {
		t := row.UsedAt.Time
		usedAt = &t
	}
	return &model.Token{
		ID:        row.ID,
		UserID:    row.UserID,
		Purpose:   string(row.Purpose),
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		UsedAt:    usedAt,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *tokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkTokenUsed(ctx, id)
}

func (r *tokenRepository) InvalidateByPurpose(ctx context.Context, userID uuid.UUID, purpose string) error {
	return r.q.InvalidateUserTokensByPurpose(ctx, sqlc.InvalidateUserTokensByPurposeParams{
		UserID:  userID,
		Purpose: sqlc.TokenPurpose(purpose),
	})
}
