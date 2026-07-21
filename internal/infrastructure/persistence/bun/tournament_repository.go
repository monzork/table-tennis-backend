package bun

import (
	"context"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TournamentRepository struct {
	db             *bun.DB
	tournamentRepo *EventRepository
}

func NewTournamentRepository(db *bun.DB, tournamentRepo *EventRepository) *TournamentRepository {
	return &TournamentRepository{
		db:             db,
		tournamentRepo: tournamentRepo,
	}
}

func (r *TournamentRepository) DB() *bun.DB { return r.db }

func (r *TournamentRepository) Save(ctx context.Context, e *tournament.Tournament) error {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return err
	}

	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {

		model := &TournamentModel{
			ID:          id,
			Name:        e.Name,
			DivisionIDs: e.DivisionIDs,
			SkipElo:     e.SkipElo,
			StartDate:   e.StartDate,
			EndDate:     e.EndDate,
			NumTables:   e.NumTables,
		}

		_, err := tx.NewInsert().Model(model).Exec(ctx)
		if err != nil {
			return err
		}

		for _, t := range e.Events {
			if err := r.tournamentRepo.SaveTx(ctx, tx, t); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *TournamentRepository) GetByID(ctx context.Context, idStr string) (*tournament.Tournament, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	model := new(TournamentModel)
	err = ExtractDB(ctx, r.db).NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}

	tourneys, _ := r.tournamentRepo.GetByEventID(ctx, id, false)

	return &tournament.Tournament{
		ID:          model.ID.String(),
		Name:        model.Name,
		DivisionIDs: model.DivisionIDs,
		SkipElo:     model.SkipElo,
		StartDate:   model.StartDate,
		EndDate:     model.EndDate,
		NumTables:   model.NumTables,
		Events:      tourneys,
	}, nil
}

func (r *TournamentRepository) GetByIDDeep(ctx context.Context, idStr string) (*tournament.Tournament, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	model := new(TournamentModel)
	err = ExtractDB(ctx, r.db).NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}

	tourneys, _ := r.tournamentRepo.GetByEventID(ctx, id, true)

	return &tournament.Tournament{
		ID:          model.ID.String(),
		Name:        model.Name,
		DivisionIDs: model.DivisionIDs,
		SkipElo:     model.SkipElo,
		StartDate:   model.StartDate,
		EndDate:     model.EndDate,
		NumTables:   model.NumTables,
		Events:      tourneys,
	}, nil
}

