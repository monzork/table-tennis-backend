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

	ID             uuid.UUID  `bun:"id,pk,type:uuid"`
	TournamentID   uuid.UUID  `bun:"tournament_id,notnull"`
	MatchType      string     `bun:"match_type,notnull,default:'singles'"`
	TeamAPlayer1ID uuid.UUID  `bun:"team_a_player_1_id,notnull"`
	TeamAPlayer2ID *uuid.UUID `bun:"team_a_player_2_id"`
	TeamBPlayer1ID uuid.UUID  `bun:"team_b_player_1_id,notnull"`
	TeamBPlayer2ID *uuid.UUID `bun:"team_b_player_2_id"`
	Status         string     `bun:"status,notnull,default:'in_progress'"`
	WinnerTeam     *string    `bun:"winner_team"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      *time.Time `bun:"updated_at,nullzero"`
}

type MatchRepository struct {
	db         *bun.DB
	playerRepo *PlayerRepository
}

func NewMatchRepository(db *bun.DB, playerRepo *PlayerRepository) *MatchRepository {
	return &MatchRepository{db: db, playerRepo: playerRepo}
}

func (r *MatchRepository) Save(ctx context.Context, m *tournament.Match) error {
	model := &MatchModel{
		ID:             m.ID,
		TournamentID:   m.TournamentID,
		MatchType:      m.MatchType,
		TeamAPlayer1ID: m.TeamA[0].ID,
		TeamBPlayer1ID: m.TeamB[0].ID,
		Status:         m.Status,
	}
	
	if len(m.TeamA) == 2 {
		id := m.TeamA[1].ID
		model.TeamAPlayer2ID = &id
	}
	if len(m.TeamB) == 2 {
		id := m.TeamB[1].ID
		model.TeamBPlayer2ID = &id
	}
	
	if m.WinnerTeam != "" {
		model.WinnerTeam = &m.WinnerTeam
	}

	_, err := r.db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	return err
}
