package bun

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/sync/errgroup"
)

type TournamentRepository struct {
	db *bun.DB
}

func NewTournamentRepository(db *bun.DB) *TournamentRepository {
	return &TournamentRepository{db: db}
}

func (r *TournamentRepository) DB() *bun.DB { return r.db }

// generateUniqueTournamentPIN generates a 4-digit PIN (1000-9999) using crypto/rand,
// not already in usedPINs, then adds it to the set to prevent future collisions.
func generateUniqueTournamentPIN(usedPINs map[string]bool) string {
	var b [4]byte
	for {
		_, _ = cryptorand.Read(b[:])
		pinVal := int(binary.BigEndian.Uint32(b[:]))%9000 + 1000
		pin := fmt.Sprintf("%04d", pinVal)
		if !usedPINs[pin] {
			usedPINs[pin] = true
			return pin
		}
	}
}

func (r *TournamentRepository) Save(ctx context.Context, t *tournament.Tournament) error {
	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {

	if err := r.saveTx(ctx, tx, t); err != nil {
		return err
	}
	return nil
	})
}

func (r *TournamentRepository) SaveTx(ctx context.Context, tx bun.IDB, t *tournament.Tournament) error {
	return r.saveTx(ctx, tx, t)
}

func (r *TournamentRepository) saveTx(ctx context.Context, tx bun.IDB, t *tournament.Tournament) error {
	tID, err := uuid.Parse(t.ID)
	if err != nil {
		return err
	}

	var eventIDPtr *uuid.UUID
	if t.EventID != nil {
		uid, err := uuid.Parse(*t.EventID)
		if err != nil {
			return err
		}
		eventIDPtr = &uid
	}

	model := &TournamentModel{
		ID:                 tID,
		Name:               t.Name,
		Type:               t.Type,
		Format:             t.Format,
		DivisionFormats:    t.DivisionFormats,
		DivisionGroupPassCounts: t.DivisionGroupPassCounts,
		Status:             t.Status,
		EventCategory:      t.EventCategory,
		StartDate:          t.StartDate,
		EndDate:            t.EndDate,
		GroupPassCount:     t.GroupPassCount,
		RegistrationOpen:   t.RegistrationOpen,
		EventID:            eventIDPtr,
		SkipElo:            t.SkipElo,
		TeamFormat:         t.TeamFormat,
		WinnerName:         t.WinnerName,
		NumTables:          t.NumTables,
		HasThirdPlaceMatch: t.HasThirdPlaceMatch,
		Metrics:            t.Metrics,
	}
	if _, err := tx.NewInsert().Model(model).Exec(ctx); err != nil {
		return err
	}

	// Save participants in bulk with unique PINs per tournament
	if len(t.Participants) > 0 {
		usedPINs := make(map[string]bool)
		partModels := make([]TournamentParticipantModel, len(t.Participants))
		for i, p := range t.Participants {
			pID, err := uuid.Parse(p.ID)
			if err != nil {
				return err
			}
			partModels[i] = TournamentParticipantModel{
				TournamentID:     tID,
				PlayerID:         pID,
				Pin:              generateUniqueTournamentPIN(usedPINs),
				EloBeforeSingles: &p.SinglesElo,
				EloBeforeDoubles: &p.DoublesElo,
			}
		}
		if _, err := tx.NewInsert().Model(&partModels).Exec(ctx); err != nil {
			return err
		}
	}

	// Save groups and group participants in bulk
	if len(t.Groups) > 0 {
		groupModels := make([]GroupModel, len(t.Groups))
		var gpModels []GroupParticipantModel
		for i, g := range t.Groups {
			gID, err := uuid.Parse(g.ID)
			if err != nil {
				return err
			}
			groupModels[i] = GroupModel{
				ID:           gID,
				TournamentID: tID,
				Name:         g.Name,
			}
			for _, p := range g.Players {
				pID, err := uuid.Parse(p.ID)
				if err != nil {
					return err
				}
				gpModels = append(gpModels, GroupParticipantModel{
					GroupID:  gID,
					PlayerID: pID,
				})
			}
		}
		if _, err := tx.NewInsert().Model(&groupModels).Exec(ctx); err != nil {
			return err
		}
		if len(gpModels) > 0 {
			if _, err := tx.NewInsert().Model(&gpModels).Exec(ctx); err != nil {
				return err
			}
		}
	}

	// Save default stage rules
	if err := saveStageRules(ctx, tx, t.StageRules); err != nil {
		return err
	}

	// Save division-specific rules
	if err := SaveDivisionRules(ctx, tx, t.ID, t.DivisionRules); err != nil {
		return err
	}

	// Save teams and team players in bulk
	if len(t.Teams) > 0 {
		teamModels := make([]TeamModel, len(t.Teams))
		var tpModels []TeamPlayerModel
		for i, team := range t.Teams {
			teamID, err := uuid.Parse(team.ID)
			if err != nil {
				return err
			}
			teamModels[i] = TeamModel{
				ID:           teamID,
				TournamentID: tID,
				Name:         team.Name,
			}
			for _, p := range team.Players {
				pID, err := uuid.Parse(p.ID)
				if err != nil {
					return err
				}
				tpModels = append(tpModels, TeamPlayerModel{
					TeamID:   teamID,
					PlayerID: pID,
				})
			}
		}
		if _, err := tx.NewInsert().Model(&teamModels).Exec(ctx); err != nil {
			return err
		}
		if len(tpModels) > 0 {
			if _, err := tx.NewInsert().Model(&tpModels).Exec(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *TournamentRepository) GetAll(ctx context.Context) ([]*tournament.Tournament, error) {
	var models []TournamentModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&models).Order("start_date DESC").Scan(ctx); err != nil {
		return nil, err
	}

	// Batch-load participant counts per tournament
	type countRow struct {
		TournamentID uuid.UUID `bun:"tournament_id"`
		Count        int       `bun:"count"`
	}
	var counts []countRow
	_ = ExtractDB(ctx, r.db).NewSelect().
		TableExpr("tournament_participants").
		ColumnExpr("tournament_id, COUNT(*) AS count").
		GroupExpr("tournament_id").
		Scan(ctx, &counts)

	countMap := make(map[uuid.UUID]int, len(counts))
	for _, c := range counts {
		countMap[c.TournamentID] = c.Count
	}

	tournaments := make([]*tournament.Tournament, len(models))
	for i, m := range models {
		// Build a placeholder Participants slice so len() returns the real count
		cnt := countMap[m.ID]
		participants := make([]*player.Player, cnt)
		for j := range participants {
			participants[j] = &player.Player{}
		}

		var eventIDPtr *string
		if m.EventID != nil {
			s := m.EventID.String()
			eventIDPtr = &s
		}

		tournaments[i] = &tournament.Tournament{
			ID:                 m.ID.String(),
			Name:               m.Name,
			Type:               m.Type,
			Format:             m.Format,
			DivisionFormats:    m.DivisionFormats,
			DivisionGroupPassCounts: m.DivisionGroupPassCounts,
			Status:             m.Status,
			EventCategory:      m.EventCategory,
			StartDate:          m.StartDate,
			EndDate:            m.EndDate,
			GroupPassCount:     m.GroupPassCount,
			RegistrationOpen:   m.RegistrationOpen,
			EventID:            eventIDPtr,
			SkipElo:            m.SkipElo,
			WinnerName:         m.WinnerName,
			NumTables:          m.NumTables,
			HasThirdPlaceMatch: m.HasThirdPlaceMatch,
			Metrics:            m.Metrics,
			Participants:       participants,
		}
	}
	return tournaments, nil
}

func (r *TournamentRepository) GetByID(ctx context.Context, idStr string) (*tournament.Tournament, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	model := new(TournamentModel)
	err = ExtractDB(ctx, r.db).NewSelect().
		Model(model).
		Relation("Participants", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Relation("Player")
		}).
		Relation("Groups", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("name ASC")
		}).
		Relation("Teams", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("name ASC")
		}).
		Relation("Matches", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Relation("TeamAPlayer1").
			Relation("TeamAPlayer2").
			Relation("TeamBPlayer1").
			Relation("TeamBPlayer2")
		}).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	db := ExtractDB(ctx, r.db)

	// --- Workaround for Bun nested has-many panics: Manually fetch nested has-many relations ---

	// 1. Fetch Group Participants
	if len(model.Groups) > 0 {
		var groupIDs []uuid.UUID
		groupMap := make(map[uuid.UUID]*GroupModel)
		for i := range model.Groups {
			groupIDs = append(groupIDs, model.Groups[i].ID)
			groupMap[model.Groups[i].ID] = &model.Groups[i]
		}
		var gps []GroupParticipantModel
		_ = db.NewSelect().Model(&gps).Where("group_id IN (?)", bun.In(groupIDs)).Relation("Player").OrderExpr("position ASC").Scan(ctx)
		for _, gp := range gps {
			if g, ok := groupMap[gp.GroupID]; ok {
				g.Participants = append(g.Participants, gp)
			}
		}
	}

	// 2. Fetch Team Players
	if len(model.Teams) > 0 {
		var teamIDs []uuid.UUID
		teamMap := make(map[uuid.UUID]*TeamModel)
		for i := range model.Teams {
			teamIDs = append(teamIDs, model.Teams[i].ID)
			teamMap[model.Teams[i].ID] = &model.Teams[i]
		}
		var tps []TeamPlayerModel
		_ = db.NewSelect().Model(&tps).Where("team_id IN (?)", bun.In(teamIDs)).Relation("Player").Scan(ctx)
		for _, tp := range tps {
			if t, ok := teamMap[tp.TeamID]; ok {
				t.TeamPlayers = append(t.TeamPlayers, tp)
			}
		}
	}

	// 3. Fetch Match Sets
	if len(model.Matches) > 0 {
		var matchIDs []uuid.UUID
		matchMap := make(map[uuid.UUID]*MatchModel)
		for i := range model.Matches {
			matchIDs = append(matchIDs, model.Matches[i].ID)
			matchMap[model.Matches[i].ID] = &model.Matches[i]
		}
		var sets []MatchSetModel
		_ = db.NewSelect().Model(&sets).Where("match_id IN (?)", bun.In(matchIDs)).OrderExpr("set_number ASC").Scan(ctx)
		for _, set := range sets {
			mID, _ := uuid.Parse(set.MatchID)
			if m, ok := matchMap[mID]; ok {
				m.Sets = append(m.Sets, set)
			}
		}
	}
	// -----------------------------------------------------------------------------------------

	// Helper to convert PlayerModel to domain player
	toPlayer := func(pm *PlayerModel) *player.Player {
		if pm == nil {
			return &player.Player{}
		}
		return &player.Player{
			ID:             pm.ID.String(),
			FirstName:      pm.FirstName,
			SecondName:     pm.SecondName,
			LastName:       pm.LastName,
			SecondLastName: pm.SecondLastName,
			Gender:         pm.Gender,
			SinglesElo:     pm.SinglesElo,
			DoublesElo:     pm.DoublesElo,
			Country:        pm.Country,
			Department:     pm.Department,
		}
	}

	// ── 1. Assemble participants ────────────────────────────────────────────
	var participantPlayers []*player.Player
	// Also build a snapshot Elo lookup keyed by player UUID for use in groups/teams.
	snapshotSinglesElo := make(map[uuid.UUID]int16, len(model.Participants))
	snapshotDoublesElo := make(map[uuid.UUID]int16, len(model.Participants))
	for _, pt := range model.Participants {
		if pt.Player != nil {
			p := toPlayer(pt.Player)
			if pt.EloBeforeSingles != nil {
				p.SinglesElo = *pt.EloBeforeSingles
				snapshotSinglesElo[pt.PlayerID] = *pt.EloBeforeSingles
			} else {
				snapshotSinglesElo[pt.PlayerID] = pt.Player.SinglesElo
			}
			if pt.EloBeforeDoubles != nil {
				p.DoublesElo = *pt.EloBeforeDoubles
				snapshotDoublesElo[pt.PlayerID] = *pt.EloBeforeDoubles
			} else {
				snapshotDoublesElo[pt.PlayerID] = pt.Player.DoublesElo
			}
			participantPlayers = append(participantPlayers, p)
		}
	}

	// ── 2. Assemble teams ───────────────────────────────────────────────────
	var teams []*tournament.Team
	teamMapDomain := make(map[uuid.UUID]*tournament.Team)
	for _, tm := range model.Teams {
		var teamPlayers []*player.Player
		for _, tp := range tm.TeamPlayers {
			if tp.Player != nil {
				teamPlayers = append(teamPlayers, toPlayer(tp.Player))
			}
		}
		t := &tournament.Team{
			ID:           tm.ID.String(),
			TournamentID: tm.TournamentID.String(),
			Name:         tm.Name,
			Players:      teamPlayers,
		}
		teams = append(teams, t)
		teamMapDomain[tm.ID] = t
	}

	// ── 3. Assemble groups ──────────────────────────────────────────────────
	isTeamType := model.Type == "doubles" || model.Type == "mixed_doubles" || model.Type == "teams"

	var groups []tournament.Group
	for _, gm := range model.Groups {
		var groupPlayers []*player.Player
		for _, gp := range gm.Participants {
			if gp.Player != nil {
				p := toPlayer(gp.Player)
				// Use snapshot Elo for display consistency with division grouping.
				if snap, ok := snapshotSinglesElo[gp.PlayerID]; ok {
					p.SinglesElo = snap
				}
				if snap, ok := snapshotDoublesElo[gp.PlayerID]; ok {
					p.DoublesElo = snap
				}
				groupPlayers = append(groupPlayers, p)
			} else if isTeamType {
				// For doubles/teams, group participants use team IDs.
				// Avg Elo is computed from each team member's snapshot Elo.
				if tm, ok := teamMapDomain[gp.PlayerID]; ok {
					avgElo := int16(1000)
					tps := tm.Players
					if len(tps) > 0 {
						sum := int32(0)
						for _, tp := range tps {
							if model.Type == "doubles" || model.Type == "mixed_doubles" {
								if e, ok := snapshotDoublesElo[uuid.MustParse(tp.ID)]; ok {
									sum += int32(e)
								} else {
									sum += int32(tp.DoublesElo)
								}
							} else {
								if e, ok := snapshotSinglesElo[uuid.MustParse(tp.ID)]; ok {
									sum += int32(e)
								} else {
									sum += int32(tp.SinglesElo)
								}
							}
						}
						avgElo = int16(sum / int32(len(tps)))
					}
					groupPlayers = append(groupPlayers, &player.Player{
						ID:         tm.ID,
						FirstName:  tm.Name,
						LastName:   "",
						SinglesElo: avgElo,
						DoublesElo: avgElo,
					})
				}
			}
		}
		groups = append(groups, tournament.Group{
			ID:      gm.ID.String(),
			Name:    gm.Name,
			Players: groupPlayers,
		})
	}

	// ── 4. Assemble matches ─────────────────────────────────────────────────
	// For doubles/teams, build a reverse map: player ID → team ID
	playerToTeam := make(map[uuid.UUID]uuid.UUID)
	if isTeamType {
		for _, tm := range model.Teams {
			for _, tp := range tm.TeamPlayers {
				playerToTeam[tp.PlayerID] = tm.ID
			}
		}
	}

	var matches []tournament.Match
	for _, mm := range model.Matches {
		wt := ""
		if mm.WinnerTeam != nil {
			wt = *mm.WinnerTeam
		}

		var sets []tournament.MatchSet
		for _, sm := range mm.Sets {
			sets = append(sets, tournament.MatchSet{
				Number: sm.SetNumber,
				ScoreA: sm.ScoreA,
				ScoreB: sm.ScoreB,
			})
		}

		teamAID := mm.TeamAPlayer1ID
		teamBID := mm.TeamBPlayer1ID
		if isTeamType && mm.TeamMatchID == nil {
			if tid, ok := playerToTeam[mm.TeamAPlayer1ID]; ok {
				teamAID = tid
			}
			if tid, ok := playerToTeam[mm.TeamBPlayer1ID]; ok {
				teamBID = tid
			}
		}

		teamAPlayer := &player.Player{ID: teamAID.String()}
		teamBPlayer := &player.Player{ID: teamBID.String()}
		if isTeamType {
			if tm, ok := teamMapDomain[teamAID]; ok {
				teamAPlayer.FirstName = tm.Name
			} else if mm.TeamAPlayer1 != nil {
				teamAPlayer.FirstName = mm.TeamAPlayer1.FirstName
				teamAPlayer.LastName = mm.TeamAPlayer1.LastName
			}
			if tm, ok := teamMapDomain[teamBID]; ok {
				teamBPlayer.FirstName = tm.Name
			} else if mm.TeamBPlayer1 != nil {
				teamBPlayer.FirstName = mm.TeamBPlayer1.FirstName
				teamBPlayer.LastName = mm.TeamBPlayer1.LastName
			}
		} else {
			if mm.TeamAPlayer1 != nil {
				teamAPlayer.FirstName = mm.TeamAPlayer1.FirstName
				teamAPlayer.LastName = mm.TeamAPlayer1.LastName
			}
			if mm.TeamBPlayer1 != nil {
				teamBPlayer.FirstName = mm.TeamBPlayer1.FirstName
				teamBPlayer.LastName = mm.TeamBPlayer1.LastName
			}
		}

		var teamMatchIDPtr *string
		if mm.TeamMatchID != nil {
			s := mm.TeamMatchID.String()
			teamMatchIDPtr = &s
		}

		var refereeIDPtr *string
		if mm.RefereeID != nil {
			s := mm.RefereeID.String()
			refereeIDPtr = &s
		}

		m := tournament.Match{
			ID:           mm.ID.String(),
			TournamentID: mm.TournamentID.String(),
			MatchType:    mm.MatchType,
			Status:       mm.Status,
			WinnerTeam:   wt,
			TeamA:        []*player.Player{teamAPlayer},
			TeamB:        []*player.Player{teamBPlayer},
			Sets:         sets,
			TeamMatchID:  teamMatchIDPtr,
			Stage:        mm.Stage,
			DivisionID:   mm.DivisionID,
			UpdatedAt:    mm.UpdatedAt,
			RefereeID:    refereeIDPtr,
			TableNumber:  mm.TableNumber,
			Pin:          mm.Pin,
			RoundNumber:  mm.RoundNumber,
		}

		// For parent team matches (MatchType=teams, no TeamMatchID), compute sub-match wins
		// and store them as a single virtual set so ScoreA()/ScoreB() reflect team scores correctly.
		if mm.MatchType == "teams" && mm.TeamMatchID == nil {
			subWinsA, subWinsB := 0, 0
			for _, other := range model.Matches {
				if other.TeamMatchID == nil || other.TeamMatchID.String() != mm.ID.String() {
					continue
				}
				if other.Status == "finished" && other.WinnerTeam != nil {
					if *other.WinnerTeam == "A" {
						subWinsA++
					} else if *other.WinnerTeam == "B" {
						subWinsB++
					}
				}
			}
			// Inject a virtual set that encodes sub-match wins so ScoreA/B work in templates
			m.Sets = []tournament.MatchSet{{Number: 1, ScoreA: subWinsA, ScoreB: subWinsB}}
		}
		matches = append(matches, m)
	}

	var eventIDPtr *string
	if model.EventID != nil {
		s := model.EventID.String()
		eventIDPtr = &s
	}

	// ── 5. Load division rules ───────────────────────────────────────────────
	divisionRules := LoadDivisionRules(ctx, r.db, model.ID.String())

	return &tournament.Tournament{
		ID:                 model.ID.String(),
		Name:               model.Name,
		Status:             model.Status,
		Type:               model.Type,
		Format:             model.Format,
		DivisionFormats:    model.DivisionFormats,
		DivisionGroupPassCounts: model.DivisionGroupPassCounts,
		EventCategory:      model.EventCategory,
		StartDate:          model.StartDate,
		EndDate:            model.EndDate,
		GroupPassCount:     model.GroupPassCount,
		RegistrationOpen:   model.RegistrationOpen,
		EventID:            eventIDPtr,
		SkipElo:            model.SkipElo,
		WinnerName:         model.WinnerName,
		Participants:       participantPlayers,
		Groups:             groups,
		Rules:              []tournament.Rule{},
		StageRules:         loadStageRules(ctx, r.db, model.ID),
		DivisionRules:      divisionRules,
		Matches:            matches,
		Teams:              teams,
		TeamFormat:         model.TeamFormat,
		NumTables:          model.NumTables,
		HasThirdPlaceMatch: model.HasThirdPlaceMatch,
		Metrics:            model.Metrics,
	}, nil
}