func (r *TournamentRepository) GetAll(ctx context.Context) ([]*tournament.Tournament, error) {
	var models []TournamentModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&models).Order("created_at DESC").Scan(ctx); err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, nil
	}

	// Collect all tournament IDs and batch-load all events across all tournaments
	eventIDs := make([]uuid.UUID, len(models))
	for i, m := range models {
		eventIDs[i] = m.ID
	}

	var allEventModels []EventModel
	_ = ExtractDB(ctx, r.db).NewSelect().Model(&allEventModels).Where("tournament_id IN (?)", bun.List(eventIDs)).Scan(ctx)

	// Collect event IDs for batch loading participants and teams
	tournamentIDs := make([]uuid.UUID, len(allEventModels))
	for i, tm := range allEventModels {
		tournamentIDs[i] = tm.ID
	}

	// Batch-load participants, teams, team players, and players
	var allPartModels []EventParticipantModel
	if len(tournamentIDs) > 0 {
		_ = ExtractDB(ctx, r.db).NewSelect().Model(&allPartModels).Where("event_id IN (?)", bun.List(tournamentIDs)).Scan(ctx)
	}

	var allTeamModels []TeamModel
	if len(tournamentIDs) > 0 {
		_ = ExtractDB(ctx, r.db).NewSelect().Model(&allTeamModels).Where("event_id IN (?)", bun.List(tournamentIDs)).Order("name ASC").Scan(ctx)
	}

	teamIDs := make([]uuid.UUID, len(allTeamModels))
	for i, tm := range allTeamModels {
		teamIDs[i] = tm.ID
	}

	var allTPModels []TeamPlayerModel
	if len(teamIDs) > 0 {
		_ = ExtractDB(ctx, r.db).NewSelect().Model(&allTPModels).Where("team_id IN (?)", bun.List(teamIDs)).Scan(ctx)
	}

	// Collect all player IDs
	playerIDSet := make(map[uuid.UUID]bool)
	for _, pt := range allPartModels {
		playerIDSet[pt.PlayerID] = true
	}
	for _, tp := range allTPModels {
		playerIDSet[tp.PlayerID] = true
	}

	playerIDs := make([]uuid.UUID, 0, len(playerIDSet))
	for pid := range playerIDSet {
		playerIDs = append(playerIDs, pid)
	}

	type playerModel = PlayerModel
	playerCache := make(map[uuid.UUID]*playerModel)
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
			Department:     pm.Department,
		}
	}

	// Index everything
	partsByTournament := make(map[uuid.UUID][]EventParticipantModel)
	for _, pt := range allPartModels {
		partsByTournament[pt.TournamentID] = append(partsByTournament[pt.TournamentID], pt)
	}
	teamsByTournament := make(map[uuid.UUID][]TeamModel)
	for _, tm := range allTeamModels {
		teamsByTournament[tm.TournamentID] = append(teamsByTournament[tm.TournamentID], tm)
	}
	tpByTeam := make(map[uuid.UUID][]TeamPlayerModel)
	for _, tp := range allTPModels {
		tpByTeam[tp.TeamID] = append(tpByTeam[tp.TeamID], tp)
	}

	// Build events indexed by tournament
	tournamentsByEvent := make(map[string][]*event.Event)
	for _, m := range allEventModels {
		var participantPlayers []*player.Player
		for _, pt := range partsByTournament[m.ID] {
			if pm, ok := playerCache[pt.PlayerID]; ok {
				participantPlayers = append(participantPlayers, toPlayer(pm))
			}
		}

		var teams []*event.Team
		for _, tm := range teamsByTournament[m.ID] {
			var teamPlayers []*player.Player
			for _, tp := range tpByTeam[tm.ID] {
				if pm, ok := playerCache[tp.PlayerID]; ok {
					teamPlayers = append(teamPlayers, toPlayer(pm))
				}
			}
			teams = append(teams, &event.Team{
				ID:           tm.ID.String(),
				TournamentID: tm.TournamentID.String(),
				Name:         tm.Name,
				Players:      teamPlayers,
			})
		}

		eidStr := ""
		if m.EventID != nil {
			eidStr = m.EventID.String()
		}

		var eventIDPtr *string
		if m.EventID != nil {
			s := m.EventID.String()
			eventIDPtr = &s
		}

		tournamentsByEvent[eidStr] = append(tournamentsByEvent[eidStr], &event.Event{
			ID:               m.ID.String(),
			Name:             m.Name,
			Type:             m.Type,
			Format:           m.Format,
			Status:           m.Status,
			EventCategory:    m.EventCategory,
			StartDate:        m.StartDate,
			EndDate:          m.EndDate,
			GroupPassCount:   m.GroupPassCount,
			RegistrationOpen: m.RegistrationOpen,
			EventID:          eventIDPtr,
			SkipElo:          m.SkipElo,
			Participants:     participantPlayers,
			Rules:            []event.Rule{},
			Matches:          []event.Match{},
			Teams:            teams,
			TeamFormat:       m.TeamFormat,
			Metrics:          m.Metrics,
		})
	}

	tournaments := make([]*tournament.Tournament, len(models))
	for i, m := range models {
		tournaments[i] = &tournament.Tournament{
			ID:          m.ID.String(),
			Name:        m.Name,
			DivisionIDs: m.DivisionIDs,
			SkipElo:     m.SkipElo,
			StartDate:   m.StartDate,
			EndDate:     m.EndDate,
			NumTables:   m.NumTables,
			Events:      tournamentsByEvent[m.ID.String()],
		}
	}
	return tournaments, nil
}

