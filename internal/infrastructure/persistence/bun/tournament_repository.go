package bun

import (
	"context"
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
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
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
			StartDate: m.StartDate,
			EndDate:   m.EndDate,
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
	return &tournament.Tournament{
		ID:        model.ID,
		Name:      model.Name,
		Type:      model.Type,
		StartDate: model.StartDate,
		EndDate:   model.EndDate,
		Rules:     []tournament.Rule{},
		Matches:   []tournament.Match{},
	}, nil
}

func (r *TournamentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().Model(&TournamentModel{}).Where("id = ?", id).Exec(ctx)
	return err
}
