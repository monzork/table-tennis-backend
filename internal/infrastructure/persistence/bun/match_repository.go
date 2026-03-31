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
	Status         string     `bun:"status,notnull,default:'scheduled'"`
	WinnerTeam     *string    `bun:"winner_team"`
	Stage          string     `bun:"stage,notnull,default:'group'"`
	RoundNumber    int        `bun:"round_number,notnull,default:1"`
	GroupID        *string    `bun:"group_id"`
	NextMatchID    *string    `bun:"next_match_id"`
	NextMatchSlot  string     `bun:"next_match_slot,default:'A'"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      *time.Time `bun:"updated_at,nullzero"`
}

type MatchSetModel struct {
	bun.BaseModel `bun:"table:match_sets"`

	ID        string `bun:"id,pk"`
	MatchID   string `bun:"match_id,notnull"`
	SetNumber int    `bun:"set_number,notnull"`
	ScoreA    int    `bun:"score_a,notnull"`
	ScoreB    int    `bun:"score_b,notnull"`
}

type MatchRepository struct {
	db         *bun.DB
	playerRepo *PlayerRepository
}

func NewMatchRepository(db *bun.DB, playerRepo *PlayerRepository) *MatchRepository {
	return &MatchRepository{db: db, playerRepo: playerRepo}
}

func (r *MatchRepository) DB() *bun.DB { return r.db }

func (r *MatchRepository) Save(ctx context.Context, m *tournament.Match) error {
	model := &MatchModel{
		ID:             m.ID,
		TournamentID:   m.TournamentID,
		MatchType:      m.MatchType,
		TeamAPlayer1ID: m.TeamA[0].ID,
		TeamBPlayer1ID: m.TeamB[0].ID,
		Status:         m.Status,
		Stage:          "group",
		RoundNumber:    1,
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

// GetByID fetches a match model (without player resolution) for score updates.
func (r *MatchRepository) GetByID(ctx context.Context, id uuid.UUID) (*MatchModel, error) {
	m := new(MatchModel)
	if err := r.db.NewSelect().Model(m).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateScore replaces all set scores, resolves winner, persists, updates players' Elo,
// and advances the winner into the next match if configured.
func (r *MatchRepository) UpdateScore(ctx context.Context, id uuid.UUID, sets []tournament.MatchSet, stageRule *StageRuleModel) error {
	m, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Resolve winner count
	needed := (stageRule.BestOf / 2) + 1
	winsA, winsB := 0, 0
	for _, s := range sets {
		if s.ScoreA > s.ScoreB {
			winsA++
		} else if s.ScoreB > s.ScoreA {
			winsB++
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Replace sets
	tx.NewDelete().TableExpr("match_sets").Where("match_id = ?", id).Exec(ctx)
	for _, s := range sets {
		setModel := &MatchSetModel{
			ID:        uuid.New().String(),
			MatchID:   id.String(),
			SetNumber: s.Number,
			ScoreA:    s.ScoreA,
			ScoreB:    s.ScoreB,
		}
		if _, err := tx.NewInsert().Model(setModel).Exec(ctx); err != nil {
			return err
		}
	}

	// Determine if match is finished
	if winsA >= needed || winsB >= needed {
		winner := "A"
		if winsB >= needed {
			winner = "B"
		}
		m.WinnerTeam = &winner
		m.Status = "finished"



		// Advance winner to next match slot if configured
		if m.NextMatchID != nil {
			nextID, _ := uuid.Parse(*m.NextMatchID)
			winnedPlayerID := m.TeamAPlayer1ID
			if winner == "B" {
				winnedPlayerID = m.TeamBPlayer1ID
			}
			if m.NextMatchSlot == "A" {
				_, _ = tx.NewUpdate().TableExpr("matches").Set("team_a_player_1_id = ?, status = 'scheduled'", winnedPlayerID).Where("id = ?", nextID).Exec(ctx)
			} else {
				_, _ = tx.NewUpdate().TableExpr("matches").Set("team_b_player_1_id = ?, status = 'scheduled'", winnedPlayerID).Where("id = ?", nextID).Exec(ctx)
			}
		}
	} else {
		m.Status = "in_progress"
	}

	now := time.Now()
	m.UpdatedAt = &now
	_, err = tx.NewUpdate().Model(m).WherePK().Column("status", "winner_team", "updated_at").Exec(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}