func (r *TournamentRepository) Update(ctx context.Context, e *tournament.Tournament) error {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return err
	}
	model := &TournamentModel{
		ID:          id,
		Name:        e.Name,
		DivisionIDs: e.DivisionIDs,
		SkipElo:     e.SkipElo,
		StartDate:   e.StartDate,
		EndDate:     e.EndDate,
		NumTables:   e.NumTables,
	}
	_, err = ExtractDB(ctx, r.db).NewUpdate().Model(model).WherePK().Column("name", "division_ids", "skip_elo", "start_date", "end_date", "num_tables").Exec(ctx)
	return err
}

func (r *TournamentRepository) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}

	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {

		// Find all events belonging to this tournament
		var tournamentIDs []uuid.UUID
		err := tx.NewSelect().
			Model((*EventModel)(nil)).
			Column("id").
			Where("tournament_id = ?", id).
			Scan(ctx, &tournamentIDs)
		if err != nil {
			return err
		}

		// Cascade-delete all event dependents in batch
		if len(tournamentIDs) > 0 {
			tx.NewDelete().TableExpr("match_sets").Where("match_id IN (SELECT id FROM matches WHERE event_id IN (?))", bun.List(tournamentIDs)).Exec(ctx)
			// Clear self-referencing FKs before deleting matches
			tx.NewUpdate().TableExpr("matches").Set("next_match_id = NULL, team_match_id = NULL").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("matches").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("event_stage_rules").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE event_id IN (?))", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("groups").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("event_participants").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("team_players").Where("team_id IN (SELECT id FROM teams WHERE event_id IN (?))", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("teams").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
		}

		// Delete the events themselves
		if len(tournamentIDs) > 0 {
			if _, err := tx.NewDelete().Model((*EventModel)(nil)).Where("tournament_id = ?", id).Exec(ctx); err != nil {
				return err
			}
		}

		// Delete the tournament
		if _, err := tx.NewDelete().Model(&TournamentModel{}).Where("id = ?", id).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}

func (r *TournamentRepository) DeleteEvents(ctx context.Context, idStrs []string) error {
	if len(idStrs) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(idStrs))
	for _, s := range idStrs {
		if uid, err := uuid.Parse(s); err == nil {
			ids = append(ids, uid)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	return RunInTx(ctx, r.db, func(ctx context.Context, tx bun.Tx) error {

		var tournamentIDs []uuid.UUID
		err := tx.NewSelect().
			Model((*EventModel)(nil)).
			Column("id").
			Where("tournament_id IN (?)", bun.List(ids)).
			Scan(ctx, &tournamentIDs)
		if err != nil {
			return err
		}

		if len(tournamentIDs) > 0 {
			tx.NewDelete().TableExpr("match_sets").Where("match_id IN (SELECT id FROM matches WHERE event_id IN (?))", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewUpdate().TableExpr("matches").Set("next_match_id = NULL, team_match_id = NULL").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("matches").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("event_stage_rules").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE event_id IN (?))", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("groups").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("event_participants").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("team_players").Where("team_id IN (SELECT id FROM teams WHERE event_id IN (?))", bun.List(tournamentIDs)).Exec(ctx)
			tx.NewDelete().TableExpr("teams").Where("event_id IN (?)", bun.List(tournamentIDs)).Exec(ctx)
		}

		if len(tournamentIDs) > 0 {
			if _, err := tx.NewDelete().Model((*EventModel)(nil)).Where("tournament_id IN (?)", bun.List(ids)).Exec(ctx); err != nil {
				return err
			}
		}

		if _, err := tx.NewDelete().Model(&TournamentModel{}).Where("id IN (?)", bun.List(ids)).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}
