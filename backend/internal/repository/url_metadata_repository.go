package repository

import (
	"context"
	"errors"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	sqlc "github.com/dauxuanhoanghung/url-shortener/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrURLMetadataNotFound = errors.New("url metadata not found")

type URLMetadataRepository interface {
	Create(ctx context.Context, urlID uuid.UUID) (*model.URLMetadata, error)
	GetByURLID(ctx context.Context, urlID uuid.UUID) (*model.URLMetadata, error)
	UpdateFetched(ctx context.Context, id uuid.UUID, result model.FetchResult) error
	ListPending(ctx context.Context, limit int) ([]*model.URLMetadata, error)
}

type urlMetadataRepository struct {
	q *sqlc.Queries
}

func NewURLMetadataRepository(db *pgxpool.Pool) URLMetadataRepository {
	return &urlMetadataRepository{q: sqlc.New(db)}
}

func (r *urlMetadataRepository) Create(ctx context.Context, urlID uuid.UUID) (*model.URLMetadata, error) {
	row, err := r.q.CreateURLMetadata(ctx, urlID)
	if err != nil {
		return nil, err
	}
	return urlMetadataFromRow(row), nil
}

func (r *urlMetadataRepository) GetByURLID(ctx context.Context, urlID uuid.UUID) (*model.URLMetadata, error) {
	row, err := r.q.GetURLMetadataByURLID(ctx, urlID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrURLMetadataNotFound
	}
	if err != nil {
		return nil, err
	}
	return urlMetadataFromRow(row), nil
}

func (r *urlMetadataRepository) UpdateFetched(ctx context.Context, id uuid.UUID, result model.FetchResult) error {
	return r.q.UpdateURLMetadataFetched(ctx, sqlc.UpdateURLMetadataFetchedParams{
		ID:          id,
		Title:       textOrNull(result.Title),
		Description: textOrNull(result.Description),
		OgImage:     textOrNull(result.OgImage),
		FaviconUrl:  textOrNull(result.FaviconURL),
		HttpStatus:  pgtype.Int4{Int32: result.HTTPStatus, Valid: true},
		FetchStatus: string(result.FetchStatus),
	})
}

func (r *urlMetadataRepository) ListPending(ctx context.Context, limit int) ([]*model.URLMetadata, error) {
	rows, err := r.q.ListPendingURLMetadata(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]*model.URLMetadata, 0, len(rows))
	for _, row := range rows {
		out = append(out, urlMetadataFromRow(row))
	}
	return out, nil
}

func urlMetadataFromRow(row sqlc.UrlMetadatum) *model.URLMetadata {
	m := &model.URLMetadata{
		ID:          row.ID,
		URLID:       row.UrlID,
		FetchStatus: model.FetchStatus(row.FetchStatus),
		CreatedAt:   row.CreatedAt.Time,
	}
	if row.Title.Valid {
		s := row.Title.String
		m.Title = &s
	}
	if row.Description.Valid {
		s := row.Description.String
		m.Description = &s
	}
	if row.OgImage.Valid {
		s := row.OgImage.String
		m.OgImage = &s
	}
	if row.FaviconUrl.Valid {
		s := row.FaviconUrl.String
		m.FaviconURL = &s
	}
	if row.HttpStatus.Valid {
		v := row.HttpStatus.Int32
		m.HTTPStatus = &v
	}
	if row.FetchedAt.Valid {
		t := row.FetchedAt.Time
		m.FetchedAt = &t
	}
	return m
}

func textOrNull(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
