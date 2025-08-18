package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/monzork/table-tennis-backend/internal/domain/players"
	"github.com/uptrace/bun"
)

type SQLitePlayersRepository struct {
	db *bun.DB
}

func NewSQLitePlayersRepository(db *bun.DB) *SQLitePlayersRepository {
	return &SQLitePlayersRepository{db: db}
}

func (r *SQLitePlayersRepository) Create(ctx context.Context, p *players.Players) error {
	// Ensure the ID is set
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now().UTC()
	if p.Created_at.IsZero() {
		p.Created_at = now
	}
	p.Updated_at = now

	_, err := r.db.NewInsert().Model(p).Exec(ctx)
	return err
}