func (r *TournamentRepository) Update(ctx context.Context, t *tournament.Tournament) error {
	tID, err := uuid.Parse(t.ID)
	if err != nil {
		return err
	}

	var eventIDPtr *uuid.UUID
	if t.EventID != nil {
		uid, err := uuid.Parse(*t.EventID)
		if err != nil {
			return err
		}
		eventIDPtr = &uid
	}

	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {

	model := &TournamentModel{
		ID:                 tID,
		Name:               t.Name,
		Type:               t.Type,
		Format:             t.Format,
		DivisionFormats:    t.DivisionFormats,
		DivisionGroupPassCounts: t.DivisionGroupPassCounts,
		Status:             t.Status,
		EventCategory:      t.EventCategory,
		StartDate:          t.StartDate,
		EndDate:            t.EndDate,
		GroupPassCount:     t.GroupPassCount,
		RegistrationOpen:   t.RegistrationOpen,
		EventID:            eventIDPtr,
		SkipElo:            t.SkipElo,
		TeamFormat:         t.TeamFormat,
		WinnerName:         t.WinnerName,
		NumTables:          t.NumTables,
		HasThirdPlaceMatch: t.HasThirdPlaceMatch,
		Metrics:            t.Metrics,
	}

	_, err = tx.NewUpdate().Model(model).WherePK().Column("name", "type", "format", "event_category", "status", "start_date", "end_date", "group_pass_count", "registration_open", "event_id", "skip_elo", "team_format", "winner_name", "num_tables", "has_third_place_match", "metrics").Exec(ctx)
	if err != nil {
		return err
	}

	// Load existing participant PINs BEFORE scrubbing, so we can re-assign them after re-insert
	existingPINs := make(map[string]string)
	{
		var existingParts []TournamentParticipantModel
		_ = tx.NewSelect().Model(&existingParts).Column("player_id", "pin").Where("tournament_id = ?", tID).Scan(ctx)
		for _, ep := range existingParts {
			existingPINs[ep.PlayerID.String()] = ep.Pin
		}
	}

	// Scrub existing groups, participants, and teams
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", tID).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", tID).Exec(ctx)
	tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", tID).Exec(ctx)
	tx.NewDelete().TableExpr("team_players").Where("team_id IN (SELECT id FROM teams WHERE tournament_id = ?)", tID).Exec(ctx)
	tx.NewDelete().TableExpr("teams").Where("tournament_id = ?", tID).Exec(ctx)

	// Refresh participants in bulk, preserving existing PINs and generating unique new ones
	if len(t.Participants) > 0 {
		// Seed the used-PIN set with all preserved existing PINs
		usedPINs := make(map[string]bool)
		for _, pin := range existingPINs {
			if pin != "" && pin != "0000" {
				usedPINs[pin] = true
			}
		}

		partModels := make([]TournamentParticipantModel, len(t.Participants))
		for i, p := range t.Participants {
			pID, err := uuid.Parse(p.ID)
			if err != nil {
				return err
			}
			pin := existingPINs[p.ID]
			if pin == "" || pin == "0000" {
				pin = generateUniqueTournamentPIN(usedPINs)
			}
			partModels[i] = TournamentParticipantModel{
				TournamentID:     tID,
				PlayerID:         pID,
				Pin:              pin,
				EloBeforeSingles: &p.SinglesElo,
				EloBeforeDoubles: &p.DoublesElo,
			}
		}
		if _, err = tx.NewInsert().Model(&partModels).Exec(ctx); err != nil {
			return err
		}
	}

	// Refresh teams and team players in bulk
	if len(t.Teams) > 0 {
		teamModels := make([]TeamModel, len(t.Teams))
		var tpModels []TeamPlayerModel
		for i, team := range t.Teams {
			teamID, err := uuid.Parse(team.ID)
			if err != nil {
				return err
			}
			teamModels[i] = TeamModel{
				ID:           teamID,
				TournamentID: tID,
				Name:         team.Name,
			}
			for _, p := range team.Players {
				pID, err := uuid.Parse(p.ID)
				if err != nil {
					return err
				}
				tpModels = append(tpModels, TeamPlayerModel{
					TeamID:   teamID,
					PlayerID: pID,
				})
			}
		}
		if _, err = tx.NewInsert().Model(&teamModels).Exec(ctx); err != nil {
			return err
		}
		if len(tpModels) > 0 {
			if _, err = tx.NewInsert().Model(&tpModels).Exec(ctx); err != nil {
				return err
			}
		}
	}

	// Refresh groups and group participants in bulk
	if len(t.Groups) > 0 {
		groupModels := make([]GroupModel, len(t.Groups))
		var gpModels []GroupParticipantModel
		for i, g := range t.Groups {
			gID, err := uuid.Parse(g.ID)
			if err != nil {
				return err
			}
			groupModels[i] = GroupModel{
				ID:           gID,
				TournamentID: tID,
				Name:         g.Name,
			}
			for idx, p := range g.Players {
				pID, err := uuid.Parse(p.ID)
				if err != nil {
					return err
				}
				gpModels = append(gpModels, GroupParticipantModel{
					GroupID:  gID,
					PlayerID: pID,
					Position: idx,
				})
			}
		}
		if _, err = tx.NewInsert().Model(&groupModels).Exec(ctx); err != nil {
			return err
		}
		if len(gpModels) > 0 {
			if _, err = tx.NewInsert().Model(&gpModels).Exec(ctx); err != nil {
				return err
			}
		}
	}

	// Replace stage rules if changed
	if len(t.StageRules) > 0 {
		if err := replaceStageRules(ctx, tx, t.ID, t.StageRules); err != nil {
			return err
		}
	}

	// Replace division rules if changed
	if len(t.DivisionRules) > 0 {
		if err := ReplaceDivisionRules(ctx, tx, t.ID, t.DivisionRules); err != nil {
			return err
		}
	}

	return nil
	})
}

