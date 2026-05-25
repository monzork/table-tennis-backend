package bun

import (
	"context"
	"table-tennis-backend/internal/domain/event"

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
		Tournaments: tourneys,
	}, nil
}

func (r *EventRepository) GetAll(ctx context.Context) ([]*event.Event, error) {
	var models []EventModel
	if err := r.db.NewSelect().Model(&models).Order("created_at DESC").Scan(ctx); err != nil {
		return nil, err
	}

	events := make([]*event.Event, len(models))
	for i, m := range models {
		tourneys, _ := r.tournamentRepo.GetByEventID(ctx, m.ID)
		events[i] = &event.Event{
			ID:          m.ID,
			Name:        m.Name,
			DivisionID:  m.DivisionID,
			SkipElo:     m.SkipElo,
			StartDate:   m.StartDate,
			EndDate:     m.EndDate,
			Tournaments: tourneys,
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
	}
	_, err := r.db.NewUpdate().Model(model).WherePK().Column("name", "division_id", "skip_elo", "start_date", "end_date").Exec(ctx)
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

	// Cascade-delete each tournament's dependents
	for _, tid := range tournamentIDs {
		tx.NewDelete().TableExpr("match_sets").Where("match_id IN (SELECT id FROM matches WHERE tournament_id = ?)", tid).Exec(ctx)
		tx.NewDelete().TableExpr("elo_snapshots").Where("tournament_id = ?", tid).Exec(ctx)
		tx.NewDelete().TableExpr("matches").Where("tournament_id = ?", tid).Exec(ctx)
		tx.NewDelete().TableExpr("stage_rules").Where("tournament_id = ?", tid).Exec(ctx)
		tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", tid).Exec(ctx)
		tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", tid).Exec(ctx)
		tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", tid).Exec(ctx)
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
