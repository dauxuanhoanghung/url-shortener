package repository

import (
	"context"
	"errors"
	"strings"

	sqlc "github.com/dauxuanhoanghung/url-shortener/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTagLimitExceeded = errors.New("tag limit exceeded")

const maxTagsPerURL = 20

type TagRepository interface {
	SetTagsForURL(ctx context.Context, urlID uuid.UUID, tags []string) error
	ListForURL(ctx context.Context, urlID uuid.UUID) ([]string, error)
	ListForURLs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID][]string, error)
	DeleteAllForURL(ctx context.Context, urlID uuid.UUID) error
}

type tagRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewTagRepository(db *pgxpool.Pool) TagRepository {
	return &tagRepository{pool: db, q: sqlc.New(db)}
}

func (r *tagRepository) SetTagsForURL(ctx context.Context, urlID uuid.UUID, tags []string) error {
	if len(tags) > maxTagsPerURL {
		return ErrTagLimitExceeded
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := r.q.WithTx(tx)

	if err := q.DeleteAllTagsForURL(ctx, urlID); err != nil {
		return err
	}

	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if err := q.UpsertURLTag(ctx, sqlc.UpsertURLTagParams{
			UrlID: urlID,
			Tag:   tag,
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *tagRepository) ListForURL(ctx context.Context, urlID uuid.UUID) ([]string, error) {
	return r.q.ListTagsForURL(ctx, urlID)
}

func (r *tagRepository) ListForURLs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	rows, err := r.q.ListTagsForURLs(ctx, urlIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]string, len(urlIDs))
	for _, row := range rows {
		result[row.UrlID] = append(result[row.UrlID], row.Tag)
	}
	return result, nil
}

func (r *tagRepository) DeleteAllForURL(ctx context.Context, urlID uuid.UUID) error {
	return r.q.DeleteAllTagsForURL(ctx, urlID)
}