func (r *TournamentRepository) UpdateGroups(ctx context.Context, t *tournament.Tournament) error {
	tID, err := uuid.Parse(t.ID)
	if err != nil {
		return err
	}

	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {

	// Scrub existing groups and group participants
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", tID).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", tID).Exec(ctx)

	// Refresh groups and group participants in bulk
	if len(t.Groups) > 0 {
		groupModels := make([]GroupModel, len(t.Groups))
		var gpModels []GroupParticipantModel
		for i, g := range t.Groups {
			gID, err := uuid.Parse(g.ID)
			if err != nil {
				return err
			}
			groupModels[i] = GroupModel{
				ID:           gID,
				TournamentID: tID,
				Name:         g.Name,
			}
			for idx, p := range g.Players {
				pID, err := uuid.Parse(p.ID)
				if err != nil {
					return err
				}
				gpModels = append(gpModels, GroupParticipantModel{
					GroupID:  gID,
					PlayerID: pID,
					Position: idx,
				})
			}
		}
		if _, err = tx.NewInsert().Model(&groupModels).Exec(ctx); err != nil {
			return err
		}
		if len(gpModels) > 0 {
			if _, err = tx.NewInsert().Model(&gpModels).Exec(ctx); err != nil {
				return err
			}
		}
	}

	return nil
	})
}

