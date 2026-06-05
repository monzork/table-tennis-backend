package bun

import (
	"context"
	"database/sql"
	"fmt"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TournamentRepository struct {
	db *bun.DB
}

func NewTournamentRepository(db *bun.DB) *TournamentRepository {
	return &TournamentRepository{db: db}
}

func (r *TournamentRepository) DB() *bun.DB { return r.db }

func (r *TournamentRepository) Save(ctx context.Context, t *tournament.Tournament) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.saveTx(ctx, tx, t); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TournamentRepository) SaveTx(ctx context.Context, tx bun.IDB, t *tournament.Tournament) error {
	return r.saveTx(ctx, tx, t)
}

func (r *TournamentRepository) saveTx(ctx context.Context, tx bun.IDB, t *tournament.Tournament) error {
	model := &TournamentModel{
		ID:        t.ID,
		Name:      t.Name,
		Type:      t.Type,
		Format:    t.Format,
		Status:    t.Status,
		EventCategory: t.EventCategory,
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
		GroupPassCount: t.GroupPassCount,
		RegistrationOpen: t.RegistrationOpen,
		EventID:   t.EventID,
		SkipElo:   t.SkipElo,
		TeamFormat: t.TeamFormat,
		WinnerName: t.WinnerName,
	}
	if _, err := tx.NewInsert().Model(model).Exec(ctx); err != nil {
		return err
	}

	// Save participants in bulk
	if len(t.Participants) > 0 {
		partModels := make([]TournamentParticipantModel, len(t.Participants))
		for i, p := range t.Participants {
			partModels[i] = TournamentParticipantModel{
				TournamentID:     t.ID,
				PlayerID:         p.ID,
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
			groupModels[i] = GroupModel{
				ID:           g.ID,
				TournamentID: t.ID,
				Name:         g.Name,
			}
			for _, p := range g.Players {
				gpModels = append(gpModels, GroupParticipantModel{
					GroupID:  g.ID,
					PlayerID: p.ID,
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

	// Save teams and team players in bulk
	if len(t.Teams) > 0 {
		teamModels := make([]TeamModel, len(t.Teams))
		var tpModels []TeamPlayerModel
		for i, team := range t.Teams {
			teamModels[i] = TeamModel{
				ID:           team.ID,
				TournamentID: t.ID,
				Name:         team.Name,
			}
			for _, p := range team.Players {
				tpModels = append(tpModels, TeamPlayerModel{
					TeamID:   team.ID,
					PlayerID: p.ID,
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
	if err := r.db.NewSelect().Model(&models).Scan(ctx); err != nil {
		return nil, err
	}
	tournaments := make([]*tournament.Tournament, len(models))
	for i, m := range models {
		tournaments[i] = &tournament.Tournament{
			ID:        m.ID,
			Name:      m.Name,
			Type:      m.Type,
			Format:    m.Format,
			Status:    m.Status,
			EventCategory: m.EventCategory,
			StartDate: m.StartDate,
			EndDate:   m.EndDate,
			GroupPassCount: m.GroupPassCount,
			RegistrationOpen: m.RegistrationOpen,
			EventID:   m.EventID,
			SkipElo:   m.SkipElo,
			WinnerName: m.WinnerName,
			Rules:     []tournament.Rule{},
			Matches:   []tournament.Match{},
		}
	}
	return tournaments, nil
}

func (r *TournamentRepository) GetByID(ctx context.Context, id uuid.UUID) (*tournament.Tournament, error) {
	model := new(TournamentModel)
	err := r.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}

	// ── 1. Load participants ────────────────────────────────────────────────
	var partModels []TournamentParticipantModel
	_ = r.db.NewSelect().Model(&partModels).Where("tournament_id = ?", id).Scan(ctx)

	// Collect all player IDs we'll need
	playerIDSet := make(map[uuid.UUID]bool)
	for _, pt := range partModels {
		playerIDSet[pt.PlayerID] = true
	}

	// ── 2. Load groups and group participants in batch ───────────────────────
	var groupModels []GroupModel
	_ = r.db.NewSelect().Model(&groupModels).Where("tournament_id = ?", id).Order("name ASC").Scan(ctx)

	groupIDs := make([]uuid.UUID, len(groupModels))
	for i, gm := range groupModels {
		groupIDs[i] = gm.ID
	}

	var allGPModels []GroupParticipantModel
	if len(groupIDs) > 0 {
		_ = r.db.NewSelect().Model(&allGPModels).Where("group_id IN (?)", bun.In(groupIDs)).Order("group_id", "position ASC").Scan(ctx)
	}

	for _, gp := range allGPModels {
		playerIDSet[gp.PlayerID] = true
	}

	// ── 3. Load teams and team players in batch ─────────────────────────────
	var teamModels []TeamModel
	_ = r.db.NewSelect().Model(&teamModels).Where("tournament_id = ?", model.ID).Order("name ASC").Scan(ctx)

	teamIDs := make([]uuid.UUID, len(teamModels))
	teamMap := make(map[uuid.UUID]*TeamModel)
	for i, tm := range teamModels {
		teamIDs[i] = tm.ID
		tmCopy := tm
		teamMap[tm.ID] = &tmCopy
	}

	var allTPModels []TeamPlayerModel
	if len(teamIDs) > 0 {
		_ = r.db.NewSelect().Model(&allTPModels).Where("team_id IN (?)", bun.In(teamIDs)).Scan(ctx)
	}

	for _, tp := range allTPModels {
		playerIDSet[tp.PlayerID] = true
	}

	// ── 4. Batch-load ALL players we need in a single query ─────────────────
	playerIDs := make([]uuid.UUID, 0, len(playerIDSet))
	for pid := range playerIDSet {
		playerIDs = append(playerIDs, pid)
	}

	playerCache := make(map[uuid.UUID]*PlayerModel)
	if len(playerIDs) > 0 {
		var allPlayers []PlayerModel
		_ = r.db.NewSelect().Model(&allPlayers).Where("id IN (?)", bun.In(playerIDs)).Scan(ctx)
		for i := range allPlayers {
			playerCache[allPlayers[i].ID] = &allPlayers[i]
		}
	}

	// Helper to convert PlayerModel to domain player
	toPlayer := func(pm *PlayerModel) *player.Player {
		return &player.Player{
			ID:         pm.ID,
			FirstName:  pm.FirstName,
			LastName:   pm.LastName,
			Gender:     pm.Gender,
			SinglesElo: pm.SinglesElo,
			DoublesElo: pm.DoublesElo,
			Country:    pm.Country,
		}
	}

	// ── 5. Assemble participants ────────────────────────────────────────────
	var participantPlayers []*player.Player
	for _, pt := range partModels {
		if pm, ok := playerCache[pt.PlayerID]; ok {
			participantPlayers = append(participantPlayers, toPlayer(pm))
		}
	}

	// ── 6. Assemble teams ───────────────────────────────────────────────────
	// Group team players by team ID
	tpByTeam := make(map[uuid.UUID][]TeamPlayerModel)
	for _, tp := range allTPModels {
		tpByTeam[tp.TeamID] = append(tpByTeam[tp.TeamID], tp)
	}

	var teams []*tournament.Team
	for _, tm := range teamModels {
		var teamPlayers []*player.Player
		for _, tp := range tpByTeam[tm.ID] {
			if pm, ok := playerCache[tp.PlayerID]; ok {
				teamPlayers = append(teamPlayers, toPlayer(pm))
			}
		}
		teams = append(teams, &tournament.Team{
			ID:           tm.ID,
			TournamentID: tm.TournamentID,
			Name:         tm.Name,
			Players:      teamPlayers,
		})
	}

	// ── 7. Assemble groups ──────────────────────────────────────────────────
	// Group participants by group ID
	gpByGroup := make(map[uuid.UUID][]GroupParticipantModel)
	for _, gp := range allGPModels {
		gpByGroup[gp.GroupID] = append(gpByGroup[gp.GroupID], gp)
	}

	isTeamType := model.Type == "doubles" || model.Type == "mixed_doubles" || model.Type == "teams"

	var groups []tournament.Group
	for _, gm := range groupModels {
		var groupPlayers []*player.Player
		for _, gp := range gpByGroup[gm.ID] {
			if pm, ok := playerCache[gp.PlayerID]; ok {
				groupPlayers = append(groupPlayers, toPlayer(pm))
			} else if isTeamType {
				// For doubles/teams, group participants use team IDs
				if tm, ok := teamMap[gp.PlayerID]; ok {
					avgElo := int16(1000)
					tps := tpByTeam[tm.ID]
					if len(tps) > 0 {
						sum := int32(0)
						for _, tp := range tps {
							if pm, ok := playerCache[tp.PlayerID]; ok {
								if model.Type == "doubles" || model.Type == "mixed_doubles" {
									sum += int32(pm.DoublesElo)
								} else {
									sum += int32(pm.SinglesElo)
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
			ID:      gm.ID,
			Name:    gm.Name,
			Players: groupPlayers,
		})
	}

	// ── 8. Load matches and sets in batch ───────────────────────────────────
	var matchModels []MatchModel
	if err := r.db.NewSelect().Model(&matchModels).Where("tournament_id = ?", id).Scan(ctx); err != nil && err != sql.ErrNoRows {
		// Just ignore if matches fail to load
	}

	// Batch-load all match sets
	matchIDs := make([]uuid.UUID, len(matchModels))
	for i, mm := range matchModels {
		matchIDs[i] = mm.ID
	}
	var allSetModels []MatchSetModel
	if len(matchIDs) > 0 {
		_ = r.db.NewSelect().Model(&allSetModels).Where("match_id IN (?)", bun.In(matchIDs)).Order("match_id", "set_number ASC").Scan(ctx)
	}
	setsByMatch := make(map[string][]MatchSetModel)
	for _, sm := range allSetModels {
		setsByMatch[sm.MatchID] = append(setsByMatch[sm.MatchID], sm)
	}

	// For doubles/teams, build a reverse map: player ID → team ID
	playerToTeam := make(map[uuid.UUID]uuid.UUID)
	if isTeamType {
		for _, tm := range teamModels {
			for _, tp := range tpByTeam[tm.ID] {
				playerToTeam[tp.PlayerID] = tm.ID
			}
		}
	}

	var matches []tournament.Match
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
		if isTeamType && mm.TeamMatchID == nil {
			if tid, ok := playerToTeam[mm.TeamAPlayer1ID]; ok {
				teamAID = tid
			}
			if tid, ok := playerToTeam[mm.TeamBPlayer1ID]; ok {
				teamBID = tid
			}
		}

		teamAPlayer := &player.Player{ID: teamAID}
		teamBPlayer := &player.Player{ID: teamBID}
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

		m := tournament.Match{
			ID:           mm.ID,
			TournamentID: mm.TournamentID,
			MatchType:    mm.MatchType,
			Status:       mm.Status,
			WinnerTeam:   wt,
			TeamA:        []*player.Player{teamAPlayer},
			TeamB:        []*player.Player{teamBPlayer},
			Sets:         sets,
			TeamMatchID:  mm.TeamMatchID,
			Stage:        mm.Stage,
			UpdatedAt:    mm.UpdatedAt,
		}

		// For parent team matches (MatchType=teams, no TeamMatchID), compute sub-match wins
		// and store them as a single virtual set so ScoreA()/ScoreB() reflect team scores correctly.
		if mm.MatchType == "teams" && mm.TeamMatchID == nil {
			subWinsA, subWinsB := 0, 0
			for _, other := range matchModels {
				if other.TeamMatchID == nil || *other.TeamMatchID != mm.ID {
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

	return &tournament.Tournament{
		ID:           model.ID,
		Name:         model.Name,
		Status:       model.Status,
		Type:         model.Type,
		Format:       model.Format,
		EventCategory: model.EventCategory,
		StartDate:    model.StartDate,
		EndDate:      model.EndDate,
		GroupPassCount: model.GroupPassCount,
		RegistrationOpen: model.RegistrationOpen,
		EventID:      model.EventID,
		SkipElo:      model.SkipElo,
		WinnerName:   model.WinnerName,
		Participants: participantPlayers,
		Groups:       groups,
		Rules:        []tournament.Rule{},
		StageRules:   loadStageRules(ctx, r.db, model.ID),
		Matches:      matches,
		Teams:        teams,
		TeamFormat:   model.TeamFormat,
	}, nil
}

func (r *TournamentRepository) Update(ctx context.Context, t *tournament.Tournament) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	model := &TournamentModel{
		ID:        t.ID,
		Name:      t.Name,
		Type:      t.Type,
		Format:    t.Format,
		Status:    t.Status,
		EventCategory: t.EventCategory,
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
		GroupPassCount: t.GroupPassCount,
		RegistrationOpen: t.RegistrationOpen,
		EventID:   t.EventID,
		SkipElo:   t.SkipElo,
		TeamFormat: t.TeamFormat,
		WinnerName: t.WinnerName,
	}

	_, err = tx.NewUpdate().Model(model).WherePK().Column("name", "type", "format", "event_category", "status", "start_date", "end_date", "group_pass_count", "registration_open", "event_id", "skip_elo", "team_format", "winner_name").Exec(ctx)
	if err != nil {
		return err
	}

	// Scrub existing groups, participants, and teams
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("team_players").Where("team_id IN (SELECT id FROM teams WHERE tournament_id = ?)", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("teams").Where("tournament_id = ?", t.ID).Exec(ctx)

	// Refresh participants in bulk
	if len(t.Participants) > 0 {
		partModels := make([]TournamentParticipantModel, len(t.Participants))
		for i, p := range t.Participants {
			partModels[i] = TournamentParticipantModel{
				TournamentID:     t.ID,
				PlayerID:         p.ID,
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
			teamModels[i] = TeamModel{
				ID:           team.ID,
				TournamentID: t.ID,
				Name:         team.Name,
			}
			for _, p := range team.Players {
				tpModels = append(tpModels, TeamPlayerModel{
					TeamID:   team.ID,
					PlayerID: p.ID,
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
			groupModels[i] = GroupModel{
				ID:           g.ID,
				TournamentID: t.ID,
				Name:         g.Name,
			}
			for idx, p := range g.Players {
				gpModels = append(gpModels, GroupParticipantModel{
					GroupID:  g.ID,
					PlayerID: p.ID,
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

	return tx.Commit()
}

func (r *TournamentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Manual cascade since SQLite FK cascade may not be enabled
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", id).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", id).Exec(ctx)
	tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", id).Exec(ctx)
	_, err = tx.NewDelete().Model(&TournamentModel{}).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TournamentRepository) GetByEventID(ctx context.Context, eventID uuid.UUID) ([]*tournament.Tournament, error) {
	var models []TournamentModel
	if err := r.db.NewSelect().Model(&models).Where("event_id = ?", eventID).Scan(ctx); err != nil {
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

	// Batch-load all participants for all tournaments in this event
	var allPartModels []TournamentParticipantModel
	_ = r.db.NewSelect().Model(&allPartModels).Where("tournament_id IN (?)", bun.In(tournamentIDs)).Scan(ctx)

	// Batch-load all teams for all tournaments
	var allTeamModels []TeamModel
	_ = r.db.NewSelect().Model(&allTeamModels).Where("tournament_id IN (?)", bun.In(tournamentIDs)).Order("name ASC").Scan(ctx)

	teamIDs := make([]uuid.UUID, len(allTeamModels))
	for i, tm := range allTeamModels {
		teamIDs[i] = tm.ID
	}

	// Batch-load all team players
	var allTPModels []TeamPlayerModel
	if len(teamIDs) > 0 {
		_ = r.db.NewSelect().Model(&allTPModels).Where("team_id IN (?)", bun.In(teamIDs)).Scan(ctx)
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
		_ = r.db.NewSelect().Model(&allPlayers).Where("id IN (?)", bun.In(playerIDs)).Scan(ctx)
		for i := range allPlayers {
			playerCache[allPlayers[i].ID] = &allPlayers[i]
		}
	}

	toPlayer := func(pm *PlayerModel) *player.Player {
		return &player.Player{
			ID:         pm.ID,
			FirstName:  pm.FirstName,
			LastName:   pm.LastName,
			Gender:     pm.Gender,
			SinglesElo: pm.SinglesElo,
			DoublesElo: pm.DoublesElo,
			Country:    pm.Country,
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
				ID:           tm.ID,
				TournamentID: tm.TournamentID,
				Name:         tm.Name,
				Players:      teamPlayers,
			})
		}

		tournaments[i] = &tournament.Tournament{
			ID:        m.ID,
			Name:      m.Name,
			Type:      m.Type,
			Format:    m.Format,
			Status:    m.Status,
			EventCategory: m.EventCategory,
			StartDate: m.StartDate,
			EndDate:   m.EndDate,
			GroupPassCount: m.GroupPassCount,
			RegistrationOpen: m.RegistrationOpen,
			EventID:   m.EventID,
			SkipElo:   m.SkipElo,
			WinnerName: m.WinnerName,
			Participants: participantPlayers,
			Rules:     []tournament.Rule{},
			Matches:   []tournament.Match{},
			Teams:     teams,
			TeamFormat: m.TeamFormat,
		}
	}
	return tournaments, nil
}

func (r *TournamentRepository) SaveTeam(ctx context.Context, team *tournament.Team) error {
	tmModel := &TeamModel{
		ID:           team.ID,
		TournamentID: team.TournamentID,
		Name:         team.Name,
	}
	_, err := r.db.NewInsert().Model(tmModel).Exec(ctx)
	return err
}

func (r *TournamentRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().Model((*TeamModel)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *TournamentRepository) AddPlayerToTeam(ctx context.Context, teamID uuid.UUID, playerID uuid.UUID) error {
	var tm TeamModel
	if err := r.db.NewSelect().Model(&tm).Where("id = ?", teamID).Scan(ctx); err != nil {
		return err
	}

	t, err := r.GetByID(ctx, tm.TournamentID)
	if err != nil {
		return err
	}

	var pm PlayerModel
	if err := r.db.NewSelect().Model(&pm).Where("id = ?", playerID).Scan(ctx); err != nil {
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
		if team.ID == teamID {
			currentTeam = team
		}
		// Check if player is already in ANY team in this tournament
		for _, p := range team.Players {
			if p.ID == playerID {
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
	_, err = r.db.NewInsert().Model(tpModel).Exec(ctx)
	return err
}

func (r *TournamentRepository) RemovePlayerFromTeam(ctx context.Context, teamID uuid.UUID, playerID uuid.UUID) error {
	_, err := r.db.NewDelete().Model((*TeamPlayerModel)(nil)).Where("team_id = ? AND player_id = ?", teamID, playerID).Exec(ctx)
	return err
}

func (r *TournamentRepository) UpdateParticipantElo(ctx context.Context, tournamentID uuid.UUID, playerID uuid.UUID, singlesElo, doublesElo int16) error {
	_, err := r.db.NewUpdate().
		TableExpr("tournament_participants").
		Set("elo_after_singles = ?, elo_after_doubles = ?", singlesElo, doublesElo).
		Where("tournament_id = ? AND player_id = ?", tournamentID, playerID).
		Exec(ctx)
	return err
}

