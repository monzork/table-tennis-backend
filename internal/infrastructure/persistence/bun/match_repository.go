package bun

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"

	"table-tennis-backend/internal/domain/player"
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
	RefereeID      *uuid.UUID `bun:"referee_id,type:uuid"`
	TableNumber    *int       `bun:"table_number"`
	Pin            string     `bun:"pin,nullzero"`
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

func (r *MatchRepository) GenerateUniquePin(ctx context.Context) string {
	for {
		var b [4]byte
		_, _ = cryptorand.Read(b[:])
		// Generate a 4-digit PIN (1000–9999) using crypto/rand
		pinVal := int(binary.BigEndian.Uint32(b[:]))%9000 + 1000
		pinStr := fmt.Sprintf("%d", pinVal)
		count, err := r.db.NewSelect().
			Model((*MatchModel)(nil)).
			Where("pin = ?", pinStr).
			Where("status != 'finished'").
			Count(ctx)
		if err == nil && count == 0 {
			return pinStr
		}
	}
}

func (r *MatchRepository) Save(ctx context.Context, m *tournament.Match) error {
	mID, err := uuid.Parse(m.ID)
	if err != nil {
		return err
	}
	tID, err := uuid.Parse(m.TournamentID)
	if err != nil {
		return err
	}
	pA1, err := uuid.Parse(m.TeamA[0].ID)
	if err != nil {
		return err
	}
	pB1, err := uuid.Parse(m.TeamB[0].ID)
	if err != nil {
		return err
	}

	var teamMatchIDPtr *uuid.UUID
	if m.TeamMatchID != nil {
		uid, err := uuid.Parse(*m.TeamMatchID)
		if err != nil {
			return err
		}
		teamMatchIDPtr = &uid
	}

	var refereeIDPtr *uuid.UUID
	if m.RefereeID != nil {
		uid, err := uuid.Parse(*m.RefereeID)
		if err != nil {
			return err
		}
		refereeIDPtr = &uid
	}

	stage := m.Stage
	if stage == "" {
		stage = "group"
	}
	if m.Pin == "" {
		m.Pin = r.GenerateUniquePin(ctx)
	}

	model := &MatchModel{
		ID:             mID,
		TournamentID:   tID,
		MatchType:      m.MatchType,
		TeamAPlayer1ID: pA1,
		TeamBPlayer1ID: pB1,
		Status:         m.Status,
		Stage:          stage,
		RoundNumber:    1,
		TeamMatchID:    teamMatchIDPtr,
		RefereeID:      refereeIDPtr,
		TableNumber:    m.TableNumber,
		Pin:            m.Pin,
	}

	if len(m.TeamA) == 2 {
		pA2, err := uuid.Parse(m.TeamA[1].ID)
		if err != nil {
			return err
		}
		model.TeamAPlayer2ID = &pA2
	}
	if len(m.TeamB) == 2 {
		pB2, err := uuid.Parse(m.TeamB[1].ID)
		if err != nil {
			return err
		}
		model.TeamBPlayer2ID = &pB2
	}

	if m.WinnerTeam != "" {
		model.WinnerTeam = &m.WinnerTeam
	}

	_, err = r.db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("status = EXCLUDED.status, winner_team = EXCLUDED.winner_team, referee_id = EXCLUDED.referee_id, table_number = EXCLUDED.table_number, pin = EXCLUDED.pin, team_a_player_1_id = EXCLUDED.team_a_player_1_id, team_b_player_1_id = EXCLUDED.team_b_player_1_id, team_a_player_2_id = EXCLUDED.team_a_player_2_id, team_b_player_2_id = EXCLUDED.team_b_player_2_id").
		Exec(ctx)
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
func (r *MatchRepository) UpdateScore(ctx context.Context, idStr string, sets []tournament.MatchSet, stageRule tournament.StageRule) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

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

func (r *MatchRepository) CountUnfinishedMatches(ctx context.Context, tournamentID string) (int, error) {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return 0, err
	}
	return r.db.NewSelect().
		Model((*MatchModel)(nil)).
		Where("tournament_id = ?", tID).
		Where("status != ?", "finished").
		Where("team_match_id IS NULL").
		Count(ctx)
}

func (r *MatchRepository) CountFinishedMatches(ctx context.Context, tournamentID string) (int, error) {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return 0, err
	}
	return r.db.NewSelect().
		Model((*MatchModel)(nil)).
		Where("tournament_id = ?", tID).
		Where("status = ?", "finished").
		Where("team_match_id IS NULL").
		Count(ctx)
}