func (r *TournamentRepository) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {
	// Manual cascade since SQLite FK cascade may not be enabled
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", id).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", id).Exec(ctx)
	tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", id).Exec(ctx)
	_, err = tx.NewDelete().Model(&TournamentModel{}).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
	})
}

func (r *TournamentRepository) GetByEventID(ctx context.Context, eventID uuid.UUID, deep bool) ([]*tournament.Tournament, error) {
	var models []TournamentModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&models).Where("event_id = ?", eventID).Order("start_date DESC").Scan(ctx); err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, nil
	}

	// Collect all tournament IDs
	tournamentIDs := make([]uuid.UUID, len(models))
	for i, m := range models {
		tournamentIDs[i] = m.ID
	}

	// Use errgroup for concurrent loading
	eg, egCtx := errgroup.WithContext(ctx)

	var allPartModels []TournamentParticipantModel
	var allTeamModels []TeamModel
	var allTPModels []TeamPlayerModel

	var matchModels []MatchModel
	var allSetModels []MatchSetModel

	eg.Go(func() error {
		return ExtractDB(ctx, r.db).NewSelect().Model(&allPartModels).Where("tournament_id IN (?)", bun.List(tournamentIDs)).Scan(egCtx)
	})

	eg.Go(func() error {
		err := ExtractDB(ctx, r.db).NewSelect().Model(&allTeamModels).Where("tournament_id IN (?)", bun.List(tournamentIDs)).Order("name ASC").Scan(egCtx)
		if err != nil {
			return err
		}
		if len(allTeamModels) > 0 {
			teamIDs := make([]uuid.UUID, len(allTeamModels))
			for i, tm := range allTeamModels {
				teamIDs[i] = tm.ID
			}
			return ExtractDB(ctx, r.db).NewSelect().Model(&allTPModels).Where("team_id IN (?)", bun.List(teamIDs)).Scan(egCtx)
		}
		return nil
	})

	if deep {
		eg.Go(func() error {
			if len(tournamentIDs) > 0 {
				if err := ExtractDB(ctx, r.db).NewSelect().Model(&matchModels).Where("tournament_id IN (?)", bun.List(tournamentIDs)).Scan(egCtx); err != nil {
					return err
				}
				matchIDs := make([]uuid.UUID, len(matchModels))
				for i, mm := range matchModels {
					matchIDs[i] = mm.ID
				}
				if len(matchIDs) > 0 {
					return ExtractDB(ctx, r.db).NewSelect().Model(&allSetModels).Where("match_id IN (?)", bun.List(matchIDs)).Order("match_id", "set_number ASC").Scan(egCtx)
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil && err != sql.ErrNoRows {
		// Just ignore if empty, or log it.
	}

	// Collect all player IDs needed
	playerIDSet := make(map[uuid.UUID]bool)
	for _, pt := range allPartModels {
		playerIDSet[pt.PlayerID] = true
	}
	for _, tp := range allTPModels {
		playerIDSet[tp.PlayerID] = true
	}

	// Batch-load all players
	playerIDs := make([]uuid.UUID, 0, len(playerIDSet))
	for pid := range playerIDSet {
		playerIDs = append(playerIDs, pid)
	}
	playerCache := make(map[uuid.UUID]*PlayerModel)
	if len(playerIDs) > 0 {
		var allPlayers []PlayerModel
		_ = ExtractDB(ctx, r.db).NewSelect().Model(&allPlayers).Where("id IN (?)", bun.List(playerIDs)).Scan(ctx)
		for i := range allPlayers {
			playerCache[allPlayers[i].ID] = &allPlayers[i]
		}
	}

	toPlayer := func(pm *PlayerModel) *player.Player {
		return &player.Player{
			ID:             pm.ID.String(),
			FirstName:      pm.FirstName,
			SecondName:     pm.SecondName,
			LastName:       pm.LastName,
			SecondLastName: pm.SecondLastName,
			Gender:         pm.Gender,
			SinglesElo:     pm.SinglesElo,
			DoublesElo:     pm.DoublesElo,
			Country:        pm.Country,
		}
	}

	// Index participants by tournament
	partsByTournament := make(map[uuid.UUID][]TournamentParticipantModel)
	for _, pt := range allPartModels {
		partsByTournament[pt.TournamentID] = append(partsByTournament[pt.TournamentID], pt)
	}

	// Index teams by tournament and team players by team
	teamsByTournament := make(map[uuid.UUID][]TeamModel)
	for _, tm := range allTeamModels {
		teamsByTournament[tm.TournamentID] = append(teamsByTournament[tm.TournamentID], tm)
	}
	tpByTeam := make(map[uuid.UUID][]TeamPlayerModel)
	for _, tp := range allTPModels {
		tpByTeam[tp.TeamID] = append(tpByTeam[tp.TeamID], tp)
	}

	// For doubles/teams, build a reverse map: player ID → team ID
	playerToTeam := make(map[uuid.UUID]uuid.UUID)
	teamMap := make(map[uuid.UUID]*TeamModel)
	for _, tm := range allTeamModels {
		tmCopy := tm
		teamMap[tm.ID] = &tmCopy
		for _, tp := range tpByTeam[tm.ID] {
			playerToTeam[tp.PlayerID] = tm.ID
		}
	}

	matchesByTournament := make(map[uuid.UUID][]tournament.Match)
	if deep {
		setsByMatch := make(map[string][]MatchSetModel)
		for _, sm := range allSetModels {
			setsByMatch[sm.MatchID] = append(setsByMatch[sm.MatchID], sm)
		}

		for _, mm := range matchModels {
			wt := ""
			if mm.WinnerTeam != nil {
				wt = *mm.WinnerTeam
			}

			var sets []tournament.MatchSet
			for _, sm := range setsByMatch[mm.ID.String()] {
				sets = append(sets, tournament.MatchSet{
					Number: sm.SetNumber,
					ScoreA: sm.ScoreA,
					ScoreB: sm.ScoreB,
				})
			}

			teamAID := mm.TeamAPlayer1ID
			teamBID := mm.TeamBPlayer1ID
			// In events, some tournaments might be team type and some singles
			var tType string
			for _, tm := range models {
				if tm.ID == mm.TournamentID {
					tType = tm.Type
					break
				}
			}
			isTeamType := tType == "doubles" || tType == "mixed_doubles" || tType == "teams"

			if isTeamType && mm.TeamMatchID == nil {
				if tid, ok := playerToTeam[mm.TeamAPlayer1ID]; ok {
					teamAID = tid
				}
				if tid, ok := playerToTeam[mm.TeamBPlayer1ID]; ok {
					teamBID = tid
				}
			}

			teamAPlayer := &player.Player{ID: teamAID.String()}
			teamBPlayer := &player.Player{ID: teamBID.String()}
			if isTeamType {
				if tm, ok := teamMap[teamAID]; ok {
					teamAPlayer.FirstName = tm.Name
				} else if pm, ok := playerCache[teamAID]; ok {
					teamAPlayer.FirstName = pm.FirstName
					teamAPlayer.LastName = pm.LastName
				}
				if tm, ok := teamMap[teamBID]; ok {
					teamBPlayer.FirstName = tm.Name
				} else if pm, ok := playerCache[teamBID]; ok {
					teamBPlayer.FirstName = pm.FirstName
					teamBPlayer.LastName = pm.LastName
				}
			} else {
				if pm, ok := playerCache[teamAID]; ok {
					teamAPlayer.FirstName = pm.FirstName
					teamAPlayer.LastName = pm.LastName
				}
				if pm, ok := playerCache[teamBID]; ok {
					teamBPlayer.FirstName = pm.FirstName
					teamBPlayer.LastName = pm.LastName
				}
			}

			var teamMatchIDPtr *string
			if mm.TeamMatchID != nil {
				s := mm.TeamMatchID.String()
				teamMatchIDPtr = &s
			}

			var refereeIDPtr *string
			if mm.RefereeID != nil {
				s := mm.RefereeID.String()
				refereeIDPtr = &s
			}

			m := tournament.Match{
				ID:           mm.ID.String(),
				TournamentID: mm.TournamentID.String(),
				MatchType:    mm.MatchType,
				Status:       mm.Status,
				WinnerTeam:   wt,
				TeamA:        []*player.Player{teamAPlayer},
				TeamB:        []*player.Player{teamBPlayer},
				Sets:         sets,
				TeamMatchID:  teamMatchIDPtr,
				Stage:        mm.Stage,
				DivisionID:   mm.DivisionID,
				UpdatedAt:    mm.UpdatedAt,
				RefereeID:    refereeIDPtr,
				TableNumber:  mm.TableNumber,
				Pin:          mm.Pin,
				RoundNumber:  mm.RoundNumber,
			}

			// Virtual set for parent team matches
			if mm.MatchType == "teams" && mm.TeamMatchID == nil {
				subWinsA, subWinsB := 0, 0
				for _, other := range matchModels {
					if other.TeamMatchID == nil || other.TeamMatchID.String() != mm.ID.String() {
						continue
					}
					if other.Status == "finished" && other.WinnerTeam != nil {
						if *other.WinnerTeam == "A" {
							subWinsA++
						} else if *other.WinnerTeam == "B" {
							subWinsB++
						}
					}
				}
				m.Sets = []tournament.MatchSet{{Number: 1, ScoreA: subWinsA, ScoreB: subWinsB}}
			}
			matchesByTournament[mm.TournamentID] = append(matchesByTournament[mm.TournamentID], m)
		}
	}

	// Assemble tournaments
	tournaments := make([]*tournament.Tournament, len(models))
	for i, m := range models {
		var participantPlayers []*player.Player
		for _, pt := range partsByTournament[m.ID] {
			if pm, ok := playerCache[pt.PlayerID]; ok {
				participantPlayers = append(participantPlayers, toPlayer(pm))
			}
		}

		var teams []*tournament.Team
		for _, tm := range teamsByTournament[m.ID] {
			var teamPlayers []*player.Player
			for _, tp := range tpByTeam[tm.ID] {
				if pm, ok := playerCache[tp.PlayerID]; ok {
					teamPlayers = append(teamPlayers, toPlayer(pm))
				}
			}
			teams = append(teams, &tournament.Team{
				ID:           tm.ID.String(),
				TournamentID: tm.TournamentID.String(),
				Name:         tm.Name,
				Players:      teamPlayers,
			})
		}

		var eventIDPtr *string
		if m.EventID != nil {
			s := m.EventID.String()
			eventIDPtr = &s
		}

		matches := matchesByTournament[m.ID]
		if matches == nil {
			matches = []tournament.Match{}
		}

		tournaments[i] = &tournament.Tournament{
			ID:                 m.ID.String(),
			Name:               m.Name,
			Type:               m.Type,
			Format:             m.Format,
			DivisionFormats:    m.DivisionFormats,
			DivisionGroupPassCounts: m.DivisionGroupPassCounts,
			Status:             m.Status,
			EventCategory:      m.EventCategory,
			StartDate:          m.StartDate,
			EndDate:            m.EndDate,
			GroupPassCount:     m.GroupPassCount,
			RegistrationOpen:   m.RegistrationOpen,
			EventID:            eventIDPtr,
			SkipElo:            m.SkipElo,
			WinnerName:         m.WinnerName,
			Participants:       participantPlayers,
			Rules:              []tournament.Rule{},
			Matches:            matches,
			Teams:              teams,
			TeamFormat:         m.TeamFormat,
			NumTables:          m.NumTables,
			HasThirdPlaceMatch: m.HasThirdPlaceMatch,
		}
	}
	return tournaments, nil
}

func (r *TournamentRepository) SaveTeam(ctx context.Context, team *tournament.Team) error {
	tID, err := uuid.Parse(team.TournamentID)
	if err != nil {
		return err
	}
	teamID, err := uuid.Parse(team.ID)
	if err != nil {
		return err
	}

	tmModel := &TeamModel{
		ID:           teamID,
		TournamentID: tID,
		Name:         team.Name,
	}
	_, err = ExtractDB(ctx, r.db).NewInsert().Model(tmModel).Exec(ctx)
	return err
}

func (r *TournamentRepository) DeleteTeam(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	_, err = ExtractDB(ctx, r.db).NewDelete().Model((*TeamModel)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *TournamentRepository) AddPlayerToTeam(ctx context.Context, teamIDStr string, playerIDStr string) error {
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return err
	}
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		return err
	}

	var tm TeamModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&tm).Where("id = ?", teamID).Scan(ctx); err != nil {
		return err
	}

	t, err := r.GetByID(ctx, tm.TournamentID.String())
	if err != nil {
		return err
	}

	var pm PlayerModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&pm).Where("id = ?", playerID).Scan(ctx); err != nil {
		return err
	}

	if t.EventCategory == "women" && pm.Gender != "F" {
		return fmt.Errorf("Only female athletes are allowed in women's tournaments")
	}
	if t.EventCategory == "men" && pm.Gender != "M" {
		return fmt.Errorf("Only male athletes are allowed in men's tournaments")
	}

	var currentTeam *tournament.Team
	for _, team := range t.Teams {
		if team.ID == teamIDStr {
			currentTeam = team
		}
		// Check if player is already in ANY team in this tournament
		for _, p := range team.Players {
			if p.ID == playerIDStr {
				return fmt.Errorf("player is already registered in another team for this tournament")
			}
		}
	}

	if t.Type == "doubles" || t.Type == "mixed_doubles" {
		if currentTeam != nil && len(currentTeam.Players) >= 2 {
			return fmt.Errorf("doubles teams can only have a maximum of two players")
		}
	}

	if t.Type == "mixed_doubles" {
		if currentTeam != nil && len(currentTeam.Players) == 1 {
			if currentTeam.Players[0].Gender == pm.Gender {
				return fmt.Errorf("mixed doubles teams must consist of one male and one female player")
			}
		}
	}

	tpModel := &TeamPlayerModel{
		TeamID:   teamID,
		PlayerID: playerID,
	}
	_, err = ExtractDB(ctx, r.db).NewInsert().Model(tpModel).Exec(ctx)
	return err
}

func (r *TournamentRepository) RemovePlayerFromTeam(ctx context.Context, teamIDStr string, playerIDStr string) error {
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return err
	}
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		return err
	}
	_, err = ExtractDB(ctx, r.db).NewDelete().Model((*TeamPlayerModel)(nil)).Where("team_id = ? AND player_id = ?", teamID, playerID).Exec(ctx)
	return err
}

