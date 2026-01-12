package bun

import (
	"context"
	"table-tennis-backend/internal/domain/player"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PlayerRepository struct {
	db *bun.DB
}

func NewPlayerRepository(db *bun.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (r *PlayerRepository) Save(ctx context.Context, p *player.Player) error {
	model := &PlayerModel{
		ID:        p.ID,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Birthdate: p.Birthdate,
		Elo:       p.Elo,
		Country:   p.Country,
	}

	_, err := r.db.NewInsert().Model(model).Exec(ctx)

	return err
}

func (r *PlayerRepository) GetAll(ctx context.Context) ([]*player.Player, error) {
	var models []PlayerModel
	err := r.db.NewSelect().Model(&models).OrderBy("elo", bun.OrderDesc).Scan(ctx)

	if err != nil {
		return nil, err
	}

	players := make([]*player.Player, len(models))

	for i, m := range models {
		players[i] = &player.Player{
			ID:        m.ID,
			FirstName: m.FirstName,
			LastName:  m.LastName,
			Birthdate: m.Birthdate,
			Elo:       m.Elo,
			Country:   m.Country,
		}
	}
	return players, nil
}

func (r *PlayerRepository) GetById(ctx context.Context, id uuid.UUID) (*player.Player, error) {
	var model PlayerModel
	err := r.db.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx)

	if err != nil {
		return nil, err
	}

	return &player.Player{
		ID:        model.ID,
		FirstName: model.FirstName,
		LastName:  model.LastName,
		Birthdate: model.Birthdate,
		Elo:       model.Elo,
		Country:   model.Country,
	}, nil

}
