package repository

import (
	"context"
	"errors"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	sqlc "github.com/dauxuanhoanghung/url-shortener/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrURLNotFound        = errors.New("url not found")
	ErrShortCodeConflict  = errors.New("short code already exists")
	pgUniqueViolationCode = "23505"
)

type URLRepository interface {
	Create(ctx context.Context, url *model.ShortURL) (*model.ShortURL, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]model.ShortURL, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.ShortURL, error)
	GetByShortCode(ctx context.Context, shortCode string) (*model.ShortURL, error)
	SoftDelete(ctx context.Context, id, userID uuid.UUID) error
	IncrementClick(ctx context.Context, shortCode string) error
}

type urlRepository struct {
	q *sqlc.Queries
}

func NewURLRepository(db *pgxpool.Pool) URLRepository {
	return &urlRepository{q: sqlc.New(db)}
}

func (r *urlRepository) Create(ctx context.Context, u *model.ShortURL) (*model.ShortURL, error) {
	row, err := r.q.CreateShortURL(ctx, sqlc.CreateShortURLParams{
		ID:          u.ID,
		UserID:      u.UserID,
		ShortCode:   u.ShortCode,
		OriginalUrl: u.OriginalURL,
		ClickCount:  pgtype.Int8{Int64: u.ClickCount, Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: u.CreatedAt, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			return nil, ErrShortCodeConflict
		}
		return nil, err
	}
	return shortURLFromRow(row), nil
}

func (r *urlRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]model.ShortURL, error) {
	rows, err := r.q.ListShortURLsByUser(ctx, sqlc.ListShortURLsByUserParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	urls := make([]model.ShortURL, 0, len(rows))
	for _, row := range rows {
		urls = append(urls, *shortURLFromRow(row))
	}
	return urls, nil
}

func (r *urlRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.q.CountShortURLsByUser(ctx, userID)
	return int(n), err
}

func (r *urlRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ShortURL, error) {
	row, err := r.q.GetShortURLByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrURLNotFound
	}
	if err != nil {
		return nil, err
	}
	return shortURLFromRow(row), nil
}

func (r *urlRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.ShortURL, error) {
	row, err := r.q.GetShortURLByShortCode(ctx, shortCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrURLNotFound
	}
	if err != nil {
		return nil, err
	}
	return shortURLFromRow(row), nil
}

func (r *urlRepository) IncrementClick(ctx context.Context, shortCode string) error {
	return r.q.IncrementShortURLClick(ctx, shortCode)
}

func (r *urlRepository) SoftDelete(ctx context.Context, id, userID uuid.UUID) error {
	n, err := r.q.SoftDeleteShortURL(ctx, sqlc.SoftDeleteShortURLParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrURLNotFound
	}
	return nil
}

func shortURLFromRow(row sqlc.ShortUrl) *model.ShortURL {
	u := &model.ShortURL{
		ID:          row.ID,
		UserID:      row.UserID,
		ShortCode:   row.ShortCode,
		OriginalURL: row.OriginalUrl,
		ClickCount:  row.ClickCount.Int64,
		CreatedAt:   row.CreatedAt.Time,
	}
	if row.LastAccessedAt.Valid {
		t := row.LastAccessedAt.Time
		u.LastAccessedAt = &t
	}
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		u.DeletedAt = &t
	}
	return u
}
