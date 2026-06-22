package bun

import (
	"context"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type EventRepository struct {
	db             *bun.DB
	tournamentRepo *TournamentRepository
}

func NewEventRepository(db *bun.DB, tournamentRepo *TournamentRepository) *EventRepository {
	return &EventRepository{
		db:             db,
		tournamentRepo: tournamentRepo,
	}
}

func (r *EventRepository) DB() *bun.DB { return r.db }

func (r *EventRepository) Save(ctx context.Context, e *event.Event) error {
	model := &EventModel{
		ID:         e.ID,
		Name:       e.Name,
		DivisionID: e.DivisionID,
		SkipElo:    e.SkipElo,
		StartDate:  e.StartDate,
		EndDate:    e.EndDate,
		NumTables:  e.NumTables,
	}
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

func (r *EventRepository) GetByID(ctx context.Context, id uuid.UUID) (*event.Event, error) {
	model := new(EventModel)
	err := r.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}

	tourneys, _ := r.tournamentRepo.GetByEventID(ctx, id)

	return &event.Event{
		ID:          model.ID,
		Name:        model.Name,
		DivisionID:  model.DivisionID,
		SkipElo:     model.SkipElo,
		StartDate:   model.StartDate,
		EndDate:     model.EndDate,
		NumTables:   model.NumTables,
		Tournaments: tourneys,
	}, nil
}

func (r *EventRepository) GetAll(ctx context.Context) ([]*event.Event, error) {
	var models []EventModel
	if err := r.db.NewSelect().Model(&models).Order("created_at DESC").Scan(ctx); err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, nil
	}

	// Collect all event IDs and batch-load all tournaments across all events
	eventIDs := make([]uuid.UUID, len(models))
	for i, m := range models {
		eventIDs[i] = m.ID
	}

	var allTournamentModels []TournamentModel
	_ = r.db.NewSelect().Model(&allTournamentModels).Where("event_id IN (?)", bun.In(eventIDs)).Scan(ctx)

	// Collect tournament IDs for batch loading participants and teams
	tournamentIDs := make([]uuid.UUID, len(allTournamentModels))
	for i, tm := range allTournamentModels {
		tournamentIDs[i] = tm.ID
	}

	// Batch-load participants, teams, team players, and players
	var allPartModels []TournamentParticipantModel
	if len(tournamentIDs) > 0 {
		_ = r.db.NewSelect().Model(&allPartModels).Where("tournament_id IN (?)", bun.In(tournamentIDs)).Scan(ctx)
	}

	var allTeamModels []TeamModel
	if len(tournamentIDs) > 0 {
		_ = r.db.NewSelect().Model(&allTeamModels).Where("tournament_id IN (?)", bun.In(tournamentIDs)).Order("name ASC").Scan(ctx)
	}

	teamIDs := make([]uuid.UUID, len(allTeamModels))
	for i, tm := range allTeamModels {
		teamIDs[i] = tm.ID
	}

	var allTPModels []TeamPlayerModel
	if len(teamIDs) > 0 {
		_ = r.db.NewSelect().Model(&allTPModels).Where("team_id IN (?)", bun.In(teamIDs)).Scan(ctx)
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
			Department: pm.Department,
		}
	}

	// Index everything
	partsByTournament := make(map[uuid.UUID][]TournamentParticipantModel)
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

	// Build tournaments indexed by event
	tournamentsByEvent := make(map[uuid.UUID][]*tournament.Tournament)
	for _, m := range allTournamentModels {
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

		eid := uuid.Nil
		if m.EventID != nil {
			eid = *m.EventID
		}
		tournamentsByEvent[eid] = append(tournamentsByEvent[eid], &tournament.Tournament{
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
		})
	}

	events := make([]*event.Event, len(models))
	for i, m := range models {
		events[i] = &event.Event{
			ID:          m.ID,
			Name:        m.Name,
			DivisionID:  m.DivisionID,
			SkipElo:     m.SkipElo,
			StartDate:   m.StartDate,
			EndDate:     m.EndDate,
			NumTables:   m.NumTables,
			Tournaments: tournamentsByEvent[m.ID],
		}
	}
	return events, nil
}

func (r *EventRepository) Update(ctx context.Context, e *event.Event) error {
	model := &EventModel{
		ID:         e.ID,
		Name:       e.Name,
		DivisionID: e.DivisionID,
		SkipElo:    e.SkipElo,
		StartDate:  e.StartDate,
		EndDate:    e.EndDate,
		NumTables:  e.NumTables,
	}
	_, err := r.db.NewUpdate().Model(model).WherePK().Column("name", "division_id", "skip_elo", "start_date", "end_date", "num_tables").Exec(ctx)
	return err
}

func (r *EventRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Find all tournaments belonging to this event
	var tournamentIDs []uuid.UUID
	err = tx.NewSelect().
		Model((*TournamentModel)(nil)).
		Column("id").
		Where("event_id = ?", id).
		Scan(ctx, &tournamentIDs)
	if err != nil {
		return err
	}

	// Cascade-delete all tournament dependents in batch
	if len(tournamentIDs) > 0 {
		tx.NewDelete().TableExpr("match_sets").Where("match_id IN (SELECT id FROM matches WHERE tournament_id IN (?))", bun.In(tournamentIDs)).Exec(ctx)
		// Clear self-referencing FKs before deleting matches
		tx.NewUpdate().TableExpr("matches").Set("next_match_id = NULL, team_match_id = NULL").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("matches").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("tournament_stage_rules").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id IN (?))", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("groups").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("team_players").Where("team_id IN (SELECT id FROM teams WHERE tournament_id IN (?))", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("teams").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
	}

	// Delete the tournaments themselves
	if len(tournamentIDs) > 0 {
		_, err = tx.NewDelete().Model((*TournamentModel)(nil)).Where("event_id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Delete the event
	_, err = tx.NewDelete().Model(&EventModel{}).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *EventRepository) DeleteEvents(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tournamentIDs []uuid.UUID
	err = tx.NewSelect().
		Model((*TournamentModel)(nil)).
		Column("id").
		Where("event_id IN (?)", bun.In(ids)).
		Scan(ctx, &tournamentIDs)
	if err != nil {
		return err
	}

	if len(tournamentIDs) > 0 {
		tx.NewDelete().TableExpr("match_sets").Where("match_id IN (SELECT id FROM matches WHERE tournament_id IN (?))", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewUpdate().TableExpr("matches").Set("next_match_id = NULL, team_match_id = NULL").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("matches").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("tournament_stage_rules").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id IN (?))", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("groups").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("team_players").Where("team_id IN (SELECT id FROM teams WHERE tournament_id IN (?))", bun.In(tournamentIDs)).Exec(ctx)
		tx.NewDelete().TableExpr("teams").Where("tournament_id IN (?)", bun.In(tournamentIDs)).Exec(ctx)
	}

	if len(tournamentIDs) > 0 {
		_, err = tx.NewDelete().Model((*TournamentModel)(nil)).Where("event_id IN (?)", bun.In(ids)).Exec(ctx)
		if err != nil {
			return err
		}
	}

	_, err = tx.NewDelete().Model(&EventModel{}).Where("id IN (?)", bun.In(ids)).Exec(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}

