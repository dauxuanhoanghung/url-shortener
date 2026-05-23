package repository

import (
	"context"
	"time"

	sqlc "github.com/dauxuanhoanghung/url-shortener/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminAuditEntry struct {
	ID         uuid.UUID
	ActorID    *uuid.UUID // nil for CLI-originated actions
	Action     string
	TargetType string
	TargetID   string
	Before     []byte // raw JSON; adapter does not parse
	After      []byte
	CreatedAt  time.Time
}

type AdminAuditRepository interface {
	Create(ctx context.Context, entry *AdminAuditEntry) error
	List(ctx context.Context, limit, offset int32) ([]AdminAuditEntry, error)
	Count(ctx context.Context) (int64, error)
}

type adminAuditRepository struct {
	q *sqlc.Queries
}

func NewAdminAuditRepository(db *pgxpool.Pool) AdminAuditRepository {
	return &adminAuditRepository{q: sqlc.New(db)}
}

func (r *adminAuditRepository) Create(ctx context.Context, entry *AdminAuditEntry) error {
	params := sqlc.CreateAdminAuditEntryParams{
		ID:         entry.ID,
		ActorID:    entry.ActorID,
		Action:     entry.Action,
		TargetType: pgtype.Text{String: entry.TargetType, Valid: entry.TargetType != ""},
		TargetID:   pgtype.Text{String: entry.TargetID, Valid: entry.TargetID != ""},
		Before:     entry.Before,
		After:      entry.After,
		CreatedAt:  pgtype.Timestamp{Time: entry.CreatedAt, Valid: true},
	}
	return r.q.CreateAdminAuditEntry(ctx, params)
}

func (r *adminAuditRepository) List(ctx context.Context, limit, offset int32) ([]AdminAuditEntry, error) {
	rows, err := r.q.ListAdminAuditEntries(ctx, sqlc.ListAdminAuditEntriesParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]AdminAuditEntry, 0, len(rows))
	for _, row := range rows {
		entry := AdminAuditEntry{
			ID:        row.ID,
			ActorID:   row.ActorID,
			Action:    row.Action,
			Before:    row.Before,
			After:     row.After,
			CreatedAt: row.CreatedAt.Time,
		}
		if row.TargetType.Valid {
			entry.TargetType = row.TargetType.String
		}
		if row.TargetID.Valid {
			entry.TargetID = row.TargetID.String
		}
		out = append(out, entry)
	}
	return out, nil
}

func (r *adminAuditRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountAdminAuditEntries(ctx)
}