func (r *MatchRepository) GetAll(ctx context.Context) ([]*tournament.Match, error) {
	var models []MatchModel
	if err := r.db.NewSelect().Model(&models).Order("created_at DESC").Scan(ctx); err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, nil
	}

	// 1. Collect all unique player IDs and match IDs
	playerIDSet := make(map[uuid.UUID]bool)
	matchIDs := make([]uuid.UUID, len(models))
	for i, m := range models {
		matchIDs[i] = m.ID
		playerIDSet[m.TeamAPlayer1ID] = true
		playerIDSet[m.TeamBPlayer1ID] = true
		if m.TeamAPlayer2ID != nil {
			playerIDSet[*m.TeamAPlayer2ID] = true
		}
		if m.TeamBPlayer2ID != nil {
			playerIDSet[*m.TeamBPlayer2ID] = true
		}
	}

	// 2. Batch-load all players in a single query
	playerIDs := make([]uuid.UUID, 0, len(playerIDSet))
	for pid := range playerIDSet {
		playerIDs = append(playerIDs, pid)
	}

	playerCache := make(map[uuid.UUID]*player.Player)
	if len(playerIDs) > 0 {
		var playerModels []PlayerModel
		if err := r.db.NewSelect().Model(&playerModels).Where("id IN (?)", bun.In(playerIDs)).Scan(ctx); err == nil {
			for _, pm := range playerModels {
				playerCache[pm.ID] = &player.Player{
					ID:             pm.ID.String(),
					FirstName:      pm.FirstName,
					SecondName:     pm.SecondName,
					LastName:       pm.LastName,
					SecondLastName: pm.SecondLastName,
					Birthdate:      pm.Birthdate,
					Gender:         pm.Gender,
					SinglesElo:     pm.SinglesElo,
					DoublesElo:     pm.DoublesElo,
					Country:        pm.Country,
					Department:     pm.Department,
					WhatsAppNumber: pm.WhatsAppNumber,
					NationalID:     pm.NationalID,
				}
			}
		}
	}

	// 3. Batch-load all match sets in a single query
	var setModels []MatchSetModel
	setsByMatch := make(map[string][]tournament.MatchSet)
	if err := r.db.NewSelect().Model(&setModels).Where("match_id IN (?)", bun.In(matchIDs)).Order("set_number ASC").Scan(ctx); err == nil {
		for _, sm := range setModels {
			setsByMatch[sm.MatchID] = append(setsByMatch[sm.MatchID], tournament.MatchSet{
				Number: sm.SetNumber,
				ScoreA: sm.ScoreA,
				ScoreB: sm.ScoreB,
			})
		}
	}

	// 4. Assemble the domain matches
	matches := make([]*tournament.Match, 0, len(models))
	for _, m := range models {
		teamA := []*player.Player{}
		if p, ok := playerCache[m.TeamAPlayer1ID]; ok {
			teamA = append(teamA, p)
		}
		if m.TeamAPlayer2ID != nil {
			if p, ok := playerCache[*m.TeamAPlayer2ID]; ok {
				teamA = append(teamA, p)
			}
		}

		teamB := []*player.Player{}
		if p, ok := playerCache[m.TeamBPlayer1ID]; ok {
			teamB = append(teamB, p)
		}
		if m.TeamBPlayer2ID != nil {
			if p, ok := playerCache[*m.TeamBPlayer2ID]; ok {
				teamB = append(teamB, p)
			}
		}

		wt := ""
		if m.WinnerTeam != nil {
			wt = *m.WinnerTeam
		}

		var teamMatchIDPtr *string
		if m.TeamMatchID != nil {
			s := m.TeamMatchID.String()
			teamMatchIDPtr = &s
		}
		var refereeIDPtr *string
		if m.RefereeID != nil {
			s := m.RefereeID.String()
			refereeIDPtr = &s
		}

		matches = append(matches, &tournament.Match{
			ID:           m.ID.String(),
			TournamentID: m.TournamentID.String(),
			MatchType:    m.MatchType,
			TeamA:        teamA,
			TeamB:        teamB,
			Status:       m.Status,
			WinnerTeam:   wt,
			Sets:         setsByMatch[m.ID.String()],
			TeamMatchID:  teamMatchIDPtr,
			Stage:        m.Stage,
			UpdatedAt:    m.UpdatedAt,
			RefereeID:    refereeIDPtr,
			TableNumber:  m.TableNumber,
			Pin:          m.Pin,
		})
	}
	return matches, nil
}

func (r *MatchRepository) GetOccupiedTablesByEvent(ctx context.Context, eventID uuid.UUID) ([]int, error) {
	var tids []uuid.UUID
	err := r.db.NewSelect().
		Model((*TournamentModel)(nil)).
		Column("id").
		Where("event_id = ?", eventID).
		Scan(ctx, &tids)
	if err != nil || len(tids) == 0 {
		return nil, err
	}

	var activeMatches []MatchModel
	err = r.db.NewSelect().
		Model(&activeMatches).
		Where("status = 'in_progress' AND table_number IS NOT NULL").
		Where("tournament_id IN (?)", bun.In(tids)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	occupied := make([]int, 0, len(activeMatches))
	for _, am := range activeMatches {
		if am.TableNumber != nil {
			occupied = append(occupied, *am.TableNumber)
		}
	}
	return occupied, nil
}

func (r *MatchRepository) GetOccupiedTablesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]int, error) {
	var activeMatches []MatchModel
	err := r.db.NewSelect().
		Model(&activeMatches).
		Where("status = 'in_progress' AND table_number IS NOT NULL").
		Where("tournament_id = ?", tournamentID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	occupied := make([]int, 0, len(activeMatches))
	for _, am := range activeMatches {
		if am.TableNumber != nil {
			occupied = append(occupied, *am.TableNumber)
		}
	}
	return occupied, nil
}
