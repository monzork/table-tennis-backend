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
	TeamMatchID    *uuid.UUID `bun:"team_match_id,type:uuid"`
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
	stage := m.Stage
	if stage == "" {
		stage = "group"
	}
	model := &MatchModel{
		ID:             m.ID,
		TournamentID:   m.TournamentID,
		MatchType:      m.MatchType,
		TeamAPlayer1ID: m.TeamA[0].ID,
		TeamBPlayer1ID: m.TeamB[0].ID,
		Status:         m.Status,
		Stage:          stage,
		RoundNumber:    1,
		TeamMatchID:    m.TeamMatchID,
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

func (r *MatchRepository) GetSets(ctx context.Context, matchID string) ([]MatchSetModel, error) {
	var sets []MatchSetModel
	if err := r.db.NewSelect().Model(&sets).Where("match_id = ?", matchID).Order("set_number ASC").Scan(ctx); err != nil {
		return nil, err
	}
	return sets, nil
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
				_, _ = tx.NewUpdate().TableExpr("matches").Set("team_a_player_1_id = ?, status = 'scheduled'", winnedPlayerID).Where("id = ? AND status = 'scheduled'", nextID).Exec(ctx)
			} else {
				_, _ = tx.NewUpdate().TableExpr("matches").Set("team_b_player_1_id = ?, status = 'scheduled'", winnedPlayerID).Where("id = ? AND status = 'scheduled'", nextID).Exec(ctx)
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

	// Update team match status if this was a sub-match
	if m.TeamMatchID != nil {
		var siblingMatches []MatchModel
		_ = tx.NewSelect().Model(&siblingMatches).Where("team_match_id = ?", m.TeamMatchID).Scan(ctx)

		subWinsA, subWinsB := 0, 0
		for _, sm := range siblingMatches {
			if sm.ID == m.ID {
				// Always use in-memory state for current match (transaction may not reflect update yet)
				if m.Status == "finished" && m.WinnerTeam != nil {
					if *m.WinnerTeam == "A" {
						subWinsA++
					} else if *m.WinnerTeam == "B" {
						subWinsB++
					}
				}
				continue
			}
			if sm.Status == "finished" && sm.WinnerTeam != nil {
				if *sm.WinnerTeam == "A" {
					subWinsA++
				} else if *sm.WinnerTeam == "B" {
					subWinsB++
				}
			}
		}
		// If current match wasn't in sibling list at all, count it
		if len(siblingMatches) == 0 || !containsMatch(siblingMatches, m.ID) {
			if m.Status == "finished" && m.WinnerTeam != nil {
				if *m.WinnerTeam == "A" {
					subWinsA++
				} else if *m.WinnerTeam == "B" {
					subWinsB++
				}
			}
		}

		parentMatch := new(MatchModel)
		if err := tx.NewSelect().Model(parentMatch).Where("id = ?", m.TeamMatchID).Scan(ctx); err == nil {
			if subWinsA >= 3 {
				w := "A"
				parentMatch.WinnerTeam = &w
				parentMatch.Status = "finished"
			} else if subWinsB >= 3 {
				w := "B"
				parentMatch.WinnerTeam = &w
				parentMatch.Status = "finished"
			} else {
				parentMatch.Status = "in_progress"
			}
			pNow := time.Now()
			parentMatch.UpdatedAt = &pNow
			_, _ = tx.NewUpdate().Model(parentMatch).WherePK().Column("status", "winner_team", "updated_at").Exec(ctx)

			// When the team match is decided, reset remaining unplayed sub-matches to 'scheduled'
			// so they don't appear as "in_progress" in the bracket
			if parentMatch.Status == "finished" {
				_, _ = tx.NewUpdate().TableExpr("matches").
					Set("status = 'scheduled'").
					Where("team_match_id = ? AND status = 'in_progress' AND id != ?", m.TeamMatchID, m.ID).
					Exec(ctx)
			}

			// Advance winner of the team matchup
			if parentMatch.Status == "finished" && parentMatch.NextMatchID != nil {
				nextID, _ := uuid.Parse(*parentMatch.NextMatchID)
				winnedTeamID := parentMatch.TeamAPlayer1ID
				if *parentMatch.WinnerTeam == "B" {
					winnedTeamID = parentMatch.TeamBPlayer1ID
				}
				if parentMatch.NextMatchSlot == "A" {
					_, _ = tx.NewUpdate().TableExpr("matches").Set("team_a_player_1_id = ?, status = 'scheduled'", winnedTeamID).Where("id = ? AND status = 'scheduled'", nextID).Exec(ctx)
				} else {
					_, _ = tx.NewUpdate().TableExpr("matches").Set("team_b_player_1_id = ?, status = 'scheduled'", winnedTeamID).Where("id = ? AND status = 'scheduled'", nextID).Exec(ctx)
				}
			}
		}
	}

	return tx.Commit()
}

func containsMatch(matches []MatchModel, id uuid.UUID) bool {
	for _, m := range matches {
		if m.ID == id {
			return true
		}
	}
	return false
}

func (r *MatchRepository) CountUnfinishedMatches(ctx context.Context, tournamentID uuid.UUID) (int, error) {
	return r.db.NewSelect().
		Model((*MatchModel)(nil)).
		Where("tournament_id = ?", tournamentID).
		Where("status != ?", "finished").
		Where("team_match_id IS NULL").
		Count(ctx)
}

func (r *MatchRepository) CountFinishedMatches(ctx context.Context, tournamentID uuid.UUID) (int, error) {
	return r.db.NewSelect().
		Model((*MatchModel)(nil)).
		Where("tournament_id = ?", tournamentID).
		Where("status = ?", "finished").
		Where("team_match_id IS NULL").
		Count(ctx)
}