func (r *TournamentRepository) UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	_, err = ExtractDB(ctx, r.db).NewUpdate().
		TableExpr("tournament_participants").
		Set("elo_after_singles = ?, elo_after_doubles = ?", singlesElo, doublesElo).
		Where("tournament_id = ? AND player_id = ?", tID, pID).
		Exec(ctx)
	return err
}

// UpdateParticipantEloBefore corrects the Elo snapshot a participant was seeded
// with for this tournament (elo_before_singles/doubles), e.g. when the player's
// stored Elo was fixed after they were already registered.
func (r *TournamentRepository) UpdateParticipantEloBefore(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	_, err = ExtractDB(ctx, r.db).NewUpdate().
		TableExpr("tournament_participants").
		Set("elo_before_singles = ?, elo_before_doubles = ?", singlesElo, doublesElo).
		Where("tournament_id = ? AND player_id = ?", tID, pID).
		Exec(ctx)
	return err
}

func (r *TournamentRepository) UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*player.Player) error {
	if len(players) == 0 {
		return nil
	}
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return err
	}

	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {

	for _, p := range players {
		pID, err := uuid.Parse(p.ID)
		if err != nil {
			return err
		}
		_, err = tx.NewUpdate().
			TableExpr("tournament_participants").
			Set("elo_after_singles = ?, elo_after_doubles = ?", p.SinglesElo, p.DoublesElo).
			Where("tournament_id = ? AND player_id = ?", tID, pID).
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
	})
}

