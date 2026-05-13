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
