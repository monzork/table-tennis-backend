package bun

import (
	"context"
	"table-tennis-backend/internal/domain/tournament"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MatchModel struct {
	bun.BaseModel `bun:"table:matches"`

	ID           uuid.UUID  `bun:"id,pk,type:uuid"`
	TournamentID uuid.UUID  `bun:"tournament_id,notnull"`
	PlayerAID    uuid.UUID  `bun:"player_a_id,notnull"`
	PlayerBID    uuid.UUID  `bun:"player_b_id,notnull"`
	Status       string     `bun:"status,notnull,default:'in_progress'"`
	WinnerID     *uuid.UUID `bun:"winner_id,nullzero"`
	CreatedAt    time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    *time.Time `bun:"updated_at,nullzero"`
}

type MatchRepository struct {
	db *bun.DB
}

func NewMatchRepository(db *bun.DB) *MatchRepository {
	return &MatchRepository{db: db}
}

func (r *MatchRepository) Save(ctx context.Context, m *tournament.Match) error {
	model := &MatchModel{
		ID:           m.ID,
		TournamentID: m.TournamentID,
		PlayerAID:    m.Players[0].ID,
		PlayerBID:    m.Players[1].ID,
		Status:       m.Status,
	}
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}