// AddParticipant inserts a single player into tournament_participants, e.g. to
// enroll a newly-created player into a tournament outside of tournament creation.
func (r *TournamentRepository) AddParticipant(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	model := &TournamentParticipantModel{
		TournamentID:     tID,
		PlayerID:         pID,
		Pin:              r.generateUniqueParticipantPIN(ctx, tID),
		EloBeforeSingles: &singlesElo,
		EloBeforeDoubles: &doublesElo,
	}
	_, err = ExtractDB(ctx, r.db).NewInsert().Model(model).Ignore().Exec(ctx)
	return err
}

// RemoveParticipant deletes a player from tournament_participants and any group they belong to.
func (r *TournamentRepository) RemoveParticipant(ctx context.Context, tournamentID string, playerID string) error {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {
	// Remove from group participants
	tx.NewDelete().TableExpr("group_participants").
		Where("player_id = ? AND group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", pID, tID).
		Exec(ctx)
	// Remove from tournament participants
	_, err = tx.NewDelete().TableExpr("tournament_participants").
		Where("tournament_id = ? AND player_id = ?", tID, pID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return nil
	})
}


func (r *TournamentRepository) generateUniqueParticipantPIN(ctx context.Context, tournamentID uuid.UUID) string {
	for {
		var b [4]byte
		_, _ = cryptorand.Read(b[:])
		pinVal := int(binary.BigEndian.Uint32(b[:]))%9000 + 1000
		pin := fmt.Sprintf("%04d", pinVal)
		count, err := ExtractDB(ctx, r.db).NewSelect().
			Model((*TournamentParticipantModel)(nil)).
			Where("tournament_id = ? AND pin = ?", tournamentID, pin).
			Count(ctx)
		if err == nil && count == 0 {
			return pin
		}
	}
}

