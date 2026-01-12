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
	model := &TournamentModel{
		ID:        t.ID,
		Name:      t.Name,
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
	}
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
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
