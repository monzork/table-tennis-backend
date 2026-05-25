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
	}
	_, err = tx.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return err
	}

	// Save participants
	for _, p := range t.Participants {
		partModel := &TournamentParticipantModel{
			TournamentID: t.ID,
			PlayerID:     p.ID,
			EloBeforeSingles: &p.SinglesElo,
			EloBeforeDoubles: &p.DoublesElo,
		}
		_, err = tx.NewInsert().Model(partModel).Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Save groups
	for _, g := range t.Groups {
		groupModel := &GroupModel{
			ID:           g.ID,
			TournamentID: t.ID,
			Name:         g.Name,
		}
		_, err = tx.NewInsert().Model(groupModel).Exec(ctx)
		if err != nil {
			return err
		}

		// Save group participants
		for _, p := range g.Players {
			gpModel := &GroupParticipantModel{
				GroupID:  g.ID,
				PlayerID: p.ID,
			}
			_, err = tx.NewInsert().Model(gpModel).Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	// Save default stage rules
	if err := saveStageRules(ctx, tx, t.StageRules); err != nil {
		return err
	}

	// Save teams
	for _, team := range t.Teams {
		tmModel := &TeamModel{
			ID:           team.ID,
			TournamentID: t.ID,
			Name:         team.Name,
		}
		_, err = tx.NewInsert().Model(tmModel).Exec(ctx)
		if err != nil {
			return err
		}

		for _, p := range team.Players {
			tpModel := &TeamPlayerModel{
				TeamID:   team.ID,
				PlayerID: p.ID,
			}
			_, err = tx.NewInsert().Model(tpModel).Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
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

	// Load participants
	var partModels []TournamentParticipantModel
	_ = r.db.NewSelect().Model(&partModels).Where("tournament_id = ?", id).Scan(ctx)

	var participantPlayers []*player.Player
	for _, pt := range partModels {
		var pm PlayerModel
		if e := r.db.NewSelect().Model(&pm).Where("id = ?", pt.PlayerID).Scan(ctx); e == nil {
			participantPlayers = append(participantPlayers, &player.Player{
				ID:         pm.ID,
				FirstName:  pm.FirstName,
				LastName:   pm.LastName,
				SinglesElo: pm.SinglesElo,
				DoublesElo: pm.DoublesElo,
				Country:    pm.Country,
			})
		}
	}

	// Load groups with their players
	var groupModels []GroupModel
	_ = r.db.NewSelect().Model(&groupModels).Where("tournament_id = ?", id).Order("name ASC").Scan(ctx)

	var groups []tournament.Group
	for _, gm := range groupModels {
		var gpModels []GroupParticipantModel
		_ = r.db.NewSelect().Model(&gpModels).Where("group_id = ?", gm.ID).Order("position ASC").Scan(ctx)

		var groupPlayers []*player.Player
		for _, gp := range gpModels {
			var pm PlayerModel
			if e := r.db.NewSelect().Model(&pm).Where("id = ?", gp.PlayerID).Scan(ctx); e == nil {
				groupPlayers = append(groupPlayers, &player.Player{
					ID:         pm.ID,
					FirstName:  pm.FirstName,
					LastName:   pm.LastName,
					SinglesElo: pm.SinglesElo,
					DoublesElo: pm.DoublesElo,
					Country:    pm.Country,
				})
			}
		}
		groups = append(groups, tournament.Group{
			ID:      gm.ID,
			Name:    gm.Name,
			Players: groupPlayers,
		})
	}

	var matchModels []MatchModel
	if err := r.db.NewSelect().Model(&matchModels).Where("tournament_id = ?", id).Scan(ctx); err != nil && err != sql.ErrNoRows {
		// Just log or ignore if matches fail to load; it shouldn't fail the tournament
	}
	var matches []tournament.Match
	for _, mm := range matchModels {
		wt := ""
		if mm.WinnerTeam != nil {
			wt = *mm.WinnerTeam
		}
		
		// Load sets for this match
		var setModels []MatchSetModel
		_ = r.db.NewSelect().Model(&setModels).Where("match_id = ?", mm.ID).Order("set_number ASC").Scan(ctx)
		
		var sets []tournament.MatchSet
		for _, sm := range setModels {
			sets = append(sets, tournament.MatchSet{
				Number: sm.SetNumber,
				ScoreA: sm.ScoreA,
				ScoreB: sm.ScoreB,
			})
		}

		m := tournament.Match{
			ID:           mm.ID,
			TournamentID: mm.TournamentID,
			MatchType:    mm.MatchType,
			Status:       mm.Status,
			WinnerTeam:   wt,
			TeamA:        []*player.Player{{ID: mm.TeamAPlayer1ID}},
			TeamB:        []*player.Player{{ID: mm.TeamBPlayer1ID}},
			Sets:         sets,
			TeamMatchID:  mm.TeamMatchID,
		}
		matches = append(matches, m)
	}

	// Load teams
	var teamModels []TeamModel
	_ = r.db.NewSelect().Model(&teamModels).Where("tournament_id = ?", model.ID).Order("name ASC").Scan(ctx)

	var teams []*tournament.Team
	for _, tm := range teamModels {
		var tpModels []TeamPlayerModel
		_ = r.db.NewSelect().Model(&tpModels).Where("team_id = ?", tm.ID).Scan(ctx)

		var teamPlayers []*player.Player
		for _, tp := range tpModels {
			var pm PlayerModel
			if e := r.db.NewSelect().Model(&pm).Where("id = ?", tp.PlayerID).Scan(ctx); e == nil {
				teamPlayers = append(teamPlayers, &player.Player{
					ID:         pm.ID,
					FirstName:  pm.FirstName,
					LastName:   pm.LastName,
					SinglesElo: pm.SinglesElo,
					DoublesElo: pm.DoublesElo,
					Country:    pm.Country,
				})
			}
		}
		teams = append(teams, &tournament.Team{
			ID:           tm.ID,
			TournamentID: tm.TournamentID,
			Name:         tm.Name,
			Players:      teamPlayers,
		})
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
	}

	_, err = tx.NewUpdate().Model(model).WherePK().Column("name", "type", "format", "event_category", "status", "start_date", "end_date", "group_pass_count", "registration_open", "event_id", "skip_elo", "team_format").Exec(ctx)
	if err != nil {
		return err
	}

	// Scrub existing groups, participants, and teams
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("team_players").Where("team_id IN (SELECT id FROM teams WHERE tournament_id = ?)", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("teams").Where("tournament_id = ?", t.ID).Exec(ctx)

	// Refresh participants
	for _, p := range t.Participants {
		partModel := &TournamentParticipantModel{
			TournamentID: t.ID,
			PlayerID:     p.ID,
			EloBeforeSingles: &p.SinglesElo,
			EloBeforeDoubles: &p.DoublesElo,
		}
		_, err = tx.NewInsert().Model(partModel).Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Refresh teams
	for _, team := range t.Teams {
		tmModel := &TeamModel{
			ID:           team.ID,
			TournamentID: t.ID,
			Name:         team.Name,
		}
		_, err = tx.NewInsert().Model(tmModel).Exec(ctx)
		if err != nil {
			return err
		}

		for _, p := range team.Players {
			tpModel := &TeamPlayerModel{
				TeamID:   team.ID,
				PlayerID: p.ID,
			}
			_, err = tx.NewInsert().Model(tpModel).Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	// Refresh groups
	for _, g := range t.Groups {
		groupModel := &GroupModel{
			ID:           g.ID,
			TournamentID: t.ID,
			Name:         g.Name,
		}
		_, err = tx.NewInsert().Model(groupModel).Exec(ctx)
		if err != nil {
			return err
		}

		for idx, p := range g.Players {
			gpModel := &GroupParticipantModel{
				GroupID:  g.ID,
				PlayerID: p.ID,
				Position: idx,
			}
			_, err = tx.NewInsert().Model(gpModel).Exec(ctx)
			if err != nil {
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
	tournaments := make([]*tournament.Tournament, len(models))
	for i, m := range models {
		// Load participants
		var partModels []TournamentParticipantModel
		_ = r.db.NewSelect().Model(&partModels).Where("tournament_id = ?", m.ID).Scan(ctx)

		var participantPlayers []*player.Player
		for _, pt := range partModels {
			var pm PlayerModel
			if e := r.db.NewSelect().Model(&pm).Where("id = ?", pt.PlayerID).Scan(ctx); e == nil {
				participantPlayers = append(participantPlayers, &player.Player{
					ID:         pm.ID,
					FirstName:  pm.FirstName,
					LastName:   pm.LastName,
					SinglesElo: pm.SinglesElo,
					DoublesElo: pm.DoublesElo,
					Country:    pm.Country,
				})
			}
		}

		// Load teams
		var teamModels []TeamModel
		_ = r.db.NewSelect().Model(&teamModels).Where("tournament_id = ?", m.ID).Order("name ASC").Scan(ctx)

		var teams []*tournament.Team
		for _, tm := range teamModels {
			var tpModels []TeamPlayerModel
			_ = r.db.NewSelect().Model(&tpModels).Where("team_id = ?", tm.ID).Scan(ctx)

			var teamPlayers []*player.Player
			for _, tp := range tpModels {
				var pm PlayerModel
				if e := r.db.NewSelect().Model(&pm).Where("id = ?", tp.PlayerID).Scan(ctx); e == nil {
					teamPlayers = append(teamPlayers, &player.Player{
						ID:         pm.ID,
						FirstName:  pm.FirstName,
						LastName:   pm.LastName,
						SinglesElo: pm.SinglesElo,
						DoublesElo: pm.DoublesElo,
						Country:    pm.Country,
					})
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