func (r *TournamentRepository) GetEventNumTables(ctx context.Context, eventID string) (int, error) {
	eID, err := uuid.Parse(eventID)
	if err != nil {
		return 0, err
	}
	var eventModel EventModel
	err = ExtractDB(ctx, r.db).NewSelect().
		Model(&eventModel).
		Column("num_tables").
		Where("id = ?", eID).
		Scan(ctx)
	if err != nil {
		return 0, err
	}
	return eventModel.NumTables, nil
}

func (r *TournamentRepository) GetParticipantSnapshots(ctx context.Context, tournamentID string) ([]tournament.ParticipantSnapshot, error) {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return nil, err
	}

	var snapshots []TournamentParticipantModel
	err = ExtractDB(ctx, r.db).NewSelect().
		Model(&snapshots).
		Where("tournament_id = ?", tID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	domainSnaps := make([]tournament.ParticipantSnapshot, len(snapshots))
	for i, s := range snapshots {
		domainSnaps[i] = tournament.ParticipantSnapshot{
			PlayerID:         s.PlayerID.String(),
			Pin:              s.Pin,
			EloBeforeSingles: s.EloBeforeSingles,
			EloAfterSingles:  s.EloAfterSingles,
			EloBeforeDoubles: s.EloBeforeDoubles,
			EloAfterDoubles:  s.EloAfterDoubles,
		}
	}
	return domainSnaps, nil
}

// GetParticipantPIN returns the PIN for a specific player in a specific tournament.
func (r *TournamentRepository) GetParticipantPIN(ctx context.Context, tournamentID, playerID string) (string, error) {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return "", err
	}
	pID, err := uuid.Parse(playerID)
	if err != nil {
		return "", err
	}
	var part TournamentParticipantModel
	err = ExtractDB(ctx, r.db).NewSelect().
		Model(&part).
		Where("tournament_id = ? AND player_id = ?", tID, pID).
		Scan(ctx)
	if err != nil {
		return "", err
	}
	return part.Pin, nil
}

// GetParticipantPINsByTournament returns a map of playerID -> PIN for all participants in a tournament.
func (r *TournamentRepository) GetParticipantPINsByTournament(ctx context.Context, tournamentID string) (map[string]string, error) {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return nil, err
	}
	var parts []TournamentParticipantModel
	err = ExtractDB(ctx, r.db).NewSelect().
		Model(&parts).
		Column("player_id", "pin").
		Where("tournament_id = ?", tID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(parts))
	for _, p := range parts {
		result[p.PlayerID.String()] = p.Pin
	}
	return result, nil
}

// GetParticipantOrOfficialByPIN checks both tournament participants and officials for a matching PIN.
func (r *TournamentRepository) GetParticipantOrOfficialByPIN(ctx context.Context, tournamentID string, pin string) (string, error) {
	if pin == "" {
		return "", fmt.Errorf("empty pin")
	}

	var playerID string

	// Check participants
	err := ExtractDB(ctx, r.db).NewSelect().Table("tournament_participants").Column("player_id").
		Where("tournament_id = ? AND pin = ?", tournamentID, pin).Scan(ctx, &playerID)
	if err == nil && playerID != "" {
		return playerID, nil
	}

	// Check officials
	err = ExtractDB(ctx, r.db).NewSelect().Table("tournament_officials").Column("player_id").
		Where("tournament_id = ? AND pin = ?", tournamentID, pin).Scan(ctx, &playerID)
	if err == nil && playerID != "" {
		return playerID, nil
	}

	return "", fmt.Errorf("no participant or official found with the given PIN")
}

func (r *TournamentRepository) AddOfficial(ctx context.Context, tournamentID string, playerID string, pin string) error {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	official := &TournamentOfficialModel{
		TournamentID: tID,
		PlayerID:     pID,
		Pin:          pin,
	}
	_, err = ExtractDB(ctx, r.db).NewInsert().Model(official).On("CONFLICT (tournament_id, player_id) DO UPDATE").Set("pin = EXCLUDED.pin").Exec(ctx)
	return err
}

func (r *TournamentRepository) RemoveOfficial(ctx context.Context, tournamentID string, playerID string) error {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	_, err = ExtractDB(ctx, r.db).NewDelete().Model((*TournamentOfficialModel)(nil)).Where("tournament_id = ? AND player_id = ?", tID, pID).Exec(ctx)
	return err
}

func (r *TournamentRepository) GetOfficials(ctx context.Context, tournamentID string) ([]tournament.ParticipantSnapshot, error) {
	tID, err := uuid.Parse(tournamentID)
	if err != nil {
		return nil, err
	}
	var officials []TournamentOfficialModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&officials).Where("tournament_id = ?", tID).Scan(ctx); err != nil {
		return nil, err
	}
	var snapshots []tournament.ParticipantSnapshot
	for _, o := range officials {
		snapshots = append(snapshots, tournament.ParticipantSnapshot{
			PlayerID: o.PlayerID.String(),
			Pin:      o.Pin,
		})
	}
	return snapshots, nil
}

func (r *TournamentRepository) UpdateEventIDBulk(ctx context.Context, tournamentIDs []string, eventID string) error {
	if len(tournamentIDs) == 0 {
		return nil
	}

	var uuids []uuid.UUID
	for _, idStr := range tournamentIDs {
		if u, err := uuid.Parse(idStr); err == nil {
			uuids = append(uuids, u)
		}
	}
	if len(uuids) == 0 {
		return nil
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return err
	}

	_, err = ExtractDB(ctx, r.db).NewUpdate().
		Model((*TournamentModel)(nil)).
		Set("event_id = ?", eventUUID).
		Where("id IN (?)", bun.List(uuids)).
		Exec(ctx)

	return err
}
